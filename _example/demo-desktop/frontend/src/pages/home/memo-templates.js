import {
  DEFAULT_VISIBILITY,
  VISIBILITY,
  buildMemoReferenceIndex,
  collectTodos,
  extractTags,
  memoBacklinkCount,
} from "../../domain/memos.js";
import { collectCodeBlocks, collectLinks, collectResources, fileDisplayName } from "../../domain/memo-resources.js";
import { normalizeProjectColor, normalizeProjectFilter, normalizeProjectID } from "../../domain/projects.js";
import {
  calendarWeekdays,
  formatDateKey,
  formatRelativeDate,
  generateCalendarDays,
  memoDateCounts,
  startOfMonth,
} from "./memo-date.js";
import { calendarDayInfo } from "./memo-calendar-info.js";
import { SVG } from "./memo-icons.js";
import {
  compactFileURL,
  inlineMarkdown,
  renderMemoMarkdown,
  renderVSCodeOpenButton,
  safeImageUrl,
  safeUrl,
} from "./memo-markdown.js";
import { escapeAttr, escapeHTML } from "./memo-utils.js";

function detachedMemoWindowTemplate() {
  return `
    <div class="memo-window-shell velo-drag" data-velo-drag>
      <header class="memo-window-titlebar velo-drag" data-velo-drag>
        <div class="memo-window-native-controls" aria-hidden="true"></div>
        <div class="memo-window-drag-region" aria-hidden="true"></div>
        <div class="memo-window-title-actions">
          <button class="memo-window-icon-button velo-no-drag" type="button" data-window-control="toggleFixed" title="悬浮在所有窗口上方" aria-label="悬浮在所有窗口上方">
            ${SVG.pin}
          </button>
          <span class="memo-window-visibility memo-visibility" data-window-visibility hidden></span>
        </div>
      </header>
      <main class="memo-window-body velo-no-drag" data-window-content></main>
      <form class="memo-window-comment-form velo-no-drag" data-window-comment-form>
        <button class="memo-window-comment-tool" type="button" data-window-comment-attach title="添加图片或附件" aria-label="添加图片或附件">
          ${SVG.plus}
        </button>
        <div class="memo-editor-switch memo-window-comment-switch">
          <div class="memo-editor-host memo-window-comment-editor" data-window-comment-editor data-editor-switch-host></div>
          <section class="memo-editor-preview memo-window-comment-preview velo-no-drag" data-window-comment-preview hidden></section>
        </div>
        <button class="memo-window-comment-tool" type="button" data-window-comment-preview-toggle title="预览" aria-label="预览评论" aria-pressed="false">
          ${SVG.eye}
        </button>
        <button class="memo-window-comment-submit" type="submit" data-window-comment-submit title="发送" aria-label="发送评论" disabled>
          ${SVG.send}
        </button>
        <input class="memo-hidden-input" type="file" multiple data-window-comment-file-input />
      </form>
      <div class="memo-toast" data-toast role="status"></div>
    </div>
  `;
}

function detachedMemoCardTemplate(memo, renderContext, options = {}) {
  const tags = extractTags(memo.content);
  const summary = memoCardSummaryTemplate(memo, tags, { interactiveTags: false });
  const backlinks = memoBacklinkCount(renderContext, memo.id);
  const comments = Array.isArray(options.comments) ? options.comments : [];
  const editingCommentId = String(options.editingCommentId || "");
  const expandedCommentIds = options.expandedCommentIds;

  return `
    <article class="memo-card memo-window-card" data-memo-id="${escapeAttr(memo.id)}">
      <header class="memo-card-head memo-window-card-head">
        <div class="memo-author">
          <div class="memo-avatar">U</div>
          <div>
            <div class="memo-author-name">You</div>
            <time datetime="${escapeAttr(memo.createdAt)}">${formatRelativeDate(memo.createdAt)}</time>
          </div>
        </div>
        <div class="memo-card-meta memo-window-card-meta">
          <div class="memo-card-head-actions">
            <button class="memo-action-button" type="button" data-action="copyMemo" title="复制" aria-label="复制">${SVG.copy}</button>
            <button class="memo-action-button" type="button" data-action="copyMemoRef" title="复制引用" aria-label="复制引用">${SVG.link}</button>
          </div>
          ${memo.pinned ? '<span class="memo-pin-label">置顶</span>' : ""}
          ${backlinks ? `<span class="memo-backlink-label">${backlinks} 引用</span>` : ""}
        </div>
      </header>
      <div class="memo-content">${renderMemoMarkdown(memo.content, renderContext)}</div>
      ${summary}
      ${comments.length ? detachedMemoCommentsTemplate(comments, renderContext, editingCommentId, expandedCommentIds) : ""}
    </article>
  `;
}

function detachedMemoCommentsTemplate(comments, renderContext, editingCommentId = "", expandedCommentIds) {
  const commentContext = {
    ...renderContext,
    showLineNumbers: false,
  };
  return `
    <section class="memo-window-comments" aria-label="评论">
      <div class="memo-window-comments-title">评论</div>
      <div class="memo-comment-list">${comments.map((comment) => detachedMemoCommentTemplate(comment, commentContext, editingCommentId, expandedCommentIds)).join("")}</div>
    </section>
  `;
}

function detachedMemoCommentTemplate(comment, renderContext, editingCommentId = "", expandedCommentIds) {
  const time = comment.updatedAt || comment.createdAt;
  const editing = comment.id === editingCommentId;
  const expanded = expandedCommentIds && typeof expandedCommentIds.has === "function" ? expandedCommentIds.has(comment.id) : false;
  const expandLabel = expanded ? "收起" : "展开";
  const content = renderMemoCommentContent(comment, renderContext);
  return `
    <article class="memo-comment memo-window-comment ${editing ? "is-editing" : ""}" data-comment-id="${escapeAttr(comment.id)}">
      <header class="memo-comment-head">
        <div class="memo-avatar memo-comment-avatar">U</div>
        <div>
          <div class="memo-comment-author">You</div>
          <time datetime="${escapeAttr(time)}">${formatRelativeDate(time)}</time>
        </div>
      </header>
      ${
        editing
          ? `
            <div class="memo-window-comment-edit">
              <div class="memo-editor-switch">
                <div class="memo-editor-host is-inline memo-window-comment-edit-host" data-window-comment-edit-host data-editor-switch-host></div>
                <section class="memo-editor-preview memo-window-comment-edit-preview" data-window-comment-edit-preview hidden></section>
              </div>
              <div class="memo-window-comment-edit-actions">
                <button class="memo-secondary-button" type="button" data-window-comment-action="preview" aria-pressed="false">${SVG.eye}<span>预览</span></button>
                <button class="memo-secondary-button" type="button" data-window-comment-action="cancel">${SVG.x}<span>取消</span></button>
                <button class="memo-primary-button" type="button" data-window-comment-action="save">${SVG.check}<span>保存</span></button>
              </div>
            </div>
          `
          : `
            <div class="memo-window-comment-bubble">
              <div class="memo-window-comment-hover-actions" aria-label="评论操作">
                <button class="memo-action-button" type="button" data-window-comment-action="edit" title="编辑评论" aria-label="编辑评论">${SVG.edit}</button>
                <button class="memo-action-button is-danger" type="button" data-window-comment-action="delete" title="删除评论" aria-label="删除评论">${SVG.trash}</button>
              </div>
              <div class="memo-window-comment-collapse ${expanded ? "is-expanded" : "is-collapsed"}" data-window-comment-collapse>
                <div class="memo-content memo-comment-content">${content}</div>
                <button class="memo-expand-button memo-window-comment-expand-button" type="button" data-window-comment-action="toggleExpand" aria-expanded="${expanded ? "true" : "false"}" title="${expandLabel}">
                  <span>${expandLabel}</span>
                  ${SVG.chevronDown}
                </button>
              </div>
            </div>
          `
      }
    </article>
  `;
}

