# AGENTS.md

本文件给 Codex 快速定位代码用。用户在对话里描述一个功能时，优先按下方“功能地图 -> 代码”查找；不要先全仓库盲搜，除非表里没有覆盖。

## 项目概览

- 仓库主体是 Go 模块 `github.com/ltaoo/velo`，Go 版本见 `go.mod`。
- 根包 `velo` 是桌面应用框架：WebView、JS bridge、内置 API 路由、窗口状态、轻量存储、HTTP/WebSocket fallback。
- `cmd/velo` 是 CLI：`velo build`、`velo dev`、`velo doctor`。
- `_example/demo-desktop` 是功能最完整的示例/业务应用：vault、memo、project、task、GTD、OSS、本地设置、输入法锁定、剪贴板、独立窗口。
- `_example/demo-*` 还有 reader、IM、notification、macostool、iOS 等较小示例。

## 快速入口

| 想找的内容 | 首看代码 |
| --- | --- |
| 框架 API、`Box`、`BoxContext`、`Get/Post`、`Run` | `velo.go` |
| JS 调 Go 的 `window.invoke`、`window.onGoMessage`、WebSocket fallback | `asset/runtime/runtime.js`, `ws.go`, `velo.go` |
| 原生 WebView 窗口实现 | `webview/webview.go`, `webview/webview_darwin_pure.go`, `webview/webview_windows.go`, `webview/webview_others.go` |
| CLI 命令入口 | `cmd/velo/main.go` |
| 打包构建、签名、公证、DMG | `cmd/velo/build.go`, `buildcfg/` |
| 开发模式运行 | `cmd/velo/dev.go` |
| 环境检查 | `cmd/velo/doctor.go` |
| 自动更新主流程 | `updater/api/api.go`, `updater/checker/`, `updater/downloader/`, `updater/applier/` |
| demo-desktop 应用启动 | `_example/demo-desktop/main.go`, `_example/demo-desktop/internal/desktopapp/app.go` |
| demo-desktop 所有后端路由分派 | `_example/demo-desktop/internal/desktopapp/api_routes.go` |
| demo-desktop 前端主入口和路由 | `_example/demo-desktop/frontend/src/index.js`, `_example/demo-desktop/frontend/src/store/routes.js` |
| demo-desktop 主 memo 工作台 | `_example/demo-desktop/frontend/src/pages/home/index.js`, `_example/demo-desktop/frontend/src/pages/home/memos.js` |

## 框架功能地图 -> 代码

