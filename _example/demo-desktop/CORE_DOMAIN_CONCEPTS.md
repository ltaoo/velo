# 核心领域与概念

本文档总结 `demo-desktop` 示例项目中的核心领域模型、概念关系和主要工作流。项目本质上是一个基于 Velo 的桌面 Memo 工作台：Go 负责桌面壳、原生能力、本地文件系统和 API；前端负责工作台交互、编辑器、筛选视图和资源呈现。

## 项目定位

`demo-desktop` 是一个本地优先的桌面应用示例，核心目标是让用户选择一个本地目录作为知识库，随后在该目录中创建、编辑、组织和管理 Markdown memo。项目同时支持本地或 S3 兼容对象存储，用于管理 memo 中引用的图片和附件。

关键运行特征：

- 桌面壳由 `github.com/ltaoo/velo` 提供，运行模式是 bridge 模式。
- 前端资源通过 Go `embed` 打包进应用。
- 用户数据以本地目录和 JSON/Markdown 文件为主，避免依赖远端服务。
- Velo Store 会随当前 vault 切换到对应 `.velo` 目录。
- 应用支持多窗口、外部链接确认打开、文件拖放、系统文件选择、快捷键和自动更新。

## 架构边界

### Go 桌面后端

Go 后端位于 `internal/desktopapp`，承担以下职责：

- 启动 Velo 应用、创建主窗口、注册快捷键和拖放回调。
- 管理 vault、project、memo、对象存储配置等本地数据。
- 通过 `/api/...` 路由向前端暴露领域操作。
- 调用系统能力，例如选择目录、确认打开外部链接、打开编辑器、打开额外窗口。
- 处理自动更新检查、下载和重启应用。

### 前端工作台

前端位于 `frontend/src`，主要职责是：

- 通过 `globalThis.invoke` 调用 Go API。
- 在非桌面 bridge 环境下使用 `localStorage` 种子数据，方便浏览器预览。
- 提供 memo 编辑器、列表、筛选、日历、任务、标签、资源和链接视图。
- 解析 memo 内容中的标签、任务、双链引用、附件和外链。
- 将上传后的资源写成稳定的 `@assets/{storageId}/{key}` 引用。

## 核心概念

### App / Desktop Shell

App 是整个桌面进程和 WebView 容器。入口在 `main.go`，它把前端资源、应用配置、图标、版本和运行模式传给 `desktopapp.Run`。

App 负责：

- 初始化日志目录 `~/.myapp/app.log`。
- 初始化自动更新器。
- 根据最近使用的 active vault 决定进入 `/vault-picker` 还是 `/desktop`。
- 创建主 WebView，默认大小为 `1024x768`。
- 注册全局快捷键：显示主窗口和隐藏主窗口。
- 将系统拖放文件转换为前端可用的文件 payload。

### Vault

Vault 是用户选择的本地工作区目录，是 memo、project 和附件存储的领域根。一个 vault 目录至少包含：

- `.velo/vault.json`：vault 自身元数据。
- `.velo/projects.json`：project 列表和 active project。
- `memo/`：memo Markdown 文件目录。
- `storage/` 或配置指定目录：默认本地对象存储根。

Vault 由 `VaultContext` 表示：

- `RootDir`：vault 根目录。
- `VeloDir`：vault 下的 `.velo` 目录。
- `MemoDir`：vault 下的 `memo` 目录。
- `Entry`：写入全局 vault registry 的摘要信息。

全局 vault registry 存在用户主目录下：

- `~/.velo/data.json`

它记录最近打开的 vault 列表和 `ActiveVaultID`。应用启动时会尝试加载 active vault；如果不可用，则进入 vault 选择页。

### Vault Registry

Vault Registry 是本机级别的 vault 索引，不属于某个具体 vault。它保存：

- schema version。
- 当前 active vault ID。
- 最近打开过的 vault 列表。

打开 vault 时，项目会：

- 清理并校验路径。
- 确认路径存在、是目录、可写。
- 创建或读取 `.velo/vault.json`。
- 创建 `memo/` 目录。
- 更新全局 registry。
- 将 Velo Store 切换到当前 vault 的 `.velo` 目录。

### Project

Project 是 memo 的轻量分组。它不是独立目录，而是保存在 `.velo/projects.json` 中的结构化记录。

`ProjectRecord` 核心字段：

- `id`：形如 `project_{random}`。
- `name`：展示名称。
- `color`：十六进制颜色，非法时回退到 `#2563eb`。
- `archived`：是否归档。
- `sortOrder`：排序值。
- `createdAt` / `updatedAt`：UTC 时间戳。

