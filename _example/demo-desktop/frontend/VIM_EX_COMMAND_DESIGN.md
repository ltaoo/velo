# ProseMirror Vim Ex 命令设计

本文档设计 `demo-desktop` ProseMirror memo 编辑器里的 Vim `:` 命令层。第一目标是让 `:w` 保存当前编辑草稿、`:wq` 发布或保存正式 memo、`:q` 退出编辑，同时把现有 `window.prompt` 方案升级成可扩展的 Ex command framework，后续可以逐步补齐常用 Vim 命令。

## 现状

- `frontend/public/vim.js` 已经有 `runExCommand(view, options)`，但它通过 `window.prompt(":", "")` 读取命令，只硬编码支持 `w/write`、`wq/x`、`q/quit`、`nohl/noh`。
- `frontend/public/prosemirror-editor.umd.js` 会把 `createVimPlugin({ onSave })` 接到 editor 的 `save` event，也有 `Mod-s` keymap。
- `frontend/src/pages/home/memo-editor.js` 已经把 ProseMirror editor 的 `onSave(instance)` 转发给 `createMiniEditor` 的 `editorOptions.onSave`。
- `frontend/src/pages/home/memos.js` 创建 composer/editor 时只传了 `onSubmit`，没有传 `onSave`。因此当前 `:w` 或 `Mod-s` 对主 memo 编辑器实际上不会写入 vault 草稿，也不会发布或保存正式 memo。
- composer 草稿目前通过 `DRAFT_STORAGE_KEY` 写入 `localStorage`，不随 vault 文件持久化，也不适合在 vault 级别同步、迁移或恢复。

## 目标

1. `:w` / `:write` 只写入草稿，不发布新 memo，也不覆盖正式 memo。
2. `:wq` 发布 composer 草稿，或保存已有 memo 的编辑内容，然后退出当前编辑上下文。
3. `:q` 退出当前编辑上下文；composer 退出输入焦点，编辑已有 memo 时关闭编辑态。
4. 草稿必须存入当前 vault 的持久化存储，不再使用 `localStorage`。
5. `:w`、`:wq`、`:q` 都支持异步结果，成功和失败都显示明确状态，不静默失败。
6. 用统一命令注册表替代 `runExCommand` 里的硬编码分支。
7. `:` 和 `/` 都使用内嵌命令行 UI，不再使用 `window.prompt`。
8. 命令层留出 range、bang、补全、历史和业务命令扩展点。
9. 保持 Vim 插件通用：`vim.js` 不直接 import memo 领域代码，业务行为通过 options 注入。

## 非目标

- 不实现完整 Vimscript。
- 不引入文件路径编辑语义，memo 是领域对象，不是裸文件 buffer。
- 不在第一阶段实现危险批量修改命令，例如 `:%s`、`:g`、`:v`。
- 不让 `:q!` 删除 composer draft，除非业务层显式实现 discard。
- 不再使用 `localStorage` 保存 composer 正文草稿；无 vault 的浏览器预览环境可以只保留内存草稿或显示草稿不可持久化。

## 架构

### 一层：Vim 插件命令 UI

`vim.js` 内新增 command-line state 和 plugin view：

```js
commandLine: {
  active: false,
  prompt: ":",
  input: "",
  historyIndex: null,
  completions: [],
  selectedCompletion: 0
}
```

按 `:` 时进入 command-line：

- 保持编辑器文档不变。
- 设置 Vim mode 为 `command` 或保留 `normal` 并设置 `commandLine.active`。建议新增 `MODES.COMMAND`，状态 badge 可显示 `COMMAND`。
- 插件 view 在 editor 容器底部渲染 `<input>`，自动 focus。
- `Enter` 执行，`Esc` 取消，`ArrowUp/ArrowDown` 走历史，`Tab` 走补全。

按 `/` 时复用同一个 command-line view，prompt 为 `/`，提交后调用现有 search executor。这样可以一起移除 `promptSearch()` 里的 `window.prompt`。

### 二层：Ex parser

新增纯函数：

```js
parseExCommand(raw) => {
  raw,
  range,
  command,
  args,
  bang,
}
```

第一阶段支持：

