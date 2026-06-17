# 云盘文件预览能力设计

本文用于记录 demo-desktop 云盘文件预览的调研结论和后续实现方案。

## 背景

当前云盘文件管理入口在 `frontend/oss-manager.html`，预览窗口入口在
`frontend/oss-preview.html`，后端预览接口是 `/api/oss/files/preview`。

现有后端实现位于：

- `internal/desktopapp/routes_storage.go`
- `internal/desktopapp/oss_storage.go`
- `internal/desktopapp/oss_local.go`

当前预览模型是后端读取完整文件内容，通过 JSON 返回文本或 base64：

- `text`：直接返回字符串。
- `image`：返回 base64，前端拼成 data URL。
- `pdf`：返回 base64，前端用 `<object>` 渲染。
- 其他类型：返回 `unknown`。
- 文件大小限制：8 MB。

这个模型适合小文本、小图片和小 PDF，不适合 Office 文档、大 PDF、音视频、
代码大文件和需要流式加载的内容。

## 总体结论

采用“前端优先 + 后端转换兜底”的混合预览方案。

- 图片、PDF、代码、文本、CSV/XLSX 数据视图优先前端预览。
- Office 复杂版式、PPT、DOC、旧二进制格式优先后端转换为 PDF。
- 大文件统一改为后端流式内容接口，不再通过 JSON/base64 传输。
- 转换结果需要缓存，并且转换任务必须有超时、并发限制和临时目录隔离。

## 文件类型能力矩阵

| 类型 | 推荐预览方式 | 说明 |
| --- | --- | --- |
| 图片 `jpg/png/webp/gif/bmp/avif/svg` | 前端直出 URL/Blob | SVG 按不可信内容处理，避免脚本风险。 |
| PDF | 前端 PDF.js + 后端流式 URL | 支持翻页、缩放、搜索，比 `<object data="data:...">` 更可控。 |
| 代码/文本 `go/js/ts/json/xml/yaml/md/sql/...` | 前端高亮 | 小文件高亮，大文件降级为纯文本或分块加载。 |
| CSV/XLSX/XLS | 前端数据视图 + 后端 PDF 版式视图 | SheetJS 可做表格数据预览；复杂样式、图表和打印版式走 PDF。 |
| DOCX | 简单文档可前端，复杂文档走后端 PDF | DOCX 到 HTML 难以完全保真，前端预览只作为快速预览。 |
| DOC/RTF/ODT | 后端 LibreOffice 转 PDF | 旧 Office 二进制和 ODF 前端支持差。 |
| PPT/PPTX/ODP | 后端转 PDF | 先做静态幻灯片预览，不处理动画。 |
| 音视频 `mp3/wav/mp4/webm` | 前端 `<audio>/<video>` + Range | 后端必须支持 HTTP Range。 |
| 压缩包 `zip/tar/gz/7z` | 后端列目录，前端树形浏览 | 按需读取条目，避免直接解压全部。 |
| HEIC/TIFF/PSD/CAD/3D | 可选后端转换或外部服务 | 建议作为后续阶段能力。 |

## 前端预览方案

适合放在 `frontend/oss-preview.html` 或后续拆成模块化 JS。

### 优点

- 实现轻，适合桌面 demo。
- 离线可用，不依赖额外服务。
- 隐私好，文件不用上传第三方。
- 图片、PDF、代码、CSV/XLSX 数据视图体验好。

### 缺点

- Office 保真度有限。
- 大文件会占用浏览器内存，容易卡顿。
- DOCX/Markdown/HTML 等转换输出必须 sanitize。
- 复杂表格、图表、PPT 动画基本无法完整还原。

### 推荐前端库

- PDF：`pdf.js`
- 代码高亮：`highlight.js` 起步；如果想要 VS Code 风格高亮，可评估 `shiki`
- Excel/CSV 数据视图：`SheetJS`
- DOCX 快速预览：`mammoth` 或 `docx-preview`，只作为非保真快速预览

## 后端预览方案

适合新增在 `internal/desktopapp/oss_preview*.go`，路由仍由
`routes_storage.go` 注册。

### 优点

- 格式覆盖更广。
- Office/PPT/DOC 转 PDF 后，前端渲染路径统一。
- 可以做缓存、权限、限流、异步任务。
- 大文件可通过流式接口访问，避免 JSON/base64 膨胀。

### 缺点

- LibreOffice/OnlyOffice/kkFileView 等依赖重。
- 转换会消耗 CPU、内存和磁盘。
- 不可信文件转换存在安全风险，需要隔离和超时。
- 跨平台打包会更复杂。

### 推荐后端转换

第一选择：LibreOffice headless 转 PDF。

典型命令：

```bash
soffice --headless --convert-to pdf --outdir <output-dir> <input-file>
```

后端需要控制：

- 临时目录隔离。
- 转换超时，例如 60-180 秒。
- 并发限制，例如 1-2 个转换任务。
- 缓存清理策略。
- 文件大小上限和页数上限。
- 转换失败时返回可解释错误。

## 推荐接口设计

### 预览准备接口

保留现有路径，改造返回结构：

```http
POST /api/oss/files/preview
```

请求：

```json
{
  "storageId": "memo-local",
  "path": "docs/report.docx"
}
```