| 功能描述/关键词 | 主要代码 | 说明 |
| --- | --- | --- |
| 创建应用、注册 API、返回统一 JSON | `velo.go` | `NewApp`, `Box.Get`, `Box.Post`, `BoxContext.Ok/Error/BindJSON/Query`。 |
| bridge 模式、HTTP 模式、BridgeHttp 模式 | `velo.go` | `ModeBridge`, `ModeBridgeHttp`, `ModeHttp`, `Box.Run`, `Box.webviewURL`。 |
| 前端调用 Go API | `asset/runtime/runtime.js`, `velo.go` | 前端 `invoke(url, {method,args,headers})` -> `Box.handleMessage` -> `get_handlers/post_handlers`。 |
| HTTP fallback API | `velo.go` | `setupMux` 把 `b.Get/b.Post` 同时注册为 HTTP handler。 |
| WebSocket fallback | `ws.go`, `asset/runtime/runtime.js` | 内部路径 `/__velo/ws`，用于无原生 bridge 时收发调用和 Go 推送消息。 |
| 注入运行时信息 | `velo.go`, `asset/runtime/runtime.js` | `injectedRuntimeJS` 注入 `window.__VELO__` 和 runtime。 |
| Go 主动推送消息给前端 | `velo.go`, `webview/webview.go`, `ws.go` | `Box.SendMessage` -> 原生 WebView 或 WebSocket；前端用 `window.onGoMessage(handler)`。 |
| 内置存储 API | `velo.go`, `store/store.go` | `/api/storage/get`, `/api/storage/set`, `/api/storage/delete`，持久化到 `storage.json`。 |
| 内置窗口状态 API | `velo.go`, `store/store.go` | `/api/window/state/snapshot`, `/api/window/state/load`。demo 另有 `/api/window/state/save/restore`。 |
| 运行时信息 API | `velo.go` | `/api/velo/info`。 |
| SPA/静态资源服务 | `frontendserver/frontendserver.go` | dev/prod 两种模式，支持 entry page fallback 和静态资源 cache header。 |
| 原生窗口控制方法 | `asset/runtime/runtime.js`, `webview/webview_darwin_pure.go`, `webview/webview_windows.go` | `__velo/window/start_drag`, `close`, `minimize`, `hide`, `set_size`, `state`, `toggle_maximize`, `maximize`, `restore`, `set_always_on_top`。Windows 当前没有 `state/set_size/hide` 的完整对应实现。 |
| 文件选择对话框 | `file/file.go`, `file/file_darwin.go`, `file/file_windows.go`, `file/file_linux.go` | `ShowFileSelectDialog`, `ShowFileSelectDialogWithTypes`。 |
| 系统托盘 | `tray/tray.go`, `tray/tray_darwin.go`, `tray/tray_windows.go`, `tray/tray_linux.go` | `NewTray`, `MenuItem`, native setup/run。 |
| 通知/推送 | `notification/` | 本地通知 `Show`，macOS/Windows/Linux 分平台，远程推送在 `push*.go`。 |
| 原生错误弹窗 | `error/` | `ShowErrorDialog`，各平台实现。 |
| 全局快捷键 | `shortcut/shortcut.go`, `shortcut/shortcut_darwin.go`, `shortcut/shortcut_windows.go` | 基于 `golang.design/x/hotkey`，demo 在 `desktopapp/app.go` 注册快捷键。 |
| 开机自启动 | `autostart/` | macOS/Windows/other 分平台。 |
| 输入法/输入源 | `inputsource/inputsource.go`, `inputsource/manager.go`, `inputsource/inputsource_darwin.go`, `inputsource/inputsource_windows.go` | 列举、读取、切换输入源，以及按前台 app 规则锁定输入源。 |
| 数据库和迁移 | `database/database.go`, `database/migration.go` | GORM + migrate，支持 sqlite/mysql/postgres。 |

## CLI/构建功能地图 -> 代码

| 功能描述/关键词 | 主要代码 | 说明 |
| --- | --- | --- |
| CLI 命令分发 | `cmd/velo/main.go` | `version`, `doctor`, `build`, `dev`。 |
| `velo build` 配置读取 | `cmd/velo/build.go`, `buildcfg/buildcfg.go` | 读取 `app-config.json`，校验 `app.name/app.version`。 |
| 图标生成 | `buildcfg/icons.go` | 生成多尺寸 PNG、ICO、macOS iconset/icns，兼容复制到 `build/`。 |
| Windows 资源 | `buildcfg/windows.go` | `GenerateWinres` 生成 `winres.json`。 |
| macOS plist/entitlements | `buildcfg/darwin.go` | `GenerateDarwinPlist`，包含 APS entitlement。 |
| Linux desktop entry | `buildcfg/linux.go` | `GenerateLinuxDesktop`。 |
| macOS app bundle/DMG | `cmd/velo/build.go` | `createDarwinApp`, `createDarwinDMG`。 |
| macOS 签名/公证 | `cmd/velo/build.go` | `.env` 读取 Apple/P12/P8 凭证，`codesign`, `notarytool`, `stapler`。 |
| dev 模式 rebuild/restart | `cmd/velo/dev.go` | `go build` 后运行二进制，按 `R` 重启，`Q` 退出。 |
| 环境诊断 | `cmd/velo/doctor.go` | Go/Git/Xcode/CocoaPods/gomobile/iOS SDK/Linux/Windows deps。 |

## 自动更新功能地图 -> 代码