Project 文件还包含 `activeProjectId`，用于记录当前活跃 project。Memo 可以通过 `projectId` 归属到某个 project；写入 memo 前会校验 project 是否存在，避免悬空引用。

### Memo

Memo 是项目最核心的业务对象。每条 memo 以 Markdown 文件存储在 vault 的 `memo/` 目录下。

`MemoRecord` 核心字段：

- `id`：形如 `memo_YYYYMMDDTHHMMSS_{random}`。
- `content`：Markdown 正文。
- `path`：相对 vault 的文件路径。
- `projectId`：可选 project 归属。
- `visibility`：`PRIVATE`、`PROTECTED` 或 `PUBLIC`。
- `pinned`：是否置顶。
- `archived`：是否归档。
- `tags`：从内容中提取的标签。
- `references`：从 `[[...]]` 语法中提取的 memo 引用。
- `createdAt` / `updatedAt`：UTC 时间戳。

Memo 文件路径按创建时间组织：

```text
memo/YYYY/MM/{memo_id}.md
```

Memo 文件使用 YAML front matter 加正文的格式：

```markdown
---
schemaVersion: 1
id: "memo_..."
projectId: "project_..."
createdAt: "..."
updatedAt: "..."
visibility: "PRIVATE"
pinned: false
archived: false
contentWhitespace: "preserve"
tags: []
references: []
---
memo content
```

`contentWhitespace: "preserve"` 表示读取时保留正文空白和尾部换行。

### Memo 内容语义

Memo 的正文不只是普通 Markdown，前后端都会从中提取结构化语义。

支持的内容语义：

- 标签：`#tag`，支持字母、数字、中文、下划线和短横线。
- 待办：`- [ ] task` / `- [x] task`。
- Wiki 风格引用：`[[target]]`、`[[target|alias]]`、`![[target]]`。
- 行选择器：例如 `[[target#L3]]` 或 `[[target#L3-L8]]`。
- Markdown 链接和图片：`[label](url)`、`![alt](url)`。
- 图片布局块：`:::image-layout grid` 到 `:::`，块内每行一个 Markdown 图片或图片 URL；`grid` 当前按微博式九宫格展示，`:::images` / `:::image-grid` 作为兼容别名。
- 原始 URL：直接写入的 `http://` 或 `https://` 链接。
- 托管附件引用：`@assets/{storageId}/{objectKey}`。

后端负责在写入 memo 时提取 `tags` 和 `references` 并保存到 front matter。前端负责构建更丰富的展示索引，例如标签统计、任务统计、双链入链/出链、未解析引用、附件列表和链接列表。

### Todo / 代办

本节描述的是当前实现。若 Todo 要升级为滴答清单/GTD 级别的一等核心领域，应采用独立任务文件模型，见 `TODO_DOMAIN_DESIGN.md`。

Todo 是从 memo 正文中派生出来的任务项，不是后端独立实体，也没有单独的数据表或 JSON 文件。它的唯一持久化来源是 memo Markdown 内容中的 task list 行。

支持的任务语法由前端正则定义：

```text
^(\s*[-*]\s+\[)([ xX])(\]\s+)(.*)$
```

也就是只识别下面两类无序列表任务：

```markdown
- [ ] 未完成任务
- [x] 已完成任务
* [ ] 星号列表任务
* [X] 大写 X 也视为完成
```

当前不会识别：

- `+ [ ] task`
- `1. [ ] task`
- `-[ ] task`
- `- [] task`
- 代码块中的任务过滤

Todo 派生对象在前端类型中称为 `TodoItem`，核心字段包括：

- `id`：`${memo.id}:${lineIndex}`，由 memo ID 和行号组成。
- `memoId`：来源 memo ID。
- `memo`：来源 memo 完整对象。
- `lineIndex`：任务所在的正文行号，零基索引。
- `checked`：是否完成。
- `text`：任务正文，不包含 `- [ ]` 或 `- [x]` 前缀。
- `projectId`：继承来源 memo 的 projectId。
- `sourceText`：用于展示来源 memo 的上下文摘要。

Todo 的解析流程：

1. 前端读取 memo 的 `content`。
2. 使用 `memoLines` 把正文按 `\n` 拆成行。
3. 每一行交给 `parseTaskLine`。
4. 匹配成功后生成一个 `TodoItem`。
5. `collectTodos(memos)` 汇总当前 memo 集合中的所有任务。
6. `getTodoStats(memos)` 基于汇总结果计算 `total`、`done` 和 `open`。