- 空命令：只关闭命令行。
- `!`：识别 `q!`、`write!` 等 bang。
- 命令别名：`w`/`write`、`q`/`quit`、`wq`、`x`、`nohl`/`noh`、`help`。
- 行号跳转：`:1`、`:42`、`:$`。
- range token 先只解析但不执行复杂编辑：`%`、`.`、`$`、`1,10`。

后续支持 `:s/pattern/replacement/flags` 时，需要专门 parser，不能简单 `split("/")`，因为 replacement 里可能有 escaped slash。

### 三层：命令注册表

`vim.js` 内维护命令注册表：

```js
const exCommands = [
  {
    names: ["write", "w"],
    usage: ":w",
    complete: null,
    execute(ctx) {}
  }
];
```

`ctx` 包含：

```js
{
  view,
  state,
  options,
  parsed,
  setMessage(message),
  closeCommandLine(),
  focusEditor(),
}
```

`execute` 返回：

```js
{
  ok: true,
  message: "draft written"
}
```

或 Promise。异步命令执行期间显示 `writing draft...` 或 `committing...`，完成后更新 message。失败按命令类型统一转为 `write draft failed: ...` 或 `commit failed: ...`。

### 四层：业务适配 options

扩展 `createVimPlugin(options)` 可用的回调：

```js
{
  onWriteDraft(ctx) => Promise<Result | string | void>,
  onCommit(ctx) => Promise<Result | string | void>,
  onQuit(ctx) => Promise<Result | string | void>,
  isDirty(ctx) => boolean,
  onDiscard(ctx) => Promise<Result | string | void>,
}
```

其中：

- `onWriteDraft` 是 `:w` 的业务语义，只保存草稿。
- `onCommit` 是 `:wq` / `:x` 的业务语义，发布或保存正式 memo。
- `onQuit` 是 `:q` 的业务语义，退出当前编辑上下文。
- `onDiscard` 只服务 `:q!`，用于显式丢弃当前编辑内容。

`ProsemirrorEditor` 需要让这些事件可以返回 Promise。建议新增 `emitAsync(type, ...args)`，并让 `vimPluginsFor()` 和 keymap 都调用同一条命令路径：

```js
requestWriteDraft(detail) {
  return this.emitAsync("writeDraft", this, detail);
}

requestCommit(detail) {
  return this.emitAsync("commit", this, detail);
}

requestQuit(detail) {
  return this.emitAsync("quit", this, detail);
}
```

`createMiniEditor` 继续负责把内部 editor 实例包装到页面选项：

```js
onWriteDraft(instance, detail) {
  if (editorOptions.onWriteDraft) return editorOptions.onWriteDraft(api, detail);
}
onCommit(instance, detail) {
  if (editorOptions.onCommit) return editorOptions.onCommit(api, detail);
}
onQuit(instance, detail) {
  if (editorOptions.onQuit) return editorOptions.onQuit(api, detail);
}
```

其中 `api` 是 `createMiniEditor` 返回的 editor facade，至少能 `getText()`、`focus()`、`setText()`。

## 草稿模型

新增 vault scoped 的 memo draft 概念，存到当前 vault 的持久化存储中。推荐第一阶段使用 `.velo/memo-drafts.json` 或 Velo Store 里的 `memoDrafts:v1` key；两者都位于当前 vault 的 `.velo` 目录，避免写到浏览器 `localStorage`。

建议数据结构：

```json
{
  "schemaVersion": 1,
  "drafts": [
    {
      "id": "draft_composer",
      "kind": "composer",
      "content": "未发布正文",
      "projectId": "project_xxx",
      "visibility": "PRIVATE",
      "updatedAt": "2026-06-09T12:00:00Z"
    },
    {
      "id": "draft_memo_...",
      "kind": "memo-edit",
      "memoId": "memo_...",
      "baseUpdatedAt": "2026-06-09T10:00:00Z",
      "content": "编辑中的正文",
      "projectId": "project_xxx",
      "visibility": "PRIVATE",
      "updatedAt": "2026-06-09T12:10:00Z"
    }
  ]
}
```

后端新增 API：

- `GET /api/memo-drafts`：读取当前 vault 的草稿列表。
- `POST /api/memo-drafts/upsert`：按 `id` 写入或覆盖草稿。
- `POST /api/memo-drafts/delete`：删除指定草稿。

