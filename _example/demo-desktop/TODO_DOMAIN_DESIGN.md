# Todo 领域设计：从 Memo 派生任务到 GTD 文件模型

本文档讨论如何把 `demo-desktop` 中的 Todo 从“memo 正文里的 task line 派生视图”升级为一等核心领域对象，以支持接近滴答清单这类完整 GTD 应用的复杂能力，同时保持项目当前的本地优先和文件持久化风格。

## 目标

Todo 和 Memo 都应该是核心领域：

- Memo 负责记录、沉淀、引用、上下文和知识整理。
- Todo 负责行动管理、计划、提醒、重复、完成和复盘。

两者可以互相引用，但不应该互相吞并。Memo 中可以出现任务，任务中也可以引用 memo；但任务的状态、日期、重复规则、提醒和排序不应依赖 memo 正文行号。

## 当前模型的问题

当前 Todo 来自 `Memo.content` 中的 Markdown task line：

```markdown
- [ ] 跟进发布清单
- [x] 处理图标资源
```

前端用 memo ID + 行号生成 `TodoItem.id`，勾选时改写 memo 的对应行。

这个模型简单，但有明显上限：

- 没有稳定任务 ID；行号变化会改变任务身份。
- 没有独立字段承载截止时间、提醒、重复规则、优先级、清单、父子任务等。
- 不能可靠地做今日、未来、日历、已过期、重复任务实例、番茄/专注、完成统计。
- 外部编辑 memo 后，任务定位和状态容易失真。
- 删除 memo 和保留 todo 的语义不清晰。
- 多端同步或冲突合并时，按整篇 memo 改写成本高。

因此目标模型应改为：Todo 有自己的文件、ID 和生命周期；Memo 里的 task line 只是输入入口、展示镜像或引用。

## 领域对象

### Task

Task 是一条可执行事项。

核心字段：

- `id`：稳定 ID，例如 `task_20260609T103012_x7a9b2c4`。
- `title`：任务标题。
- `status`：`open`、`completed`、`cancelled`、`archived`。
- `listId`：所属清单。
- `projectId`：可选，复用当前 Project 或引入 Task Project。
- `priority`：`none`、`low`、`medium`、`high`。
- `tags`：任务标签。
- `contexts`：GTD 场景，例如 `@home`、`@office`、`@phone`。
- `startAt`：开始时间。
- `dueAt`：截止时间。
- `reminders`：提醒规则列表。
- `repeat`：重复规则。
- `parentId`：父任务 ID。
- `subtaskIds`：子任务 ID。
- `source`：来源信息，例如从 memo 抽取。
- `links`：关联 memo、文件、URL、asset。
- `notes`：任务备注，支持 Markdown。
- `createdAt`、`updatedAt`、`completedAt`、`cancelledAt`。

### Task List

Task List 是清单或文件夹中的列表。

字段建议：

- `id`
- `name`
- `color`
- `icon`
- `sortOrder`
- `archived`
- `folderId`

系统内置清单不一定落文件，例如：

- Inbox
- Today
- Next
- Scheduled
- Someday
- Completed

这些更适合作为 Smart List，也就是由查询条件生成的视图。

### Smart List

Smart List 是任务查询视图，不直接持久化任务。

典型视图：

- Inbox：没有 list 或没有明确计划的任务。
- Today：今天开始、今天截止、今天提醒或逾期未完成。
- Next：下一步行动。
- Scheduled：未来有日期的任务。
- Someday：暂不行动。
- Completed：已完成任务。
- Overdue：已过期未完成。

### Reminder

Reminder 是提醒定义，不等于 due date。

示例：

```json
{
  "type": "absolute",
  "at": "2026-06-09T09:30:00+08:00"
}
```

或：

```json
{
  "type": "relative",
  "base": "dueAt",
  "offsetMinutes": -30
}
```

### Repeat Rule

Repeat Rule 定义重复任务的生成规则。

建议使用接近 iCalendar RRULE 的结构，但先保持 JSON 化，避免字符串解析过早复杂化：

```json
{
  "frequency": "weekly",
  "interval": 1,
  "weekdays": ["MO", "WE", "FR"],
  "end": {
    "type": "never"
  }
}
```

重复任务需要区分：