Todo 的上下文摘要由 `memoSourceText` 生成。它会优先寻找任务行之前最近的一行非任务文本；如果没有，则使用 memo 中第一行非任务文本；如果整条 memo 只有任务，则显示默认文案 `仅包含任务的 memo`。摘要会清理 Markdown 标记并压缩到固定长度。

Todo 的完成状态更新不是修改独立任务记录，而是改写来源 memo 的原始行：

1. 用户在 memo 渲染视图或 Todo view 中点击 checkbox。
2. 页面层根据 `data-memo-id` 和 `data-task-line` 找到来源 memo 与行号。
3. `toggleTask` 把 memo content 拆行。
4. `updateTaskLine` 只替换该行 checkbox 标记，保留前缀、缩进和任务正文。
5. 页面调用通用 `updateMemo`。
6. 后端 `/api/memos/update` 写回整个 memo Markdown 文件。

因此 Todo 的几个重要约束是：

- 行号是 Todo 的定位方式；如果 memo 内容被外部编辑器改动，前端需要刷新 memo 列表后重新派生 Todo。
- Todo 继承 memo 的 project、visibility、tag 和 archived 状态，本身没有独立属性。
- Todo 搜索会匹配任务文本、来源摘要、完整 memo 内容和 visibility。
- Todo view 会按未完成优先排序；同完成状态下按来源 memo 创建时间排序，再按行号排序。
- 勾选任务会更新整个 memo 的 `updatedAt`。
- Todo 统计和导航角标基于当前 project scope 下的 memo 集合计算。

删除 memo 时，Todo 有一个特殊保留逻辑。如果待删除 memo 中包含任务，确认框会出现 `同时删除 todo 项` 选项。用户取消该选项时，前端会先把该 memo 中所有任务行提取出来，创建一条新的 memo，再删除原 memo。新 memo 会继承原 memo 的 visibility 和 projectId。这个行为的含义是“保留任务文本”，不是保留原 memo 的完整上下文。

从领域建模角度看，Todo 当前是 `Memo.content` 的派生视图：

```text
Memo.content
  └─ Markdown task lines
       └─ TodoItem[]
            ├─ Todo view
            ├─ stats / nav badge
            ├─ checkbox update
            └─ delete memo preserve flow
```

这种设计的优点是简单、本地优先、与 Markdown 文件兼容；代价是 Todo 不能拥有独立截止日期、负责人、优先级、重复规则或稳定任务 ID。若后续要扩展为完整任务系统，需要考虑把任务元数据写入 Markdown 约定、front matter，或引入独立 task index。

### Visibility

Visibility 是 memo 的可见性标记，目前是本地领域属性，不直接对应权限系统。

取值：

- `PRIVATE`：仅自己。
- `PROTECTED`：工作区。
- `PUBLIC`：公开。

非法或空值会回退为 `PRIVATE`。

### Pin / Archive

Memo 和 Project 都支持归档语义：

- Memo 的 `archived` 控制是否进入归档状态。
- Project 的 `archived` 控制分组是否归档。

Memo 还支持 `pinned`，用于在工作台中置顶展示。

### Cloud Storage / OSS

Cloud Storage 是 memo 附件和图片的存储配置集合。虽然命名中有 OSS，实际抽象是 S3 兼容对象存储加本地对象存储。

设置保存在 Velo Store 中，key 为：

```text
demo-desktop:settings:cloud-storage:v1
```

`CloudStorageSettings` 包含：

- `activeStorageId`：当前活跃存储。
- `defaultsInitialized`：是否已经初始化默认配置。
- `storages`：多个 `OSSConfig`。

`OSSConfig` 支持：

- S3 兼容存储：endpoint、bucket、access key、secret、region、path prefix、public base URL 等。
- 本地存储：provider 为 `local` 或 `local-oss`，默认 ID 为 `memo-local`，默认 bucket 为 `memos`。

上传资源后会返回：

- `url`：可访问 URL。
- `ref`：稳定资源引用，格式为 `@assets/{storageId}/{key}`。

Memo 删除时可以清理该 memo 独占引用的托管附件。如果同一附件仍被其他 memo 引用，则会跳过删除。

### Asset Reference

Asset Reference 是 memo 正文中用于引用托管附件的稳定标识：

```text
@assets/{storageId}/{key}
```

它把正文和具体存储配置解耦：

- 前端可以根据当前 storage settings 解析成实际 URL。
- 后端可以根据 storage ID 和 key 删除对象。
- 本地存储可以通过 `/api/oss/assets` 代理访问。
- 远程存储可以重定向到公开对象 URL。

### Resource / Link

Resource 是从 memo 内容中解析出的附件或图片；Link 是从 memo 内容中解析出的外链。