前端新增 `domain/memo-drafts.js`：

- `loadMemoDraftsFromVault()`
- `upsertMemoDraftInVault(draft)`
- `deleteMemoDraftInVault(id)`

composer 初始化时从 `draft_composer` 恢复正文、project 和 visibility。编辑已有 memo 时从 `draft_memo_{memoId}` 恢复未提交编辑；如果 `baseUpdatedAt` 和当前 memo `updatedAt` 不一致，提示存在草稿冲突，先采用草稿内容但保留用户可丢弃入口。

## Memo 场景语义

### Composer

`memos.js` 创建 composer 时新增：

```js
onWriteDraft() {
  return writeComposerDraft({ source: "vim-write" });
}
onCommit() {
  return publishComposerDraft({ source: "vim-wq" });
}
onQuit() {
  return exitComposer();
}
```

语义：

- `:w`：内容非空则写入 `draft_composer`；成功后保留 editor 内容，显示 `draft written`。
- `:wq`：先写入/确认草稿，再发布正式 memo；成功后删除 `draft_composer`，清空 composer。
- `:q`：退出 composer 编辑焦点，不发布、不清空已持久化草稿。UI 可以 blur editor 或回到 normal mode。
- 空内容执行 `:w`：可以删除空草稿并返回 `empty draft cleared`；空内容执行 `:wq` 返回 `先写点内容`。
- 保存中再次 `:w` 或 `:wq`：返回 `正在保存` 或复用 existing `state.saving` guard。

### 编辑已有 memo

编辑态 `createMiniEditor` 新增：

```js
onWriteDraft() {
  return writeEditDraft(memo.id, { source: "vim-write" });
}
onCommit() {
  return commitEditDraft(memo.id, { source: "vim-wq" });
}
onQuit() {
  return exitEdit(memo.id);
}
```

语义：

- `:w`：写入 `draft_memo_{memoId}`，不覆盖正式 memo，不退出编辑态。
- `:wq`：保存正式 memo，成功后删除 `draft_memo_{memoId}` 并退出编辑态。
- `:q`：退出编辑态，不提交当前内容；如果内容尚未写入草稿，先自动写入草稿再退出，避免误丢。
- `:q!`：退出编辑态并删除对应草稿，显式丢弃未提交内容。
- `:x`：如果存在未提交改动，等同 `:wq`；否则等同 `:q`。

### 普通阅读态

没有活跃 editor 时不存在 `:` 命令入口。分离 memo 窗口是 readonly，不纳入第一阶段。

## 第一阶段命令清单

| 命令 | 别名 | 行为 |
| --- | --- | --- |
| `:w` | `:write` | 调用 `onWriteDraft`，只写草稿。 |
| `:w!` | `:write!` | 第一阶段等同 `:w`，但 parser 保留 bang。 |
| `:update` | `:up` | 如果 `isDirty` 为 true 才写草稿，否则显示 `draft unchanged`。没有 `isDirty` 时等同 `:w`。 |
| `:q` | `:quit` | 调用 `onQuit({ force: false })`，退出当前编辑上下文。 |
| `:q!` | `:quit!` | 调用 `onDiscard` 或 `onQuit({ force: true })`，退出并丢弃草稿。 |
| `:wq` |  | 调用 `onCommit`，发布或保存正式 memo，然后退出。 |
| `:x` |  | dirty 时等同 `:wq`，不 dirty 时等同 `:q`。 |
| `:nohl` | `:noh` | 清理 `searchRange`。 |
| `:help` |  | 显示支持命令摘要。 |
| `:1` / `:42` / `:$` |  | 跳转到对应 textblock 行。 |

## 第二阶段命令清单

| 命令 | 行为 |
| --- | --- |
| `:s/foo/bar/` | 当前行替换第一个匹配。 |
| `:s/foo/bar/g` | 当前行替换全部匹配。 |
| `:%s/foo/bar/g` | 全文替换全部匹配，执行前可先不做确认；后续再支持 `c` flag。 |
| `:d` | 删除当前行或 range。 |
| `:y` | yank 当前行或 range 到 register。 |
| `:put` | 在当前行后插入 register。 |
| `:copy {line}` | 复制 range 到目标行后。 |
| `:move {line}` | 移动 range 到目标行后。 |