| 功能描述/关键词 | 主要代码 | 说明 |
| --- | --- | --- |
| 应用级更新器 API | `updater/api/api.go` | `NewUpdaterWithOptions`, check/download/apply/restart/skip 等聚合入口。 |
| 更新配置类型 | `updater/types/types.go`, `buildcfg/buildcfg.go` | `UpdateConfig`, `UpdateSource`, `ReleaseInfo`；`app-config.json` 的 `update` 会转为 updater config。 |
| 版本/开发模式判断 | `updater/version/version.go` | `(dev)`、semver 比较、更新模式。 |
| 更新源选择和缓存 | `updater/checker/checker.go`, `updater/cache/cache.go` | 多源按优先级检查，缓存和跳过版本状态。 |
| GitHub Releases 更新源 | `updater/checker/github_checker.go` | 解析 release 和平台 asset/checksum。 |
| HTTP manifest 更新源 | `updater/checker/http_checker.go`, `updater/checker/manifest.go` | manifest 下载、校验、平台 asset 匹配。 |
| 下载、断点续传、校验 | `updater/downloader/download.go` | HTTPS 校验、resume、progress callback、SHA256。 |
| 应用更新 | `updater/applier/` | `applier_darwin.go`, `applier_windows.go`, `applier_unix.go`，备份、替换、回滚、重启。 |
| demo-desktop 更新接口 | `_example/demo-desktop/internal/desktopapp/routes_update_window.go` | `/api/update/check`, `/download`, `/restart`, `/skip`，下载进度通过 `download_progress` 消息推送。 |

## demo-desktop 后端功能地图 -> 代码

后端所有业务路由从 `_example/demo-desktop/internal/desktopapp/api_routes.go` 注册。