前端会把以下内容归类为 resource：

- Markdown 图片。
- 常见图片扩展名 URL。
- 常见文件扩展名 URL。
- `local://`、`blob:`、`data:` 链接。
- `@assets/...` 引用。

普通 `http(s)` 和 `mailto:` 链接归类为 link。桌面环境中打开外部 `http(s)` 链接会经过 Go 后端确认，再交给系统默认浏览器。

### Detached Memo Window

Detached Memo Window 是独立 memo 窗口。前端请求 `/api/memo-window/open` 后，后端会把 memo payload 暂存在内存缓存中，并打开一个 frameless WebView。

特点：

- 每个窗口按 memo ID 命名。
- 入口页面是 `memo-window.html`。
- 可传入单条 memo，也可传入 memo 列表作为渲染上下文。
- 数据只缓存在当前进程内，不是持久化模型。

### Window / Utility Pages

除主工作台外，应用还支持多个工具窗口：

- `/settings`：设置页。
- `/oss-manager`：OSS 文件管理。
- `/oss-storage-editor`：存储配置编辑。
- `/oss-preview`：OSS 文件预览。
- `/memo-slim`：轻量 memo 窗口。

这些窗口由 `/api/open_window` 统一创建，并根据 pathname 决定入口 HTML、窗口标题、尺寸和窗口名称。

### Update

Update 是桌面应用级能力，不属于 memo 领域，但属于项目核心基础设施。

更新配置来自 `app-config.json` 的 `update` 字段。后端暴露：

- `/api/update/check`
- `/api/update/download`
- `/api/update/restart`
- `/api/update/skip`

下载进度通过 Velo message 推送给前端。

## 主要工作流

### 启动与进入工作台

1. `main.go` 嵌入前端资源和配置。
2. `desktopapp.Run` 初始化日志、更新器和 Velo App。
3. 后端读取 `~/.velo/data.json`。
4. 如果 active vault 可用，则打开该 vault 并进入 `/desktop`。
5. 如果没有 active vault，则进入 `/vault-picker`。
6. 主 WebView 创建后，前端路由把 `/desktop` 映射到 memo 工作台。

### 选择或创建 Vault

1. 前端调用 `/api/vault/status` 读取本机 vault registry。
2. 用户输入路径或通过系统目录选择器选择路径。
3. 前端调用 `/api/vault/open`。
4. 后端校验目录、创建 `.velo` 和 `memo/`，读取或创建 `vault.json`。
5. 后端更新全局 registry，并把 Store 切换到该 vault 的 `.velo`。
6. 前端跳转到 `/desktop`。

### 创建 Memo

1. 用户在 mini editor 中输入 Markdown。
2. 前端调用 `/api/memos/create`，传入 content、visibility 和可选 projectId。
3. 后端校验内容非空、校验 projectId。
4. 后端生成 memo ID，提取 tags 和 references。
5. 后端渲染 front matter 和正文，原子写入 `memo/YYYY/MM/{id}.md`。
6. 前端刷新 memo 列表并重建标签、任务、引用和资源索引。

### 更新 Memo

1. 前端提交 patch 到 `/api/memos/update`。
2. 后端按 memo ID 在 `memo/` 下查找 Markdown 文件。
3. 后端读取 front matter 和正文，合并 patch。
4. 如果内容变更，则重新提取 tags 和 references。
5. 后端更新 `updatedAt` 并原子写回原文件。

### 删除 Memo 与附件清理

1. 前端调用 `/api/memos/delete`。
2. 后端定位 memo 文件。
3. 如果开启 `cleanupAssets`，后端提取该 memo 中的 `@assets/...` 引用。
4. 后端扫描其他 memo，跳过仍被其他 memo 引用的附件。
5. 后端删除 memo 文件。
6. 后端删除独占附件，并返回删除数量、跳过数量和错误列表。

### 上传与引用附件

1. 用户拖放、粘贴或选择文件。
2. 前端把文件转为 base64 data URL。
3. 前端调用 OSS 上传 API。
4. 后端按 active storage 写入本地或 S3 兼容存储。
5. 后端返回对象 URL 和 `@assets/...` ref。
6. 前端把 Markdown 图片或文件链接插入 memo 正文。

## 持久化模型