- 模板任务：保存重复规则。
- 实例任务：某一次发生的任务。
- 跳过/改期记录：单次例外。

### Task Source

Task Source 描述任务从哪里来。

从 memo 抽取任务时，应保存：

```json
{
  "type": "memo",
  "memoId": "memo_...",
  "memoPath": "memo/2026/06/memo_....md",
  "line": 12,
  "text": "- [ ] 跟进发布清单"
}
```

注意：`line` 只是来源快照，不再作为任务身份。即使 memo 变化，task 仍然有效。

## 推荐文件持久化方案

推荐采用“任务单文件 + 可重建索引 + 事件日志”的组合。

```text
{vault}/
  tasks/
    open/
      2026/
        06/
          task_20260609T103012_x7a9b2c4.json
    completed/
      2026/
        task_20260608T180210_abcd1234.json
    archived/
  .velo/
    task-lists.json
    task-index.json
    task-events/
      2026-06.jsonl
```

### 为什么不是单个 tasks.json

单文件 JSON 实现快，但不适合复杂 Todo：

- 任意小修改都会改写大文件。
- 冲突合并困难。
- 外部工具不方便按单任务查看和编辑。
- 历史和审计需要额外结构。
- 已完成任务长期增长会拖慢读取。

### 为什么不是只用 Markdown task line

Markdown task line 人类友好，但字段表达能力弱。完整 GTD 需要结构化数据承载状态、日期、重复、提醒、排序和来源关系。

### 为什么选择单任务文件

单任务文件的优点：

- 稳定 ID 和独立生命周期。
- 修改粒度小，适合本地优先和同步。
- 可直接被外部编辑器打开和校验。
- 结构化字段天然适合日期、提醒、重复和子任务。
- 已完成任务可以按年归档，降低活跃读取成本。
- 索引损坏时可以扫描 `tasks/` 重建。

## Task 文件格式

建议使用纯 JSON。Task 只承载结构化行动状态；富文本内容、附件和长记录由关联的 memo/note 承担。

```json
{
  "schemaVersion": 1,
  "id": "task_20260609T103012_x7a9b2c4",
  "title": "跟进发布清单",
  "status": "open",
  "listId": "inbox",
  "projectId": "project_release",
  "priority": "high",
  "tags": ["release"],
  "contexts": ["office"],
  "startAt": "2026-06-09T09:00:00+08:00",
  "dueAt": "2026-06-09T18:00:00+08:00",
  "timezone": "Asia/Shanghai",
  "reminders": [
    {
      "type": "relative",
      "base": "dueAt",
      "offsetMinutes": -30
    }
  ],
  "repeat": {
    "frequency": "none"
  },
  "parentId": "",
  "subtaskIds": [],
  "source": {
    "type": "memo",
    "memoId": "memo_20260609T095500_aabbccdd",
    "memoPath": "memo/2026/06/memo_20260609T095500_aabbccdd.md",
    "line": 12
  },
  "links": [
    {
      "type": "memo",
      "id": "memo_20260609T095500_aabbccdd"
    }
  ],
  "noteRefs": [
    {
      "memoId": "memo_20260609T110000_note",
      "role": "note",
      "sortOrder": 0,
      "createdAt": "2026-06-09T11:00:00+08:00"
    }
  ],
  "createdAt": "2026-06-09T10:30:12+08:00",
  "updatedAt": "2026-06-09T10:30:12+08:00",
  "completedAt": ""
}
```

文件路径建议：

```text
tasks/open/YYYY/MM/{task_id}.json
tasks/completed/YYYY/{task_id}.json
tasks/archived/YYYY/{task_id}.json
```

路径由状态和时间决定，但 task identity 只由 `id` 决定。移动文件不应改变任务身份。

## 索引文件

`.velo/task-index.json` 是性能优化，不是权威数据源。损坏或丢失时可以扫描 `tasks/` 重建。

建议结构：

```json
{
  "schemaVersion": 1,
  "rebuiltAt": "2026-06-09T10:40:00+08:00",
  "tasks": {
    "task_20260609T103012_x7a9b2c4": {
      "path": "tasks/open/2026/06/task_20260609T103012_x7a9b2c4.json",
      "status": "open",
      "title": "跟进发布清单",
      "listId": "inbox",
      "projectId": "project_release",
      "priority": "high",
      "tags": ["release"],
      "startAt": "2026-06-09T09:00:00+08:00",
      "dueAt": "2026-06-09T18:00:00+08:00",
      "updatedAt": "2026-06-09T10:30:12+08:00"
    }
  }
}
```