function detachedMemoRenderContext(state, sourceId, options = {}) {
  const index = state.memoRefIndex || buildMemoReferenceIndex(state.memos);
  state.memoRefIndex = index;
  return {
    depth: options.depth || 0,
    index,
    maxDepth: options.maxDepth || 2,
    readonly: Boolean(options.readonly),
    editorSettings: state.editorSettings,
    showLineNumbers: options.showLineNumbers !== false,
    sourceId: sourceId || "",
    stack: options.stack || (sourceId ? [sourceId] : []),
  };
}

function activeViewMeta(view) {
  const metas = {
    files: {
      hideComposer: true,
      searchPlaceholder: "搜索文件、图片或来源 memo",
      subtitle: "从所有 memo 中汇总文件和图片",
      title: "文件",
    },
    codeblocks: {
      hideComposer: true,
      searchPlaceholder: "搜索代码片段、别名、命令或来源 memo",
      subtitle: "标记片段优先，未标记代码块沉底",
      title: "代码片段",
    },
    links: {
      hideComposer: true,
      searchPlaceholder: "搜索链接或来源 memo",
      subtitle: "从所有 memo 中汇总超链接",
      title: "超链接",
    },
    clipboard: {
      hideComposer: true,
      searchPlaceholder: "搜索当前粘贴板内容",
      subtitle: "显示当前粘贴板的文本、链接或图片",
      title: "粘贴板",
    },
    memos: {
      hideComposer: false,
      searchPlaceholder: "搜索 memos",
      subtitle: "捕捉、整理、回看",
      title: "Inbox",
    },
    todos: {
      hideComposer: true,
      searchPlaceholder: "搜索任务、清单或上下文",
      subtitle: "Inbox、Today、Scheduled 与任务 notes",
      title: "GTD",
    },
    items: {
      hideComposer: true,
      searchPlaceholder: "搜索开放事项、标签或决策",
      subtitle: "像 Issue 一样管理 open loops",
      title: "Open Loops",
    },
    milestones: {
      hideComposer: true,
      searchPlaceholder: "搜索阶段目标",
      subtitle: "像 Milestone 一样管理阶段收敛",
      title: "Milestones",
    },
  };
  return metas[view] || metas.memos;
}

function shellTemplate() {
  return `
    <div class="memo-shell">
      <aside class="memo-sidebar" aria-label="Memo navigation">
        <div class="memo-brand">
          <div class="memo-brand-mark">M</div>
          <div>
            <div class="memo-brand-title">Memos</div>
            <div class="memo-brand-subtitle">Local workspace</div>
          </div>
        </div>
        <nav class="memo-nav" aria-label="Memo filters">
          ${filterButtonTemplate("all", "全部", SVG.hash)}
          ${filterButtonTemplate("pinned", "置顶", SVG.pin)}
          ${filterButtonTemplate("private", "仅自己", SVG.lock)}
          ${filterButtonTemplate("public", "公开", SVG.globe)}
          ${filterButtonTemplate("archive", "归档", SVG.archive)}
        </nav>
        <div class="memo-sidebar-section">
          <div class="memo-sidebar-heading">
            <span>Project</span>
            <span data-project-summary></span>
          </div>
          <div class="memo-project-list" data-project-list></div>
          <button class="memo-nav-button memo-project-create" type="button" data-action="createProject">
            ${SVG.plus}
            <span>新建 Project</span>
          </button>
        </div>
        <div class="memo-sidebar-section">
          <div class="memo-sidebar-heading">
            <span>聚合</span>
          </div>
          <nav class="memo-nav memo-collection-nav" aria-label="Memo collections">
            ${viewNavButtonTemplate("todos", "代办", SVG.check, "data-todo-nav-count")}
            ${viewNavButtonTemplate("items", "事项", SVG.hash, "data-item-nav-count")}
            ${viewNavButtonTemplate("milestones", "里程碑", SVG.clock, "data-milestone-nav-count")}
            ${viewNavButtonTemplate("links", "超链接", SVG.link, "data-link-nav-count")}
            ${viewNavButtonTemplate("codeblocks", "代码片段", SVG.code, "data-code-nav-count")}
            ${viewNavButtonTemplate("files", "文件", SVG.paperclip, "data-file-nav-count")}
            ${viewNavButtonTemplate("clipboard", "粘贴板", SVG.copy, "data-clipboard-nav-count")}
          </nav>
        </div>
        <div class="memo-sidebar-section">
          <div class="memo-sidebar-heading">
            <span>标签</span>
            <span data-tag-summary></span>
          </div>
          <div class="memo-tag-list" data-tag-list></div>
        </div>
        <div class="memo-sidebar-footer">
          <button class="memo-nav-button memo-settings-button" type="button" data-action="openSettings">
            ${SVG.settings}
            <span>设置</span>
          </button>
        </div>
      </aside>

      <main class="memo-main">
        <header class="memo-topbar">
          <div>
            <h1 data-main-title>Inbox</h1>
            <p data-main-subtitle>捕捉、整理、回看</p>
          </div>
          <div class="memo-topbar-actions">
            <button class="memo-icon-text-button" type="button" data-action="openSlimMemos" title="打开精简版">
              ${SVG.list}
              <span>精简版</span>
            </button>
            <button class="memo-icon-text-button" type="button" data-action="sortMemos" title="排序">
              ${SVG.sort}
              <span>排序</span>
            </button>
          </div>
        </header>

        <section class="memo-composer" aria-label="Create memo" data-composer>
          <div class="memo-composer-head">
            <div class="memo-tool-group memo-tool-group-head" aria-label="命令">
              ${toolButtonTemplate("bold", "粗体", SVG.bold)}
              ${toolButtonTemplate("italic", "斜体", SVG.italic)}
              ${toolButtonTemplate("code", "代码", SVG.code)}
              ${toolButtonTemplate("list", "列表", SVG.list)}
              ${toolButtonTemplate("checklist", "任务", SVG.check)}
              ${toolButtonTemplate("tag", "标签", SVG.hash)}
              ${toolButtonTemplate("link", "链接", SVG.link)}
              ${toolButtonTemplate("image", "图片", SVG.image)}
              ${toolButtonTemplate("attach", "附件", SVG.paperclip)}
              ${toolButtonTemplate("date", "时间", SVG.clock)}
            </div>
            <label class="memo-select-wrap">
              <span class="memo-select-icon">${SVG.hash}</span>
              <select data-project-select aria-label="Project">
                ${projectOptionsTemplate([], "")}
              </select>
              ${SVG.chevronDown}
            </label>
            <label class="memo-select-wrap">
              <span class="memo-select-icon">${SVG.lock}</span>
              <select data-visibility-select aria-label="可见性">
                ${visibilityOptionsTemplate(DEFAULT_VISIBILITY)}
              </select>
              ${SVG.chevronDown}
            </label>
          </div>
          <div class="memo-editor-switch memo-composer-switch">
            <div class="memo-editor-host" data-composer-host data-editor-switch-host></div>
            <section class="memo-editor-preview memo-composer-preview" data-composer-preview hidden></section>
          </div>
          <div class="memo-composer-toolbar">
            <div class="memo-composer-status-line">
              <span data-composer-vim-status></span>
              <span data-composer-status></span>
            </div>
            <div class="memo-composer-actions">
              <button class="memo-secondary-button" type="button" data-action="toggleComposerPreview" aria-pressed="false">
                ${SVG.eye}
                <span>预览</span>
              </button>
              <button class="memo-primary-button" type="button" data-action="createMemo">
                ${SVG.send}
                <span>发布</span>
              </button>
            </div>
          </div>
          <input class="memo-hidden-input" type="file" multiple data-attach-input />
        </section>

        <section class="memo-feed-tools" aria-label="Memo search">
          <label class="memo-search">
            ${SVG.search}
            <input type="search" placeholder="搜索 memos" data-search-input />
          </label>
          <button class="memo-clear-button" type="button" data-action="clearFilters">重置</button>
          <span class="memo-feed-count" data-feed-count></span>
        </section>

        <section class="memo-list" data-memo-list aria-label="Memo list"></section>
      </main>

      <aside class="memo-inspector" aria-label="Memo details">
        <section class="memo-inspector-section">
          <div class="memo-inspector-title">日历</div>
          <div class="memo-calendar" data-calendar></div>
        </section>
        <section class="memo-inspector-section">
          <div class="memo-inspector-title">置顶</div>
          <div class="memo-pinned-list" data-pinned-list></div>
        </section>
        <section class="memo-inspector-section">
          <div class="memo-inspector-title">概览</div>
          <div class="memo-stats" data-stats></div>
        </section>
      </aside>
      <div class="memo-command-palette" data-memo-search-palette hidden>
        <div class="memo-command-panel" role="dialog" aria-modal="true" aria-label="搜索 memo 和代码片段">
          <label class="memo-command-search">
            ${SVG.search}
            <input type="search" data-memo-search-input placeholder="搜索 memo / 代码片段" autocomplete="off" />
          </label>
          <div class="memo-command-results" data-memo-search-results role="listbox"></div>
        </div>
      </div>
      <section class="memo-clipboard-card" data-clipboard-card hidden></section>
      <div class="memo-toast" data-toast role="status"></div>
    </div>
  `;
}