| 功能描述/关键词 | API 路由 | 后端入口 | 领域实现 |
| --- | --- | --- | --- |
| 应用启动、主窗口、快捷键 | - | `_example/demo-desktop/internal/desktopapp/app.go` | logger、updater、vault 初始化、`NewApp`、`NewWebview`、全局快捷键。 |
| ping/app 信息 | `/api/ping`, `/api/app` | `_example/demo-desktop/internal/desktopapp/routes_vault_memo.go` | `appVersion`, `appMode`, `velo.GetVersion`。 |
| vault 状态、选择目录、打开 vault | `/api/vault/status`, `/api/vault/select-directory`, `/api/vault/open` | `_example/demo-desktop/internal/desktopapp/routes_vault_memo.go` | `_example/demo-desktop/internal/desktopapp/vault_project.go`, `_example/demo-desktop/internal/desktopapp/platform/native_select_directory_*.go`。 |
| project 列表/创建/更新/激活 | `/api/projects*` | `_example/demo-desktop/internal/desktopapp/routes_vault_memo.go` | `_example/demo-desktop/internal/desktopapp/project.go`, `_example/demo-desktop/internal/desktopapp/vault_project.go`。 |
| memo 列表/创建/更新/删除 | `/api/memos*` | `_example/demo-desktop/internal/desktopapp/routes_vault_memo.go` | `_example/demo-desktop/internal/desktopapp/memo.go`, `_example/demo-desktop/internal/desktopapp/memo_assets.go`, `_example/demo-desktop/internal/desktopapp/memo_markdown.go`, `_example/demo-desktop/internal/desktopapp/memo_task_delete.go`。 |
| memo 评论 | `/api/memo-comments*` | `_example/demo-desktop/internal/desktopapp/routes_vault_memo.go` | `_example/demo-desktop/internal/desktopapp/memo_comment.go`, `_example/demo-desktop/internal/desktopapp/memo_assets.go`。 |
| memo 草稿 | `/api/memo-drafts*` | `_example/demo-desktop/internal/desktopapp/routes_vault_memo.go` | `_example/demo-desktop/internal/desktopapp/memo_draft.go`。 |
| task 列表/详情/创建/更新/完成/删除 | `/api/tasks*` | `_example/demo-desktop/internal/desktopapp/routes_tasks.go` | `_example/demo-desktop/internal/desktopapp/task.go`, `_example/demo-desktop/internal/desktopapp/task_index.go`, `_example/demo-desktop/internal/desktopapp/task_file.go`。 |
| task note | `/api/tasks/notes/create` | `_example/demo-desktop/internal/desktopapp/routes_tasks.go` | `_example/demo-desktop/internal/desktopapp/task_notes.go`, `_example/demo-desktop/internal/desktopapp/memo_task_sync.go`。 |
| 从 memo 行拆 task | `/api/tasks/extract-from-memo` | `_example/demo-desktop/internal/desktopapp/routes_tasks.go` | `_example/demo-desktop/internal/desktopapp/task_notes.go`, `_example/demo-desktop/internal/desktopapp/memo_task_sync.go`。 |
| 重建 task index | `/api/task-index/rebuild` | `_example/demo-desktop/internal/desktopapp/routes_tasks.go` | `_example/demo-desktop/internal/desktopapp/task_index.go`。 |
| GTD item | `/api/gtd/items*` | `_example/demo-desktop/internal/desktopapp/routes_gtd.go` | `_example/demo-desktop/internal/desktopapp/gtd_item.go`。 |
| GTD milestone | `/api/gtd/milestones*` | `_example/demo-desktop/internal/desktopapp/routes_gtd.go` | `_example/demo-desktop/internal/desktopapp/gtd_milestone.go`。 |
| 编辑器设置 | `/api/settings/editor*` | `_example/demo-desktop/internal/desktopapp/routes_storage.go` | `_example/demo-desktop/internal/desktopapp/editor_settings.go`。 |
| 云/OSS 存储设置 | `/api/settings/cloud-storage*` | `_example/demo-desktop/internal/desktopapp/routes_storage.go` | `_example/demo-desktop/internal/desktopapp/cloud_storage_settings.go`, `_example/demo-desktop/internal/desktopapp/oss_helpers.go`, `_example/demo-desktop/internal/desktopapp/oss_storage.go`, `_example/demo-desktop/internal/desktopapp/oss_local.go`。 |
| OSS 上传/文件管理/预览/资产代理 | `/api/oss/*` | `_example/demo-desktop/internal/desktopapp/routes_storage.go` | `_example/demo-desktop/internal/desktopapp/oss_storage.go`, `_example/demo-desktop/internal/desktopapp/oss_local.go`, `_example/demo-desktop/internal/desktopapp/oss_helpers.go`。 |
| 窗口显示隐藏/状态保存恢复 | `/api/window/show`, `/api/window/hide`, `/api/window/state/save`, `/api/window/state/restore` | `_example/demo-desktop/internal/desktopapp/routes_desktop.go` | `store.Store`，主窗口由 `_example/demo-desktop/internal/desktopapp/app.go` 管。 |
| 打开任意辅助窗口 | `/api/open_window` | `_example/demo-desktop/internal/desktopapp/routes_update_window.go` | `_example/demo-desktop/internal/desktopapp/windowing/open_window.go` 生成 name/title/path/entry page。 |
| 独立 memo 窗口 | `/api/memo-window/open`, `/api/memo-window/get` | `_example/demo-desktop/internal/desktopapp/routes_desktop.go` | `_example/demo-desktop/frontend/memo-window.html`, `_example/demo-desktop/frontend/src/memo-window.js`, `_example/demo-desktop/frontend/src/pages/home/memos.js` 的 `mountDetachedMemoWindow`。 |
| 文件选择、图片 data URL | `/api/file/select`, `/api/file/select-data-url` | `_example/demo-desktop/internal/desktopapp/routes_desktop.go` | `_example/demo-desktop/internal/desktopapp/desktop_io.go`, `file/`。 |
| 外部编辑器打开文件 | `/api/editor/open`, `/api/editor/apps` | `_example/demo-desktop/internal/desktopapp/routes_desktop.go` | `_example/demo-desktop/internal/desktopapp/editor_external.go`, `_example/demo-desktop/internal/desktopapp/editor_apps.go`, `_example/demo-desktop/internal/desktopapp/editor_settings.go`。 |
| 外部浏览器打开链接 | `/api/external/open` | `_example/demo-desktop/internal/desktopapp/routes_desktop.go` | `_example/demo-desktop/internal/desktopapp/external/browser.go`, `_example/demo-desktop/internal/desktopapp/platform/native_confirm_*.go`。 |
| 输入法锁定设置/状态 | `/api/settings/input-source-lock*`, `/api/input-source/status` | `_example/demo-desktop/internal/desktopapp/routes_input_source_lock.go` | `_example/demo-desktop/internal/desktopapp/input_source_lock.go`, 根包 `inputsource/`。 |
| 剪贴板最新内容 | `/api/clipboard/latest` | `_example/demo-desktop/internal/desktopapp/clipboard.go` | `clipboard-go` watcher，返回文本/HTML/PNG snapshot。 |
| snippet 搜索/启动器 | `/api/snippets/search`, `/api/snippet-launcher/open` | `_example/demo-desktop/internal/desktopapp/snippets.go` | 从 memo markdown code fence 收集片段；启动器窗口 `_example/demo-desktop/frontend/snippet-launcher.html`。 |
| 拖拽文件进 memo | Go 推送 `memo_file_drop` | `_example/demo-desktop/internal/desktopapp/app.go` | `mainWindowOptions.OnDragDrop` -> `b.SendMessage`；前端在 `_example/demo-desktop/frontend/src/pages/home/memos.js` 监听。 |
| 主窗口重新聚焦 | Go 推送 `main_window_focus` | `_example/demo-desktop/internal/desktopapp/app.go` | `showMainWindow`；前端在 `_example/demo-desktop/frontend/src/pages/home/memos.js` 监听。 |