响应示例：

```json
{
  "viewer": "office-pdf",
  "name": "report.docx",
  "path": "docs/report.docx",
  "mimeType": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
  "size": 1048576,
  "sourceUrl": "/api/oss/files/content?...",
  "convertedUrl": "/api/oss/files/preview-artifact?...",
  "jobId": "preview-job-id",
  "status": "ready"
}
```

`viewer` 可选值：

- `image`
- `pdf`
- `code`
- `text`
- `spreadsheet`
- `office-pdf`
- `media`
- `archive`
- `unsupported`

### 原始内容流式接口

新增：

```http
GET /api/oss/files/content?storageId=...&path=...&token=...
```

要求：

- 支持 `Range`。
- 设置正确 `Content-Type`。
- 设置 `Content-Disposition: inline`。
- 支持 `ETag` 或 `Last-Modified`。
- 本地存储使用 `http.ServeContent`。
- S3/OSS 使用 `GetObject` 的 Range 参数透传。

### 转换产物接口

新增：

```http
GET /api/oss/files/preview-artifact?jobId=...
```

用于返回 PDF、缩略图或其他转换产物。

### 转换状态接口

新增：

```http
GET /api/oss/files/preview-status?jobId=...
```

响应：

```json
{
  "status": "pending|running|ready|failed",
  "progress": 0,
  "message": ""
}
```

## 前端渲染设计

`oss-preview.html` 内部建立 viewer registry：

```js
const renderers = {
  image: renderImagePreview,
  pdf: renderPDFPreview,
  code: renderCodePreview,
  text: renderTextPreview,
  spreadsheet: renderSpreadsheetPreview,
  "office-pdf": renderOfficePDFPreview,
  media: renderMediaPreview,
  archive: renderArchivePreview,
  unsupported: renderUnsupportedPreview,
};
```

渲染策略：

- `image`：使用 `<img src="sourceUrl">`。
- `pdf`：使用 PDF.js 加载 `sourceUrl` 或 `convertedUrl`。
- `code`：fetch 文本，按扩展名选择语言并高亮。
- `text`：fetch 文本，纯文本渲染。
- `spreadsheet`：fetch ArrayBuffer，SheetJS 解析 sheet tabs。
- `office-pdf`：轮询状态，ready 后用 PDF.js 渲染 `convertedUrl`。
- `media`：使用 `<audio>` 或 `<video>`，依赖 Range。
- `archive`：调用后端目录接口，展示树。
- `unsupported`：展示文件信息和打开/下载入口。

## 缓存设计

转换缓存建议放在：

```text
<store-dir>/preview-cache
```

缓存 key：

```text
storageId + objectPath + etag/mtime + size + converterVersion
```

缓存内容：

- 转换后的 PDF。
- 缩略图，可选。
- 元数据 JSON。

清理策略：

- 最大缓存体积，例如 1 GB。
- 最大保留时间，例如 7-30 天。
- LRU 清理。

## 安全要求

- 不信任任何文件内容。
- SVG、HTML、Markdown、DOCX 转 HTML 输出必须 sanitize。
- 压缩包必须防 zip slip 和炸弹包。
- Office 转换使用单独临时目录。
- 转换进程必须有超时和并发限制。
- 不把 OSS 私有访问凭证暴露给前端。
- `sourceUrl` 和 `convertedUrl` 使用短期 token 或仅在本地进程内可访问。

## 实现阶段

### 第一期

- 新增 `/api/oss/files/content` 流式接口。
- `oss-preview.html` 改用 source URL，不再用 base64 渲染图片/PDF。
- 接入 PDF.js。
- 接入代码高亮。
- 音视频预览支持 Range。

### 第二期

- 接入 SheetJS 做 CSV/XLSX 数据预览。
- 新增 LibreOffice 转 PDF。
- 新增转换缓存和异步状态接口。
- Office/PPT/DOC 默认走 PDF 预览。

### 第三期

- DOCX 快速前端预览。
- 压缩包目录预览。
- 更多图片格式转换。

### 第四期

- 评估 ONLYOFFICE 或 kkFileView 作为可配置外部预览引擎。
- 适合需要多人协作、Office 编辑或非常广格式覆盖时再引入。

## 外部预览服务评估

### ONLYOFFICE Docs

适合：

- 需要 Office 文档编辑。
- 需要协作能力。
- 文件服务可被 Document Server 访问。

代价：

- 需要部署 Document Server。
- 需要处理 JWT、回调、文件 URL 权限。
- 对桌面 demo 来说偏重。

### kkFileView

适合：

- 想快速覆盖大量格式。
- 可接受独立 Java/Spring Boot 服务。

代价：

- 技术栈较重。
- 和当前 Go 桌面 demo 集成成本高。
- 打包分发复杂。

## 推荐落点

优先实现本地可控的混合方案：

1. 前端预览图片、PDF、代码、文本、表格数据。
2. 后端负责原始文件流式读取和 Office 转 PDF。
3. 外部预览服务只作为可配置增强，不作为默认路径。

核心原则：代码、文本、图片、PDF 和表格数据浏览交给前端；Office 版式、
PPT、DOC 和复杂格式交给后端转换。