索引只存列表页、筛选、排序和统计需要的摘要字段。打开任务详情时再读 task 文件。

## 事件日志

`.velo/task-events/YYYY-MM.jsonl` 用于记录关键变更，解决两个问题：

- 复杂功能需要审计和回放，例如完成、延期、重复实例生成。
- 未来同步时，事件日志比只看最终文件更容易合并。

示例：

```jsonl
{"id":"evt_...","taskId":"task_...","type":"created","at":"2026-06-09T10:30:12+08:00"}
{"id":"evt_...","taskId":"task_...","type":"completed","at":"2026-06-09T17:42:00+08:00"}
{"id":"evt_...","taskId":"task_...","type":"rescheduled","from":"2026-06-09","to":"2026-06-10","at":"2026-06-09T18:00:00+08:00"}
```

第一阶段可以先实现 task 文件和索引，事件日志只记录 create/update/complete/delete。后续再让同步和统计依赖它。

## Memo 与 Todo 的关系

### 关系原则

Memo 和 Todo 应该松耦合：

- Memo 可以引用 Task。
- Task 可以链接 Memo。
- Task 状态不依赖 memo 行号。
- Memo 删除不默认删除 Task，除非用户明确选择。
- Task 完成不必须修改 Memo 正文，除非该 memo 中存在 task mirror。

### Memo 中引用 Task

建议引入稳定引用语法：

```markdown
[[task:task_20260609T103012_x7a9b2c4]]
```

也可以支持嵌入卡片：

```markdown
![[task:task_20260609T103012_x7a9b2c4]]
```

### Memo 中的轻量任务入口

保留 Markdown task line 作为快速捕捉入口，但保存 memo 时可以提供“抽取为 Task”的能力。

原文：

```markdown
- [ ] 跟进发布清单 #release
```

抽取后可以变成：

```markdown
- [ ] [[task:task_20260609T103012_x7a9b2c4|跟进发布清单]]
```

这样 memo 保留阅读上下文，真正的任务状态存在 task 文件中。

### Task 回链 Memo

Task JSON 中保存 `source`、`links` 和 `noteRefs`：

```json
{
  "source": {
    "type": "memo",
    "memoId": "memo_...",
    "line": 12
  },
  "links": [
    {
      "type": "memo",
      "id": "memo_..."
    }
  ],
  "noteRefs": [
    {
      "memoId": "memo_...",
      "role": "note",
      "sortOrder": 0
    }
  ]
}
```

这样任务详情页可以展示“来源 memo”和上下文摘要。

### Task Note

Task Note 复用 Memo 的 Markdown 能力。它仍然保存在 `memo/YYYY/MM/*.md`，但 front matter 带上任务归属：

```markdown
---
schemaVersion: 1
id: "memo_..."
kind: "task_note"
taskId: "task_..."
createdAt: "..."
updatedAt: "..."
tags: []
references: []
---
任务执行记录、附件、链接和上下文。

- [ ] 可以在 note 中继续捕捉子任务
```

Task 通过 `noteRefs` 关联多个 note。Note 中的 Markdown todo 行可以抽取为子任务：子任务写入独立 Task JSON，`parentId` 指向父任务，`source` 指向 note 的 memo ID 和行号。

## GTD 能力映射

### 收集

Inbox 是默认入口。任何没有明确清单和日期的任务进入 Inbox。

支持入口：

- 快速新增任务。
- 从 memo task line 抽取。
- 从选中文本创建任务。
- 从文件/链接创建任务。
- 从全局快捷键打开轻量任务窗口。

### 澄清

任务需要支持从 Inbox 分流：

- 立即完成。
- 安排日期。
- 移入清单。
- 关联 Project。
- 标记 Someday。
- 拆分为子任务。

### 组织

组织维度不要只靠一个字段：

- 清单：用户主观分组。
- Project：结果导向的较大目标。
- Tag：横向标签。
- Context：GTD 场景。
- Date：日程维度。
- Priority：执行优先级。

### 回顾

需要从文件和事件日志支持：