## demo-desktop 前端功能地图 -> 代码

| 功能描述/关键词 | 前端入口 | 调用后端/领域模块 |
| --- | --- | --- |
| 应用入口、路由挂载、窗口状态轮询 | `_example/demo-desktop/frontend/src/index.js` | `store/index.js`, `/api/window/state/restore`, `/api/window/state/snapshot`。 |
| 路由表 | `_example/demo-desktop/frontend/src/store/routes.js` | `/desktop` 映射到 `/home/index`。 |
| HTTP/bridge 客户端 | `_example/demo-desktop/frontend/src/store/http_client.js`, `_example/demo-desktop/frontend/src/domain/native.js` | `invoke(...)`, `callNativeAPI(...)`。 |
| Vault 选择页 | `_example/demo-desktop/frontend/src/pages/vault-picker/index.js` | `_example/demo-desktop/frontend/src/domain/vaults.js` -> `/api/vault/*`。 |
| 主工作台挂载 | `_example/demo-desktop/frontend/src/pages/home/index.js` | `mountMemosHome`。 |
| Memo 工作台主逻辑 | `_example/demo-desktop/frontend/src/pages/home/memos.js` | 体量最大：memo 列表、编辑器、筛选、评论、task/GTD 视图、资源、链接、日历、拖拽、独立窗口入口。 |
| Memo 渲染模板 | `_example/demo-desktop/frontend/src/pages/home/memo-templates.js` | card、评论、detached window 模板。 |
| Memo markdown/引用/标签/todo 解析 | `_example/demo-desktop/frontend/src/domain/memos.js`, `_example/demo-desktop/frontend/src/pages/home/memo-markdown.js`, `_example/demo-desktop/frontend/src/pages/home/memo-utils.js` | wikilink、embed、task line、tag、markdown 渲染。 |
| Memo/project 数据调用 | `_example/demo-desktop/frontend/src/domain/memo-repository.js`, `_example/demo-desktop/frontend/src/domain/projects.js` | `/api/memos*`, `/api/projects*`。 |
| Memo 评论调用 | `_example/demo-desktop/frontend/src/domain/memo-comments.js` | `/api/memo-comments*`。 |
| Memo 草稿调用 | `_example/demo-desktop/frontend/src/domain/memo-drafts.js` | `/api/memo-drafts*`。 |
| Task 数据和 normalize | `_example/demo-desktop/frontend/src/domain/tasks.js` | `/api/tasks*`, `/api/task-index/rebuild`。 |
| GTD 数据和 normalize | `_example/demo-desktop/frontend/src/domain/gtd.js` | `/api/gtd/*`。 |
| 资源/链接/代码块收集 | `_example/demo-desktop/frontend/src/domain/memo-resources.js` | 纯前端扫描 memo 内容；snippet 后端也有类似扫描。 |
| 云存储设置、asset URL | `_example/demo-desktop/frontend/src/domain/storage.js` | `/api/settings/cloud-storage*`, `/api/oss/assets`。 |
| 设置窗口 | `_example/demo-desktop/frontend/settings.html` | 内联 JS/CSS，调用 `/api/settings/editor`, `/api/settings/cloud-storage`, `/api/settings/input-source-lock`, `/api/open_window`。 |
| OSS 文件管理 | `_example/demo-desktop/frontend/oss-manager.html` | 内联 JS/CSS，调用 `/api/oss/files/*`, `/api/open_window`。 |
| OSS 存储编辑 | `_example/demo-desktop/frontend/oss-storage-editor.html` | 内联 JS/CSS，调用 cloud storage settings。 |
| OSS 文件预览 | `_example/demo-desktop/frontend/oss-preview.html` | 内联 JS/CSS，调用 `/api/oss/files/preview`。 |
| 图片预览/标注窗口 | `_example/demo-desktop/frontend/image-preview.html`, `_example/demo-desktop/frontend/src/image-preview.js`, `_example/demo-desktop/frontend/src/components/image-preview.js` | 原生窗口控制 `__velo/window/*`，本地 preview state。 |
| 独立 memo 窗口 | `_example/demo-desktop/frontend/memo-window.html`, `_example/demo-desktop/frontend/src/memo-window.js`, `_example/demo-desktop/frontend/src/pages/home/memos.js` | `/api/memo-window/get`, `/api/window/state/save`。 |
| 迷你 memo 窗口 | `_example/demo-desktop/frontend/memo-slim.html`, `_example/demo-desktop/frontend/src/memo-slim.js` | `/api/memos*`, `__velo/window/*`。 |
| Snippet launcher | `_example/demo-desktop/frontend/snippet-launcher.html`, `_example/demo-desktop/frontend/src/snippet-launcher.js` | `/api/snippets/search`, `__velo/window/hide`, `__velo/window/set_size`。 |