这些命令都要基于 ProseMirror transaction 和现有 textblock/linewise helpers 实现，不直接拼接整篇 Markdown 字符串。

## UI 设计

命令行 DOM 由 `vim.js` 插件 view 创建，默认挂到 `editorView.dom.parentElement`：

```html
<div class="vim-command-line" data-active="true">
  <span class="vim-command-prompt">:</span>
  <input class="vim-command-input" />
  <div class="vim-command-completions"></div>
</div>
```

样式放在 `frontend/public/index.css`：

- 固定在当前 editor 底部，不遮挡输入行。
- 使用和 `.memo-vim-status` 接近的深色背景。
- message 可以复用同一容器短暂显示，例如 `draft written`、`committed`、`unknown command: foo`。
- command mode 下 badge 显示 `COMMAND`，颜色用中性深色。

## 错误处理

- unknown command：`Not an editor command: {command}`。
- 草稿回调缺失：`write draft is not available`。
- 草稿写入失败：`write draft failed: {message}`。
- 发布/保存失败：`commit failed: {message}`。
- 退出失败：`quit failed: {message}`。
- 无法跳转行：`line number out of range`。

命令执行失败不应该抛出到浏览器 console，除非是未预期异常；用户可见信息通过 Vim message 显示。

## 测试计划

### 纯函数

- `parseExCommand("w")`、`parseExCommand("write!")`、`parseExCommand("q!")`。
- range parser：`%s/a/b/g`、`1,3d`、`$`、`42`。
- command registry alias resolution：最短别名和 unknown command。

### Vim 插件 VM 测试

沿用 `VIM_MODE_CONTEXT.md` 里的方式，在 browser-like Node VM 中加载 `prosemirror.umd.min.js` 和 `vim.js`：

- normal mode 输入 `:` 后 command line active。
- 输入 `w` + Enter 调用 fake `onWriteDraft` 一次，不调用 fake `onCommit`。
- 输入 `wq` + Enter 调用 fake `onCommit` 一次。
- fake `onWriteDraft` resolve 后 message 为 `draft written`。
- fake `onWriteDraft` reject 后 message 为 `write draft failed: ...`。
- `:nohl` 清理 searchRange。
- `:42` 跳转到第 42 个 textblock。
- `Esc` 从 command mode 回到 normal mode。

### 页面集成

- composer 中 `:w` 写入 vault 草稿，不发布 memo，不清空 editor。
- composer 中 `:wq` 发布 memo，删除 vault 草稿，清空 editor。
- edit editor 中 `:w` 写入 vault 草稿，不覆盖正式 memo，不退出编辑态。
- edit editor 中 `:wq` 保存正式 memo，删除 vault 草稿，退出编辑态。
- edit editor 中 `:q` 退出编辑态，未写入的改动先自动落草稿。
- edit editor 中 `:q!` 退出编辑态并删除对应草稿。
- `Mod-s` 和 `:w` 走同一写草稿路径。
- 输入法在 command-line input 里可以正常组合，不触发 normal-mode IME block。

## 实施顺序

1. 新增 vault draft 后端 API 和 `domain/memo-drafts.js`，把 composer 的 `DRAFT_STORAGE_KEY` 从 `localStorage` 迁移到 vault 持久化草稿。
2. 把 `memos.js` 的 composer/edit editor 接上 `onWriteDraft`、`onCommit`、`onQuit`、`onDiscard`，并让写草稿、发布、保存、退出都返回 Promise 结果。
3. 调整 `ProsemirrorEditor` 的事件模型，使 `writeDraft`、`commit`、`quit` 返回值可以传回 `vim.js` 和 keymap。
4. 在 `vim.js` 中引入 command parser 与 registry，先保留 `window.prompt` UI，但命令执行走新 registry，降低一次改动的风险。
5. 用 command-line plugin view 替换 `window.prompt`，同步替换 `/` search prompt。
6. 增加 CSS、cache query version，并补 VM 测试记录。
7. 第二阶段再实现 range edit commands 和 substitute。