| 数据 | 位置 | 格式 | 说明 |
| --- | --- | --- | --- |
| 全局 vault registry | `~/.velo/data.json` | JSON | 本机最近 vault 和 active vault |
| Vault 元数据 | `{vault}/.velo/vault.json` | JSON | vault ID、名称、schema version |
| Project 列表 | `{vault}/.velo/projects.json` | JSON | projects 和 activeProjectId |
| Memo | `{vault}/memo/YYYY/MM/*.md` | Markdown + front matter | memo 正文和元数据 |
| Cloud storage settings | Velo Store key `demo-desktop:settings:cloud-storage:v1` | JSON | 当前 vault 的存储配置 |
| 默认本地附件 | `{vault}/storage/memos/...` | 文件 | provider 为 local 的默认对象存储 |
| 窗口状态 | Velo Store window state | JSON | 窗口位置和尺寸 |

## API 边界

主要 API 族：

- `/api/vault/*`：vault 状态、目录选择、打开 vault。
- `/api/projects/*`：project 列表、创建、更新、激活。
- `/api/memos/*`：memo 列表、创建、更新、删除。
- `/api/settings/cloud-storage*`：对象存储配置读取、保存、删除。
- `/api/oss/*`：上传、列表、预览、建目录、删除、资源访问。
- `/api/window/*`：显示、隐藏、恢复窗口状态。
- `/api/editor/open`：用外部编辑器打开文件。
- `/api/external/open`：确认并打开外部浏览器链接。
- `/api/memo-window/*`：打开和读取独立 memo 窗口数据。
- `/api/update/*`：检查、下载、应用和跳过更新。
- `/api/open_window`：打开设置、OSS 管理、预览等工具窗口。

## 关键约束与不变量

- 只有 active vault 存在时，project 和 memo API 才能工作。
- Vault path 必须是存在、可写的目录。
- Memo 文件必须写在当前 vault 的 `memo/` 目录内，不能通过相对路径逃逸 vault。
- Memo 内容不能为空。
- Memo 的 projectId 必须为空或指向已存在 project。
- Memo visibility 只允许 `PRIVATE`、`PROTECTED`、`PUBLIC`，否则回退到 `PRIVATE`。
- Project ID 和 storage ID 都会被清理成受限字符集。
- Project color 必须是 `#RRGGBB`，否则回退到默认蓝色。
- Markdown 文件写入和 JSON 文件写入使用临时文件加 rename 的原子写入策略。
- 删除 memo 时，只清理没有被其他 memo 共享引用的托管附件。
- 外部 `http(s)` 链接在桌面环境中由后端校验并确认后打开。

## 前端领域视图

工作台并不是简单的 memo 列表，它会从 memo 集合派生多个视图：

- Memo feed：按时间、置顶、归档、可见性、搜索、project、tag 等条件展示。
- Project filter：按 project 或未分配 memo 过滤。
- Tag view：从内容中提取标签并按数量排序。
- Todo view：从 Markdown task list 中提取任务并支持勾选更新。
- Calendar view：按 memo 创建日期组织。
- Link view：聚合 memo 中的外部链接。
- Resource view：聚合 memo 中的图片和附件。
- Reference index：构建 memo 双链的入链、出链和未解析引用。

这些视图大多是前端派生状态，持久化来源仍然是 vault 中的 memo Markdown 文件和 project JSON。

## 概念关系图

```text
App
  ├─ Velo desktop shell
  ├─ WebView frontend
  ├─ Velo Store
  └─ Active Vault
       ├─ Vault metadata (.velo/vault.json)
       ├─ Projects (.velo/projects.json)
       │    └─ ProjectRecord
       ├─ Memos (memo/YYYY/MM/*.md)
       │    ├─ tags
       │    ├─ references
       │    ├─ tasks
       │    ├─ links
       │    └─ asset references
       └─ Storage settings
            ├─ local storage
            └─ S3-compatible storage
```

## 代码入口索引

- 应用入口：`main.go`
- 桌面启动：`internal/desktopapp/app.go`
- 路由注册：`internal/desktopapp/api_routes.go`
- Vault：`internal/desktopapp/vault_project.go`
- Project：`internal/desktopapp/project.go`
- Memo：`internal/desktopapp/memo.go`
- Memo Markdown：`internal/desktopapp/memo_markdown.go`
- Memo 附件引用：`internal/desktopapp/memo_assets.go`
- 存储配置：`internal/desktopapp/cloud_storage_settings.go`
- OSS / local storage：`internal/desktopapp/oss_storage.go`、`internal/desktopapp/oss_local.go`
- 桌面能力路由：`internal/desktopapp/routes_desktop.go`
- 更新与多窗口：`internal/desktopapp/routes_update_window.go`
- 前端领域层：`frontend/src/domain/`
- Memo 工作台：`frontend/src/pages/home/memos.js`
- Vault 选择页：`frontend/src/pages/vault-picker/index.js`