- 今日完成数。
- 逾期任务。
- 各清单未完成数量。
- Project 任务进度。
- 重复任务完成情况。
- 已完成归档。

### 执行

执行视图应由查询生成：

- Today：今天要做。
- Next：下一步行动。
- Calendar：按日期展示。
- Focus：只展示选中的一个任务或一个清单。
- Waiting：等待他人。
- Someday：未来可能做。

## API 设计草案

建议新增独立 API，不再复用 `/api/memos/*`。

```text
GET  /api/tasks
GET  /api/tasks/get?id=...
POST /api/tasks/create
POST /api/tasks/update
POST /api/tasks/complete
POST /api/tasks/cancel
POST /api/tasks/delete
POST /api/tasks/move
POST /api/tasks/extract-from-memo

GET  /api/task-lists
POST /api/task-lists/create
POST /api/task-lists/update
POST /api/task-lists/archive

GET  /api/task-index/rebuild
```

`/api/tasks` 默认返回索引摘要，详情按需读取文件。

## 后端模块建议

新增 Go 文件：

```text
internal/desktopapp/task.go
internal/desktopapp/task_markdown.go
internal/desktopapp/task_index.go
internal/desktopapp/task_events.go
internal/desktopapp/routes_tasks.go
```

职责划分：

- `task.go`：领域结构、校验、创建、更新、完成、删除。
- `task_markdown.go`：front matter 读写、路径生成、迁移。
- `task_index.go`：扫描任务文件、维护索引、查询摘要。
- `task_events.go`：追加事件日志。
- `routes_tasks.go`：API 边界。

## 前端模块建议

新增前端领域层：

```text
frontend/src/domain/tasks.js
frontend/src/domain/task-lists.js
frontend/src/domain/task-repository.js
frontend/src/types/tasks.d.ts
```

页面层可以先从现有 Todo view 演进：

- `tasks` 视图替代当前 `todos` 派生视图。
- 保留 memo 内 checkbox 渲染，但优先识别 `[[task:...]]`。
- 新增任务详情侧栏或独立窗口。
- 新增 Today、Inbox、Scheduled、Completed 等 smart list。

## 迁移路径

### 第一阶段：引入 Task 文件，但不破坏现有 Memo

新增 task 文件模型和 API。当前 memo task line 仍按旧逻辑工作。

交付：

- 创建任务。
- 列出任务。
- 完成任务。
- task 文件读写。
- task index 重建。

### 第二阶段：从 Memo 抽取任务

为 memo task line 增加“抽取为任务”操作。

抽取时：

1. 创建 task 文件。
2. 保存 source memo 信息。
3. 可选地把原 task line 替换为 `[[task:...]]` 引用。
4. 刷新 Task view。

### 第三阶段：Task 引用渲染

在 memo markdown 渲染中支持：

```markdown
[[task:task_id]]
![[task:task_id]]
```

渲染为任务卡片，状态从 task 文件读取。

### 第四阶段：GTD 完整能力

逐步加入：

- 清单和文件夹。
- Today / Scheduled / Next / Completed。
- due date、start date、reminder。
- repeat rule。
- priority、context、tag。
- subtasks。
- 事件日志统计。

## 关键不变量

- Task 的身份由 `id` 决定，不由文件路径、标题或 memo 行号决定。
- Task 文件是权威数据源，索引是缓存。
- Memo 引用 Task 时只保存 task ID。
- Task 可以没有来源 memo。
- 删除 Memo 不应隐式删除 Task。
- 删除 Task 时，关联 Memo 中的引用可以保留为失效引用，或由用户选择清理。
- 完成重复任务时，不直接把模板任务关闭，而是生成下一次实例或记录本次完成。
- 所有文件写入继续使用临时文件加 rename 的原子写入策略。

## 推荐决策

建议把 Todo 升级为独立文件模型：

```text
tasks/{status}/YYYY/MM/{task_id}.json
```

并用 `.velo/task-index.json` 做可重建索引，用 `.velo/task-events/*.jsonl` 记录关键变更。Memo 中的 Markdown task line 保留为快速捕捉和展示语法，但复杂 Todo 功能应基于 Task 文件实现。

这个方案和当前项目的 vault、本地优先、Markdown 可读、JSON 元数据、原子写入模式一致，同时为 GTD 能力留下足够空间。