## 示例应用地图

| 示例 | 入口 | 说明 |
| --- | --- | --- |
| demo-desktop | `_example/demo-desktop/main.go` | 最完整业务示例，主要维护对象。 |
| demo-reader | `_example/demo-reader/main.go` | 阅读器、SQLite/migrations、文件读取、窗口 pinned/show/hide。 |
| demo-im | `_example/demo-im/main.go` | 简单 IM/消息示例。 |
| demo-notification | `_example/demo-notification/main.go` | 通知能力示例。 |
| demo-macostool | `_example/demo-macostool/main.go` | macOS 工具类示例，含前端静态资源。 |
| demo-ios | `_example/demo-ios/main.go` | iOS 相关实验入口。 |
| frontendserver example | `_example/frontendserver/main.go` | 前端 server 示例。 |

## 定位规则

1. 如果用户说的是“框架能力”（WebView、bridge、窗口、storage、CLI、updater、native 包），先看根目录包和 `cmd/velo`。
2. 如果用户说的是“桌面示例里的业务功能”（memo、task、vault、OSS、settings、snippet、输入法锁定），先看 `_example/demo-desktop/internal/desktopapp/routes_*.go` 找 API，再看同名领域文件。
3. 前端问题先从 `_example/demo-desktop/frontend/src/pages/...` 或对应 HTML 入口开始，再追到 `_example/demo-desktop/frontend/src/domain/*`。
4. `_example/demo-desktop/frontend/src/pages/home/memos.js` 是主工作台大文件。涉及 memo 列表、筛选、编辑、评论、task/GTD 面板、资源视图、拖拽、独立 memo 窗口时，通常都要查它。
5. 后端新增 API 应走 `b.Get`/`b.Post`，业务路由放到对应 `_example/demo-desktop/internal/desktopapp/routes_*.go`，并在 `_example/demo-desktop/internal/desktopapp/api_routes.go` 注册。
6. 前端新增 native 调用优先通过 `domain/native.js` 的 `callNativeAPI` 或已有 domain 模块封装，不要在页面里散落重复错误处理。
7. 窗口类需求先区分：框架内置窗口控制在 `__velo/window/*`；demo 业务窗口打开在 `/api/open_window` 或 `/api/memo-window/open`。

## 对话输出约定

- 每次对话最后都追加 `喵~`。

## 常用验证命令

```bash
go test ./...
go test ./updater/...
go test ./_example/demo-desktop/internal/desktopapp/...
go build ./cmd/velo
```

前端 demo 没有完整构建脚本，`_example/demo-desktop/frontend/package.json` 目前主要提供 ESLint。