function filterButtonTemplate(filter, label, icon) {
  return `
    <button class="memo-nav-button" type="button" data-filter="${filter}">
      ${icon}
      <span>${label}</span>
    </button>
  `;
}

function projectFilterTemplate(filter, label, count, color, activeFilter) {
  const active = normalizeProjectFilter(filter) === normalizeProjectFilter(activeFilter);
  const swatch = color ? `<span class="memo-project-dot" style="--project-color: ${escapeAttr(color)}"></span>` : SVG.hash;
  return `
    <button class="memo-nav-button memo-project-filter ${active ? "is-active" : ""}" type="button" data-project-filter="${escapeAttr(filter)}">
      ${swatch}
      <span>${escapeHTML(label)}</span>
      <strong>${count ? escapeHTML(String(count)) : ""}</strong>
    </button>
  `;
}

function projectOptionsTemplate(projects, selected) {
  const selectedID = normalizeProjectID(selected);
  const activeProjects = Array.isArray(projects) ? projects : [];
  const options = ['<option value="">未归属</option>'].concat(
    activeProjects
      .filter((project) => !project.archived)
      .map((project) => `<option value="${escapeAttr(project.id)}" ${project.id === selectedID ? "selected" : ""}>${escapeHTML(project.name)}</option>`),
  );
  if (!selectedID) options[0] = '<option value="" selected>未归属</option>';
  return options.join("");
}

function projectBadgeTemplate(projectId, projects = []) {
  const id = normalizeProjectID(projectId);
  if (!id) return '<span class="memo-project-badge">未归属</span>';
  const project = (Array.isArray(projects) ? projects : []).find((item) => item && item.id === id);
  const label = project ? project.name : "未知 Project";
  const color = normalizeProjectColor(project && project.color);
  return `<span class="memo-project-badge" style="--project-color: ${escapeAttr(color)}">${escapeHTML(label)}</span>`;
}

function viewNavButtonTemplate(view, label, icon, countAttr) {
  return `
    <button class="memo-nav-button" type="button" data-view="${view}">
      ${icon}
      <span>${label}</span>
      <strong ${countAttr}></strong>
    </button>
  `;
}

function toolButtonTemplate(command, label, icon) {
  return `
    <button class="memo-tool-button" type="button" data-command="${command}" title="${label}" aria-label="${label}">
      ${icon}
    </button>
  `;
}

function statTemplate(label, value) {
  return `
    <div class="memo-stat">
      <span>${label}</span>
      <strong>${value}</strong>
    </div>
  `;
}

function calendarTemplate(monthDate, memos, selectedDate, weekStart) {
  const month = startOfMonth(monthDate);
  const counts = memoDateCounts(memos);
  const weekdays = calendarWeekdays(weekStart);
  const days = generateCalendarDays(month, weekStart);
  const todayKey = formatDateKey(new Date());

  return `
    <div class="memo-calendar-head">
      <button class="memo-calendar-nav" type="button" data-calendar-action="prevMonth" title="上个月" aria-label="上个月">
        ${SVG.chevronLeft}
      </button>
      <div class="memo-calendar-title">
        <strong>${month.getFullYear()} 年 ${month.getMonth() + 1} 月</strong>
        <span>${selectedDate || "未选择日期"}</span>
      </div>
      <button class="memo-calendar-nav" type="button" data-calendar-action="nextMonth" title="下个月" aria-label="下个月">
        ${SVG.chevronRight}
      </button>
    </div>
    <div class="memo-calendar-toolbar ${selectedDate ? "" : "is-single"}">
      <button class="memo-calendar-today" type="button" data-calendar-action="today">今天</button>
      ${selectedDate ? '<button class="memo-calendar-clear" type="button" data-calendar-action="clearDate">清除</button>' : ""}
    </div>
    <div class="memo-calendar-weekdays">
      ${weekdays.map((day) => `<span>${day}</span>`).join("")}
    </div>
    <div class="memo-calendar-grid">
      ${days
        .map((day) => {
          const count = counts.get(day.key) || 0;
          const info = calendarDayInfo(day.date);
          const classes = [
            "memo-calendar-day",
            day.inMonth ? "" : "is-outside",
            day.key === todayKey ? "is-today" : "",
            day.key === selectedDate ? "is-selected" : "",
            count ? "has-memo" : "",
            info.festivalLabel ? "has-festival" : "",
            info.holidayStatus ? "is-" + info.holidayStatus : "",
          ]
            .filter(Boolean)
            .join(" ");
          const ariaLabel = [day.key, info.title, count ? `${count} 条 memo` : ""].filter(Boolean).join("，");
          return `
            <button
              class="${classes}"
              type="button"
              data-calendar-date="${escapeAttr(day.key)}"
              aria-label="${escapeAttr(ariaLabel)}"
              title="${escapeAttr(ariaLabel)}"
            >
              <span class="memo-calendar-solar">${day.date.getDate()}</span>
              <span class="memo-calendar-lunar">${escapeHTML(info.lunarLabel)}</span>
              ${info.festivalLabel ? `<em class="memo-calendar-festival">${escapeHTML(info.festivalLabel)}</em>` : ""}
              ${info.holidayBadge ? `<span class="memo-calendar-holiday-badge" aria-hidden="true">${info.holidayBadge}</span>` : ""}
              ${count ? `<strong class="memo-calendar-memo-count">${count}</strong>` : ""}
            </button>
          `;
        })
        .join("")}
    </div>
  `;
}

function memoTemplate(memo, editingId, renderContext, expanded = false, projects = [], options = {}) {
  const visibility = VISIBILITY[memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
  const tags = extractTags(memo.content);
  const summary = memoCardSummaryTemplate(memo, tags, { interactiveTags: true });
  const archived = memo.archived;
  const editing = memo.id === editingId;
  const backlinks = memoBacklinkCount(renderContext, memo.id);
  const expandLabel = expanded ? "收起" : "展开";
  const projectBadge = projectBadgeTemplate(memo.projectId, projects);
  const comments = Array.isArray(options.comments) ? options.comments : [];
  const commenting = options.commenting === true;
  const editingCommentId = String(options.editingCommentId || "");
  const sourceEditing = options.sourceEditing === true;
  const sourceDraft = String(options.sourceDraft || "");

  return `
    <article class="memo-card ${memo.pinned ? "is-pinned" : ""} ${archived ? "is-archived" : ""}" data-memo-id="${escapeAttr(memo.id)}">
      <header class="memo-card-head">
        <div class="memo-author">
          <div class="memo-avatar">U</div>
          <div>
            <div class="memo-author-name">You</div>
            <time datetime="${escapeAttr(memo.createdAt)}">${formatRelativeDate(memo.createdAt)}</time>
          </div>
        </div>
        <div class="memo-card-meta">
          <div class="memo-card-head-actions">
            <button class="memo-action-button" type="button" data-action="togglePin" title="${memo.pinned ? "取消置顶" : "置顶"}">${SVG.pin}</button>
            <button class="memo-action-button" type="button" data-action="detachMemo" title="分离为窗口">${SVG.external}</button>
            <button class="memo-action-button" type="button" data-action="copyMemo" title="复制">${SVG.copy}</button>
          </div>
          <span class="memo-visibility">${SVG[visibility.icon]} ${visibility.label}</span>
          ${projectBadge}
          ${memo.pinned ? '<span class="memo-pin-label">置顶</span>' : ""}
          ${backlinks ? `<span class="memo-backlink-label">${backlinks} 引用</span>` : ""}
        </div>
      </header>
      ${
        editing
          ? editTemplate(memo, projects)
          : sourceEditing
            ? memoSourceTemplate(sourceDraft)
            : `
            <div class="memo-list-collapse ${expanded ? "is-expanded" : "is-collapsed"}" data-memo-collapse>
              <div class="memo-content">${renderMemoMarkdown(memo.content, renderContext)}</div>
              <button class="memo-expand-button" type="button" data-action="toggleMemoExpand" aria-expanded="${expanded ? "true" : "false"}" title="${expandLabel}">
                <span>${expandLabel}</span>
                ${SVG.chevronDown}
              </button>
            </div>
            ${summary}
          `
      }
      ${!editing && !sourceEditing && (comments.length || commenting) ? memoCommentSectionTemplate(comments, commenting, renderContext, editingCommentId) : ""}
      <footer class="memo-card-actions">
        <button class="memo-action-button" type="button" data-action="copyMemoRef" title="复制引用">${SVG.link}</button>
        <button class="memo-action-button" type="button" data-action="commentMemo" title="评论">
          ${SVG.comment}
          ${comments.length ? `<span class="memo-action-count">${comments.length}</span>` : ""}
        </button>
        <button class="memo-action-button" type="button" data-action="editMemo" title="编辑">${SVG.edit}</button>
        <button class="memo-action-button" type="button" data-action="editMemoSource" title="编辑源数据" aria-label="编辑源数据">${SVG.code}</button>
        ${
          archived
            ? `<button class="memo-action-button" type="button" data-action="restoreMemo" title="恢复">${SVG.restore}</button>`
            : `<button class="memo-action-button" type="button" data-action="archiveMemo" title="归档">${SVG.archive}</button>`
        }
        <button class="memo-action-button is-danger" type="button" data-action="deleteMemo" title="删除">${SVG.trash}</button>
      </footer>
    </article>
  `;
}

function memoSourceTemplate(sourceDraft) {
  return `
    <div class="memo-inline-editor memo-source-editor">
      <textarea class="memo-source-textarea" data-memo-source-yaml spellcheck="false" rows="10">${escapeHTML(sourceDraft)}</textarea>
      <div class="memo-inline-actions">
        <div class="memo-inline-status-line">YAML frontmatter</div>
        <button class="memo-secondary-button" type="button" data-action="cancelMemoSource">${SVG.x}<span>取消</span></button>
        <button class="memo-primary-button" type="button" data-action="saveMemoSource">${SVG.check}<span>保存</span></button>
      </div>
    </div>
  `;
}

function memoCardSummaryTemplate(memo, tags, options = {}) {
  const stats = memoCardStatItems(memo);
  const tagMarkup = memoCardTagsTemplate(tags, options);
  if (!stats.length && !tagMarkup) return "";

  return `
    <div class="memo-card-summary">
      ${stats.length ? `<div class="memo-card-stats">${stats.map(memoCardStatTemplate).join("")}</div>` : ""}
      ${tagMarkup}
    </div>
  `;
}

function memoCardStatItems(memo) {
  const content = String((memo && memo.content) || "");
  const list = [{ label: `${Array.from(content).length} 字符` }];
  const resources = collectResources([memo]);
  const files = resources.filter((resource) => resource.type === "file").length;
  const images = resources.filter((resource) => resource.type === "image").length;
  const todos = collectTodos([memo]).length;
  const codeBlocks = collectCodeBlocks([memo]).length;
  const links = collectLinks([memo]).length;

  if (files) list.push({ label: `${files} 文件` });
  if (images) list.push({ label: `${images} 图片` });
  if (todos) list.push({ label: `${todos} 代办` });
  if (codeBlocks) list.push({ label: `${codeBlocks} 代码块` });
  if (links) list.push({ label: `${links} 链接` });

  return list;
}

function memoCardStatTemplate(item) {
  return `<span class="memo-card-stat">${escapeHTML(item.label)}</span>`;
}

function memoCardTagsTemplate(tags, options = {}) {
  if (!Array.isArray(tags) || !tags.length) return "";
  const interactive = options.interactiveTags === true;
  const items = tags.map(function (tag) {
    return interactive
      ? `<button type="button" data-tag="${escapeAttr(tag)}">#${escapeHTML(tag)}</button>`
      : `<span>#${escapeHTML(tag)}</span>`;
  });
  return `<div class="memo-card-tags">${items.join("")}</div>`;
}

function memoCommentSectionTemplate(comments, commenting, renderContext, editingCommentId = "") {
  const commentContext = {
    ...renderContext,
    showLineNumbers: false,
  };
  return `
    <section class="memo-comments" aria-label="评论">
      ${comments.length ? `<div class="memo-comment-list">${comments.map((comment) => memoCommentTemplate(comment, commentContext, editingCommentId)).join("")}</div>` : ""}
      ${
        commenting
          ? `
            <div class="memo-comment-editor">
              <div class="memo-editor-switch">
                <div class="memo-editor-host is-inline" data-comment-host data-editor-switch-host></div>
                <section class="memo-editor-preview memo-comment-preview" data-comment-preview hidden></section>
              </div>
              <div class="memo-inline-actions memo-comment-actions">
                <div class="memo-inline-status-line" data-comment-vim-status></div>
                <button class="memo-secondary-button" type="button" data-action="toggleCommentPreview" aria-pressed="false">${SVG.eye}<span>预览</span></button>
                <button class="memo-secondary-button" type="button" data-action="cancelComment">${SVG.x}<span>取消</span></button>
                <button class="memo-primary-button" type="button" data-action="saveComment">${SVG.check}<span>评论</span></button>
              </div>
            </div>
          `
          : ""
      }
    </section>
  `;
}

function memoCommentTemplate(comment, renderContext, editingCommentId = "") {
  const time = comment.updatedAt || comment.createdAt;
  const editing = comment.id === editingCommentId;
  const content = renderMemoCommentContent(comment, renderContext);
  return `
    <article class="memo-comment ${editing ? "is-editing" : ""}" data-comment-id="${escapeAttr(comment.id)}">
      <header class="memo-comment-head">
        <div class="memo-avatar memo-comment-avatar">U</div>
        <div>
          <div class="memo-comment-author">You</div>
          <time datetime="${escapeAttr(time)}">${formatRelativeDate(time)}</time>
        </div>
      </header>
      ${
        editing
          ? `
            <div class="memo-comment-edit">
              <div class="memo-editor-switch">
                <div class="memo-editor-host is-inline memo-comment-edit-host" data-comment-edit-host data-editor-switch-host></div>
                <section class="memo-editor-preview memo-comment-edit-preview" data-comment-edit-preview hidden></section>
              </div>
              <div class="memo-inline-actions memo-comment-edit-actions">
                <div class="memo-inline-status-line" data-comment-edit-vim-status></div>
                <button class="memo-secondary-button" type="button" data-action="toggleCommentEditPreview" aria-pressed="false">${SVG.eye}<span>预览</span></button>
                <button class="memo-secondary-button" type="button" data-action="cancelCommentEdit">${SVG.x}<span>取消</span></button>
                <button class="memo-primary-button" type="button" data-action="saveCommentEdit">${SVG.check}<span>保存</span></button>
              </div>
            </div>
          `
          : `
            <div class="memo-comment-bubble">
              <div class="memo-comment-hover-actions" aria-label="评论操作">
                <button class="memo-action-button" type="button" data-action="editComment" title="编辑评论" aria-label="编辑评论">${SVG.edit}</button>
                <button class="memo-action-button is-danger" type="button" data-action="deleteComment" title="删除评论" aria-label="删除评论">${SVG.trash}</button>
              </div>
              <div class="memo-content memo-comment-content">${content}</div>
            </div>
          `
      }
    </article>
  `;
}

function renderMemoCommentContent(comment, renderContext) {
  try {
    return renderMemoMarkdown(comment.content || "", renderContext);
  } catch (err) {
    return `<p>${escapeHTML(comment.content || "")}</p>`;
  }
}

function todoGroupTemplate(label, todos, renderContextFor, projects = []) {
  if (!todos.length) return "";
  return `
    <section class="memo-todo-group" aria-label="${escapeAttr(label)}">
      <div class="memo-todo-group-head">
        <span>${escapeHTML(label)}</span>
        <strong>${todos.length}</strong>
      </div>
      ${todos.map((todo) => todoTemplate(todo, renderContextFor ? renderContextFor(todo) : {}, projects)).join("")}
    </section>
  `;
}

function todoTemplate(todo, renderContext, projects = []) {
  const visibility = VISIBILITY[todo.memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
  const tags = extractTags(todo.memo.content);
  const projectBadge = projectBadgeTemplate(todo.memo.projectId, projects);
  return `
    <article class="memo-todo-card ${todo.checked ? "is-complete" : ""}" data-memo-id="${escapeAttr(todo.memoId)}">
      <div class="memo-todo-check">
        <input type="checkbox" data-task-line="${todo.lineIndex}" ${todo.checked ? "checked" : ""} />
        <span>${inlineMarkdown(todo.text, renderContext)}</span>
      </div>
      <div class="memo-todo-source">
        ${sourceMemoMarkerTemplate(todo.memoId)}
        <div class="memo-todo-meta">
          <time datetime="${escapeAttr(todo.memo.createdAt)}">${formatRelativeDate(todo.memo.createdAt)}</time>
          ${projectBadge}
          <span>${SVG[visibility.icon]} ${visibility.label}</span>
          ${tags.slice(0, 3).map((tag) => `<span>#${escapeHTML(tag)}</span>`).join("")}
        </div>
      </div>
    </article>
  `;
}

function taskWorkspaceTemplate(options) {
  const state = options || {};
  const filters = [
    ["inbox", "Inbox"],
    ["today", "Today"],
    ["overdue", "已过期"],
    ["scheduled", "Scheduled"],
    ["next", "Next"],
    ["completed", "Completed"],
    ["all", "All"],
  ];
  return `
    <section class="memo-task-workspace">
      <form class="memo-task-create" data-task-create-form>
        <input name="title" type="text" placeholder="添加任务到 Inbox" autocomplete="off" />
        <select name="priority" aria-label="优先级">
          <option value="none">无优先级</option>
          <option value="low">低</option>
          <option value="medium">中</option>
          <option value="high">高</option>
        </select>
        <input name="dueAt" type="date" aria-label="截止日期" />
        <button class="memo-primary-button" type="submit">${SVG.plus}<span>添加</span></button>
      </form>
      <div class="memo-task-tabs" role="tablist" aria-label="Task filters">
        ${filters
          .map(([value, label]) => `
            <button class="memo-task-tab ${state.filter === value ? "is-active" : ""}" type="button" data-task-filter="${value}">
              <span>${label}</span>
              <strong>${state.counts && state.counts[value] ? state.counts[value] : ""}</strong>
            </button>
          `)
          .join("")}
      </div>
    </section>
  `;
}

function taskGroupTemplate(label, tasks, context) {
  if (!tasks.length) return "";
  return `
    <section class="memo-todo-group memo-task-group" aria-label="${escapeAttr(label)}">
      <div class="memo-todo-group-head">
        <span>${escapeHTML(label)}</span>
        <strong>${tasks.length}</strong>
      </div>
      ${tasks.map((task) => taskCardTemplate(task, context || {})).join("")}
    </section>
  `;
}

function taskCardTemplate(task, context) {
  const projects = (context && context.projects) || [];
  const projectBadge = projectBadgeTemplate(task.projectId, projects);
  const priority = task.priority || "none";
  const complete = task.status === "completed";
  const dueLabel = task.dueAt ? formatTaskDate(task.dueAt) : "";
  const startLabel = task.startAt ? formatTaskDate(task.startAt) : "";
  const completedLabel = complete && task.completedAt ? formatTaskDateTime(task.completedAt) : "";
  const sourceMemoId = taskLinkedMemoId(task);
  return `
    <article class="memo-task-card ${complete ? "is-complete" : ""} is-priority-${escapeAttr(priority)}" data-task-id="${escapeAttr(task.id)}">
      <label class="memo-task-check">
        <input type="checkbox" data-task-complete ${complete ? "checked" : ""} />
        <span></span>
      </label>
      <div class="memo-task-body">
        <div class="memo-task-title-row">
          <strong>${escapeHTML(task.title)}</strong>
          <span class="memo-task-priority">${taskPriorityLabel(priority)}</span>
        </div>
        <div class="memo-task-meta">
          ${projectBadge}
          <span>${escapeHTML(task.listId || "inbox")}</span>
          ${task.parentId ? `<span>子任务</span>` : ""}
          ${dueLabel ? `<time datetime="${escapeAttr(task.dueAt)}">截止 ${escapeHTML(dueLabel)}</time>` : ""}
          ${startLabel ? `<time datetime="${escapeAttr(task.startAt)}">开始 ${escapeHTML(startLabel)}</time>` : ""}
          ${completedLabel ? `<time datetime="${escapeAttr(task.completedAt)}">完成 ${escapeHTML(completedLabel)}</time>` : ""}
          ${task.noteCount ? `<span>${task.noteCount} notes</span>` : ""}
          ${task.subtaskCount ? `<span>${task.subtaskCount} subtasks</span>` : ""}
          ${sourceMemoId ? sourceMemoMarkerTemplate(sourceMemoId) : ""}
          ${(task.contexts || []).slice(0, 3).map((item) => `<span>@${escapeHTML(item)}</span>`).join("")}
          ${(task.tags || []).slice(0, 3).map((tag) => `<span>#${escapeHTML(tag)}</span>`).join("")}
        </div>
      </div>
      <div class="memo-task-actions">
        <button class="memo-action-button" type="button" data-action="addTaskNote" title="添加 note">${SVG.edit}</button>
        <button class="memo-action-button" type="button" data-action="copyTaskRef" title="复制引用">${SVG.link}</button>
      </div>
    </article>
  `;
}

function taskLinkedMemoId(task) {
  const source = task && task.source ? task.source : {};
  const memoId = String(source.memoId || "").trim();
  return memoId;
}

function sourceMemoMarkerTemplate(memoId) {
  return `
    <button class="memo-source-marker" type="button" data-action="openSourceMemo" data-memo-id="${escapeAttr(memoId)}" title="有关联 memo" aria-label="有关联 memo">
      ${SVG.link}
    </button>
  `;
}

function emptyTasksTemplate() {
  return `
    <div class="memo-empty-state">
      <div class="memo-empty-icon">${SVG.check}</div>
      <h2>没有匹配的任务</h2>
      <button class="memo-secondary-button" type="button" data-action="clearFilters">查看全部</button>
    </div>
  `;
}

function formatTaskDate(value) {
  const date = taskDateValue(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString();
}

function formatTaskDateTime(value) {
  const date = taskDateValue(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString([], {
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    month: "numeric",
  });
}

function taskDateValue(value) {
  const raw = String(value || "").trim();
  const dateOnly = raw.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (dateOnly) {
    return new Date(Number(dateOnly[1]), Number(dateOnly[2]) - 1, Number(dateOnly[3]));
  }
  return new Date(raw);
}

function taskPriorityLabel(priority) {
  switch (priority) {
    case "high":
      return "高";
    case "medium":
      return "中";
    case "low":
      return "低";
    default:
      return "无";
  }
}

function gtdItemWorkspaceTemplate(options) {
  const milestones = (options && options.milestones) || [];
  return `
    <section class="memo-task-workspace">
      <form class="memo-task-create" data-gtd-item-create-form>
        <input name="title" type="text" placeholder="捕捉开放事项、bug、想法或问题" autocomplete="off" />
        <select name="type" aria-label="事项类型">
          <option value="idea">想法</option>
          <option value="feature">功能</option>
          <option value="bug">Bug</option>
          <option value="question">问题</option>
          <option value="chore">杂项</option>
        </select>
        <select name="milestoneId" aria-label="里程碑">
          <option value="">无里程碑</option>
          ${milestones.filter((item) => item.status !== "completed" && item.status !== "cancelled").map((item) => `<option value="${escapeAttr(item.id)}">${escapeHTML(item.title)}</option>`).join("")}
        </select>
        <button class="memo-primary-button" type="submit">${SVG.plus}<span>添加</span></button>
      </form>
    </section>
  `;
}

function gtdItemGroupTemplate(label, items, context) {
  if (!items.length) return "";
  return `
    <section class="memo-todo-group memo-task-group" aria-label="${escapeAttr(label)}">
      <div class="memo-todo-group-head">
        <span>${escapeHTML(label)}</span>
        <strong>${items.length}</strong>
      </div>
      ${items.map((item) => gtdItemCardTemplate(item, context || {})).join("")}
    </section>
  `;
}

function gtdItemCardTemplate(item, context) {
  const milestone = ((context && context.milestones) || []).find((entry) => entry.id === item.milestoneId);
  const projectBadge = projectBadgeTemplate(item.projectId, (context && context.projects) || []);
  const closed = item.status === "closed" || item.status === "resolved";
  return `
    <article class="memo-task-card ${closed ? "is-complete" : ""} is-priority-none" data-gtd-item-id="${escapeAttr(item.id)}">
      <label class="memo-task-check">
        <input type="checkbox" data-gtd-item-complete ${closed ? "checked" : ""} />
        <span></span>
      </label>
      <div class="memo-task-body">
        <div class="memo-task-title-row">
          <strong>${escapeHTML(item.title)}</strong>
          <span class="memo-task-priority">${gtdItemTypeLabel(item.type)}</span>
        </div>
        <div class="memo-task-meta">
          ${projectBadge}
          <span>${gtdItemStatusLabel(item.status)}</span>
          ${milestone ? `<span>${escapeHTML(milestone.title)}</span>` : ""}
          ${item.linkedTaskIds.length ? `<span>${item.linkedTaskIds.length} tasks</span>` : ""}
          ${item.linkedMemoIds.length ? `<span>${item.linkedMemoIds.length} memos</span>` : ""}
          ${(item.labels || []).slice(0, 4).map((label) => `<span>#${escapeHTML(label)}</span>`).join("")}
        </div>
        ${item.decision ? `<p class="memo-task-note">${escapeHTML(item.decision)}</p>` : ""}
      </div>
      <div class="memo-task-actions">
        ${item.status === "open" ? `<button class="memo-action-button" type="button" data-action="triageGTDItem" title="标记已澄清">${SVG.check}</button>` : ""}
        ${!closed ? `<button class="memo-action-button" type="button" data-action="waitGTDItem" title="标记等待">${SVG.clock}</button>` : ""}
        ${!closed ? `<button class="memo-action-button" type="button" data-action="closeGTDItem" title="关闭">${SVG.archive}</button>` : ""}
      </div>
    </article>
  `;
}

function gtdMilestoneWorkspaceTemplate() {
  return `
    <section class="memo-task-workspace">
      <form class="memo-task-create" data-gtd-milestone-create-form>
        <input name="title" type="text" placeholder="新增阶段目标，例如 v0.2 GTD Inbox" autocomplete="off" />
        <select name="status" aria-label="状态">
          <option value="planned">计划中</option>
          <option value="active">进行中</option>
        </select>
        <input name="targetAt" type="date" aria-label="目标日期" />
        <button class="memo-primary-button" type="submit">${SVG.plus}<span>添加</span></button>
      </form>
    </section>
  `;
}

function gtdMilestoneGroupTemplate(label, milestones, context) {
  if (!milestones.length) return "";
  return `
    <section class="memo-todo-group memo-task-group" aria-label="${escapeAttr(label)}">
      <div class="memo-todo-group-head">
        <span>${escapeHTML(label)}</span>
        <strong>${milestones.length}</strong>
      </div>
      ${milestones.map((milestone) => gtdMilestoneCardTemplate(milestone, context || {})).join("")}
    </section>
  `;
}

function gtdMilestoneCardTemplate(milestone, context) {
  const items = (context.items || []).filter((item) => item.milestoneId === milestone.id || milestone.itemIds.includes(item.id));
  const tasks = (context.tasks || []).filter((task) => milestone.taskIds.includes(task.id));
  const openItems = items.filter((item) => item.status !== "closed" && item.status !== "resolved").length;
  const openTasks = tasks.filter((task) => task.status !== "completed" && task.status !== "cancelled" && task.status !== "archived").length;
  const target = milestone.targetAt ? formatTaskDate(milestone.targetAt) : "";
  const complete = milestone.status === "completed";
  return `
    <article class="memo-task-card ${complete ? "is-complete" : ""} is-priority-none" data-gtd-milestone-id="${escapeAttr(milestone.id)}">
      <span class="memo-task-check" aria-hidden="true"></span>
      <div class="memo-task-body">
        <div class="memo-task-title-row">
          <strong>${escapeHTML(milestone.title)}</strong>
          <span class="memo-task-priority">${gtdMilestoneStatusLabel(milestone.status)}</span>
        </div>
        <div class="memo-task-meta">
          ${target ? `<time datetime="${escapeAttr(milestone.targetAt)}">目标 ${escapeHTML(target)}</time>` : ""}
          <span>${openItems} open items</span>
          <span>${openTasks} open tasks</span>
          <span>${items.length} items</span>
          <span>${tasks.length} tasks</span>
        </div>
      </div>
      <div class="memo-task-actions">
        ${milestone.status === "planned" ? `<button class="memo-action-button" type="button" data-action="activateGTDMilestone" title="开始">${SVG.check}</button>` : ""}
        ${!complete ? `<button class="memo-action-button" type="button" data-action="completeGTDMilestone" title="完成">${SVG.archive}</button>` : ""}
      </div>
    </article>
  `;
}

function gtdItemTypeLabel(type) {
  const labels = {
    bug: "Bug",
    chore: "杂项",
    feature: "功能",
    idea: "想法",
    question: "问题",
  };
  return labels[type] || labels.idea;
}

function gtdItemStatusLabel(status) {
  const labels = {
    closed: "已关闭",
    open: "Open",
    resolved: "已解决",
    triaged: "已澄清",
    waiting: "等待",
  };
  return labels[status] || labels.open;
}

function gtdMilestoneStatusLabel(status) {
  const labels = {
    active: "进行中",
    cancelled: "已取消",
    completed: "已完成",
    planned: "计划中",
  };
  return labels[status] || labels.planned;
}

function linkTemplate(link) {
  const visibility = VISIBILITY[link.memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
  const tags = extractTags(link.memo.content);
  const href = safeUrl(link.url);
  return `
    <article class="memo-resource-card is-link" data-memo-id="${escapeAttr(link.memoId)}" data-link-url="${escapeAttr(link.url)}">
      <a class="memo-resource-target" href="${escapeAttr(href)}" target="_blank" rel="noreferrer">
        <span class="memo-resource-icon">${SVG.link}</span>
        <span class="memo-resource-body">
          <span class="memo-resource-title">${escapeHTML(link.label || link.url)}</span>
          <span class="memo-resource-url">${escapeHTML(compactFileURL(link.url))}</span>
        </span>
      </a>
      <button class="memo-action-button memo-link-copy-button" type="button" data-action="copyLink" title="复制链接" aria-label="复制链接">${SVG.copy}</button>
      <div class="memo-resource-source">
        ${sourceMemoMarkerTemplate(link.memoId)}
        <div class="memo-todo-meta">
          <time datetime="${escapeAttr(link.memo.createdAt)}">${formatRelativeDate(link.memo.createdAt)}</time>
          <span>${SVG[visibility.icon]} ${visibility.label}</span>
          ${tags.slice(0, 3).map((tag) => `<span>#${escapeHTML(tag)}</span>`).join("")}
        </div>
      </div>
    </article>
  `;
}

function codeBlockTemplate(block) {
  const visibility = VISIBILITY[block.memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
  const tags = extractTags(block.memo.content);
  const code = block.code || "";
  const preview = code.trim() || "空代码块";
  const markerLabel = block.marked ? "已标记" : "未标记";
  const aliases = Array.isArray(block.aliases) ? block.aliases : [];
  const lineRange = block.endLineIndex > block.lineIndex
    ? `${block.lineIndex + 1}-${block.endLineIndex + 1}`
    : String(block.lineIndex + 1);
  return `
    <article class="memo-resource-card is-code ${block.marked ? "is-snippet" : "is-unmarked"}" data-memo-id="${escapeAttr(block.memoId)}" data-code-block-id="${escapeAttr(block.id)}">
      <div class="memo-code-block-head">
        <span class="memo-resource-icon">${SVG.code}</span>
        <span class="memo-resource-body">
          <span class="memo-resource-title">
            ${escapeHTML(block.label || "代码片段")}
            <span class="memo-code-block-badge">${escapeHTML(markerLabel)}</span>
          </span>
          <span class="memo-resource-url">
            第 ${escapeHTML(lineRange)} 行${block.language ? ` / ${escapeHTML(block.language)}` : ""}
            ${aliases.length ? ` / ${aliases.map((alias) => escapeHTML(alias)).join(" ")}` : ""}
          </span>
        </span>
        <button class="memo-action-button" type="button" data-action="copyCodeBlock" title="复制代码">${SVG.copy}</button>
      </div>
      <pre class="memo-code-block-preview"><code>${escapeHTML(preview)}</code></pre>
      <div class="memo-resource-source">
        ${sourceMemoMarkerTemplate(block.memoId)}
        <div class="memo-todo-meta">
          <time datetime="${escapeAttr(block.memo.createdAt)}">${formatRelativeDate(block.memo.createdAt)}</time>
          <span>${SVG[visibility.icon]} ${visibility.label}</span>
          ${tags.slice(0, 3).map((tag) => `<span>#${escapeHTML(tag)}</span>`).join("")}
        </div>
      </div>
    </article>
  `;
}

function resourceGroupTemplate(label, resources) {
  if (!resources.length) return "";
  return `
    <section class="memo-todo-group" aria-label="${escapeAttr(label)}">
      <div class="memo-todo-group-head">
        <span>${escapeHTML(label)}</span>
        <strong>${resources.length}</strong>
      </div>
      ${resources.map(resourceTemplate).join("")}
    </section>
  `;
}

function resourceTemplate(resource) {
  const visibility = VISIBILITY[resource.memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
  const tags = extractTags(resource.memo.content);
  const preview = resource.type === "image" ? resourcePreviewTemplate(resource) : "";
  const openButton = renderVSCodeOpenButton(resource.url);
  const href = safeUrl(resource.url);
  const targetBody = `
    ${preview || `<span class="memo-resource-icon">${SVG.paperclip}</span>`}
    <span class="memo-resource-body">
      <span class="memo-resource-title">${escapeHTML(resource.label || fileDisplayName("", resource.url))}</span>
      <span class="memo-resource-url">${escapeHTML(compactFileURL(resource.url))}</span>
    </span>
  `;
  const target =
    href !== "#" && !openButton
      ? `<a class="memo-resource-target" href="${escapeAttr(href)}" target="_blank" rel="noreferrer">${targetBody}</a>`
      : `<div class="memo-resource-target${openButton ? " has-editor-open" : ""}">${targetBody}${openButton}</div>`;

  return `
    <article class="memo-resource-card is-${resource.type}" data-memo-id="${escapeAttr(resource.memoId)}">
      ${target}
      <div class="memo-resource-source">
        ${sourceMemoMarkerTemplate(resource.memoId)}
        <div class="memo-todo-meta">
          <time datetime="${escapeAttr(resource.memo.createdAt)}">${formatRelativeDate(resource.memo.createdAt)}</time>
          <span>${SVG[visibility.icon]} ${visibility.label}</span>
          ${tags.slice(0, 3).map((tag) => `<span>#${escapeHTML(tag)}</span>`).join("")}
        </div>
      </div>
    </article>
  `;
}

function resourcePreviewTemplate(resource) {
  const src = safeImageUrl(resource.url);
  if (!src) return `<span class="memo-resource-icon">${SVG.image}</span>`;
  return `
    <span class="memo-resource-preview">
      <img src="${escapeAttr(src)}" alt="${escapeAttr(resource.label || "image")}" loading="lazy" />
    </span>
  `;
}

function clipboardCurrentTemplate(item, options = {}) {
  const hasItem = item && item.id;
  if (!hasItem) return emptyClipboardTemplate();

  const typeLabel = options.typeLabel || clipboardFallbackTypeLabel(item.type);
  const actionLabel = options.actionLabel || "保存";
  const capturedAt = clipboardCapturedAtLabel(item.capturedAt);
  const preview = item.type === "image" && item.dataURL
    ? `<img class="memo-clipboard-current-image" src="${escapeAttr(item.dataURL)}" alt="当前粘贴板图片" />`
    : `<pre class="memo-clipboard-current-text">${escapeHTML(item.content || "空内容")}</pre>`;

  return `
    <article class="memo-resource-card memo-clipboard-current is-${escapeAttr(item.type || "text")}">
      <header class="memo-clipboard-current-head">
        <div class="memo-resource-target memo-clipboard-current-summary">
          <span class="memo-resource-icon">${clipboardIcon(item.type)}</span>
          <span class="memo-resource-body">
            <span class="memo-resource-title">当前粘贴板的内容</span>
            <span class="memo-resource-url">
              ${escapeHTML(typeLabel)}
              ${capturedAt ? ` / ${escapeHTML(capturedAt)}` : ""}
              ${item.rawType ? ` / ${escapeHTML(item.rawType)}` : ""}
            </span>
          </span>
        </div>
        <div class="memo-clipboard-current-actions">
          <button class="memo-secondary-button" type="button" data-action="clipboardRefresh">${SVG.restore}<span>刷新</span></button>
          <button class="memo-primary-button" type="button" data-action="clipboardAccept" ${options.working ? "disabled" : ""}>
            ${SVG.plus}
            <span>${escapeHTML(actionLabel)}</span>
          </button>
        </div>
      </header>
      <div class="memo-clipboard-current-preview">
        ${preview}
      </div>
    </article>
  `;
}

function emptyClipboardTemplate() {
  return `
    <div class="memo-empty-state">
      <div class="memo-empty-icon">${SVG.copy}</div>
      <h2>暂无粘贴板内容</h2>
      <button class="memo-secondary-button" type="button" data-action="clipboardRefresh">刷新</button>
    </div>
  `;
}

function clipboardIcon(type) {
  if (type === "link") return SVG.link;
  if (type === "image") return SVG.image;
  return SVG.copy;
}

function clipboardFallbackTypeLabel(type) {
  if (type === "link") return "链接";
  if (type === "image") return "图片";
  return "文本";
}

function clipboardCapturedAtLabel(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleString([], {
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    month: "numeric",
  });
}

function editTemplate(memo, projects = []) {
  return `
    <div class="memo-inline-editor">
      <div class="memo-editor-switch">
        <div class="memo-editor-host is-inline" data-edit-host data-editor-switch-host></div>
        <section class="memo-editor-preview memo-edit-preview" data-edit-preview hidden></section>
      </div>
      <div class="memo-inline-actions">
        <div class="memo-inline-status-line" data-edit-vim-status></div>
        <label class="memo-select-wrap is-compact">
          <select data-edit-project aria-label="编辑 Project">
            ${projectOptionsTemplate(projects, memo.projectId || "")}
          </select>
          ${SVG.chevronDown}
        </label>
        <label class="memo-select-wrap is-compact">
          <select data-edit-visibility aria-label="编辑可见性">
            ${visibilityOptionsTemplate(memo.visibility)}
          </select>
          ${SVG.chevronDown}
        </label>
        <button class="memo-secondary-button" type="button" data-action="toggleEditPreview" aria-pressed="false">${SVG.eye}<span>预览</span></button>
        <button class="memo-secondary-button" type="button" data-action="cancelEdit">${SVG.x}<span>取消</span></button>
        <button class="memo-primary-button" type="button" data-action="saveEdit">${SVG.check}<span>保存</span></button>
      </div>
    </div>
  `;
}

function emptyFeedTemplate() {
  return `
    <div class="memo-empty-state">
      <div class="memo-empty-icon">${SVG.search}</div>
      <h2>没有匹配的 memo</h2>
      <button class="memo-secondary-button" type="button" data-action="clearFilters">查看全部</button>
    </div>
  `;
}

function emptyTodosTemplate() {
  return `
    <div class="memo-empty-state">
      <div class="memo-empty-icon">${SVG.check}</div>
      <h2>没有匹配的代办</h2>
      <button class="memo-secondary-button" type="button" data-action="clearFilters">查看全部</button>
    </div>
  `;
}

function emptyLinksTemplate() {
  return `
    <div class="memo-empty-state">
      <div class="memo-empty-icon">${SVG.link}</div>
      <h2>没有匹配的超链接</h2>
      <button class="memo-secondary-button" type="button" data-action="clearFilters">查看全部</button>
    </div>
  `;
}

function emptyCodeBlocksTemplate() {
  return `
    <div class="memo-empty-state">
      <div class="memo-empty-icon">${SVG.code}</div>
      <h2>没有匹配的代码片段</h2>
      <button class="memo-secondary-button" type="button" data-action="clearFilters">查看全部</button>
    </div>
  `;
}

function emptyFilesTemplate() {
  return `
    <div class="memo-empty-state">
      <div class="memo-empty-icon">${SVG.paperclip}</div>
      <h2>没有匹配的文件或图片</h2>
      <button class="memo-secondary-button" type="button" data-action="clearFilters">查看全部</button>
    </div>
  `;
}

function visibilityOptionsTemplate(selected) {
  return Object.entries(VISIBILITY)
    .map(([value, item]) => `<option value="${value}" ${value === selected ? "selected" : ""}>${item.label}</option>`)
    .join("");
}

export {
  activeViewMeta,
  calendarTemplate,
  clipboardCurrentTemplate,
  codeBlockTemplate,
  detachedMemoCardTemplate,
  detachedMemoRenderContext,
  detachedMemoWindowTemplate,
  emptyCodeBlocksTemplate,
  emptyFeedTemplate,
  emptyFilesTemplate,
  emptyLinksTemplate,
  emptyTasksTemplate,
  emptyTodosTemplate,
  gtdItemCardTemplate,
  gtdItemGroupTemplate,
  gtdItemWorkspaceTemplate,
  gtdMilestoneGroupTemplate,
  gtdMilestoneWorkspaceTemplate,
  linkTemplate,
  memoTemplate,
  projectFilterTemplate,
  projectOptionsTemplate,
  resourceGroupTemplate,
  shellTemplate,
  statTemplate,
  taskCardTemplate,
  taskGroupTemplate,
  taskWorkspaceTemplate,
  todoGroupTemplate,
};
