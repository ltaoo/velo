import {
  DEFAULT_VISIBILITY,
  VISIBILITY,
  buildMemoReferenceIndex,
  collectTags,
  collectTodos,
  compactText,
  extractTags,
  getTodoStats,
  isMemoFenceClosingLine,
  memoReferenceAlias,
  memoTitle,
  normalizeMemoPayload,
  parseMemoFenceLine,
  parseTaskLine,
  updateTaskLine,
} from "../../domain/memos.js";
import {
  normalizeProjectFilter,
  normalizeProjectID,
  normalizeProjectPayload,
} from "../../domain/projects.js";
import { parseAssetReference } from "../../domain/storage.js";
import {
  completeTask,
  createTask,
  createTaskNote,
  loadTasks,
  normalizeTaskSummary,
  updateTask,
} from "../../domain/tasks.js";
import {
  closeGTDItem,
  createGTDItem,
  createGTDMilestone,
  loadGTDItems,
  loadGTDMilestones,
  normalizeGTDItem,
  normalizeGTDMilestone,
  updateGTDItem,
  updateGTDMilestone,
} from "../../domain/gtd.js";
import {
  collectCodeBlocks,
  collectLinks,
  collectResources,
  getResourceStats,
  sortMemoReference,
} from "../../domain/memo-resources.js";
import {
  createMemoCommentInVault,
  deleteMemoCommentInVault,
  loadMemoCommentsFromVault,
  normalizeMemoCommentPayload,
  updateMemoCommentInVault,
} from "../../domain/memo-comments.js";
import {
  createMemoInVault,
  createProjectInVault,
  deleteMemoInVault,
  errorMessage,
  loadMemoFromLocal,
  loadMemos,
  loadMemosFromVault,
  loadProjects,
  loadProjectsFromVault,
  saveMemos,
  saveProjects,
  updateMemoInVault,
} from "../../domain/memo-repository.js";
import {
  COMPOSER_DRAFT_ID,
  deleteMemoDraftInVault,
  loadMemoDraftsFromVault,
  memoEditDraftId,
  normalizeMemoDraftPayload,
  upsertMemoDraftInVault,
} from "../../domain/memo-drafts.js";
import { SVG } from "./memo-icons.js";
import {
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
} from "./memo-templates.js";
import {
  EDITOR_SETTINGS_STORAGE_KEY,
  createMiniEditor,
  fileInfoToUploadURL,
  filesToMarkdown,
  insertPlainTextIntoEditor,
  loadEditorSettings,
  loadEditorSettingsFromVault,
  normalizeEditorSettings,
  refreshCloudStorageSettings,
  uploadErrorMessage,
} from "./memo-editor.js";
import { renderMemoMarkdown } from "./memo-markdown.js";
import { addMonths, dateFromKey, formatDateKey, memoDateKey, startOfMonth } from "./memo-date.js";
import { registerWindowSession } from "../../window-state.js";
import {
  closestAnchor,
  closestElement,
  copyText,
  escapeAttr,
  escapeCSSIdent,
  escapeHTML,
  externalBrowserURLFromAnchor,
} from "./memo-utils.js";
import { openImagePreviewFromElement } from "../../components/image-preview.js";

const LAST_PROJECT_STORAGE_KEY = "demo-desktop:memos:last-project:v1";
const SHORTCUTS_STORAGE_KEY = "demo-desktop:settings:shortcuts:v1";
const TASK_FILTER_STORAGE_KEY = "demo-desktop:gtd:task-filter:v1";
const CLIPBOARD_AUTO_HIDE_MS = 5000;
const CLIPBOARD_EXIT_MS = 180;
const CLIPBOARD_FOREGROUND_MAX_AGE_MS = 60 * 1000;
const CLIPBOARD_MIN_VISIBLE_MS = 1500;
const DETACHED_WINDOW_STATE_POLL_INTERVAL = 250;
const DETACHED_WINDOW_STATE_SNAPSHOT_DEBOUNCE = 800;
const TASK_FILTERS = new Set(["all", "completed", "inbox", "next", "overdue", "scheduled", "today"]);

function normalizeTaskFilter(value) {
  const filter = String(value || "").trim().toLowerCase();
  return TASK_FILTERS.has(filter) ? filter : "today";
}

function loadTaskFilter() {
  return normalizeTaskFilter(localStorage.getItem(TASK_FILTER_STORAGE_KEY));
}

function rememberTaskFilter(filter) {
  localStorage.setItem(TASK_FILTER_STORAGE_KEY, normalizeTaskFilter(filter));
}

function copyCodeBlockFromAction(action, memos, notify) {
  const blockNode = closestElement(action, "[data-code-block-id]") || closestElement(action, ".memo-fenced-code-block");
  const blockId = blockNode && blockNode.dataset ? blockNode.dataset.codeBlockId : "";
  const block = blockId ? collectCodeBlocks(Array.isArray(memos) ? memos : []).find((item) => item.id === blockId) : null;
  const code = block ? block.code : codeBlockTextFromNode(blockNode);
  if (code === null) return;
  copyText(code).then(
    () => notify("已复制代码片段"),
    () => notify("复制失败"),
  );
}

function copyInlineLinkFromAction(action, notify) {
  const linkNode = closestElement(action, "[data-inline-link-url]");
  const url = linkNode && linkNode.dataset ? linkNode.dataset.inlineLinkUrl : "";
  if (!url) return;
  copyText(url).then(
    () => notify("已复制链接"),
    () => notify("复制失败"),
  );
}

function codeBlockTextFromNode(blockNode) {
  if (!blockNode || typeof blockNode.querySelector !== "function") return null;
  const codeNode = blockNode.querySelector("[data-code-block-code]") || blockNode.querySelector("pre code");
  if (!codeNode) return null;
  return codeNode.textContent || "";
}

function handleMemoRenderedCopy(event) {
  if (event.defaultPrevented || !event.clipboardData) return;

  const text = selectedMemoRenderedText(event.currentTarget);
  if (text === null) return;

  event.preventDefault();
  event.clipboardData.setData("text/plain", text);
}

function selectedMemoRenderedText(root) {
  const doc = (root && root.ownerDocument) || document;
  const selection = doc.getSelection ? doc.getSelection() : null;
  if (!selection || selection.isCollapsed || selection.rangeCount === 0) return null;

  const parts = [];
  for (let index = 0; index < selection.rangeCount; index += 1) {
    const text = selectedMemoRangeText(root, selection.getRangeAt(index));
    if (text !== null) parts.push(text);
  }

  return parts.length ? parts.join("\n") : null;
}

function selectedMemoRangeText(root, range) {
  if (!root || !range) return null;
  const selectedLines = Array.from(root.querySelectorAll(".memo-source-line"))
    .filter(function (line) {
      const body = memoLineBody(line);
      return body && rangeIntersectsNode(range, body);
    })
    .filter(function (line, _index, lines) {
      return !hasSelectedSourceLineAncestor(line, lines);
    });

  if (!selectedLines.length) return null;

  return selectedLines
    .map(function (line) {
      return selectedMemoLineText(range, memoLineBody(line));
    })
    .join("\n");
}

function memoLineBody(line) {
  if (!line || !line.children) return null;
  return Array.from(line.children).find(function (child) {
    return child && child.classList && child.classList.contains("memo-line-body");
  }) || null;
}

function hasSelectedSourceLineAncestor(line, selectedLines) {
  let node = line ? line.parentElement : null;
  while (node) {
    if (node.classList && node.classList.contains("memo-source-line") && selectedLines.includes(node)) return true;
    node = node.parentElement;
  }
  return false;
}

function rangeIntersectsNode(range, node) {
  if (!range || !node || typeof range.intersectsNode !== "function") return false;
  try {
    return range.intersectsNode(node);
  } catch (_) {
    return false;
  }
}

function selectedMemoLineText(range, body) {
  if (!body) return "";
  const doc = body.ownerDocument || document;
  const bodyRange = doc.createRange();
  bodyRange.selectNodeContents(body);

  const lineRange = range.cloneRange();
  if (range.compareBoundaryPoints(Range.START_TO_START, bodyRange) < 0) {
    lineRange.setStart(bodyRange.startContainer, bodyRange.startOffset);
  }
  if (range.compareBoundaryPoints(Range.END_TO_END, bodyRange) > 0) {
    lineRange.setEnd(bodyRange.endContainer, bodyRange.endOffset);
  }

  const fragment = lineRange.cloneContents();
  bodyRange.detach();
  lineRange.detach();

  return cleanMemoClipboardLineText(memoFragmentClipboardText(fragment));
}

function memoFragmentClipboardText(fragment) {
  if (!fragment || typeof fragment.querySelectorAll !== "function") return "";
  fragment.querySelectorAll(".memo-line-number, .memo-fenced-code-toolbar, button, input, style, script").forEach(function (node) {
    node.remove();
  });
  fragment.querySelectorAll("br").forEach(function (node) {
    node.replaceWith((fragment.ownerDocument || document).createTextNode("\n"));
  });
  return fragment.textContent || "";
}

function cleanMemoClipboardLineText(value) {
  const lines = String(value || "").replace(/\r\n?/g, "\n").replace(/\u00a0/g, " ").split("\n");
  while (lines.length && lines[0].trim() === "") lines.shift();
  while (lines.length && lines[lines.length - 1].trim() === "") lines.pop();
  return lines.join("\n");
}

function editorFileOpenSettingsKey(settings) {
  const raw = settings && typeof settings === "object" ? settings : {};
  return JSON.stringify({
    fileEditor: raw.fileEditor || null,
    fileEditorRules: raw.fileEditorRules || [],
  });
}

export function mountMemosHome(root) {
  const state = {
    activeFilter: "all",
    activeTag: "",
    activeView: "memos",
    calendarMonth: startOfMonth(new Date()),
    commentDraft: "",
    commentEditDraft: "",
    commentEditingId: "",
    commentEditPreviewVisible: false,
    commentPreviewVisible: false,
    commentingMemoId: "",
    comments: [],
    commentsLoaded: false,
    commentSaving: false,
    editingId: "",
    editDraft: "",
    editProjectId: "",
    editVisibility: DEFAULT_VISIBILITY,
    sourceDraft: "",
    sourceEditingId: "",
    highlightMemoId: "",
    highlightTimer: null,
    expandedMemoIds: new Set(),
    draftsLoaded: false,
    editorSettings: loadEditorSettings(),
    editPreviewVisible: false,
    gtdItems: [],
    gtdLoading: false,
    gtdMilestones: [],
    activeProjectFilter: "all",
    composerProjectId: localStorage.getItem(LAST_PROJECT_STORAGE_KEY) || "",
    composerPreviewVisible: false,
    lastComposerProjectId: localStorage.getItem(LAST_PROJECT_STORAGE_KEY) || "",
    memoRefIndex: null,
    memoSearchActiveIndex: 0,
    memoSearchOpen: false,
    memoSearchQuery: "",
    memoDrafts: [],
    memos: loadMemos(),
    projects: loadProjects(),
    query: "",
    selectedCalendarDate: "",
    sortDesc: true,
    saving: false,
    taskDetails: new Map(),
    taskFilter: loadTaskFilter(),
    retainedCompletedTaskFilters: new Map(),
    tasks: [],
    tasksLoading: false,
    clipboardItem: null,
    clipboardDisplayedId: "",
    clipboardForeground: true,
    clipboardLastAppearedId: "",
    clipboardLeaving: false,
    clipboardShownAt: 0,
    clipboardVisible: false,
    clipboardWorking: false,
    clipboardLeaveTimer: null,
    clipboardTimer: null,
    toastTimer: null,
    visibility: DEFAULT_VISIBILITY,
  };

  let composerEditor = null;
  let commentEditEditor = null;
  let commentEditEditorCommentId = "";
  let commentEditor = null;
  let commentEditorMemoId = "";
  let editEditor = null;
  let editEditorMemoId = "";

  root.innerHTML = shellTemplate();

  const els = {
    attachInput: root.querySelector("[data-attach-input]"),
    calendar: root.querySelector("[data-calendar]"),
    composer: root.querySelector("[data-composer]"),
    composerHost: root.querySelector("[data-composer-host]"),
    composerStatus: root.querySelector("[data-composer-status]"),
    composerVimStatus: root.querySelector("[data-composer-vim-status]"),
    codeNavCount: root.querySelector("[data-code-nav-count]"),
    clipboardCard: root.querySelector("[data-clipboard-card]"),
    clipboardNavCount: root.querySelector("[data-clipboard-nav-count]"),
    createButton: root.querySelector('[data-action="createMemo"]'),
    feedCount: root.querySelector("[data-feed-count]"),
    fileNavCount: root.querySelector("[data-file-nav-count]"),
    itemNavCount: root.querySelector("[data-item-nav-count]"),
    linkNavCount: root.querySelector("[data-link-nav-count]"),
    mainSubtitle: root.querySelector("[data-main-subtitle]"),
    mainTitle: root.querySelector("[data-main-title]"),
    milestoneNavCount: root.querySelector("[data-milestone-nav-count]"),
    memoList: root.querySelector("[data-memo-list]"),
    memoSearchInput: root.querySelector("[data-memo-search-input]"),
    memoSearchPalette: root.querySelector("[data-memo-search-palette]"),
    memoSearchResults: root.querySelector("[data-memo-search-results]"),
    pinnedList: root.querySelector("[data-pinned-list]"),
    projectList: root.querySelector("[data-project-list]"),
    projectSelect: root.querySelector("[data-project-select]"),
    projectSummary: root.querySelector("[data-project-summary]"),
    searchInput: root.querySelector("[data-search-input]"),
    stats: root.querySelector("[data-stats]"),
    tagList: root.querySelector("[data-tag-list]"),
    tagSummary: root.querySelector("[data-tag-summary]"),
    todoNavCount: root.querySelector("[data-todo-nav-count]"),
    toast: root.querySelector("[data-toast]"),
    visibilitySelect: root.querySelector("[data-visibility-select]"),
  };

  composerEditor = createComposerEditor("");

  renderAll();
  renderComposerStatus(composerEditor.getText());
  bindGoMessages();
  refreshProjectsFromVault();
  refreshMemosFromVault();
  refreshMemoCommentsFromVault();
  refreshMemoDraftsFromVault();
  refreshTasksFromVault();
  refreshGTDFromVault();
  refreshEditorSettings({ silent: true });
  refreshStorageForRender();

  window.addEventListener("click", handleExternalLinkClick, true);
  root.addEventListener("click", handleClick);
  root.addEventListener("copy", handleMemoRenderedCopy);
  root.addEventListener("input", handleInput);
  root.addEventListener("change", handleChange);
  root.addEventListener("submit", handleSubmit);
  window.addEventListener("focus", handleWindowFocus);
  window.addEventListener("blur", handleWindowBlur);
  window.addEventListener("keydown", handleKeydown);
  window.addEventListener("storage", handleStorage);

  return {
    destroy() {
      window.removeEventListener("click", handleExternalLinkClick, true);
      root.removeEventListener("click", handleClick);
      root.removeEventListener("copy", handleMemoRenderedCopy);
      root.removeEventListener("input", handleInput);
      root.removeEventListener("change", handleChange);
      root.removeEventListener("submit", handleSubmit);
      window.removeEventListener("focus", handleWindowFocus);
      window.removeEventListener("blur", handleWindowBlur);
      window.removeEventListener("keydown", handleKeydown);
      window.removeEventListener("storage", handleStorage);
      if (state.toastTimer) window.clearTimeout(state.toastTimer);
      if (state.clipboardTimer) window.clearTimeout(state.clipboardTimer);
      if (state.clipboardLeaveTimer) window.clearTimeout(state.clipboardLeaveTimer);
      if (state.highlightTimer) window.clearTimeout(state.highlightTimer);
      if (composerEditor) composerEditor.destroy();
      if (commentEditor) commentEditor.destroy();
      if (commentEditEditor) commentEditEditor.destroy();
      if (editEditor) editEditor.destroy();
      commentEditEditorCommentId = "";
      commentEditorMemoId = "";
      editEditorMemoId = "";
      root.innerHTML = "";
    },
  };

  function createComposerEditor(value) {
    return createMiniEditor(els.composerHost, {
      memoItems() {
        return state.memos;
      },
      tagItems: editorTagItems,
      onChange(nextValue) {
        renderComposerStatus(nextValue);
        renderComposerPreview();
      },
      onCommit() {
        return createMemo({ source: "vim-wq" });
      },
      onDiscard() {
        return clearComposerDraft({ clearEditor: true, message: "草稿已丢弃" });
      },
      onQuit() {
        return exitComposer();
      },
      onSave() {
        return writeComposerDraft();
      },
      onSubmit() {
        createMemo();
      },
      onWriteDraft() {
        return writeComposerDraft();
      },
      placeholder: "记录想法、任务或链接...",
      value: value || "",
      vim: editorVimEnabled(),
      vimStatusHost: els.composerVimStatus,
    });
  }

  function editorTagItems() {
    return collectTags(scopedMemos().filter((memo) => !memo.archived));
  }

  function editorVimEnabled() {
    return state.editorSettings && state.editorSettings.vimMode === true;
  }

  function calendarWeekStart() {
    return state.editorSettings && state.editorSettings.calendarWeekStart === "sunday" ? "sunday" : "monday";
  }

  function handleStorage(event) {
    if (event.key !== EDITOR_SETTINGS_STORAGE_KEY) return;
    refreshEditorSettings();
  }

  function refreshEditorSettings(options = {}) {
    return loadEditorSettingsFromVault().then(function (next) {
      applyEditorSettings(next, options);
    }, function (err) {
      if (!options.silent) showToast("读取编辑器设置失败: " + errorMessage(err));
    });
  }

  function handleWindowFocus() {
    state.clipboardForeground = true;
    refreshEditorSettings();
    requestClipboardLatest({ maxAgeMs: CLIPBOARD_FOREGROUND_MAX_AGE_MS });
  }

  function handleWindowBlur() {
    state.clipboardForeground = false;
    hideClipboardCard();
  }

  function isClipboardForeground() {
    if (state.clipboardForeground) return true;
    return typeof document.hasFocus === "function" && document.hasFocus();
  }

  function applyEditorSettings(nextSettings, options = {}) {
    const next = normalizeEditorSettings(nextSettings || loadEditorSettings());
    const vimChanged = next.vimMode !== editorVimEnabled();
    const calendarChanged = next.calendarWeekStart !== calendarWeekStart();
    const fileEditorChanged = editorFileOpenSettingsKey(next) !== editorFileOpenSettingsKey(state.editorSettings);
    if (!vimChanged && !calendarChanged && !fileEditorChanged) return;

    const composerText = composerEditor ? composerEditor.getText() : "";
    state.editorSettings = next;

    if (vimChanged) {
      if (editEditor) syncEditDraftFromEditor();
      if (commentEditor) syncCommentDraftFromEditor();
      if (commentEditEditor) syncCommentEditDraftFromEditor();
      if (composerEditor) composerEditor.destroy();
      els.composerHost.innerHTML = "";
      composerEditor = createComposerEditor(composerText);
      renderComposerStatus(composerEditor.getText());

      if (state.activeView === "memos" && (state.editingId || state.commentingMemoId || state.commentEditingId)) renderFeed();
      if (calendarChanged) renderCalendar();
      if (fileEditorChanged) renderAll();
      if (!options.silent) showToast(next.vimMode ? "已启用 Vim 模式" : "已关闭 Vim 模式");
      return;
    }

    if (calendarChanged) {
      renderCalendar();
      if (!options.silent) showToast(next.calendarWeekStart === "sunday" ? "日历已设为周日开始" : "日历已设为周一开始");
    }

    if (fileEditorChanged) {
      renderAll();
      if (!options.silent) showToast("本地文件打开应用已更新为 " + editorFileEditorLabel(next.fileEditor));
    }
  }

  function editorFileEditorLabel(fileEditor) {
    const raw = fileEditor && typeof fileEditor === "object" ? fileEditor : {};
    return raw.name || raw.id || "编辑器";
  }

  function handleExternalLinkClick(event) {
    if (event.defaultPrevented || event.button !== 0) return;
    const link = closestAnchor(event.target);
    if (!link || !root.contains(link)) return;

    const url = externalBrowserURLFromAnchor(link);
    if (!url) return;

    event.preventDefault();
    event.stopPropagation();
    confirmOpenExternalLink(url);
  }

  function refreshStorageForRender() {
    refreshCloudStorageSettings().then(function () {
      renderAll();
    }, function () {});
  }

  function handleClick(event) {
    const searchResult = closestElement(event.target, "[data-memo-search-result]");
    if (searchResult && root.contains(searchResult)) {
      openMemoSearchResult(searchResult.dataset.memoSearchResult || "");
      return;
    }

    if (event.target === els.memoSearchPalette) {
      closeMemoSearchPalette();
      return;
    }

    const command = closestElement(event.target, "[data-command]");
    if (command && root.contains(command)) {
      runComposerCommand(command.dataset.command);
      return;
    }

    const filter = closestElement(event.target, "[data-filter]");
    if (filter && root.contains(filter)) {
      state.activeView = "memos";
      state.activeFilter = filter.dataset.filter;
      state.activeTag = "";
      state.selectedCalendarDate = "";
      renderAll();
      return;
    }

    const view = closestElement(event.target, "[data-view]");
    if (view && root.contains(view)) {
      clearRetainedCompletedTasks();
      state.activeView = view.dataset.view;
      state.activeFilter = "all";
      state.activeTag = "";
      state.editingId = "";
      state.editPreviewVisible = false;
      state.sourceDraft = "";
      state.sourceEditingId = "";
      state.commentPreviewVisible = false;
      state.commentingMemoId = "";
      state.commentDraft = "";
      state.query = "";
      state.selectedCalendarDate = "";
      els.searchInput.value = "";
      renderAll();
      return;
    }

    const taskFilter = closestElement(event.target, "[data-task-filter]");
    if (taskFilter && root.contains(taskFilter)) {
      clearRetainedCompletedTasks();
      state.taskFilter = normalizeTaskFilter(taskFilter.dataset.taskFilter);
      rememberTaskFilter(state.taskFilter);
      renderAll();
      return;
    }

    const projectFilter = closestElement(event.target, "[data-project-filter]");
    if (projectFilter && root.contains(projectFilter)) {
      selectProjectFilter(projectFilter.dataset.projectFilter || "all");
      return;
    }

    const tag = closestElement(event.target, "[data-tag]");
    if (tag && root.contains(tag)) {
      state.activeTag = state.activeTag === tag.dataset.tag ? "" : tag.dataset.tag;
      state.activeFilter = "all";
      state.selectedCalendarDate = "";
      renderAll();
      return;
    }

    const calendarAction = closestElement(event.target, "[data-calendar-action]");
    if (calendarAction && root.contains(calendarAction)) {
      runCalendarAction(calendarAction.dataset.calendarAction);
      return;
    }

    const calendarDate = closestElement(event.target, "[data-calendar-date]");
    if (calendarDate && root.contains(calendarDate)) {
      selectCalendarDate(calendarDate.dataset.calendarDate);
      return;
    }

    const editorOpen = closestElement(event.target, "[data-editor-open]");
    if (editorOpen && root.contains(editorOpen)) {
      event.preventDefault();
      event.stopPropagation();
      openFileInSelectedEditor(editorOpen);
      return;
    }

    const imagePreview = closestElement(event.target, "[data-image-preview-src]");
    if (imagePreview && root.contains(imagePreview)) {
      event.preventDefault();
      event.stopPropagation();
      openImagePreview(imagePreview);
      return;
    }

    const memoRefTarget = closestElement(event.target, "[data-memo-ref-target]");
    if (memoRefTarget && root.contains(memoRefTarget)) {
      event.preventDefault();
      focusMemo(memoRefTarget.dataset.memoRefTarget);
      return;
    }

    const action = closestElement(event.target, "[data-action]");
    if (!action || !root.contains(action)) return;

    const memoNode = closestElement(action, "[data-memo-id]");
    const memoId = memoNode ? memoNode.dataset.memoId : "";
    const commentNode = closestElement(action, "[data-comment-id]");
    const commentId = commentNode ? commentNode.dataset.commentId : "";
    const taskNode = closestElement(action, "[data-task-id]");
    const taskId = taskNode ? taskNode.dataset.taskId : "";
    const gtdItemNode = closestElement(action, "[data-gtd-item-id]");
    const gtdItemId = gtdItemNode ? gtdItemNode.dataset.gtdItemId : "";
    const gtdMilestoneNode = closestElement(action, "[data-gtd-milestone-id]");
    const gtdMilestoneId = gtdMilestoneNode ? gtdMilestoneNode.dataset.gtdMilestoneId : "";

    switch (action.dataset.action) {
      case "addTaskNote":
        addTaskNote(taskId);
        break;
      case "archiveMemo":
        updateMemo(memoId, { archived: true });
        break;
      case "cancelEdit":
        cancelEdit();
        break;
      case "cancelComment":
        cancelComment();
        break;
      case "cancelCommentEdit":
        cancelCommentEdit();
        break;
      case "clearFilters":
        state.activeFilter = "all";
        state.activeTag = "";
        state.activeProjectFilter = "all";
        state.composerProjectId = state.lastComposerProjectId || "";
        state.commentingMemoId = "";
        state.commentDraft = "";
        state.commentEditingId = "";
        state.commentEditDraft = "";
        state.commentEditPreviewVisible = false;
        state.commentPreviewVisible = false;
        state.editingId = "";
        state.editPreviewVisible = false;
        state.sourceDraft = "";
        state.sourceEditingId = "";
        state.query = "";
        state.selectedCalendarDate = "";
        els.searchInput.value = "";
        renderAll();
        break;
      case "copyMemo":
        copyMemo(memoId);
        break;
      case "copyMemoRef":
        copyMemoRef(memoId);
        break;
      case "commentMemo":
        startComment(memoId);
        break;
      case "deleteComment":
        deleteComment(commentId);
        break;
      case "toggleCommentPreview":
        toggleCommentPreview(memoId);
        break;
      case "toggleCommentEditPreview":
        toggleCommentEditPreview(commentId);
        break;
      case "toggleComposerPreview":
        toggleComposerPreview();
        break;
      case "toggleEditPreview":
        toggleEditPreview(memoId);
        break;
      case "copyCodeBlock":
        copyCodeBlock(action);
        break;
      case "copyInlineLink":
        event.preventDefault();
        event.stopPropagation();
        copyInlineLinkFromAction(action, showToast);
        break;
      case "copyLink":
        event.preventDefault();
        event.stopPropagation();
        copyLink(action);
        break;
      case "copyTaskRef":
        copyTaskRef(taskId);
        break;
      case "triageGTDItem":
        updateExistingGTDItem(gtdItemId, { status: "triaged" }, "已标记为已澄清");
        break;
      case "waitGTDItem":
        updateExistingGTDItem(gtdItemId, { status: "waiting" }, "已标记为等待");
        break;
      case "closeGTDItem":
        closeExistingGTDItem(gtdItemId);
        break;
      case "activateGTDMilestone":
        updateExistingGTDMilestone(gtdMilestoneId, { status: "active" }, "里程碑已开始");
        break;
      case "completeGTDMilestone":
        updateExistingGTDMilestone(gtdMilestoneId, { status: "completed" }, "里程碑已完成");
        break;
      case "createMemo":
        createMemo();
        break;
      case "clipboardAccept":
        acceptClipboardItem();
        break;
      case "clipboardDismiss":
        hideClipboardCard({ forceAppeared: true });
        break;
      case "clipboardRefresh":
        requestClipboardLatest();
        break;
      case "createProject":
        createProjectFromPrompt();
        break;
      case "deleteMemo":
        deleteMemo(memoId);
        break;
      case "detachMemo":
        detachMemo(memoId);
        break;
      case "editMemo":
        startEdit(memoId);
        break;
      case "editComment":
        startCommentEdit(commentId);
        break;
      case "editMemoSource":
        startSourceEdit(memoId);
        break;
      case "cancelMemoSource":
        cancelSourceEdit();
        break;
      case "saveMemoSource":
        saveMemoSource(memoId);
        break;
      case "toggleMemoExpand":
        toggleMemoExpand(memoId);
        break;
      case "openSettings":
        openSettings();
        break;
      case "openSlimMemos":
        openSlimMemos();
        break;
      case "openSourceMemo":
        openSourceMemo(memoId);
        break;
      case "restoreMemo":
        updateMemo(memoId, { archived: false });
        break;
      case "saveEdit":
        saveEdit(memoId);
        break;
      case "saveComment":
        saveComment(memoId);
        break;
      case "saveCommentEdit":
        saveCommentEdit();
        break;
      case "sortMemos":
        state.sortDesc = !state.sortDesc;
        renderAll();
        break;
      case "togglePin":
        togglePin(memoId);
        break;
      default:
        break;
    }
  }

  function handleSubmit(event) {
    const taskForm = event.target.closest("[data-task-create-form]");
    if (taskForm && root.contains(taskForm)) {
      event.preventDefault();
      createTaskFromForm(taskForm);
      return;
    }

    const itemForm = event.target.closest("[data-gtd-item-create-form]");
    if (itemForm && root.contains(itemForm)) {
      event.preventDefault();
      createGTDItemFromForm(itemForm);
      return;
    }

    const milestoneForm = event.target.closest("[data-gtd-milestone-create-form]");
    if (milestoneForm && root.contains(milestoneForm)) {
      event.preventDefault();
      createGTDMilestoneFromForm(milestoneForm);
    }
  }

  function bindGoMessages() {
    if (!window.onGoMessage) return;
    window.onGoMessage(function (payload) {
      if (!payload) return;
      if (payload.type === "main_window_focus") {
        state.clipboardForeground = true;
        requestClipboardLatest({ maxAgeMs: CLIPBOARD_FOREGROUND_MAX_AGE_MS });
      }
    });
  }

  function requestClipboardLatest(options = {}) {
    if (typeof invoke !== "function") return;
    const maxAgeMs = Number(options.maxAgeMs || 0);
    const url = maxAgeMs > 0
      ? "/api/clipboard/latest?maxAgeSeconds=" + encodeURIComponent(String(Math.ceil(maxAgeMs / 1000)))
      : "/api/clipboard/latest";
    invoke(url, { method: "GET" }).then(
      function (resp) {
        if (!resp || resp.code !== 0 || !resp.data) return;
        if (!resp.data.found) {
          state.clipboardItem = null;
          if (state.activeView === "clipboard") renderMainContent();
          renderViewButtons();
          return;
        }
        const item = normalizeClipboardItem(resp.data.item);
        if (!item || !item.id) return;
        state.clipboardItem = item;
        if (state.activeView === "clipboard") renderMainContent();
        renderViewButtons();
        if (maxAgeMs > 0 && resp.data.fresh === false) {
          hideClipboardCard({ forceAppeared: true });
          return;
        }
        if (isClipboardForeground()) showClipboardCard();
      },
      function () {},
    );
  }

  function normalizeClipboardItem(item) {
    if (!item || typeof item !== "object") return null;
    return {
      capturedAt: String(item.capturedAt || ""),
      changedAt: String(item.changedAt || ""),
      content: String(item.content || ""),
      contentBase64: String(item.contentBase64 || ""),
      dataURL: String(item.dataURL || ""),
      id: String(item.id || ""),
      mimeType: String(item.mimeType || ""),
      name: String(item.name || ""),
      rawType: String(item.rawType || ""),
      size: Number(item.size || 0),
      type: String(item.type || "text"),
    };
  }

  function showClipboardCard() {
    if (!state.clipboardItem || !els.clipboardCard) return;
    const itemId = String(state.clipboardItem.id || "");
    const sameActiveItem = (state.clipboardVisible || state.clipboardLeaving) && itemId && state.clipboardDisplayedId === itemId;
    if (itemId && state.clipboardLastAppearedId === itemId) return;
    if (sameActiveItem) {
      if (state.clipboardLeaving) {
        if (state.clipboardLeaveTimer) {
          window.clearTimeout(state.clipboardLeaveTimer);
          state.clipboardLeaveTimer = null;
        }
        state.clipboardLeaving = false;
        state.clipboardVisible = true;
        state.clipboardShownAt = Date.now();
        renderClipboardCard();
        scheduleClipboardAutoHide();
      }
      return;
    }

    state.clipboardDisplayedId = itemId;
    state.clipboardLastAppearedId = "";
    state.clipboardShownAt = Date.now();
    if (state.clipboardLeaveTimer) {
      window.clearTimeout(state.clipboardLeaveTimer);
      state.clipboardLeaveTimer = null;
    }
    state.clipboardLeaving = false;
    state.clipboardVisible = true;
    renderClipboardCard();
    scheduleClipboardAutoHide();
  }

  function scheduleClipboardAutoHide() {
    if (state.clipboardTimer) window.clearTimeout(state.clipboardTimer);
    state.clipboardTimer = window.setTimeout(function () {
      if (!state.clipboardWorking) hideClipboardCard();
    }, CLIPBOARD_AUTO_HIDE_MS);
  }

  function hideClipboardCard(options = {}) {
    if (state.clipboardTimer) {
      window.clearTimeout(state.clipboardTimer);
      state.clipboardTimer = null;
    }
    if (!state.clipboardVisible && !state.clipboardLeaving) {
      renderClipboardCard();
      return;
    }
    markClipboardAppearedIfReady(options);
    state.clipboardLeaving = true;
    renderClipboardCard();
    if (state.clipboardLeaveTimer) window.clearTimeout(state.clipboardLeaveTimer);
    state.clipboardLeaveTimer = window.setTimeout(function () {
      state.clipboardVisible = false;
      state.clipboardLeaving = false;
      state.clipboardLeaveTimer = null;
      renderClipboardCard();
    }, CLIPBOARD_EXIT_MS);
  }

  function markClipboardAppearedIfReady(options = {}) {
    const itemId = String(state.clipboardDisplayedId || "");
    if (!itemId) return;
    const visibleFor = Date.now() - Number(state.clipboardShownAt || 0);
    if (options.forceAppeared || visibleFor >= CLIPBOARD_MIN_VISIBLE_MS) {
      state.clipboardLastAppearedId = itemId;
    }
  }

  function renderClipboardCard() {
    if (!els.clipboardCard) return;
    if ((!state.clipboardVisible && !state.clipboardLeaving) || !state.clipboardItem) {
      els.clipboardCard.hidden = true;
      els.clipboardCard.innerHTML = "";
      return;
    }

    const item = state.clipboardItem;
    const meta = clipboardTypeLabel(item.type);
    const action = clipboardActionLabel(item.type);
    const preview = item.type === "image" && item.dataURL
      ? `<img class="memo-clipboard-image" src="${escapeAttr(item.dataURL)}" alt="Clipboard image preview" />`
      : `<p class="memo-clipboard-text">${escapeHTML(compactText(item.content, 180))}</p>`;
    els.clipboardCard.hidden = false;
    els.clipboardCard.classList.toggle("is-leaving", state.clipboardLeaving);
    els.clipboardCard.innerHTML = `
      <header class="memo-clipboard-head">
        <span class="memo-clipboard-type">${escapeHTML(meta)}</span>
        <button class="memo-clipboard-close" type="button" data-action="clipboardDismiss" title="关闭" aria-label="关闭">×</button>
      </header>
      ${preview}
      <footer class="memo-clipboard-actions">
        <button class="memo-secondary-button" type="button" data-action="clipboardDismiss">忽略</button>
        <button class="memo-primary-button" type="button" data-action="clipboardAccept" ${state.clipboardWorking ? "disabled" : ""}>${escapeHTML(action)}</button>
      </footer>
    `;
  }

  function clipboardTypeLabel(type) {
    if (type === "link") return "链接";
    if (type === "image") return "图片";
    return "文本";
  }

  function clipboardActionLabel(type) {
    if (type === "link") return "保存链接";
    if (type === "image") return "上传文件";
    return "创建 memo";
  }

  function acceptClipboardItem() {
    const item = state.clipboardItem;
    if (!item || state.clipboardWorking) return;
    state.clipboardWorking = true;
    renderClipboardCard();
    if (state.activeView === "clipboard") renderClipboardView();

    let task;
    if (item.type === "image") {
      task = uploadClipboardImage(item);
    } else if (item.type === "link") {
      task = createMemoFromContent(item.content, "链接已保存");
    } else {
      task = createMemoFromContent(item.content, "已创建 memo");
    }

    task.then(
      function () {
        hideClipboardCard({ forceAppeared: true });
      },
      function (err) {
        showToast(errorMessage(err));
      },
    ).finally(function () {
      state.clipboardWorking = false;
      renderClipboardCard();
      if (state.activeView === "clipboard") renderClipboardView();
    });
  }

  function uploadClipboardImage(item) {
    if (!item.contentBase64 && !item.dataURL) {
      return Promise.reject(new Error("剪贴板图片为空"));
    }
    return fileInfoToUploadURL({
      name: item.name || "clipboard.png",
      type: item.mimeType || "image/png",
      url: item.dataURL || item.contentBase64,
    }).then(function (uploaded) {
      const name = uploaded.name || item.name || "clipboard.png";
      const url = uploaded.ref || uploaded.url || item.dataURL;
      const content = `![${name}](${url})`;
      return createMemoFromContent(content, "图片已上传并保存");
    });
  }

  function createMemoFromContent(content, successMessage) {
    const text = String(content || "").trim();
    if (!text) return Promise.reject(new Error("剪贴板内容为空"));
    return createMemoInVault(text, state.visibility, state.composerProjectId).then(function (memo) {
      const normalized = normalizeMemoPayload(memo);
      if (!normalized) throw new Error("创建 memo 失败");
      state.memos = [normalized].concat(state.memos);
      saveMemos(state.memos);
      rememberComposerProject(state.composerProjectId);
      state.activeView = "memos";
      state.activeFilter = "all";
      state.activeTag = "";
      state.selectedCalendarDate = "";
      renderAll();
      refreshTasksFromVault();
      showToast(successMessage || "已保存");
      return normalized;
    });
  }

  function openFileInSelectedEditor(button) {
    const file = button.dataset.editorFile || "";
    const label = button.dataset.editorLabel || button.dataset.editorAppName || "编辑器";
    if (!file) {
      showToast("没有可打开的本地文件");
      return;
    }
    if (typeof invoke !== "function") {
      showToast("当前环境不支持打开 " + label);
      return;
    }

    const line = button.dataset.editorLine || "1";
    const col = button.dataset.editorCol || "1";
    const appId = button.dataset.editorAppId || "";
    const appName = button.dataset.editorAppName || "";
    const appPath = button.dataset.editorAppPath || "";
    let url =
      "/api/editor/open?file=" +
      encodeURIComponent(file) +
      "&line=" +
      encodeURIComponent(line) +
      "&col=" +
      encodeURIComponent(col);
    if (appId) url += "&app=" + encodeURIComponent(appId);
    if (appName) url += "&appName=" + encodeURIComponent(appName);
    if (appPath) url += "&appPath=" + encodeURIComponent(appPath);
    button.disabled = true;
    invoke(url, { method: "GET" }).then(
      function (resp) {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || ("打开 " + label + " 失败"));
          return;
        }
        showToast("已在 " + label + " 中打开");
      },
      function (err) {
        showToast("打开 " + label + " 失败: " + err);
      },
    ).finally(function () {
      button.disabled = false;
    });
  }

  function confirmOpenExternalLink(url) {
    openExternalLinkInDefaultBrowser(url);
  }

  function openExternalLinkInDefaultBrowser(url) {
    if (typeof invoke !== "function") {
      window.open(url, "_blank", "noopener");
      return;
    }

    invoke("/api/external/open?url=" + encodeURIComponent(url), { method: "GET" }).then(
      function (resp) {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || "打开链接失败");
        }
      },
      function (err) {
        showToast("打开链接失败: " + err);
      },
    );
  }

  function openImagePreview(element) {
    openImagePreviewFromElement(element).catch(function (err) {
      showToast("打开图片预览失败: " + errorMessage(err));
    });
  }

  function openSettings() {
    if (typeof invoke !== "function") {
      window.open("settings.html");
      return;
    }
    invoke("/api/open_window?pathname=%2Fsettings", { method: "GET" }).then(
      function (resp) {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || "打开设置失败");
        }
      },
      function (err) {
        showToast("打开设置失败: " + err);
      },
    );
  }

  function openSlimMemos() {
    if (typeof invoke !== "function") {
      window.open("memo-slim.html", "_blank", "noopener");
      return;
    }

    invoke("/api/open_window?pathname=%2Fmemo-slim", { method: "GET" }).then(
      function (resp) {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || "打开精简版失败");
          return;
        }
        invoke("__velo/window/close", { args: {} }).catch(function () {
          window.close();
        });
      },
      function (err) {
        showToast("打开精简版失败: " + err);
      },
    );
  }

  function detachMemo(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;

    if (typeof invoke !== "function") {
      window.open("memo-window.html?id=" + encodeURIComponent(memo.id), "_blank", "noopener");
      return;
    }

    invoke("/api/memo-window/open", {
      method: "POST",
      args: {
        memo: memo,
        memos: state.memos,
      },
    }).then(
      function (resp) {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || "分离 memo 失败");
          return;
        }
        showToast("已分离为独立窗口");
      },
      function (err) {
        showToast("分离 memo 失败: " + err);
      },
    );
  }

  function openMemoSearchPalette() {
    state.memoSearchOpen = true;
    state.memoSearchQuery = "";
    state.memoSearchActiveIndex = 0;
    renderMemoSearchPalette();
    window.requestAnimationFrame(function () {
      if (!els.memoSearchInput) return;
      els.memoSearchInput.focus();
      els.memoSearchInput.select();
    });
  }

  function closeMemoSearchPalette() {
    state.memoSearchOpen = false;
    state.memoSearchQuery = "";
    state.memoSearchActiveIndex = 0;
    renderMemoSearchPalette();
  }

  function openMemoSearchResult(memoId) {
    if (!memoId) return;
    const result = memoSearchResults().find((item) => item.key === memoId || item.id === memoId);
    if (result && result.kind === "codeblock") {
      closeMemoSearchPalette();
      copyText(result.block.code).then(
        () => showToast("已复制代码片段"),
        () => showToast("复制失败"),
      );
      return;
    }

    const targetMemoId = result && result.memoId ? result.memoId : memoId;
    const memo = findMemo(targetMemoId);
    if (!memo) {
      showToast("找不到 memo 或代码片段");
      return;
    }
    closeMemoSearchPalette();
    detachMemo(memo.id);
  }

  function renderMemoSearchPalette() {
    if (!els.memoSearchPalette) return;
    els.memoSearchPalette.hidden = !state.memoSearchOpen;
    if (!state.memoSearchOpen) {
      if (els.memoSearchInput) els.memoSearchInput.value = "";
      if (els.memoSearchResults) els.memoSearchResults.innerHTML = "";
      return;
    }

    if (els.memoSearchInput && els.memoSearchInput.value.trim() !== state.memoSearchQuery) {
      els.memoSearchInput.value = state.memoSearchQuery;
    }

    const results = memoSearchResults();
    if (!els.memoSearchResults) return;
    if (!results.length) {
      els.memoSearchResults.innerHTML = '<div class="memo-command-empty">没有匹配的 memo 或代码片段</div>';
      return;
    }

    state.memoSearchActiveIndex = Math.max(0, Math.min(state.memoSearchActiveIndex, results.length - 1));
    els.memoSearchResults.innerHTML = results.map(function (result, index) {
      return [
        '<button class="memo-command-result ' + (index === state.memoSearchActiveIndex ? "is-active" : "") + '" type="button" role="option" aria-selected="' + (index === state.memoSearchActiveIndex ? "true" : "false") + '" data-memo-search-result="' + escapeAttr(result.key) + '">',
        '<span class="memo-command-result-title"><span class="memo-command-result-kind">' + escapeHTML(result.kindLabel) + '</span>' + escapeHTML(result.title) + '</span>',
        '<span class="memo-command-result-summary">' + escapeHTML(result.summary) + '</span>',
        '<span class="memo-command-result-meta">' + escapeHTML(result.meta) + '</span>',
        '</button>',
      ].join("");
    }).join("");

    const active = els.memoSearchResults.querySelector(".memo-command-result.is-active");
    if (active) active.scrollIntoView({ block: "nearest" });
  }

  function memoSearchResults() {
    const query = state.memoSearchQuery.toLowerCase();
    const memoResults = state.memos
      .filter(function (memo) {
        if (!query) return true;
        return matchesSearchQuery([
          memo.id,
          memoTitle(memo),
          memo.content,
          memo.visibility,
          projectLabel(memo.projectId),
        ].join(" "), query);
      })
      .sort(function (a, b) {
        if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
        return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
      })
      .map(function (memo) {
        return {
          id: memo.id,
          key: "memo:" + memo.id,
          kind: "memo",
          kindLabel: "MEMO",
          memoId: memo.id,
          priority: 2,
          summary: compactText(memo.content, 112),
          time: new Date(memo.createdAt).getTime() || 0,
          title: memoTitle(memo),
          meta: [
            memo.archived ? "归档" : "",
            memo.pinned ? "置顶" : "",
            projectLabel(memo.projectId),
            formatRelativeDate(memo.createdAt),
          ].filter(Boolean).join(" · "),
        };
      });

    const codeResults = collectCodeBlocks(state.memos)
      .filter(function (block) {
        if (!query) return block.marked;
        return matchesSearchQuery(codeBlockSearchText(block), query);
      })
      .sort(sortCodeBlocks)
      .map(function (block) {
        return {
          block,
          id: block.id,
          key: "codeblock:" + block.id,
          kind: "codeblock",
          kindLabel: block.marked ? "SNIP" : "CODE",
          memoId: block.memoId,
          priority: block.marked ? 0 : 4,
          summary: compactText(block.code, 112),
          time: new Date(block.memo.createdAt).getTime() || 0,
          title: block.label || "代码片段",
          meta: [
            block.marked ? "代码片段" : "未标记代码块",
            block.language,
            block.aliases && block.aliases.length ? block.aliases.join(" ") : "",
            projectLabel(block.memo.projectId),
            "Enter 复制",
          ].filter(Boolean).join(" · "),
        };
      });

    return codeResults.concat(memoResults)
      .sort(function (a, b) {
        if (query && a.priority !== b.priority) return a.priority - b.priority;
        if (!query && a.kind !== b.kind) return a.kind === "memo" ? -1 : 1;
        return state.sortDesc ? b.time - a.time : a.time - b.time;
      })
      .slice(0, 12);
  }

  function openMemoSearchShortcut() {
    try {
      const settings = JSON.parse(localStorage.getItem(SHORTCUTS_STORAGE_KEY) || "null") || {};
      if (settings.enabled === false) return "";
      return normalizeShortcut(settings.openMemoSearch) || "Ctrl+O";
    } catch (_) {
      return "Ctrl+O";
    }
  }

  function matchesShortcut(event, shortcut) {
    if (!shortcut) return false;
    const parts = shortcut.split("+");
    const key = parts.pop();
    const wanted = {
      alt: parts.includes("Alt"),
      ctrl: parts.includes("Ctrl"),
      meta: parts.includes("Meta"),
      shift: parts.includes("Shift"),
    };
    if (Boolean(event.altKey) !== wanted.alt) return false;
    if (Boolean(event.ctrlKey) !== wanted.ctrl) return false;
    if (Boolean(event.metaKey) !== wanted.meta) return false;
    if (Boolean(event.shiftKey) !== wanted.shift) return false;
    return shortcutKeyName(event.key) === key;
  }

  function normalizeShortcut(value) {
    const raw = String(value || "").trim();
    if (!raw) return "";
    const parts = raw.split("+").map((part) => part.trim()).filter(Boolean);
    if (!parts.length) return "";
    const modifiers = [];
    let key = "";
    parts.forEach(function (part) {
      const lower = part.toLowerCase();
      if (lower === "ctrl" || lower === "control") {
        if (!modifiers.includes("Ctrl")) modifiers.push("Ctrl");
      } else if (lower === "alt" || lower === "option") {
        if (!modifiers.includes("Alt")) modifiers.push("Alt");
      } else if (lower === "shift") {
        if (!modifiers.includes("Shift")) modifiers.push("Shift");
      } else if (lower === "meta" || lower === "cmd" || lower === "command") {
        if (!modifiers.includes("Meta")) modifiers.push("Meta");
      } else {
        key = shortcutKeyName(part);
      }
    });
    if (!key) return "";
    if (!modifiers.length && key.length === 1) return "";
    return modifiers.concat(key).join("+");
  }

  function shortcutKeyName(key) {
    const original = String(key || "");
    if (original === " ") return "Space";
    const value = original.trim();
    if (!value) return "";
    const lower = value.toLowerCase();
    if (lower === "control" || lower === "shift" || lower === "alt" || lower === "meta") return "";
    if (lower === "escape" || lower === "esc") return "Esc";
    if (lower === "arrowup") return "Up";
    if (lower === "arrowdown") return "Down";
    if (lower === "arrowleft") return "Left";
    if (lower === "arrowright") return "Right";
    if (value.length === 1) return value.toUpperCase();
    return value.charAt(0).toUpperCase() + value.slice(1);
  }

  function focusMemo(memoId) {
    const memo = findMemo(memoId);
    if (!memo) {
      showToast("找不到引用的 memo");
      return;
    }

    state.activeView = "memos";
    state.activeFilter = memo.archived ? "archive" : "all";
    state.activeTag = "";
    state.activeProjectFilter = "all";
    state.editingId = "";
    state.editPreviewVisible = false;
    state.sourceDraft = "";
    state.sourceEditingId = "";
    state.query = "";
    state.selectedCalendarDate = "";
    els.searchInput.value = "";
    renderAll();

    window.requestAnimationFrame(function () {
      const target = els.memoList.querySelector(`[data-memo-id="${escapeCSSIdent(memoId)}"]`);
      if (!target) return;
      target.scrollIntoView({ block: "center", behavior: "smooth" });
      target.classList.add("is-highlighted");
      if (state.highlightTimer) window.clearTimeout(state.highlightTimer);
      state.highlightTimer = window.setTimeout(function () {
        target.classList.remove("is-highlighted");
        state.highlightTimer = null;
      }, 1500);
    });
  }

  function openSourceMemo(memoId) {
    focusMemo(memoId);
  }

  function selectProjectFilter(value) {
    const next = normalizeProjectFilter(value);
    clearRetainedCompletedTasks();
    state.activeProjectFilter = next;
    state.activeTag = "";
    state.selectedCalendarDate = "";
    if (next === "unassigned") {
      state.composerProjectId = "";
    } else if (next !== "all") {
      state.composerProjectId = next;
      rememberComposerProject(next);
    } else {
      state.composerProjectId = state.lastComposerProjectId || "";
    }
    renderAll();
  }

  function rememberComposerProject(projectId) {
    state.lastComposerProjectId = projectId || "";
    localStorage.setItem(LAST_PROJECT_STORAGE_KEY, state.lastComposerProjectId);
  }

  function handleInput(event) {
    if (event.target.matches("[data-memo-search-input]")) {
      state.memoSearchQuery = event.target.value.trim();
      state.memoSearchActiveIndex = 0;
      renderMemoSearchPalette();
      return;
    }

    if (event.target.matches("[data-search-input]")) {
      state.query = event.target.value.trim();
      renderAll();
      return;
    }

    if (event.target.matches("[data-memo-source-yaml]")) {
      state.sourceDraft = event.target.value;
    }
  }

  function handleKeydown(event) {
    if (state.memoSearchOpen) {
      handleMemoSearchKeydown(event);
      return;
    }

    if (event.key === "Enter" || event.key === " ") {
      const imagePreview = closestElement(event.target, "[data-image-preview-src]");
      if (imagePreview && root.contains(imagePreview)) {
        event.preventDefault();
        openImagePreview(imagePreview);
        return;
      }
    }

    if (!matchesShortcut(event, openMemoSearchShortcut())) return;
    event.preventDefault();
    event.stopPropagation();
    openMemoSearchPalette();
  }

  function handleMemoSearchKeydown(event) {
    if (event.key === "Escape") {
      event.preventDefault();
      closeMemoSearchPalette();
      return;
    }

    const results = memoSearchResults();
    if (event.key === "ArrowDown") {
      event.preventDefault();
      state.memoSearchActiveIndex = results.length ? (state.memoSearchActiveIndex + 1) % results.length : 0;
      renderMemoSearchPalette();
      return;
    }

    if (event.key === "ArrowUp") {
      event.preventDefault();
      state.memoSearchActiveIndex = results.length ? (state.memoSearchActiveIndex - 1 + results.length) % results.length : 0;
      renderMemoSearchPalette();
      return;
    }

    if (event.key === "Enter") {
      event.preventDefault();
      if (!results.length) return;
      const active = results[Math.min(state.memoSearchActiveIndex, results.length - 1)];
      if (active) openMemoSearchResult(active.id);
    }
  }

  function handleChange(event) {
    if (event.target.matches("[data-visibility-select]")) {
      state.visibility = event.target.value;
      return;
    }

    if (event.target.matches("[data-project-select]")) {
      state.composerProjectId = normalizeProjectID(event.target.value);
      rememberComposerProject(state.composerProjectId);
      renderComposerProjectSelect();
      return;
    }

    if (event.target.matches("[data-edit-visibility]")) {
      state.editVisibility = event.target.value;
      return;
    }

    if (event.target.matches("[data-edit-project]")) {
      state.editProjectId = normalizeProjectID(event.target.value);
      return;
    }

    if (event.target.matches("[data-task-line]")) {
      const memoNode = closestElement(event.target, "[data-memo-id]");
      if (!memoNode) return;
      toggleTask(memoNode.dataset.memoId, Number(event.target.dataset.taskLine), event.target.checked);
      return;
    }

    if (event.target.matches("[data-task-complete]")) {
      const taskNode = closestElement(event.target, "[data-task-id]");
      if (!taskNode) return;
      toggleExistingTaskCompletion(taskNode.dataset.taskId, event.target);
      return;
    }

    if (event.target.matches("[data-gtd-item-complete]")) {
      const itemNode = closestElement(event.target, "[data-gtd-item-id]");
      if (!itemNode) return;
      toggleExistingGTDItemCompletion(itemNode.dataset.gtdItemId, event.target);
      return;
    }

    if (event.target.matches("[data-attach-input]")) {
      insertFiles(event.target.files);
      event.target.value = "";
    }
  }

  function runComposerCommand(command) {
    if (!composerEditor) return;

    const commands = {
      attach() {
        requestFilesForComposer("");
      },
      bold() {
        composerEditor.wrap("**", "**", "加粗文本");
      },
      checklist() {
        composerEditor.insertBlock("- [ ] ");
      },
      code() {
        composerEditor.wrap("`", "`", "code");
      },
      date() {
        composerEditor.insertText(new Date().toLocaleString());
      },
      image() {
        requestFilesForComposer("image/*");
      },
      italic() {
        composerEditor.wrap("*", "*", "斜体文本");
      },
      link() {
        composerEditor.wrap("[", "](https://)", "链接文本");
      },
      list() {
        composerEditor.insertBlock("- ");
      },
      tag() {
        composerEditor.insertText("#");
      },
    };

    if (commands[command]) commands[command]();
    composerEditor.focus();
  }

  function toggleComposerPreview() {
    state.composerPreviewVisible = !state.composerPreviewVisible;
    renderComposerPreview();
    if (!state.composerPreviewVisible && composerEditor) composerEditor.focus();
  }

  function toggleEditPreview(memoId) {
    if (!memoId || state.editingId !== memoId) return;
    syncEditDraftFromEditor();
    state.editPreviewVisible = !state.editPreviewVisible;
    renderEditPreview(memoId);
    if (!state.editPreviewVisible && editEditor) editEditor.focus();
  }

  function toggleCommentPreview(memoId) {
    if (!memoId || state.commentingMemoId !== memoId) return;
    syncCommentDraftFromEditor();
    state.commentPreviewVisible = !state.commentPreviewVisible;
    renderCommentPreview(memoId);
    if (!state.commentPreviewVisible && commentEditor) commentEditor.focus();
  }

  function toggleCommentEditPreview(commentId) {
    if (!commentId || state.commentEditingId !== commentId) return;
    syncCommentEditDraftFromEditor();
    state.commentEditPreviewVisible = !state.commentEditPreviewVisible;
    renderCommentEditPreview(commentId);
    if (!state.commentEditPreviewVisible && commentEditEditor) commentEditEditor.focus();
  }

  function renderEditablePreviews() {
    renderComposerPreview();
    if (state.editingId) renderEditPreview(state.editingId);
    if (state.commentingMemoId) renderCommentPreview(state.commentingMemoId);
    if (state.commentEditingId) renderCommentEditPreview(state.commentEditingId);
  }

  function renderComposerPreview() {
    const content = composerEditor ? composerEditor.getText() : "";
    renderEditorPreviewPanel(
      root.querySelector("[data-composer-preview]"),
      root.querySelector('[data-action="toggleComposerPreview"]'),
      state.composerPreviewVisible,
      content,
      memoRenderContext("", { readonly: true }),
    );
  }

  function renderEditPreview(memoId) {
    const content = editEditor ? editEditor.getText() : state.editDraft;
    renderEditorPreviewPanel(
      els.memoList.querySelector("[data-edit-preview]"),
      els.memoList.querySelector('[data-action="toggleEditPreview"]'),
      state.editPreviewVisible,
      content,
      memoRenderContext(memoId, { readonly: true }),
    );
  }

  function renderCommentPreview(memoId) {
    const content = commentEditor ? commentEditor.getText() : state.commentDraft;
    renderEditorPreviewPanel(
      els.memoList.querySelector("[data-comment-preview]"),
      els.memoList.querySelector('[data-action="toggleCommentPreview"]'),
      state.commentPreviewVisible,
      content,
      memoRenderContext(memoId, { readonly: true, showLineNumbers: false }),
    );
  }

  function renderCommentEditPreview(commentId) {
    const comment = findComment(commentId);
    const content = commentEditEditor ? commentEditEditor.getText() : state.commentEditDraft;
    renderEditorPreviewPanel(
      els.memoList.querySelector("[data-comment-edit-preview]"),
      els.memoList.querySelector('[data-action="toggleCommentEditPreview"]'),
      state.commentEditPreviewVisible,
      content,
      memoRenderContext(comment && comment.memoId, { readonly: true, showLineNumbers: false }),
    );
  }

  function renderEditorPreviewPanel(panel, button, visible, content, context) {
    updateEditorPreviewButton(button, visible);
    if (!panel) return;
    const switcher = closestElement(panel, ".memo-editor-switch");
    const host = switcher && switcher.querySelector("[data-editor-switch-host]");
    if (host) host.hidden = visible;
    panel.hidden = !visible;
    panel.classList.toggle("is-visible", visible);
    if (!visible) {
      panel.innerHTML = "";
      return;
    }
    panel.innerHTML = editorPreviewHTML(content, context);
  }

  function updateEditorPreviewButton(button, visible) {
    if (!button) return;
    button.setAttribute("aria-pressed", visible ? "true" : "false");
    button.title = visible ? "编辑" : "预览";
    button.setAttribute("aria-label", visible ? "编辑" : "预览");
    const label = button.querySelector("span");
    if (label) label.textContent = visible ? "编辑" : "预览";
  }

  function editorPreviewHTML(content, context) {
    const text = String(content || "");
    if (!text.trim()) return '<div class="memo-editor-preview-empty">暂无预览内容</div>';
    try {
      return `<div class="memo-content">${renderMemoMarkdown(text, context || {})}</div>`;
    } catch (err) {
      return `<div class="memo-content"><p>${escapeHTML(text)}</p></div>`;
    }
  }

  function createMemo(options = {}) {
    if (state.saving) return Promise.resolve({ ok: false, message: "正在保存" });
    const content = composerEditor.getText();
    if (!content.trim()) {
      showToast("先写点内容");
      composerEditor.focus();
      return Promise.resolve({ ok: false, message: "先写点内容" });
    }

    state.saving = true;
    renderComposerStatus(content);
    return createMemoInVault(content, state.visibility, state.composerProjectId).then(
      function (memo) {
        const normalized = normalizeMemoPayload(memo);
        state.memos = [normalized].filter(Boolean).concat(state.memos);
        saveMemos(state.memos);
        rememberComposerProject(state.composerProjectId);
        composerEditor.setText("");
        removeDraftFromState(COMPOSER_DRAFT_ID);
        state.composerPreviewVisible = false;
        state.activeView = "memos";
        state.activeFilter = "all";
        state.activeTag = "";
        state.selectedCalendarDate = "";
        renderAll();
        renderComposerStatus("");
        refreshTasksFromVault();
        showToast("已发布到 " + projectLabel(normalized && normalized.projectId));
        deleteMemoDraftInVault(COMPOSER_DRAFT_ID).catch(function (err) {
          showToast("清理草稿失败: " + errorMessage(err));
        });
        if (options.source !== "vim-wq") {
          window.requestAnimationFrame(() => {
            if (composerEditor && els.composerHost.isConnected) composerEditor.focus();
          });
        }
        return { ok: true, message: "已发布" };
      },
      function (err) {
        showToast("发布失败: " + errorMessage(err));
        return { ok: false, message: "发布失败: " + errorMessage(err) };
      },
    ).finally(function () {
      state.saving = false;
      renderComposerStatus(composerEditor.getText());
    });
  }

  function writeComposerDraft() {
    if (!composerEditor) return Promise.resolve({ ok: false, message: "没有可保存的草稿" });
    const content = composerEditor.getText();
    if (!content.trim()) {
      return clearComposerDraft({ clearEditor: false, message: "空草稿已清理" });
    }

    return upsertMemoDraftInVault({
      content,
      id: COMPOSER_DRAFT_ID,
      kind: "composer",
      projectId: state.composerProjectId,
      visibility: state.visibility,
    }).then(
      function (draft) {
        upsertDraftInState(draft);
        showToast("草稿已保存");
        return { ok: true, message: "draft written" };
      },
      function (err) {
        showToast("保存草稿失败: " + errorMessage(err));
        return { ok: false, message: "保存草稿失败: " + errorMessage(err) };
      },
    );
  }

  function clearComposerDraft(options = {}) {
    removeDraftFromState(COMPOSER_DRAFT_ID);
    if (options.clearEditor && composerEditor) {
      composerEditor.setText("");
      renderComposerStatus("");
    }
    if (options.clearEditor) {
      state.composerPreviewVisible = false;
      renderComposerPreview();
    }
    return deleteMemoDraftInVault(COMPOSER_DRAFT_ID).then(
      function () {
        if (options.message) showToast(options.message);
        return { ok: true, message: options.message || "empty draft cleared" };
      },
      function (err) {
        showToast("删除草稿失败: " + errorMessage(err));
        return { ok: false, message: "删除草稿失败: " + errorMessage(err) };
      },
    );
  }

  function exitComposer() {
    if (composerEditor && typeof composerEditor.blur === "function") composerEditor.blur();
    return Promise.resolve({ ok: true, message: "quit" });
  }

  function createTaskFromForm(form) {
    const data = new FormData(form);
    const title = String(data.get("title") || "").trim();
    if (!title) {
      showToast("任务标题不能为空");
      return;
    }
    let dueAt = String(data.get("dueAt") || "").trim();
    if (!dueAt && state.taskFilter === "today") {
      dueAt = dateKey(new Date());
    }
    const priority = String(data.get("priority") || "none").trim();
    const projectId = state.activeProjectFilter && state.activeProjectFilter !== "all" && state.activeProjectFilter !== "unassigned"
      ? state.activeProjectFilter
      : "";
    const payload = {
      dueAt,
      listId: state.taskFilter === "inbox" ? "inbox" : "",
      priority,
      projectId,
      title,
    };
    createTask(payload).then(
      function (task) {
        const summary = normalizeTaskSummary(task);
        if (summary) state.tasks = [summary].concat(state.tasks);
        form.reset();
        renderAll();
        refreshTasksFromVault();
        showToast("已创建任务");
      },
      function (err) {
        showToast("创建任务失败: " + errorMessage(err));
      },
    );
  }

  function createGTDItemFromForm(form) {
    const data = new FormData(form);
    const title = String(data.get("title") || "").trim();
    if (!title) {
      showToast("事项标题不能为空");
      return;
    }
    const projectId = state.activeProjectFilter && state.activeProjectFilter !== "all" && state.activeProjectFilter !== "unassigned"
      ? state.activeProjectFilter
      : "";
    createGTDItem({
      milestoneId: String(data.get("milestoneId") || "").trim(),
      projectId,
      title,
      type: String(data.get("type") || "idea").trim(),
    }).then(
      function (item) {
        state.gtdItems = [item].concat(state.gtdItems);
        form.reset();
        renderAll();
        showToast("已添加开放事项");
      },
      function (err) {
        showToast("添加事项失败: " + errorMessage(err));
      },
    );
  }

  function createGTDMilestoneFromForm(form) {
    const data = new FormData(form);
    const title = String(data.get("title") || "").trim();
    if (!title) {
      showToast("里程碑标题不能为空");
      return;
    }
    const projectIds = state.activeProjectFilter && state.activeProjectFilter !== "all" && state.activeProjectFilter !== "unassigned"
      ? [state.activeProjectFilter]
      : [];
    createGTDMilestone({
      projectIds,
      status: String(data.get("status") || "planned").trim(),
      targetAt: String(data.get("targetAt") || "").trim(),
      title,
    }).then(
      function (milestone) {
        state.gtdMilestones = [milestone].concat(state.gtdMilestones);
        form.reset();
        renderAll();
        showToast("已添加里程碑");
      },
      function (err) {
        showToast("添加里程碑失败: " + errorMessage(err));
      },
    );
  }

  function updateExistingGTDItem(itemId, patch, message) {
    const id = String(itemId || "").trim();
    if (!id) return;
    updateGTDItem(id, patch).then(
      function (item) {
        state.gtdItems = state.gtdItems.map((entry) => entry.id === id ? item : entry);
        renderAll();
        showToast(message || "已更新事项");
      },
      function (err) {
        showToast("更新事项失败: " + errorMessage(err));
        refreshGTDFromVault();
      },
    );
  }

  function closeExistingGTDItem(itemId) {
    const id = String(itemId || "").trim();
    if (!id) return;
    closeGTDItem(id).then(
      function (item) {
        state.gtdItems = state.gtdItems.map((entry) => entry.id === id ? item : entry);
        renderAll();
        showToast("已关闭事项");
      },
      function (err) {
        showToast("关闭事项失败: " + errorMessage(err));
        refreshGTDFromVault();
      },
    );
  }

  function toggleExistingGTDItemCompletion(itemId, checkbox) {
    const id = String(itemId || "").trim();
    if (!id || !checkbox) return;
    const checked = checkbox.checked;
    const itemCard = closestElement(checkbox, "[data-gtd-item-id]");
    checkbox.disabled = true;
    const request = checked
      ? closeGTDItem(id)
      : updateGTDItem(id, { status: "open" });
    request.then(
      function (item) {
        state.gtdItems = state.gtdItems.map((entry) => entry.id === id ? item : entry);
        replaceGTDItemCard(itemCard, item);
        showToast(checked ? "已关闭事项" : "已重新打开事项");
      },
      function (err) {
        checkbox.checked = !checked;
        checkbox.disabled = false;
        showToast((checked ? "关闭事项失败: " : "重新打开事项失败: ") + errorMessage(err));
      },
    );
  }

  function updateExistingGTDMilestone(milestoneId, patch, message) {
    const id = String(milestoneId || "").trim();
    if (!id) return;
    updateGTDMilestone(id, patch).then(
      function (milestone) {
        state.gtdMilestones = state.gtdMilestones.map((entry) => entry.id === id ? milestone : entry);
        renderAll();
        showToast(message || "已更新里程碑");
      },
      function (err) {
        showToast("更新里程碑失败: " + errorMessage(err));
        refreshGTDFromVault();
      },
    );
  }

  function toggleExistingTaskCompletion(taskId, checkbox) {
    const id = String(taskId || "").trim();
    if (!id || !checkbox) return;
    const checked = checkbox.checked;
    const completedInFilter = state.taskFilter;
    const taskCard = checkbox ? closestElement(checkbox, "[data-task-id]") : null;
    checkbox.disabled = true;
    const request = checked
      ? completeTask(id)
      : updateTask(id, { completedAt: "", status: "open" });
    request.then(
      function (task) {
        const summary = normalizeTaskSummary(task);
        if (checked) {
          retainCompletedTaskInFilter(id, completedInFilter);
        } else {
          state.retainedCompletedTaskFilters.delete(id);
        }
        state.tasks = state.tasks.map((item) => item.id === id && summary ? summary : item);
        replaceTaskCard(taskCard, summary);
        showToast(checked ? "已完成任务" : "已取消完成");
      },
      function (err) {
        checkbox.checked = !checked;
        checkbox.disabled = false;
        showToast((checked ? "完成任务失败: " : "取消完成失败: ") + errorMessage(err));
      },
    );
  }

  function replaceGTDItemCard(card, item) {
    if (!card || !item) return;
    card.outerHTML = gtdItemCardTemplate(item, {
      milestones: state.gtdMilestones,
      projects: state.projects,
    });
  }

  function replaceTaskCard(card, task) {
    if (!card || !task) return;
    card.outerHTML = taskCardTemplate(task, {
      memos: state.memos,
      projects: state.projects,
    });
  }

  function addTaskNote(taskId) {
    const task = findTask(taskId);
    if (!task) return;
    const content = window.prompt("添加 task note，支持 Markdown 和 todo 行", "");
    if (content === null) return;
    if (!content.trim()) {
      showToast("note 内容不能为空");
      return;
    }
    createTaskNote(task.id, { content, visibility: DEFAULT_VISIBILITY }).then(
      function (result) {
        const summary = normalizeTaskSummary(result.task);
        if (summary) state.tasks = state.tasks.map((item) => item.id === task.id ? summary : item);
        if (result.memo) {
          const memo = normalizeMemoPayload(result.memo);
          if (memo) state.memos = [memo].concat(state.memos.filter((item) => item.id !== memo.id));
        }
        saveMemos(state.memos);
        renderAll();
        refreshTasksFromVault();
        showToast("已添加 note");
      },
      function (err) {
        showToast("添加 note 失败: " + errorMessage(err));
      },
    );
  }

  function copyTaskRef(taskId) {
    const task = findTask(taskId);
    if (!task) return;
    copyText(`[[task:${task.id}|${task.title}]]`).then(
      () => showToast("已复制 task 引用"),
      () => showToast("复制失败"),
    );
  }


  function startComment(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    if (commentEditor && state.commentingMemoId === memoId) {
      commentEditor.focus();
      return;
    }
    if (commentEditor) syncCommentDraftFromEditor();
    state.commentingMemoId = memoId;
    state.commentDraft = "";
    state.commentPreviewVisible = false;
    state.commentEditingId = "";
    state.commentEditDraft = "";
    state.commentEditPreviewVisible = false;
    state.editingId = "";
    state.editDraft = "";
    state.editPreviewVisible = false;
    state.sourceDraft = "";
    state.sourceEditingId = "";
    if (!state.expandedMemoIds.has(memoId)) state.expandedMemoIds.add(memoId);
    renderFeed();
  }

  function startCommentEdit(commentId) {
    const comment = findComment(commentId);
    if (!comment) return;
    if (commentEditEditor && state.commentEditingId === comment.id) {
      commentEditEditor.focus();
      return;
    }
    if (commentEditEditor) syncCommentEditDraftFromEditor();
    state.commentEditingId = comment.id;
    state.commentEditDraft = comment.content || "";
    state.commentEditPreviewVisible = false;
    state.commentingMemoId = "";
    state.commentDraft = "";
    state.commentPreviewVisible = false;
    state.editingId = "";
    state.editDraft = "";
    state.editPreviewVisible = false;
    state.sourceDraft = "";
    state.sourceEditingId = "";
    if (comment.memoId && !state.expandedMemoIds.has(comment.memoId)) state.expandedMemoIds.add(comment.memoId);
    renderFeed();
  }

  function cancelCommentEdit(options = {}) {
    state.commentEditingId = "";
    state.commentEditDraft = "";
    state.commentEditPreviewVisible = false;
    renderFeed();
    if (options.message) showToast(options.message);
    return Promise.resolve({ ok: true, message: options.message || "comment edit cancelled" });
  }

  function cancelComment(options = {}) {
    state.commentingMemoId = "";
    state.commentDraft = "";
    state.commentPreviewVisible = false;
    renderFeed();
    if (options.message) showToast(options.message);
    return Promise.resolve({ ok: true, message: options.message || "comment cancelled" });
  }

  function saveComment(memoId, options = {}) {
    const memo = findMemo(memoId);
    if (!memo) return Promise.resolve({ ok: false, message: "找不到 memo" });
    if (state.commentSaving) return Promise.resolve({ ok: false, message: "正在保存" });
    const content = commentEditor ? commentEditor.getText() : state.commentDraft;
    if (!content.trim()) {
      showToast("评论不能为空");
      if (commentEditor) commentEditor.focus();
      return Promise.resolve({ ok: false, message: "评论不能为空" });
    }

    state.commentSaving = true;
    return createMemoCommentInVault(memoId, content).then(
      function (comment) {
        upsertCommentInState(comment);
        state.commentingMemoId = "";
        state.commentDraft = "";
        state.commentPreviewVisible = false;
        renderFeed();
        if (options.source !== "vim-wq") showToast("已添加评论");
        return { ok: true, message: "已添加评论" };
      },
      function (err) {
        showToast("评论失败: " + errorMessage(err));
        return { ok: false, message: "评论失败: " + errorMessage(err) };
      },
    ).finally(function () {
      state.commentSaving = false;
    });
  }

  function saveCommentEdit() {
    const comment = findComment(state.commentEditingId);
    if (!comment) return Promise.resolve({ ok: false, message: "找不到评论" });
    if (state.commentSaving) return Promise.resolve({ ok: false, message: "正在保存" });
    const content = commentEditEditor ? commentEditEditor.getText() : state.commentEditDraft;
    if (!content.trim()) {
      showToast("评论不能为空");
      if (commentEditEditor) commentEditEditor.focus();
      return Promise.resolve({ ok: false, message: "评论不能为空" });
    }

    state.commentSaving = true;
    return updateMemoCommentInVault(comment.id, { content }).then(
      function (updated) {
        upsertCommentInState(updated);
        state.commentEditingId = "";
        state.commentEditDraft = "";
        state.commentEditPreviewVisible = false;
        renderFeed();
        showToast("已保存评论");
        return { ok: true, message: "已保存评论" };
      },
      function (err) {
        showToast("保存评论失败: " + errorMessage(err));
        return { ok: false, message: "保存评论失败: " + errorMessage(err) };
      },
    ).finally(function () {
      state.commentSaving = false;
    });
  }

  function deleteComment(commentId) {
    const comment = findComment(commentId);
    if (!comment || state.commentSaving) return;
    if (!window.confirm("删除这条评论？")) return;

    state.commentSaving = true;
    deleteMemoCommentInVault(comment.id, { cleanupAssets: true }).then(
      function () {
        state.comments = state.comments.filter((item) => item.id !== comment.id);
        if (state.commentEditingId === comment.id) {
          state.commentEditingId = "";
          state.commentEditDraft = "";
          state.commentEditPreviewVisible = false;
        }
        renderFeed();
        showToast("已删除评论");
      },
      function (err) {
        showToast("删除评论失败: " + errorMessage(err));
      },
    ).finally(function () {
      state.commentSaving = false;
    });
  }

  function exitComment(memoId) {
    const memo = findMemo(memoId);
    if (!memo) {
      state.commentingMemoId = "";
      state.commentDraft = "";
      state.commentEditingId = "";
      state.commentEditDraft = "";
      state.commentEditPreviewVisible = false;
      state.commentPreviewVisible = false;
      renderFeed();
      return Promise.resolve({ ok: true, message: "quit" });
    }
    syncCommentDraftFromEditor();
    state.commentingMemoId = "";
    state.commentPreviewVisible = false;
    renderFeed();
    return Promise.resolve({ ok: true, message: "quit" });
  }

  function startEdit(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    const draft = findDraft(memoEditDraftId(memoId));
    state.commentingMemoId = "";
    state.commentDraft = "";
    state.commentPreviewVisible = false;
    state.commentEditingId = "";
    state.commentEditDraft = "";
    state.commentEditPreviewVisible = false;
    state.sourceDraft = "";
    state.sourceEditingId = "";
    state.editingId = memoId;
    state.editDraft = draft ? draft.content : memo.content;
    state.editPreviewVisible = false;
    state.editProjectId = draft ? normalizeProjectID(draft.projectId) : memo.projectId || "";
    state.editVisibility = draft ? draft.visibility || DEFAULT_VISIBILITY : memo.visibility || DEFAULT_VISIBILITY;
    renderFeed();
    if (draft) showToast("已恢复编辑草稿");
  }

  function startSourceEdit(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    state.commentingMemoId = "";
    state.commentDraft = "";
    state.commentPreviewVisible = false;
    state.commentEditingId = "";
    state.commentEditDraft = "";
    state.commentEditPreviewVisible = false;
    state.editingId = "";
    state.editDraft = "";
    state.editPreviewVisible = false;
    state.sourceEditingId = memoId;
    state.sourceDraft = memoSourceYaml(memo);
    renderFeed();
    window.requestAnimationFrame(function () {
      const input = els.memoList.querySelector("[data-memo-source-yaml]");
      if (input) input.focus();
    });
  }

  function toggleMemoExpand(memoId) {
    if (!memoId) return;
    if (state.expandedMemoIds.has(memoId)) {
      state.expandedMemoIds.delete(memoId);
    } else {
      state.expandedMemoIds.add(memoId);
    }
    renderPinned();
    if (state.activeView === "memos") renderFeed();
  }

  function cancelEdit() {
    const draftId = state.editingId ? memoEditDraftId(state.editingId) : "";
    state.editingId = "";
    state.editDraft = "";
    state.editPreviewVisible = false;
    renderFeed();
    if (draftId) {
      removeDraftFromState(draftId);
      deleteMemoDraftInVault(draftId).catch(function (err) {
        showToast("删除草稿失败: " + errorMessage(err));
      });
    }
  }

  function cancelSourceEdit() {
    state.sourceDraft = "";
    state.sourceEditingId = "";
    renderFeed();
  }

  function saveMemoSource(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return Promise.resolve({ ok: false, message: "找不到 memo" });
    const parsed = parseMemoSourceYaml(state.sourceDraft, memo);
    if (parsed.error) {
      showToast(parsed.error);
      return Promise.resolve({ ok: false, message: parsed.error });
    }
    state.sourceDraft = "";
    state.sourceEditingId = "";
    return updateMemo(memoId, parsed.patch).then(function (result) {
      if (!result || result.ok !== false) showToast("源数据已保存");
      return result;
    });
  }

  function saveEdit(memoId, options = {}) {
    const memo = findMemo(memoId);
    if (!memo) return Promise.resolve({ ok: false, message: "找不到 memo" });
    const content = editEditor ? editEditor.getText() : state.editDraft;
    if (!content.trim()) {
      showToast("内容不能为空");
      return Promise.resolve({ ok: false, message: "内容不能为空" });
    }
    state.editingId = "";
    state.editDraft = "";
    state.editPreviewVisible = false;
    return updateMemo(memoId, {
      content,
      projectId: state.editProjectId,
      updatedAt: new Date().toISOString(),
      visibility: state.editVisibility,
    }).then(function (result) {
      if (result && result.ok === false) return result;
      removeDraftFromState(memoEditDraftId(memoId));
      return deleteMemoDraftInVault(memoEditDraftId(memoId)).then(
        function () {
          if (options.source === "vim-wq") return { ok: true, message: "committed" };
          return result || { ok: true, message: "已保存" };
        },
        function (err) {
          showToast("清理草稿失败: " + errorMessage(err));
          return { ok: false, message: "清理草稿失败: " + errorMessage(err) };
        },
      );
    });
  }

  function updateMemo(memoId, patch) {
    let nextMemo = null;
    state.memos = state.memos.map((memo) => {
      if (memo.id !== memoId) return memo;
      nextMemo = {
        ...memo,
        ...patch,
        updatedAt: patch.updatedAt || memo.updatedAt,
      };
      return nextMemo;
    });
    saveMemos(state.memos);
    renderAll();
    if (nextMemo) {
      return updateMemoInVault(memoId, patch).then(
        function (memo) {
          const normalized = normalizeMemoPayload(memo);
          if (!normalized) return { ok: false, message: "保存失败" };
          state.memos = state.memos.map((item) => item.id === memoId ? normalized : item);
          saveMemos(state.memos);
          renderAll();
          if (Object.prototype.hasOwnProperty.call(patch, "content")) {
            refreshTasksFromVault();
          }
          return { ok: true, message: "已保存" };
        },
        function (err) {
          showToast("保存失败: " + errorMessage(err));
          refreshMemosFromVault();
          return { ok: false, message: "保存失败: " + errorMessage(err) };
        },
      );
    }
    return Promise.resolve({ ok: false, message: "找不到 memo" });
  }

  function memoSourceYaml(memo) {
    const lines = [
      ["id", memo.id || ""],
      ["createdAt", memo.createdAt || ""],
      ["updatedAt", memo.updatedAt || ""],
      ["visibility", memo.visibility || DEFAULT_VISIBILITY],
      ["pinned", Boolean(memo.pinned)],
      ["archived", Boolean(memo.archived)],
      ["projectId", memo.projectId || ""],
      ["kind", memo.kind || ""],
      ["taskId", memo.taskId || ""],
    ];
    return lines.map(function ([key, value]) {
      if (typeof value === "boolean") return `${key}: ${value ? "true" : "false"}`;
      return `${key}: ${yamlQuote(value)}`;
    }).join("\n");
  }

  function parseMemoSourceYaml(value, memo) {
    const meta = {};
    const allowed = new Set(["id", "createdAt", "created_at", "updatedAt", "updated_at", "visibility", "pinned", "archived", "projectId", "kind", "taskId"]);
    const ignored = new Set(["schemaVersion", "contentWhitespace", "tags", "references"]);
    const lines = String(value || "").replace(/\r\n?/g, "\n").split("\n");
    for (let index = 0; index < lines.length; index += 1) {
      const line = lines[index].trim();
      if (!line || line.startsWith("#") || line === "---") continue;
      if (/^-\s/.test(line)) continue;
      const colon = line.indexOf(":");
      if (colon < 0) return { error: `源数据第 ${index + 1} 行缺少冒号` };
      const key = line.slice(0, colon).trim();
      if (!key) return { error: `源数据第 ${index + 1} 行缺少字段名` };
      if (ignored.has(key)) continue;
      if (!allowed.has(key)) return { error: `不支持的源数据字段: ${key}` };
      meta[key] = yamlScalarValue(line.slice(colon + 1));
    }

    const id = String(meta.id || "").trim();
    if (!id) return { error: "源数据必须保留 id" };
    if (id !== memo.id) return { error: "暂不支持修改 memo id" };

    const createdAt = String(meta.createdAt || meta.created_at || "").trim();
    if (!createdAt) return { error: "createdAt 不能为空" };
    if (!isValidMemoTime(createdAt)) return { error: "createdAt 必须是 RFC3339 时间" };

    const updatedAt = String(meta.updatedAt || meta.updated_at || "").trim();
    if (updatedAt && !isValidMemoTime(updatedAt)) return { error: "updatedAt 必须是 RFC3339 时间" };

    const visibility = String(meta.visibility || DEFAULT_VISIBILITY).trim().toUpperCase();
    if (!Object.prototype.hasOwnProperty.call(VISIBILITY, visibility)) return { error: "visibility 只能是 PRIVATE、PROTECTED 或 PUBLIC" };

    const pinned = parseYAMLBool(meta.pinned, Boolean(memo.pinned));
    if (pinned === null) return { error: "pinned 必须是 true 或 false" };
    const archived = parseYAMLBool(meta.archived, Boolean(memo.archived));
    if (archived === null) return { error: "archived 必须是 true 或 false" };

    return {
      patch: {
        archived,
        createdAt,
        kind: String(meta.kind || "").trim(),
        pinned,
        projectId: normalizeProjectID(meta.projectId || ""),
        taskId: String(meta.taskId || "").trim(),
        updatedAt,
        visibility,
      },
    };
  }

  function yamlQuote(value) {
    return JSON.stringify(String(value || ""));
  }

  function yamlScalarValue(value) {
    const raw = String(value || "").trim();
    if (!raw) return "";
    if (raw[0] === '"' || raw[0] === "'") {
      try {
        return JSON.parse(raw);
      } catch (_) {
        return raw.replace(/^['"]|['"]$/g, "");
      }
    }
    return raw.replace(/\s+#.*$/, "").trim();
  }

  function parseYAMLBool(value, fallback) {
    if (value === undefined) return fallback;
    switch (String(value).trim().toLowerCase()) {
      case "true":
      case "yes":
      case "1":
        return true;
      case "false":
      case "no":
      case "0":
        return false;
      default:
        return null;
    }
  }

  function isValidMemoTime(value) {
    const date = new Date(value);
    return !Number.isNaN(date.getTime());
  }

  function writeEditDraft(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return Promise.resolve({ ok: false, message: "找不到 memo" });
    const content = editEditor ? editEditor.getText() : state.editDraft;
    if (!content.trim()) {
      return discardEditDraft(memoId, { exit: false, message: "空草稿已清理" });
    }
    return upsertMemoDraftInVault({
      baseUpdatedAt: memo.updatedAt || "",
      content,
      id: memoEditDraftId(memoId),
      kind: "memo-edit",
      memoId,
      projectId: state.editProjectId,
      visibility: state.editVisibility,
    }).then(
      function (draft) {
        upsertDraftInState(draft);
        state.editDraft = content;
        showToast("草稿已保存");
        return { ok: true, message: "draft written" };
      },
      function (err) {
        showToast("保存草稿失败: " + errorMessage(err));
        return { ok: false, message: "保存草稿失败: " + errorMessage(err) };
      },
    );
  }

  function exitEdit(memoId) {
    const memo = findMemo(memoId);
    if (!memo) {
      state.editingId = "";
      state.editDraft = "";
      state.editPreviewVisible = false;
      renderFeed();
      return Promise.resolve({ ok: true, message: "quit" });
    }

    const content = editEditor ? editEditor.getText() : state.editDraft;
    const changed =
      content !== memo.content ||
      normalizeProjectID(state.editProjectId) !== normalizeProjectID(memo.projectId) ||
      (state.editVisibility || DEFAULT_VISIBILITY) !== (memo.visibility || DEFAULT_VISIBILITY);

    const finish = function () {
      state.editingId = "";
      state.editDraft = "";
      state.editPreviewVisible = false;
      renderFeed();
      return { ok: true, message: "quit" };
    };

    if (!changed) return Promise.resolve(finish());
    return writeEditDraft(memoId).then(function (result) {
      if (result && result.ok === false) return result;
      return finish();
    });
  }

  function discardEditDraft(memoId, options = {}) {
    const draftId = memoEditDraftId(memoId);
    removeDraftFromState(draftId);
    if (options.exit) {
      state.editingId = "";
      state.editDraft = "";
      state.editPreviewVisible = false;
      renderFeed();
    }
    return deleteMemoDraftInVault(draftId).then(
      function () {
        if (options.message) showToast(options.message);
        return { ok: true, message: options.message || "draft discarded" };
      },
      function (err) {
        showToast("删除草稿失败: " + errorMessage(err));
        return { ok: false, message: "删除草稿失败: " + errorMessage(err) };
      },
    );
  }

  function togglePin(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    updateMemo(memoId, { pinned: !memo.pinned });
  }

  function toggleTask(memoId, lineIndex, checked) {
    const memo = findMemo(memoId);
    if (!memo) return;
    const lines = memo.content.split("\n");
    if (!lines[lineIndex]) return;
    lines[lineIndex] = updateTaskLine(lines[lineIndex], checked);
    updateMemo(memoId, {
      content: lines.join("\n"),
      updatedAt: new Date().toISOString(),
    });
  }

  function deleteMemo(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    confirmDeleteMemo(memo).then(function (options) {
      if (!options) return;
      deleteMemoWithOptions(memo, options);
    });
  }

  function deleteMemoWithOptions(memo, options) {
    const memoId = memo.id;
    const preserveTodos = options.todoCount > 0 && !options.deleteTodos;
    const preservePromise = preserveTodos ? createMemoFromTodoItems(memo) : Promise.resolve(null);

    preservePromise.then(function (preservedMemo) {
      deleteMemoInVault(memoId, { cleanupAssets: options.deleteFiles, deleteTasks: options.deleteTodos }).then(
        function (result) {
          state.memos = state.memos.filter((item) => item.id !== memoId);
          state.comments = state.comments.filter((comment) => comment.memoId !== memoId);
          if (state.commentingMemoId === memoId) {
            state.commentingMemoId = "";
            state.commentDraft = "";
            state.commentPreviewVisible = false;
          }
          if (preservedMemo) {
            state.memos = [preservedMemo].concat(state.memos);
          }
          saveMemos(state.memos);
          renderAll();
          refreshTasksFromVault();
          if (result && Array.isArray(result.assetErrors) && result.assetErrors.length) {
            showToast("已删除 memo，部分文件删除失败");
          } else if (preservedMemo) {
            showToast("已删除 memo，todo 已保留");
          } else if (result && result.tasksDeleted) {
            showToast(`已删除 memo 和 ${result.tasksDeleted} 个任务`);
          } else {
            showToast("已删除 memo");
          }
        },
        function (err) {
          showToast("删除失败: " + errorMessage(err));
          if (preservedMemo) refreshMemosFromVault();
        },
      );
    }, function (err) {
      showToast("保留 todo 失败: " + errorMessage(err));
    });
  }

  function createMemoFromTodoItems(memo) {
    const lines = String(memo.content || "").replace(/\r\n/g, "\n").split("\n");
    let activeFence = null;
    const todoLines = lines.filter(function (line) {
      const fence = parseMemoFenceLine(line);
      if (activeFence) {
        if (fence && isMemoFenceClosingLine(line, activeFence)) activeFence = null;
        return false;
      }
      if (fence) {
        activeFence = fence;
        return false;
      }
      return parseTaskLine(line);
    });
    const content = todoLines.join("\n").trim();
    if (!content) return Promise.resolve(null);
    return createMemoInVault(content, memo.visibility || DEFAULT_VISIBILITY, memo.projectId || "").then(function (created) {
      const normalized = normalizeMemoPayload(created);
      if (!normalized) throw new Error("无法创建 todo memo");
      return normalized;
    });
  }

  function confirmDeleteMemo(memo) {
    const fileCount = collectManagedResources([memo].concat(commentsForMemo(memo.id))).length;
    const todoCount = collectTodos([memo]).length;

    return new Promise(function (resolve) {
      const dialog = document.createElement("div");
      dialog.className = "memo-delete-dialog";
      dialog.innerHTML = deleteMemoDialogTemplate(memo, fileCount, todoCount);
      root.appendChild(dialog);

      function close(value) {
        document.removeEventListener("keydown", handleKeydown);
        dialog.remove();
        resolve(value);
      }

      function handleKeydown(event) {
        if (event.key === "Escape") {
          event.preventDefault();
          close(null);
        }
      }

      dialog.addEventListener("click", function (event) {
        if (event.target === dialog) {
          close(null);
          return;
        }
        const action = closestElement(event.target, "[data-delete-dialog-action]");
        if (!action || !dialog.contains(action)) return;
        if (action.dataset.deleteDialogAction === "cancel") {
          close(null);
          return;
        }
        const filesInput = dialog.querySelector("[data-delete-files]");
        const todosInput = dialog.querySelector("[data-delete-todos]");
        close({
          deleteFiles: filesInput ? filesInput.checked : true,
          deleteTodos: todosInput ? todosInput.checked : true,
          fileCount,
          todoCount,
        });
      });

      document.addEventListener("keydown", handleKeydown);
      window.requestAnimationFrame(function () {
        const cancel = dialog.querySelector('[data-delete-dialog-action="cancel"]');
        if (cancel) cancel.focus();
      });
    });
  }

  function collectManagedResources(memos) {
    const seen = new Set();
    return collectResources(memos).filter(function (resource) {
      const asset = parseAssetReference(resource.url);
      if (!asset) return false;
      const key = asset.storageId + "/" + asset.key;
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
  }

  function deleteMemoDialogTemplate(memo, fileCount, todoCount) {
    const title = memoTitle(memo);
    const fileOption = fileCount
      ? `
        <label class="memo-delete-option">
          <input type="checkbox" data-delete-files checked />
          <span>
            <strong>同时删除文件和图片</strong>
            <small>${fileCount} 个已上传资源</small>
          </span>
        </label>
      `
      : "";
    const todoOption = todoCount
      ? `
        <label class="memo-delete-option">
          <input type="checkbox" data-delete-todos checked />
          <span>
            <strong>同时删除 todo 项</strong>
            <small>${todoCount} 个 todo</small>
          </span>
        </label>
      `
      : "";

    return `
      <section class="memo-delete-panel" role="dialog" aria-modal="true" aria-labelledby="memo-delete-title">
        <header class="memo-delete-head">
          <span class="memo-delete-icon">${SVG.trash}</span>
          <div>
            <h2 id="memo-delete-title">删除 memo？</h2>
            <p>${escapeHTML(compactText(title, 72))}</p>
          </div>
        </header>
        ${fileOption || todoOption ? `<div class="memo-delete-options">${fileOption}${todoOption}</div>` : ""}
        <footer class="memo-delete-actions">
          <button class="memo-secondary-button" type="button" data-delete-dialog-action="cancel">取消</button>
          <button class="memo-primary-button is-danger" type="button" data-delete-dialog-action="confirm">删除</button>
        </footer>
      </section>
    `;
  }

  function copyMemo(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    copyText(memo.content).then(
      function () {
        showToast("已复制");
      },
      function () {
        showToast("复制失败");
      },
    );
  }

  function copyMemoRef(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    copyText(`[[memo:${memo.id}|${memoReferenceAlias(memoTitle(memo))}]]`).then(
      () => showToast("已复制 memo 引用"),
      () => showToast("复制失败"),
    );
  }

  function copyCodeBlock(action) {
    copyCodeBlockFromAction(action, scopedMemos(), showToast);
  }

  function copyLink(action) {
    const linkCard = closestElement(action, "[data-link-url]");
    const url = linkCard && linkCard.dataset ? linkCard.dataset.linkUrl : "";
    if (!url) return;
    copyText(url).then(
      () => showToast("已复制链接"),
      () => showToast("复制失败"),
    );
  }

  function insertFiles(files) {
    if (!files || files.length === 0) return;
    if (composerEditor.insertFiles) {
      composerEditor.insertFiles(files);
    } else {
      filesToMarkdown(files).then(function (markdown) {
        if (markdown) composerEditor.insertBlock(markdown);
      }).catch(function (err) {
        showToast(uploadErrorMessage(err));
      });
    }
    composerEditor.focus();
  }

  function requestFilesForComposer(accept) {
    if (composerEditor && composerEditor.requestFiles) {
      composerEditor.requestFiles(accept || "");
      return;
    }
    if (accept) els.attachInput.setAttribute("accept", accept);
    else els.attachInput.removeAttribute("accept");
    els.attachInput.click();
  }

  function renderAll() {
    state.memoRefIndex = buildMemoReferenceIndex(state.memos);
    renderMainChrome();
    renderProjects();
    renderComposerProjectSelect();
    renderViewButtons();
    renderFilterButtons();
    renderCalendar();
    renderStats();
    renderTags();
    renderPinned();
    renderMainContent();
  }

  function renderMainChrome() {
    const viewMeta = activeViewMeta(state.activeView);
    els.mainTitle.textContent = viewMeta.title;
    els.mainSubtitle.textContent = viewMeta.subtitle;
    els.composer.classList.toggle("hidden", viewMeta.hideComposer);
    els.searchInput.placeholder = viewMeta.searchPlaceholder;
    els.memoList.classList.toggle("is-todo-list", state.activeView === "todos" || state.activeView === "items" || state.activeView === "milestones");
    els.memoList.classList.toggle("is-resource-list", state.activeView === "links" || state.activeView === "files" || state.activeView === "codeblocks");
    els.memoList.classList.toggle("is-clipboard-list", state.activeView === "clipboard");
  }

  function renderMainContent() {
    if (state.activeView !== "memos" && commentEditor) {
      commentEditor.destroy();
      commentEditor = null;
      commentEditorMemoId = "";
    }
    if (state.activeView !== "memos" && commentEditEditor) {
      commentEditEditor.destroy();
      commentEditEditor = null;
      commentEditEditorCommentId = "";
    }
    switch (state.activeView) {
      case "todos":
        renderTodos();
        return;
      case "items":
        renderGTDItems();
        return;
      case "milestones":
        renderGTDMilestones();
        return;
      case "links":
        renderLinks();
        return;
      case "codeblocks":
        renderCodeBlocks();
        return;
      case "files":
        renderFiles();
        return;
      case "clipboard":
        renderClipboardView();
        return;
      default:
        renderFeed();
    }
  }

  function syncEditDraftFromEditor() {
    if (!editEditor || !state.editingId || editEditorMemoId !== state.editingId) return;
    state.editDraft = editEditor.getText();
  }

  function syncCommentDraftFromEditor() {
    if (!commentEditor || !state.commentingMemoId || commentEditorMemoId !== state.commentingMemoId) return;
    state.commentDraft = commentEditor.getText();
  }

  function syncCommentEditDraftFromEditor() {
    if (!commentEditEditor || !state.commentEditingId || commentEditEditorCommentId !== state.commentEditingId) return;
    state.commentEditDraft = commentEditEditor.getText();
  }

  function renderProjects() {
    if (!els.projectList) return;
    const activeProjects = state.projects.filter((project) => !project.archived);
    const unassignedCount = state.memos.filter((memo) => !memo.projectId && !memo.archived).length;
    els.projectSummary.textContent = activeProjects.length ? `${activeProjects.length} 个项目` : "暂无项目";
    els.projectList.innerHTML = [
      projectFilterTemplate("all", "全部", state.memos.filter((memo) => !memo.archived).length, "", state.activeProjectFilter),
      projectFilterTemplate("unassigned", "未归属", unassignedCount, "", state.activeProjectFilter),
      ...activeProjects.map((project) =>
        projectFilterTemplate(project.id, project.name, projectMemoCount(project.id), project.color, state.activeProjectFilter),
      ),
    ].join("");
  }

  function renderComposerProjectSelect() {
    if (!els.projectSelect) return;
    els.projectSelect.innerHTML = projectOptionsTemplate(state.projects, state.composerProjectId);
    els.projectSelect.value = state.composerProjectId || "";
  }

  function renderViewButtons() {
    root.querySelectorAll("[data-view]").forEach((button) => {
      button.classList.toggle("is-active", button.dataset.view === state.activeView);
    });
    const memos = scopedMemos();
    const todoStats = getTaskStats(scopedTasks());
    const linkCount = collectLinks(memos).length;
    const codeBlocks = collectCodeBlocks(memos);
    const codeSnippetCount = codeBlocks.filter((block) => block.marked).length;
    const codeBlockCount = codeBlocks.length;
    const resourceCount = collectResources(memos).length;
    const openItemCount = scopedGTDItems().filter((item) => item.status !== "closed" && item.status !== "resolved").length;
    const activeMilestoneCount = scopedGTDMilestones().filter((milestone) => milestone.status === "active" || milestone.status === "planned").length;
    els.todoNavCount.textContent = todoStats.open ? String(todoStats.open) : "";
    if (els.itemNavCount) els.itemNavCount.textContent = openItemCount ? String(openItemCount) : "";
    if (els.milestoneNavCount) els.milestoneNavCount.textContent = activeMilestoneCount ? String(activeMilestoneCount) : "";
    els.linkNavCount.textContent = linkCount ? String(linkCount) : "";
    if (els.codeNavCount) els.codeNavCount.textContent = codeBlockCount ? (codeSnippetCount ? `${codeSnippetCount}/${codeBlockCount}` : String(codeBlockCount)) : "";
    els.fileNavCount.textContent = resourceCount ? String(resourceCount) : "";
    if (els.clipboardNavCount) els.clipboardNavCount.textContent = state.clipboardItem && state.clipboardItem.id ? "1" : "";
  }

  function renderFilterButtons() {
    root.querySelectorAll("[data-filter]").forEach((button) => {
      button.classList.toggle(
        "is-active",
        state.activeView === "memos" && button.dataset.filter === state.activeFilter && !state.activeTag,
      );
    });
  }

  function renderCalendar() {
    els.calendar.innerHTML = calendarTemplate(state.calendarMonth, scopedMemos(), state.selectedCalendarDate, calendarWeekStart());
  }

  function renderStats() {
    const memos = scopedMemos();
    const active = memos.filter((memo) => !memo.archived);
    const archived = memos.length - active.length;
    const publicCount = active.filter((memo) => memo.visibility === "PUBLIC").length;
    const tags = collectTags(active);
    const taskStats = getTaskStats(scopedTasks());
    const openItemCount = scopedGTDItems().filter((item) => item.status !== "closed" && item.status !== "resolved").length;
    const milestoneCount = scopedGTDMilestones().filter((milestone) => milestone.status === "active" || milestone.status === "planned").length;
    const linkCount = collectLinks(memos).length;
    const codeSnippetCount = collectCodeBlocks(memos).filter((block) => block.marked).length;
    const resourceStats = getResourceStats(memos);

    els.stats.innerHTML = [
      statTemplate("全部", active.length),
      statTemplate("公开", publicCount),
      statTemplate("标签", tags.length),
      statTemplate("任务", taskStats.total),
      statTemplate("未完成", taskStats.open),
      statTemplate("事项", openItemCount),
      statTemplate("里程碑", milestoneCount),
      statTemplate("链接", linkCount),
      statTemplate("代码片段", codeSnippetCount),
      statTemplate("文件", resourceStats.files),
      statTemplate("图片", resourceStats.images),
      statTemplate("归档", archived),
    ].join("");
  }

  function renderTags() {
    const tags = collectTags(scopedMemos().filter((memo) => !memo.archived));
    els.tagSummary.textContent = tags.length ? `${tags.length} 个标签` : "暂无标签";
    els.tagList.innerHTML = tags.length
      ? tags
          .map(
            ([tag, count]) => `
              <button class="memo-tag-filter ${state.activeTag === tag ? "is-active" : ""}" type="button" data-tag="${escapeAttr(tag)}">
                <span>#${escapeHTML(tag)}</span>
                <span>${count}</span>
              </button>
            `,
          )
          .join("")
      : '<div class="memo-empty-mini">暂无标签</div>';
  }

  function renderPinned() {
    const pinned = scopedMemos().filter((memo) => memo.pinned && !memo.archived).slice(0, 3);
    els.pinnedList.innerHTML = pinned.length
      ? pinned
          .map(function (memo) {
            const expanded = state.expandedMemoIds.has(memo.id);
            const expandLabel = expanded ? "收起" : "展开";
            return `
              <article class="memo-pinned-item" data-memo-id="${escapeAttr(memo.id)}">
                <header class="memo-pinned-head">
                  <div class="memo-pinned-actions">
                    <button class="memo-action-button" type="button" data-action="togglePin" title="取消置顶" aria-label="取消置顶">${SVG.pin}</button>
                    <button class="memo-action-button" type="button" data-action="detachMemo" title="分离为窗口" aria-label="分离为窗口">${SVG.external}</button>
                  </div>
                </header>
                <div class="memo-pinned-collapse memo-list-collapse ${expanded ? "is-expanded" : "is-collapsed"}" data-memo-collapse>
                  <div class="memo-pinned-content memo-content">${renderMemoMarkdown(memo.content, memoRenderContext(memo.id, { readonly: true, showLineNumbers: false }))}</div>
                  <button class="memo-expand-button memo-pinned-expand-button" type="button" data-action="toggleMemoExpand" aria-expanded="${expanded ? "true" : "false"}" title="${expandLabel}">
                    <span>${expandLabel}</span>
                    ${SVG.chevronDown}
                  </button>
                </div>
              </article>
            `;
          })
          .join("")
      : '<div class="memo-empty-mini">暂无置顶</div>';
    syncMemoExpandControls();
  }

  function renderFeed() {
    if (commentEditor) {
      if (state.commentingMemoId && state.commentingMemoId === commentEditorMemoId) {
        syncCommentDraftFromEditor();
      }
      commentEditor.destroy();
      commentEditor = null;
      commentEditorMemoId = "";
    }
    if (commentEditEditor) {
      if (state.commentEditingId && state.commentEditingId === commentEditEditorCommentId) {
        syncCommentEditDraftFromEditor();
      }
      commentEditEditor.destroy();
      commentEditEditor = null;
      commentEditEditorCommentId = "";
    }
    if (editEditor) {
      syncEditDraftFromEditor();
      editEditor.destroy();
      editEditor = null;
      editEditorMemoId = "";
    }

    const memos = visibleMemos();
    els.feedCount.textContent = `${memos.length} 条`;
    els.memoList.innerHTML = memos.length
      ? memos
          .map((memo) => safeMemoTemplate(memo))
          .join("")
      : emptyFeedTemplate();

    if (state.editingId) {
      const memo = findMemo(state.editingId);
      const host = els.memoList.querySelector("[data-edit-host]");
      const statusHost = els.memoList.querySelector("[data-edit-vim-status]");
      if (memo && host) {
        editEditor = createMiniEditor(host, {
          memoItems() {
            return state.memos;
          },
          tagItems: editorTagItems,
          onChange(value) {
            state.editDraft = value;
            renderEditPreview(memo.id);
          },
          onCommit() {
            return saveEdit(memo.id, { source: "vim-wq" });
          },
          onDiscard() {
            return discardEditDraft(memo.id, { exit: true, message: "草稿已丢弃" });
          },
          onQuit() {
            return exitEdit(memo.id);
          },
          onSave() {
            return writeEditDraft(memo.id);
          },
          onSubmit() {
            saveEdit(memo.id);
          },
          onWriteDraft() {
            return writeEditDraft(memo.id);
          },
          placeholder: "编辑 memo...",
          sourceMemoId: memo.id,
          value: state.editDraft,
          vim: editorVimEnabled(),
          vimStatusHost: statusHost,
        });
        editEditorMemoId = memo.id;
        editEditor.focus();
      }
    }

    if (state.commentingMemoId) {
      const memo = findMemo(state.commentingMemoId);
      const host = els.memoList.querySelector("[data-comment-host]");
      const statusHost = els.memoList.querySelector("[data-comment-vim-status]");
      if (memo && host) {
        commentEditor = createMiniEditor(host, {
          memoItems() {
            return state.memos;
          },
          tagItems: editorTagItems,
          onChange(value) {
            state.commentDraft = value;
            renderCommentPreview(memo.id);
          },
          onCommit() {
            return saveComment(memo.id, { source: "vim-wq" });
          },
          onDiscard() {
            return cancelComment({ message: "评论已取消" });
          },
          onQuit() {
            return exitComment(memo.id);
          },
          onSubmit() {
            return saveComment(memo.id);
          },
          placeholder: "添加评论...",
          sourceMemoId: memo.id,
          value: state.commentDraft,
          vim: editorVimEnabled(),
          vimStatusHost: statusHost,
        });
        commentEditorMemoId = memo.id;
        commentEditor.focus();
      }
    }

    if (state.commentEditingId) {
      const comment = findComment(state.commentEditingId);
      const host = els.memoList.querySelector("[data-comment-edit-host]");
      const editHost = host ? closestElement(host, ".memo-comment-edit") : null;
      const statusHost = editHost ? editHost.querySelector("[data-comment-edit-vim-status]") : null;
      if (comment && host) {
        commentEditEditor = createMiniEditor(host, {
          memoItems() {
            return state.memos;
          },
          tagItems: editorTagItems,
          onChange(value) {
            state.commentEditDraft = value;
            renderCommentEditPreview(comment.id);
          },
          onCommit() {
            return saveCommentEdit();
          },
          onDiscard() {
            return cancelCommentEdit({ message: "评论编辑已取消" });
          },
          onQuit() {
            return cancelCommentEdit();
          },
          onSubmit() {
            return saveCommentEdit();
          },
          placeholder: "编辑评论...",
          sourceMemoId: comment.memoId,
          value: state.commentEditDraft,
          vim: editorVimEnabled(),
          vimStatusHost: statusHost,
        });
        commentEditEditorCommentId = comment.id;
        commentEditEditor.focus();
      }
    }

    syncMemoExpandControls();
    renderEditablePreviews();
  }

  function safeMemoTemplate(memo) {
    try {
      return memoTemplate(
        memo,
        state.editingId,
        memoRenderContext(memo.id),
        state.expandedMemoIds.has(memo.id),
        state.projects,
        {
          comments: commentsForMemo(memo.id),
          commenting: state.commentingMemoId === memo.id,
          editingCommentId: state.commentEditingId,
          sourceDraft: state.sourceEditingId === memo.id ? state.sourceDraft : "",
          sourceEditing: state.sourceEditingId === memo.id,
        },
      );
    } catch (err) {
      return `
        <article class="memo-card is-archived" data-memo-id="${escapeAttr(memo.id)}">
          <div class="memo-empty-mini">memo 渲染失败: ${escapeHTML(errorMessage(err))}</div>
        </article>
      `;
    }
  }

  function syncMemoExpandControls() {
    const collapsibleItems = root.querySelectorAll("[data-memo-collapse]");
    collapsibleItems.forEach(function (item) {
      const content = item.querySelector(".memo-content");
      if (!content) return;

      item.classList.remove("is-short");
      if (item.classList.contains("is-collapsed") && content.scrollHeight <= content.clientHeight + 1) {
        item.classList.add("is-short");
      }

      content.querySelectorAll("img").forEach(function (image) {
        if (image.complete) return;
        if (image.dataset.memoExpandWatch) return;
        image.dataset.memoExpandWatch = "true";
        image.addEventListener("load", syncMemoExpandControls, { once: true });
      });
    });
  }

  function renderTodos() {
    if (editEditor) {
      syncEditDraftFromEditor();
      editEditor.destroy();
      editEditor = null;
      editEditorMemoId = "";
    }

    const tasks = visibleTasks();
    const openTasks = tasks.filter((task) => task.status !== "completed");
    els.feedCount.textContent = state.tasksLoading
      ? "正在读取任务"
      : tasks.length
        ? `${openTasks.length} 未完成 / ${tasks.length} 项`
        : "0 项";
    const groups = groupedVisibleTasks(tasks);
    const workspace = taskWorkspaceTemplate({
      counts: taskFilterCounts(scopedTasks()),
      filter: state.taskFilter,
    });
    const taskContent = tasks.length
      ? groups.map((group) => taskGroupTemplate(group.label, group.tasks, { memos: state.memos, projects: state.projects })).join("")
      : emptyTasksTemplate();
    els.memoList.innerHTML = workspace + taskContent;
  }

  function renderGTDItems() {
    if (editEditor) {
      syncEditDraftFromEditor();
      editEditor.destroy();
      editEditor = null;
      editEditorMemoId = "";
    }

    const items = visibleGTDItems();
    const openItems = items.filter((item) => item.status !== "closed" && item.status !== "resolved");
    els.feedCount.textContent = state.gtdLoading
      ? "正在读取事项"
      : items.length
        ? `${openItems.length} open / ${items.length} 项`
        : "0 项";
    const groups = groupedVisibleGTDItems(items);
    const workspace = gtdItemWorkspaceTemplate({ milestones: scopedGTDMilestones() });
    const content = items.length
      ? groups.map((group) => gtdItemGroupTemplate(group.label, group.items, {
          milestones: state.gtdMilestones,
          projects: state.projects,
        })).join("")
      : emptyTasksTemplate();
    els.memoList.innerHTML = workspace + content;
  }

  function renderGTDMilestones() {
    if (editEditor) {
      syncEditDraftFromEditor();
      editEditor.destroy();
      editEditor = null;
      editEditorMemoId = "";
    }

    const milestones = visibleGTDMilestones();
    const activeCount = milestones.filter((milestone) => milestone.status === "active" || milestone.status === "planned").length;
    els.feedCount.textContent = state.gtdLoading
      ? "正在读取里程碑"
      : milestones.length
        ? `${activeCount} active / ${milestones.length} 个`
        : "0 个";
    const groups = groupedVisibleGTDMilestones(milestones);
    const workspace = gtdMilestoneWorkspaceTemplate();
    const content = milestones.length
      ? groups.map((group) => gtdMilestoneGroupTemplate(group.label, group.milestones, {
          items: state.gtdItems,
          tasks: state.tasks,
        })).join("")
      : emptyTasksTemplate();
    els.memoList.innerHTML = workspace + content;
  }

  function renderLinks() {
    if (editEditor) {
      syncEditDraftFromEditor();
      editEditor.destroy();
      editEditor = null;
      editEditorMemoId = "";
    }

    const links = visibleLinks();
    els.feedCount.textContent = links.length ? `${links.length} 个链接` : "0 个链接";
    els.memoList.innerHTML = links.length ? links.map(linkTemplate).join("") : emptyLinksTemplate();
  }

  function renderCodeBlocks() {
    if (editEditor) {
      syncEditDraftFromEditor();
      editEditor.destroy();
      editEditor = null;
      editEditorMemoId = "";
    }

    const blocks = visibleCodeBlocks();
    const markedCount = blocks.filter((block) => block.marked).length;
    els.feedCount.textContent = blocks.length
      ? `${markedCount} 个片段 / ${blocks.length - markedCount} 个未标记`
      : "0 个代码片段";
    els.memoList.innerHTML = blocks.length ? blocks.map(codeBlockTemplate).join("") : emptyCodeBlocksTemplate();
  }

  function renderFiles() {
    if (editEditor) {
      syncEditDraftFromEditor();
      editEditor.destroy();
      editEditor = null;
      editEditorMemoId = "";
    }

    const resources = visibleResources();
    const images = resources.filter((resource) => resource.type === "image");
    const files = resources.filter((resource) => resource.type === "file");
    els.feedCount.textContent = resources.length
      ? `${files.length} 个文件 / ${images.length} 张图片`
      : "0 个文件";
    els.memoList.innerHTML = resources.length
      ? [
          resourceGroupTemplate("文件", files),
          resourceGroupTemplate("图片", images),
        ].join("")
      : emptyFilesTemplate();
  }

  function renderClipboardView() {
    if (editEditor) {
      syncEditDraftFromEditor();
      editEditor.destroy();
      editEditor = null;
      editEditorMemoId = "";
    }

    const item = state.clipboardItem;
    els.feedCount.textContent = item && item.id ? clipboardTypeLabel(item.type) : "0 项";
    els.memoList.innerHTML = clipboardCurrentTemplate(item, {
      actionLabel: clipboardActionLabel(item && item.type),
      typeLabel: clipboardTypeLabel(item && item.type),
      working: state.clipboardWorking,
    });
  }

  function renderComposerStatus(value) {
    const text = String(value || "");
    const tagCount = extractTags(text).length;
    const chars = text.trim().length;
    els.composerStatus.textContent = `${chars} 字符 / ${tagCount} 标签`;
    els.createButton.disabled = chars === 0 || state.saving;
  }

  function refreshProjectsFromVault() {
    loadProjectsFromVault().then(
      function (payload) {
        state.projects = payload.projects.map(normalizeProjectPayload).filter(Boolean);
        saveProjects(state.projects);
        if (
          payload.activeProjectId &&
          state.activeProjectFilter === "all" &&
          !(composerEditor && composerEditor.getText().trim())
        ) {
          state.composerProjectId = payload.activeProjectId;
          rememberComposerProject(payload.activeProjectId);
        }
        renderAll();
      },
      function (err) {
        showToast("读取 project 失败: " + errorMessage(err));
      },
    );
  }

  function createProjectFromPrompt() {
    const name = window.prompt("Project 名称");
    if (name === null) return;
    const trimmed = name.trim();
    if (!trimmed) {
      showToast("Project 名称不能为空");
      return;
    }
    createProjectInVault(trimmed).then(
      function (project) {
        const normalized = normalizeProjectPayload(project);
        if (!normalized) return;
        state.projects = state.projects.concat(normalized);
        saveProjects(state.projects);
        selectProjectFilter(normalized.id);
        showToast("已创建 Project");
      },
      function (err) {
        showToast("创建 Project 失败: " + errorMessage(err));
      },
    );
  }

  function refreshMemosFromVault() {
    loadMemosFromVault().then(
      function (memos) {
        state.memos = memos.map(normalizeMemoPayload).filter(Boolean);
        saveMemos(state.memos);
        renderAll();
      },
      function (err) {
        showToast("读取 vault memo 失败: " + errorMessage(err));
      },
    );
  }

  function refreshMemoCommentsFromVault() {
    loadMemoCommentsFromVault().then(
      function (comments) {
        state.comments = comments.map(normalizeMemoCommentPayload).filter(Boolean);
        state.commentsLoaded = true;
        renderAll();
      },
      function (err) {
        if (typeof globalThis.invoke === "function") {
          showToast("读取评论失败: " + errorMessage(err));
        }
      },
    );
  }

  function refreshMemoDraftsFromVault() {
    loadMemoDraftsFromVault().then(
      function (drafts) {
        state.memoDrafts = drafts.map(normalizeMemoDraftPayload).filter(Boolean);
        state.draftsLoaded = true;
        applyComposerDraft();
        renderComposerProjectSelect();
        renderComposerStatus(composerEditor ? composerEditor.getText() : "");
      },
      function (err) {
        if (typeof globalThis.invoke === "function") {
          showToast("读取草稿失败: " + errorMessage(err));
        }
      },
    );
  }

  function commentsForMemo(memoId) {
    const id = String(memoId || "").trim();
    return state.comments
      .filter((comment) => comment && comment.memoId === id)
      .sort(function (a, b) {
        const left = new Date(a.createdAt || 0).getTime() || 0;
        const right = new Date(b.createdAt || 0).getTime() || 0;
        if (left === right) return a.id.localeCompare(b.id);
        return left - right;
      });
  }

  function findComment(commentId) {
    const id = String(commentId || "").trim();
    return state.comments.find((comment) => comment && comment.id === id) || null;
  }

  function upsertCommentInState(comment) {
    const normalized = normalizeMemoCommentPayload(comment);
    if (!normalized) return;
    const index = state.comments.findIndex((item) => item.id === normalized.id);
    if (index >= 0) {
      state.comments[index] = normalized;
    } else {
      state.comments.push(normalized);
    }
  }

  function findDraft(draftId) {
    return state.memoDrafts.find((draft) => draft && draft.id === draftId) || null;
  }

  function upsertDraftInState(draft) {
    const normalized = normalizeMemoDraftPayload(draft);
    if (!normalized) return;
    const index = state.memoDrafts.findIndex((item) => item.id === normalized.id);
    if (index >= 0) {
      state.memoDrafts[index] = normalized;
    } else {
      state.memoDrafts.push(normalized);
    }
  }

  function removeDraftFromState(draftId) {
    state.memoDrafts = state.memoDrafts.filter((draft) => draft && draft.id !== draftId);
  }

  function applyComposerDraft() {
    const draft = findDraft(COMPOSER_DRAFT_ID);
    if (!draft || !composerEditor) return;
    if (composerEditor.getText().trim()) return;
    state.composerProjectId = normalizeProjectID(draft.projectId);
    state.visibility = draft.visibility || DEFAULT_VISIBILITY;
    if (els.visibilitySelect) els.visibilitySelect.value = state.visibility;
    composerEditor.setText(draft.content || "");
    renderComposerStatus(draft.content || "");
  }

  function refreshTasksFromVault() {
    state.tasksLoading = true;
    loadTasks().then(
      function (payload) {
        state.tasks = payload.tasks.map(normalizeTaskSummary).filter(Boolean);
        renderAll();
      },
      function (err) {
        if (typeof globalThis.invoke === "function") {
          showToast("读取 task 失败: " + errorMessage(err));
        }
      },
    ).finally(function () {
      state.tasksLoading = false;
      renderAll();
    });
  }

  function refreshGTDFromVault() {
    state.gtdLoading = true;
    Promise.all([loadGTDItems(), loadGTDMilestones()]).then(
      function (results) {
        state.gtdItems = results[0].map(normalizeGTDItem).filter(Boolean);
        state.gtdMilestones = results[1].map(normalizeGTDMilestone).filter(Boolean);
        renderAll();
      },
      function (err) {
        if (typeof globalThis.invoke === "function") {
          showToast("读取 GTD 事项失败: " + errorMessage(err));
        }
      },
    ).finally(function () {
      state.gtdLoading = false;
      renderAll();
    });
  }

  function visibleMemos() {
    const query = state.query.toLowerCase();
    return scopedMemos()
      .filter((memo) => {
        if (state.activeFilter === "archive") return memo.archived;
        if (memo.archived) return false;
        if (state.activeFilter === "pinned" && !memo.pinned) return false;
        if (state.activeFilter === "public" && memo.visibility !== "PUBLIC") return false;
        if (state.activeFilter === "private" && memo.visibility !== "PRIVATE") return false;
        if (state.activeTag && !extractTags(memo.content).includes(state.activeTag)) return false;
        if (state.selectedCalendarDate && memoDateKey(memo) !== state.selectedCalendarDate) return false;
        if (!query) return true;
        const commentText = commentsForMemo(memo.id).map((comment) => comment.content).join("\n");
        return `${memo.content} ${commentText} ${memo.visibility}`.toLowerCase().includes(query);
      })
      .sort((a, b) => {
        const result = new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime();
        return state.sortDesc ? -result : result;
      });
  }

  function visibleTodos() {
    const query = state.query.toLowerCase();
    return collectTodos(scopedMemos())
      .filter((todo) => {
        if (state.activeTag && !extractTags(todo.memo.content).includes(state.activeTag)) return false;
        if (!query) return true;
        return `${todo.text} ${todo.sourceText} ${todo.memo.content} ${todo.memo.visibility}`.toLowerCase().includes(query);
      })
      .sort((a, b) => {
        if (a.checked !== b.checked) return a.checked ? 1 : -1;
        const created = new Date(a.memo.createdAt).getTime() - new Date(b.memo.createdAt).getTime();
        if (created !== 0) return state.sortDesc ? -created : created;
        return a.lineIndex - b.lineIndex;
      });
  }

  function visibleTasks() {
    const query = state.query.toLowerCase();
    return scopedTasks()
      .filter((task) => taskMatchesFilter(task, state.taskFilter))
      .filter((task) => {
        if (!query) return true;
        return [
          task.title,
          task.listId,
          task.priority,
          task.status,
          task.projectId,
          task.source && task.source.memoId,
          task.source && task.source.text,
          (task.tags || []).join(" "),
          (task.contexts || []).join(" "),
        ].join(" ").toLowerCase().includes(query);
      })
      .sort(sortTasksForView);
  }

  function groupedVisibleTasks(tasks) {
    if (state.taskFilter === "completed") {
      return [{ label: "已完成", tasks }];
    }
    if (state.taskFilter === "overdue") {
      return [{ label: "已过期", tasks }];
    }
    if (state.taskFilter === "scheduled") {
      return [
        { label: "已过期", tasks: tasks.filter(isTaskOverdue) },
        { label: "今天", tasks: tasks.filter((task) => !isTaskOverdue(task) && isTaskToday(task)) },
        { label: "未来", tasks: tasks.filter((task) => !isTaskOverdue(task) && !isTaskToday(task)) },
      ].filter((group) => group.tasks.length);
    }
    return [
      { label: "未完成", tasks: tasks.filter((task) => task.status !== "completed") },
      { label: "已完成", tasks: tasks.filter((task) => task.status === "completed") },
    ].filter((group) => group.tasks.length);
  }

  function scopedTasks() {
    if (state.activeProjectFilter === "unassigned") {
      return state.tasks.filter((task) => !task.projectId);
    }
    if (state.activeProjectFilter && state.activeProjectFilter !== "all") {
      return state.tasks.filter((task) => task.projectId === state.activeProjectFilter);
    }
    return state.tasks;
  }

  function scopedGTDItems() {
    if (state.activeProjectFilter === "unassigned") {
      return state.gtdItems.filter((item) => !item.projectId);
    }
    if (state.activeProjectFilter && state.activeProjectFilter !== "all") {
      return state.gtdItems.filter((item) => item.projectId === state.activeProjectFilter);
    }
    return state.gtdItems;
  }

  function scopedGTDMilestones() {
    if (state.activeProjectFilter === "unassigned") {
      return state.gtdMilestones.filter((milestone) => !milestone.projectIds.length);
    }
    if (state.activeProjectFilter && state.activeProjectFilter !== "all") {
      return state.gtdMilestones.filter((milestone) => milestone.projectIds.includes(state.activeProjectFilter));
    }
    return state.gtdMilestones;
  }

  function visibleGTDItems() {
    const query = state.query.toLowerCase();
    return scopedGTDItems()
      .filter((item) => {
        if (!query) return true;
        const milestone = state.gtdMilestones.find((entry) => entry.id === item.milestoneId);
        return [
          item.title,
          item.type,
          item.status,
          item.decision,
          item.projectId,
          milestone && milestone.title,
          (item.labels || []).join(" "),
        ].join(" ").toLowerCase().includes(query);
      })
      .sort(sortGTDItemsForView);
  }

  function visibleGTDMilestones() {
    const query = state.query.toLowerCase();
    return scopedGTDMilestones()
      .filter((milestone) => {
        if (!query) return true;
        return [
          milestone.title,
          milestone.status,
          milestone.targetAt,
          milestone.reviewMemoId,
        ].join(" ").toLowerCase().includes(query);
      })
      .sort(sortGTDMilestonesForView);
  }

  function groupedVisibleGTDItems(items) {
    return [
      { label: "Open", items: items.filter((item) => item.status === "open") },
      { label: "已澄清", items: items.filter((item) => item.status === "triaged") },
      { label: "等待", items: items.filter((item) => item.status === "waiting") },
      { label: "已关闭", items: items.filter((item) => item.status === "closed" || item.status === "resolved") },
    ].filter((group) => group.items.length);
  }

  function groupedVisibleGTDMilestones(milestones) {
    return [
      { label: "进行中", milestones: milestones.filter((milestone) => milestone.status === "active") },
      { label: "计划中", milestones: milestones.filter((milestone) => milestone.status === "planned") },
      { label: "已完成", milestones: milestones.filter((milestone) => milestone.status === "completed") },
      { label: "已取消", milestones: milestones.filter((milestone) => milestone.status === "cancelled") },
    ].filter((group) => group.milestones.length);
  }

  function sortGTDItemsForView(a, b) {
    const status = gtdItemStatusWeight(a.status) - gtdItemStatusWeight(b.status);
    if (status !== 0) return status;
    const created = taskTimeValue(b.createdAt || b.updatedAt) - taskTimeValue(a.createdAt || a.updatedAt);
    if (created !== 0) return created;
    return String(b.id || "").localeCompare(String(a.id || ""));
  }

  function sortGTDMilestonesForView(a, b) {
    const status = gtdMilestoneStatusWeight(a.status) - gtdMilestoneStatusWeight(b.status);
    if (status !== 0) return status;
    const target = taskTimeValue(a.targetAt) - taskTimeValue(b.targetAt);
    if (target !== 0) return target;
    return taskTimeValue(b.updatedAt || b.createdAt) - taskTimeValue(a.updatedAt || a.createdAt);
  }

  function gtdItemStatusWeight(status) {
    if (status === "open") return 0;
    if (status === "triaged") return 1;
    if (status === "waiting") return 2;
    return 3;
  }

  function gtdMilestoneStatusWeight(status) {
    if (status === "active") return 0;
    if (status === "planned") return 1;
    if (status === "completed") return 2;
    return 3;
  }

  function taskMatchesFilter(task, filter) {
    if (isRetainedCompletedTask(task, filter)) return true;
    switch (filter) {
      case "all":
        return true;
      case "completed":
        return task.status === "completed";
      case "inbox":
        return task.status !== "completed" && (task.listId === "inbox" || !task.listId);
      case "overdue":
        return isTaskOverdue(task);
      case "scheduled":
        return task.status !== "completed" && Boolean(task.startAt || task.dueAt);
      case "next":
        return task.status !== "completed" && !task.parentId;
      case "today":
      default:
        return task.status !== "completed" && (isTaskToday(task) || isTaskOverdue(task));
    }
  }

  function retainCompletedTaskInFilter(taskId, filter) {
    const id = String(taskId || "").trim();
    const taskFilter = normalizeTaskFilter(filter);
    if (!id || taskFilter === "all" || taskFilter === "completed") return;
    state.retainedCompletedTaskFilters.set(id, taskFilter);
  }

  function isRetainedCompletedTask(task, filter) {
    if (!task || task.status !== "completed") return false;
    return state.retainedCompletedTaskFilters.get(task.id) === normalizeTaskFilter(filter);
  }

  function clearRetainedCompletedTasks() {
    if (state.retainedCompletedTaskFilters.size) state.retainedCompletedTaskFilters.clear();
  }

  function taskFilterCounts(tasks) {
    return {
      all: tasks.length,
      completed: tasks.filter((task) => task.status === "completed").length,
      inbox: tasks.filter((task) => taskMatchesFilter(task, "inbox")).length,
      next: tasks.filter((task) => taskMatchesFilter(task, "next")).length,
      overdue: tasks.filter((task) => taskMatchesFilter(task, "overdue")).length,
      scheduled: tasks.filter((task) => taskMatchesFilter(task, "scheduled")).length,
      today: tasks.filter((task) => taskMatchesFilter(task, "today")).length,
    };
  }

  function getTaskStats(tasks) {
    const total = tasks.length;
    const done = tasks.filter((task) => task.status === "completed").length;
    return {
      done,
      open: total - done,
      total,
    };
  }

  function sortTasksForView(a, b) {
    if (state.taskFilter === "completed") {
      return taskTimeValue(b.completedAt || b.updatedAt || b.createdAt) - taskTimeValue(a.completedAt || a.updatedAt || a.createdAt);
    }
    if (a.status !== b.status) {
      if (a.status === "completed") return 1;
      if (b.status === "completed") return -1;
    }
    const due = taskTimeValue(a.dueAt || a.startAt) - taskTimeValue(b.dueAt || b.startAt);
    if (due !== 0) return due;
    const priority = taskPriorityWeight(b.priority) - taskPriorityWeight(a.priority);
    if (priority !== 0) return priority;
    return taskTimeValue(b.updatedAt || b.createdAt) - taskTimeValue(a.updatedAt || a.createdAt);
  }

  function isTaskToday(task) {
    const today = dateKey(new Date());
    return [task.startAt, task.dueAt].some((value) => value && dateKey(taskDateValue(value)) === today);
  }

  function isTaskOverdue(task) {
    if (!task.dueAt || task.status === "completed") return false;
    const due = taskDateValue(task.dueAt);
    if (Number.isNaN(due.getTime())) return false;
    return dateKey(due) < dateKey(new Date());
  }

  function dateKey(date) {
    if (!(date instanceof Date) || Number.isNaN(date.getTime())) return "";
    const month = String(date.getMonth() + 1).padStart(2, "0");
    const day = String(date.getDate()).padStart(2, "0");
    return `${date.getFullYear()}-${month}-${day}`;
  }

  function taskTimeValue(value) {
    if (!value) return Number.MAX_SAFE_INTEGER;
    const date = taskDateValue(value);
    return Number.isNaN(date.getTime()) ? Number.MAX_SAFE_INTEGER : date.getTime();
  }

  function taskDateValue(value) {
    const raw = String(value || "").trim();
    const dateOnly = raw.match(/^(\d{4})-(\d{2})-(\d{2})$/);
    if (dateOnly) {
      return new Date(Number(dateOnly[1]), Number(dateOnly[2]) - 1, Number(dateOnly[3]));
    }
    return new Date(raw);
  }

  function taskPriorityWeight(priority) {
    switch (priority) {
      case "high":
        return 3;
      case "medium":
        return 2;
      case "low":
        return 1;
      default:
        return 0;
    }
  }

  function visibleLinks() {
    const query = state.query.toLowerCase();
    return collectLinks(scopedMemos())
      .filter((link) => {
        if (state.activeTag && !extractTags(link.memo.content).includes(state.activeTag)) return false;
        if (!query) return true;
        return `${link.label} ${link.url} ${link.sourceText} ${link.memo.content} ${link.memo.visibility}`.toLowerCase().includes(query);
      })
      .sort((a, b) => sortMemoReference(a, b, state.sortDesc));
  }

  function visibleCodeBlocks() {
    const query = state.query.toLowerCase();
    return collectCodeBlocks(scopedMemos())
      .filter((block) => {
        if (state.activeTag && !extractTags(block.memo.content).includes(state.activeTag)) return false;
        if (!query) return true;
        return matchesSearchQuery(codeBlockSearchText(block), query);
      })
      .sort(sortCodeBlocks);
  }

  function sortCodeBlocks(a, b) {
    if (a.marked !== b.marked) return a.marked ? -1 : 1;
    return sortMemoReference(a, b, state.sortDesc);
  }

  function codeBlockSearchText(block) {
    return [
      block.label,
      block.title,
      block.language,
      Array.isArray(block.aliases) ? block.aliases.join(" ") : "",
      block.code,
      block.sourceText,
      block.memo.content,
      block.memo.visibility,
      projectLabel(block.memo.projectId),
    ].join(" ");
  }

  function matchesSearchQuery(value, query) {
    const haystack = String(value || "").toLowerCase();
    const needle = String(query || "").trim().toLowerCase();
    if (!needle) return true;
    if (haystack.includes(needle)) return true;
    const terms = needle.split(/\s+/).filter(Boolean);
    return terms.length > 0 && terms.every((term) => haystack.includes(term));
  }

  function visibleResources() {
    const query = state.query.toLowerCase();
    return collectResources(scopedMemos())
      .filter((resource) => {
        if (state.activeTag && !extractTags(resource.memo.content).includes(state.activeTag)) return false;
        if (!query) return true;
        return `${resource.label} ${resource.url} ${resource.sourceText} ${resource.memo.content} ${resource.memo.visibility} ${resource.type}`.toLowerCase().includes(query);
      })
      .sort((a, b) => sortMemoReference(a, b, state.sortDesc));
  }

  function scopedMemos() {
    if (state.activeProjectFilter === "unassigned") {
      return state.memos.filter((memo) => !memo.projectId);
    }
    if (state.activeProjectFilter && state.activeProjectFilter !== "all") {
      return state.memos.filter((memo) => memo.projectId === state.activeProjectFilter);
    }
    return state.memos;
  }

  function projectMemoCount(projectId) {
    return state.memos.filter((memo) => memo.projectId === projectId && !memo.archived).length;
  }

  function projectLabel(projectId) {
    projectId = normalizeProjectID(projectId);
    if (!projectId) return "未归属";
    const project = state.projects.find((item) => item.id === projectId);
    return project ? project.name : "未知 Project";
  }

  function findMemo(memoId) {
    return state.memos.find((memo) => memo.id === memoId);
  }

  function findTask(taskId) {
    return state.tasks.find((task) => task.id === taskId);
  }

  function runCalendarAction(action) {
    switch (action) {
      case "nextMonth":
        state.calendarMonth = addMonths(state.calendarMonth, 1);
        break;
      case "prevMonth":
        state.calendarMonth = addMonths(state.calendarMonth, -1);
        break;
      case "clearDate":
        state.selectedCalendarDate = "";
        break;
      case "today":
        state.calendarMonth = startOfMonth(new Date());
        state.selectedCalendarDate = formatDateKey(new Date());
        state.activeView = "memos";
        state.activeFilter = "all";
        state.activeTag = "";
        state.query = "";
        state.editingId = "";
        state.editPreviewVisible = false;
        state.sourceDraft = "";
        state.sourceEditingId = "";
        els.searchInput.value = "";
        break;
      default:
        break;
    }
    renderAll();
  }

  function selectCalendarDate(dateKey) {
    if (!dateKey) return;
    state.selectedCalendarDate = state.selectedCalendarDate === dateKey ? "" : dateKey;
    state.calendarMonth = startOfMonth(dateFromKey(dateKey));
    state.activeView = "memos";
    state.activeFilter = "all";
    state.activeTag = "";
    state.query = "";
    state.editingId = "";
    state.editPreviewVisible = false;
    state.sourceDraft = "";
    state.sourceEditingId = "";
    els.searchInput.value = "";
    renderAll();
  }

  function memoRenderContext(sourceId, options = {}) {
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

  function showToast(message) {
    els.toast.textContent = message;
    els.toast.classList.add("is-visible");
    if (state.toastTimer) window.clearTimeout(state.toastTimer);
    state.toastTimer = window.setTimeout(() => {
      els.toast.classList.remove("is-visible");
    }, 1800);
  }
}

export function mountDetachedMemoWindow(root, options = {}) {
  const params = new URLSearchParams(window.location.search);
  const state = {
    commentDraft: "",
    commentEditDraft: "",
    commentEditingId: "",
    commentExpandedIds: new Set(),
    commentEditPreviewVisible: false,
    commentPreviewVisible: false,
    commentSaving: false,
    comments: [],
    fixed: params.get("fixed") === "1" || Boolean(options.fixed),
    lastWindowState: null,
    memo: null,
    memoRefIndex: null,
    memos: [],
    editorSettings: loadEditorSettings(),
    snapshotDebounceTimer: null,
    snapshotInFlight: false,
    snapshotPollTimer: null,
    toastTimer: null,
    windowSession: null,
    windowName: detachedMemoWindowName(params.get("id") || ""),
  };

  root.innerHTML = detachedMemoWindowTemplate();

  const els = {
    commentEditorHost: root.querySelector("[data-window-comment-editor]"),
    commentFileInput: root.querySelector("[data-window-comment-file-input]"),
    commentForm: root.querySelector("[data-window-comment-form]"),
    commentPreview: root.querySelector("[data-window-comment-preview]"),
    commentPreviewToggle: root.querySelector("[data-window-comment-preview-toggle]"),
    commentSubmit: root.querySelector("[data-window-comment-submit]"),
    content: root.querySelector("[data-window-content]"),
    fixedButton: root.querySelector('[data-window-control="toggleFixed"]'),
    toast: root.querySelector("[data-toast]"),
    visibility: root.querySelector("[data-window-visibility]"),
  };
  state.windowSession = registerWindowSession({
    entryPage: "memo-window.html",
    fixed: state.fixed,
    getState: detachedWindowSessionState,
    kind: "memo_window",
    restoreState: restoreDetachedWindowSessionState,
    title: "Memo",
  });
  let detachedCommentEditor = createDetachedCommentEditor(state.commentDraft);
  let detachedCommentEditEditor = null;

  renderFixedButton();
  applyFixedState();
  renderDetachedState("正在加载 memo...");
  loadDetachedMemo();
  refreshDetachedEditorSettings();

  window.addEventListener("click", handleExternalLinkClick, true);
  window.addEventListener("beforeunload", handleDetachedBeforeUnload);
  window.addEventListener("resize", scheduleDetachedWindowStateSnapshot);
  root.addEventListener("click", handleClick);
  root.addEventListener("change", handleChange);
  root.addEventListener("keydown", handleKeydown, true);
  root.addEventListener("submit", handleSubmit);
  root.addEventListener("copy", handleMemoRenderedCopy);
  startDetachedWindowStateSnapshots();
  syncDetachedCommentForm();

  return {
    destroy() {
      stopDetachedWindowStateSnapshots();
      window.removeEventListener("click", handleExternalLinkClick, true);
      window.removeEventListener("beforeunload", handleDetachedBeforeUnload);
      window.removeEventListener("resize", scheduleDetachedWindowStateSnapshot);
      root.removeEventListener("click", handleClick);
      root.removeEventListener("change", handleChange);
      root.removeEventListener("keydown", handleKeydown, true);
      root.removeEventListener("submit", handleSubmit);
      root.removeEventListener("copy", handleMemoRenderedCopy);
      if (detachedCommentEditor) detachedCommentEditor.destroy();
      if (detachedCommentEditEditor) detachedCommentEditEditor.destroy();
      if (state.toastTimer) window.clearTimeout(state.toastTimer);
      root.innerHTML = "";
    },
  };

  function createDetachedCommentEditor(value) {
    if (!els.commentEditorHost) return null;
    return createMiniEditor(els.commentEditorHost, {
      memoItems() {
        return state.memos;
      },
      tagItems: detachedEditorTagItems,
      onChange(nextValue) {
        state.commentDraft = nextValue;
        renderDetachedCommentPreview();
        syncDetachedCommentForm();
      },
      onCommit() {
        return submitDetachedComment();
      },
      onDiscard() {
        state.commentDraft = "";
        state.commentPreviewVisible = false;
        renderDetachedCommentPreview();
        syncDetachedCommentForm();
        showToast("评论已清空");
      },
      onRequestFiles(accept) {
        if (detachedCommentEditor && detachedCommentEditor.requestFiles) {
          detachedCommentEditor.requestFiles(accept || "");
        }
      },
      onQuit() {
        if (detachedCommentEditor) detachedCommentEditor.blur();
      },
      onSave() {
        return submitDetachedComment();
      },
      onSubmit() {
        return submitDetachedComment();
      },
      onWriteDraft() {
        return submitDetachedComment();
      },
      placeholder: "有想法，直接问，Shift+Enter 换行",
      sourceMemoId: state.memo && state.memo.id,
      value: value || "",
      vim: detachedEditorVimEnabled(),
    });
  }

  function recreateDetachedCommentEditor() {
    const value = detachedCommentEditor ? detachedCommentEditor.getText() : state.commentDraft;
    if (detachedCommentEditor) detachedCommentEditor.destroy();
    if (els.commentEditorHost) els.commentEditorHost.innerHTML = "";
    detachedCommentEditor = createDetachedCommentEditor(value);
    state.commentDraft = value || "";
    syncDetachedCommentForm();
  }

  function detachedEditorTagItems() {
    return collectTags(state.memos.filter((memo) => memo && !memo.archived));
  }

  function detachedEditorVimEnabled() {
    return state.editorSettings && state.editorSettings.vimMode === true;
  }

  function createDetachedCommentEditEditor(value) {
    const host = els.content && els.content.querySelector("[data-window-comment-edit-host]");
    if (!host) return null;
    return createMiniEditor(host, {
      memoItems() {
        return state.memos;
      },
      tagItems: detachedEditorTagItems,
      onChange(nextValue) {
        state.commentEditDraft = nextValue;
        renderDetachedCommentEditPreview();
      },
      onCommit() {
        return saveDetachedCommentEdit();
      },
      onDiscard() {
        return cancelDetachedCommentEdit();
      },
      onRequestFiles(accept) {
        if (detachedCommentEditEditor && detachedCommentEditEditor.requestFiles) {
          detachedCommentEditEditor.requestFiles(accept || "");
        }
      },
      onQuit() {
        return cancelDetachedCommentEdit();
      },
      onSave() {
        return saveDetachedCommentEdit();
      },
      onSubmit() {
        return saveDetachedCommentEdit();
      },
      onWriteDraft() {
        return saveDetachedCommentEdit();
      },
      placeholder: "编辑评论...",
      sourceMemoId: state.memo && state.memo.id,
      value: value || "",
      vim: detachedEditorVimEnabled(),
    });
  }

  function toggleDetachedCommentPreview() {
    if (detachedCommentEditor) state.commentDraft = detachedCommentEditor.getText();
    state.commentPreviewVisible = !state.commentPreviewVisible;
    renderDetachedCommentPreview();
    if (!state.commentPreviewVisible && detachedCommentEditor) detachedCommentEditor.focus();
    scheduleDetachedWindowSessionSnapshot();
  }

  function toggleDetachedCommentEditPreview() {
    if (!state.commentEditingId) return;
    if (detachedCommentEditEditor) state.commentEditDraft = detachedCommentEditEditor.getText();
    state.commentEditPreviewVisible = !state.commentEditPreviewVisible;
    renderDetachedCommentEditPreview();
    if (!state.commentEditPreviewVisible && detachedCommentEditEditor) detachedCommentEditEditor.focus();
    scheduleDetachedWindowSessionSnapshot();
  }

  function renderDetachedCommentPreview() {
    renderDetachedEditorPreviewPanel(
      els.commentPreview,
      els.commentPreviewToggle,
      state.commentPreviewVisible,
      state.commentDraft,
      detachedCommentPreviewContext(),
    );
  }

  function renderDetachedCommentEditPreview() {
    renderDetachedEditorPreviewPanel(
      els.content && els.content.querySelector("[data-window-comment-edit-preview]"),
      els.content && els.content.querySelector('[data-window-comment-action="preview"]'),
      state.commentEditPreviewVisible,
      state.commentEditDraft,
      detachedCommentPreviewContext(),
    );
  }

  function detachedCommentPreviewContext() {
    return detachedMemoRenderContext(state, state.memo && state.memo.id, { readonly: true, showLineNumbers: false });
  }

  function renderDetachedEditorPreviewPanel(panel, button, visible, content, context) {
    updateDetachedEditorPreviewButton(button, visible);
    if (!panel) return;
    const switcher = closestElement(panel, ".memo-editor-switch");
    const host = switcher && switcher.querySelector("[data-editor-switch-host]");
    if (host) host.hidden = visible;
    panel.hidden = !visible;
    panel.classList.toggle("is-visible", visible);
    if (!visible) {
      panel.innerHTML = "";
      return;
    }
    panel.innerHTML = detachedEditorPreviewHTML(content, context);
  }

  function updateDetachedEditorPreviewButton(button, visible) {
    if (!button) return;
    button.setAttribute("aria-pressed", visible ? "true" : "false");
    button.title = visible ? "编辑" : "预览";
    button.setAttribute("aria-label", visible ? "编辑" : "预览");
    const label = button.querySelector("span");
    if (label) label.textContent = visible ? "编辑" : "预览";
  }

  function detachedEditorPreviewHTML(content, context) {
    const text = String(content || "");
    if (!text.trim()) return '<div class="memo-editor-preview-empty">暂无预览内容</div>';
    try {
      return `<div class="memo-content">${renderMemoMarkdown(text, context || {})}</div>`;
    } catch (err) {
      return `<div class="memo-content"><p>${escapeHTML(text)}</p></div>`;
    }
  }

  function destroyDetachedCommentEditEditor(options = {}) {
    if (detachedCommentEditEditor) {
      if (options.preserveDraft !== false) {
        state.commentEditDraft = detachedCommentEditEditor.getText();
      }
      detachedCommentEditEditor.destroy();
      detachedCommentEditEditor = null;
    }
  }

  function loadDetachedMemo() {
    const memoId = params.get("id") || "";
    if (!memoId) {
      renderDetachedState("缺少 memo id");
      return;
    }

    if (typeof invoke !== "function") {
      loadDetachedMemoFromLocal(memoId);
      return;
    }

    invoke("/api/memo-window/get?id=" + encodeURIComponent(memoId), { method: "GET" }).then(
      function (resp) {
        const data = resp && resp.code === 0 ? resp.data || {} : {};
        if (data.found && data.memo) {
          if (typeof data.fixed === "boolean") state.fixed = data.fixed;
          if (data.windowName) state.windowName = data.windowName;
          setDetachedPayload(data.memo, data.memos);
          renderDetachedMemo();
          loadDetachedComments(state.memo && state.memo.id);
          applyFixedState();
          scheduleDetachedWindowSessionSnapshot();
          return;
        }
        loadDetachedMemoFromLocal(memoId);
      },
      function () {
        loadDetachedMemoFromLocal(memoId);
      },
    );
  }

  function loadDetachedMemoFromLocal(memoId) {
    const payload = loadMemoFromLocal(memoId);
    const memo = payload.memo;
    if (!memo) {
      renderDetachedState("找不到 memo");
      return;
    }
    setDetachedPayload(memo, payload.memos);
    renderDetachedMemo();
    loadDetachedComments(state.memo && state.memo.id);
    scheduleDetachedWindowSessionSnapshot();
  }

  function refreshDetachedEditorSettings() {
    loadEditorSettingsFromVault().then(function (settings) {
      const next = normalizeEditorSettings(settings);
      const sameFileSettings = editorFileOpenSettingsKey(next) === editorFileOpenSettingsKey(state.editorSettings);
      const sameVimMode = next.vimMode === state.editorSettings.vimMode;
      if (sameFileSettings && sameVimMode) return;
      state.editorSettings = next;
      recreateDetachedCommentEditor();
      renderDetachedMemo();
    }, function () {});
  }

  function detachedMemoWindowName(memoId) {
    const suffix = sanitizeDetachedMemoWindowID(memoId) || "memo";
    return "memo-window-" + suffix;
  }

  function sanitizeDetachedMemoWindowID(value) {
    let text = String(value || "").trim().toLowerCase();
    let output = "";
    let lastDash = false;
    for (const char of text) {
      const ok = /[a-z0-9_-]/.test(char);
      if (ok) {
        output += char;
        lastDash = false;
      } else if (!lastDash) {
        output += "-";
        lastDash = true;
      }
    }
    return output.replace(/^[-_]+|[-_]+$/g, "");
  }

  function handleDetachedBeforeUnload() {
    snapshotDetachedWindowStateSync();
  }

  function startDetachedWindowStateSnapshots() {
    if (typeof invoke !== "function" || state.snapshotPollTimer) return;
    snapshotDetachedWindowStateIfChanged();
    state.snapshotPollTimer = window.setInterval(function () {
      snapshotDetachedWindowStateIfChanged();
    }, DETACHED_WINDOW_STATE_POLL_INTERVAL);
  }

  function stopDetachedWindowStateSnapshots() {
    if (state.snapshotPollTimer) {
      window.clearInterval(state.snapshotPollTimer);
      state.snapshotPollTimer = null;
    }
    if (state.snapshotDebounceTimer) {
      window.clearTimeout(state.snapshotDebounceTimer);
      state.snapshotDebounceTimer = null;
    }
  }

  function scheduleDetachedWindowStateSnapshot() {
    if (typeof invoke !== "function") return;
    if (state.snapshotDebounceTimer) {
      window.clearTimeout(state.snapshotDebounceTimer);
    }
    state.snapshotDebounceTimer = window.setTimeout(function () {
      state.snapshotDebounceTimer = null;
      snapshotDetachedWindowState();
    }, DETACHED_WINDOW_STATE_SNAPSHOT_DEBOUNCE);
  }

  function snapshotDetachedWindowState() {
    return readDetachedWindowState().then(function (nextWindowState) {
      if (!nextWindowState) return null;
      state.lastWindowState = nextWindowState;
      return saveDetachedWindowState(nextWindowState);
    });
  }

  function snapshotDetachedWindowStateIfChanged() {
    if (typeof invoke !== "function" || state.snapshotInFlight) return;
    state.snapshotInFlight = true;
    readDetachedWindowState().then(
      function (nextWindowState) {
        if (!nextWindowState) return null;
        if (isSameDetachedWindowStateHint(state.lastWindowState, nextWindowState)) return null;
        state.lastWindowState = nextWindowState;
        return saveDetachedWindowState(nextWindowState);
      },
      function () {},
    ).finally(function () {
      state.snapshotInFlight = false;
    });
  }

  function snapshotDetachedWindowStateSync() {
    const payload = detachedWindowStatePayload(state.lastWindowState || readDetachedWindowStateHint());
    if (!payload) return;
    try {
      const xhr = new XMLHttpRequest();
      xhr.open("POST", "/api/window/state/save", false);
      xhr.setRequestHeader("Content-Type", "application/json");
      xhr.send(JSON.stringify(payload));
    } catch (_) {}
  }

  function saveDetachedWindowState(windowState) {
    const payload = detachedWindowStatePayload(windowState);
    if (!payload || typeof invoke !== "function") return Promise.resolve(null);
    return invoke("/api/window/state/save", { method: "POST", args: payload }).catch(function () {});
  }

  function detachedWindowStatePayload(windowState) {
    const name = String(state.windowName || "").trim();
    if (!name) return null;
    if (!windowState || windowState.width <= 0 || windowState.height <= 0) return null;
    return {
      fixed: state.fixed,
      name,
      x: windowState.x,
      y: windowState.y,
      width: windowState.width,
      height: windowState.height,
    };
  }

  function readDetachedWindowState() {
    if (typeof invoke !== "function") {
      return Promise.resolve(readDetachedWindowStateHint());
    }
    return callNativeWindow("__velo/window/state").then(
      function (resp) {
        if (!resp || resp.success === false || resp.width <= 0 || resp.height <= 0) {
          return readDetachedWindowStateHint();
        }
        return {
          x: Math.round(Number(resp.x || 0)),
          y: Math.round(Number(resp.y || 0)),
          width: Math.round(Number(resp.width || 0)),
          height: Math.round(Number(resp.height || 0)),
        };
      },
      function () {
        return readDetachedWindowStateHint();
      },
    );
  }

  function readDetachedWindowStateHint() {
    return {
      x: Math.round(Number(window.screenX ?? window.screenLeft ?? 0)),
      y: Math.round(Number(window.screenY ?? window.screenTop ?? 0)),
      width: Math.round(Number(window.outerWidth ?? window.innerWidth ?? 0)),
      height: Math.round(Number(window.outerHeight ?? window.innerHeight ?? 0)),
    };
  }

  function isSameDetachedWindowStateHint(a, b) {
    return Boolean(a && b && a.x === b.x && a.y === b.y && a.width === b.width && a.height === b.height);
  }

  function setDetachedPayload(memo, memos) {
    state.memo = normalizeMemoPayload(memo);
    state.comments = [];
    state.commentDraft = "";
    state.commentEditingId = "";
    state.commentEditDraft = "";
    state.commentExpandedIds = new Set();
    state.memos = Array.isArray(memos)
      ? memos.map(normalizeMemoPayload).filter(Boolean)
      : [];
    if (state.memo && !state.memos.some((item) => item.id === state.memo.id)) {
      state.memos.unshift(state.memo);
    }
    state.memoRefIndex = null;
    if (detachedCommentEditor) recreateDetachedCommentEditor();
  }

  function handleExternalLinkClick(event) {
    if (event.defaultPrevented || event.button !== 0) return;
    const link = closestAnchor(event.target);
    if (!link || !root.contains(link)) return;

    const url = externalBrowserURLFromAnchor(link);
    if (!url) return;

    event.preventDefault();
    event.stopPropagation();
    openExternalLinkInDefaultBrowser(url);
  }

  function handleClick(event) {
    const control = closestElement(event.target, "[data-window-control]");
    if (control && root.contains(control)) {
      runWindowControl(control.dataset.windowControl);
      return;
    }

    const attach = closestElement(event.target, "[data-window-comment-attach]");
    if (attach && root.contains(attach)) {
      event.preventDefault();
      if (detachedCommentEditor && detachedCommentEditor.requestFiles && !state.commentSaving) {
        detachedCommentEditor.requestFiles("");
      } else if (els.commentFileInput && !state.commentSaving) {
        els.commentFileInput.click();
      }
      return;
    }

    const previewToggle = closestElement(event.target, "[data-window-comment-preview-toggle]");
    if (previewToggle && root.contains(previewToggle)) {
      event.preventDefault();
      toggleDetachedCommentPreview();
      return;
    }

    const commentAction = closestElement(event.target, "[data-window-comment-action]");
    if (commentAction && root.contains(commentAction)) {
      event.preventDefault();
      const commentNode = closestElement(commentAction, "[data-comment-id]");
      const commentId = commentNode && commentNode.dataset ? commentNode.dataset.commentId : "";
      runDetachedCommentAction(commentAction.dataset.windowCommentAction, commentId);
      return;
    }

    const editorOpen = closestElement(event.target, "[data-editor-open]");
    if (editorOpen && root.contains(editorOpen)) {
      event.preventDefault();
      event.stopPropagation();
      openFileInSelectedEditor(editorOpen);
      return;
    }

    const imagePreview = closestElement(event.target, "[data-image-preview-src]");
    if (imagePreview && root.contains(imagePreview)) {
      event.preventDefault();
      event.stopPropagation();
      openDetachedImagePreview(imagePreview);
      return;
    }

    const memoRefTarget = closestElement(event.target, "[data-memo-ref-target]");
    if (memoRefTarget && root.contains(memoRefTarget)) {
      event.preventDefault();
      focusDetachedMemo(memoRefTarget.dataset.memoRefTarget);
      return;
    }

    const action = closestElement(event.target, "[data-action]");
    if (!action || !root.contains(action)) return;

    switch (action.dataset.action) {
      case "copyMemo":
        copyDetachedMemo();
        break;
      case "copyMemoRef":
        copyDetachedMemoRef();
        break;
      case "copyCodeBlock":
        copyCodeBlockFromAction(action, state.memos, showToast);
        break;
      case "copyInlineLink":
        event.preventDefault();
        event.stopPropagation();
        copyInlineLinkFromAction(action, showToast);
        break;
      default:
        break;
    }
  }

  function handleChange(event) {
    if (event.target !== els.commentFileInput) return;
    const files = Array.from(els.commentFileInput.files || []);
    els.commentFileInput.value = "";
    if (!files.length) return;
    insertDetachedCommentFiles(files);
  }

  function handleKeydown(event) {
    if (!isDetachedCommentEditorTarget(event.target)) return;
    if (event.isComposing) return;
    if (event.key !== "Enter" || event.shiftKey || event.metaKey || event.ctrlKey || event.altKey) return;
    if (hasOpenMemoEditorMenu()) return;
    event.preventDefault();
    event.stopPropagation();
    submitDetachedComment();
  }

  function handleSubmit(event) {
    if (event.target !== els.commentForm) return;
    event.preventDefault();
    submitDetachedComment();
  }

  function runDetachedCommentAction(action, commentId) {
    switch (action) {
      case "edit":
        startDetachedCommentEdit(commentId);
        break;
      case "save":
        saveDetachedCommentEdit();
        break;
      case "cancel":
        cancelDetachedCommentEdit();
        break;
      case "preview":
        toggleDetachedCommentEditPreview();
        break;
      case "delete":
        deleteDetachedComment(commentId);
        break;
      case "toggleExpand":
        toggleDetachedCommentExpand(commentId);
        break;
      default:
        break;
    }
  }

  function toggleDetachedCommentExpand(commentId) {
    if (!commentId) return;
    if (state.commentExpandedIds.has(commentId)) {
      state.commentExpandedIds.delete(commentId);
    } else {
      state.commentExpandedIds.add(commentId);
    }
    renderDetachedMemo();
  }

  function startDetachedCommentEdit(commentId) {
    const comment = state.comments.find((item) => item && item.id === commentId);
    if (!comment) return;
    destroyDetachedCommentEditEditor({ preserveDraft: false });
    state.commentEditingId = comment.id;
    state.commentEditDraft = comment.content || "";
    state.commentEditPreviewVisible = false;
    renderDetachedMemo();
    if (detachedCommentEditEditor) detachedCommentEditEditor.focus();
  }

  function cancelDetachedCommentEdit() {
    destroyDetachedCommentEditEditor({ preserveDraft: false });
    state.commentEditingId = "";
    state.commentEditDraft = "";
    state.commentEditPreviewVisible = false;
    renderDetachedMemo();
    return Promise.resolve({ ok: true, message: "cancelled" });
  }

  function saveDetachedCommentEdit() {
    const commentId = state.commentEditingId;
    if (!commentId || state.commentSaving) return Promise.resolve({ ok: false, message: "没有正在编辑的评论" });
    if (detachedCommentEditEditor) state.commentEditDraft = detachedCommentEditEditor.getText();
    const content = String(state.commentEditDraft || "");
    if (!content.trim()) {
      showToast("评论不能为空");
      if (detachedCommentEditEditor) detachedCommentEditEditor.focus();
      return Promise.resolve({ ok: false, message: "评论不能为空" });
    }

    state.commentSaving = true;
    syncDetachedCommentForm();
    return updateMemoCommentInVault(commentId, { content }).then(
      function (comment) {
        const normalized = normalizeMemoCommentPayload(comment);
        if (normalized) {
          state.comments = state.comments.map((item) => item.id === normalized.id ? normalized : item);
        }
        destroyDetachedCommentEditEditor({ preserveDraft: false });
        state.commentEditingId = "";
        state.commentEditDraft = "";
        state.commentEditPreviewVisible = false;
        renderDetachedMemo();
        showToast("已保存评论");
        return { ok: true, message: "已保存评论" };
      },
      function (err) {
        showToast("保存评论失败: " + errorMessage(err));
        return { ok: false, message: "保存评论失败: " + errorMessage(err) };
      },
    ).finally(function () {
      state.commentSaving = false;
      syncDetachedCommentForm();
    });
  }

  function deleteDetachedComment(commentId) {
    const comment = state.comments.find((item) => item && item.id === commentId);
    if (!comment || state.commentSaving) return;
    if (!window.confirm("删除这条评论？")) return;
    state.commentSaving = true;
    syncDetachedCommentForm();
    deleteMemoCommentInVault(comment.id, { cleanupAssets: true }).then(
      function () {
        state.comments = state.comments.filter((item) => item.id !== comment.id);
        state.commentExpandedIds.delete(comment.id);
        if (state.commentEditingId === comment.id) {
          destroyDetachedCommentEditEditor({ preserveDraft: false });
          state.commentEditingId = "";
          state.commentEditDraft = "";
          state.commentEditPreviewVisible = false;
        }
        renderDetachedMemo();
        showToast("已删除评论");
      },
      function (err) {
        showToast("删除评论失败: " + errorMessage(err));
      },
    ).finally(function () {
      state.commentSaving = false;
      syncDetachedCommentForm();
    });
  }

  function insertDetachedCommentFiles(files) {
    if (!detachedCommentEditor || state.commentSaving) return;
    filesToMarkdown(files).then(
      function (markdown) {
        if (!markdown) return;
        insertTextIntoDetachedComment(markdown);
      },
      function (err) {
        showToast(uploadErrorMessage(err));
      },
    );
  }

  function insertTextIntoDetachedComment(text) {
    if (!detachedCommentEditor) return;
    detachedCommentEditor.insertBlock(String(text || ""));
    detachedCommentEditor.focus();
    syncDetachedCommentForm();
  }

  function isDetachedCommentEditorTarget(target) {
    return Boolean(els.commentEditorHost && target && (target === els.commentEditorHost || els.commentEditorHost.contains(target)));
  }

  function hasOpenMemoEditorMenu() {
    return Boolean(document.querySelector([
      ".file-picker-menu:not(.hidden)",
      ".memo-ref-menu:not(.hidden)",
      ".slash-command-menu:not(.hidden)",
      ".tag-picker-menu:not(.hidden)",
      ".time-picker-menu:not(.hidden)",
    ].join(",")));
  }

  function loadDetachedComments(memoId) {
    const id = String(memoId || "").trim();
    if (!id) {
      state.comments = [];
      renderDetachedMemo();
      return;
    }
    loadMemoCommentsFromVault(id).then(
      function (comments) {
        state.comments = comments.map(normalizeMemoCommentPayload).filter(Boolean);
        renderDetachedMemo();
      },
      function (err) {
        state.comments = [];
        renderDetachedMemo();
        showToast("读取评论失败: " + errorMessage(err));
      },
    );
  }

  function submitDetachedComment() {
    const memo = state.memo;
    if (detachedCommentEditor) state.commentDraft = detachedCommentEditor.getText();
    if (!memo || state.commentSaving) return Promise.resolve();
    const content = String(state.commentDraft || "").trim();
    if (!content) {
      syncDetachedCommentForm();
      return Promise.resolve();
    }

    state.commentSaving = true;
    syncDetachedCommentForm();
    return createMemoCommentInVault(memo.id, state.commentDraft).then(
      function (comment) {
        const normalized = normalizeMemoCommentPayload(comment);
        if (normalized) state.comments = commentsForDetachedMemo(memo.id).concat(normalized);
        state.commentDraft = "";
        state.commentPreviewVisible = false;
        if (detachedCommentEditor) detachedCommentEditor.setText("");
        renderDetachedMemo();
        syncDetachedCommentForm();
        showToast("已评论");
      },
      function (err) {
        showToast("评论失败: " + errorMessage(err));
      },
    ).finally(function () {
      state.commentSaving = false;
      syncDetachedCommentForm();
    });
  }

  function commentsForDetachedMemo(memoId) {
    const id = String(memoId || "").trim();
    return state.comments
      .filter((comment) => comment && comment.memoId === id)
      .sort(function (a, b) {
        const left = new Date(a.createdAt || 0).getTime() || 0;
        const right = new Date(b.createdAt || 0).getTime() || 0;
        if (left === right) return a.id.localeCompare(b.id);
        return left - right;
      });
  }

  function syncDetachedCommentForm() {
    if (!els.commentSubmit) return;
    if (detachedCommentEditor && detachedCommentEditor.getText() !== state.commentDraft) {
      detachedCommentEditor.setText(state.commentDraft);
    }
    els.commentSubmit.disabled = !state.memo || state.commentSaving || !String(state.commentDraft || "").trim();
    if (els.commentEditorHost) {
      els.commentEditorHost.classList.toggle("is-disabled", !state.memo || state.commentSaving);
    }
  }

  function syncDetachedCommentExpandControls() {
    if (!els.content) return;
    const collapsibleItems = els.content.querySelectorAll("[data-window-comment-collapse]");
    collapsibleItems.forEach(function (item) {
      const content = item.querySelector(".memo-comment-content");
      if (!content) return;

      item.classList.remove("is-short");
      if (item.classList.contains("is-collapsed") && content.scrollHeight <= content.clientHeight + 1) {
        item.classList.add("is-short");
      }

      content.querySelectorAll("img").forEach(function (image) {
        if (image.complete) return;
        if (image.dataset.memoWindowCommentExpandWatch) return;
        image.dataset.memoWindowCommentExpandWatch = "true";
        image.addEventListener("load", syncDetachedCommentExpandControls, { once: true });
      });
    });
  }

  function runWindowControl(control) {
    switch (control) {
      case "close":
        snapshotDetachedWindowState().finally(function () {
          forgetDetachedWindowOpenState().finally(function () {
            callNativeWindow("__velo/window/close").catch(function () {
              window.close();
            });
          });
        });
        break;
      case "minimize":
        callNativeWindow("__velo/window/minimize").catch(function () {});
        break;
      case "toggleMaximize":
        callNativeWindow("__velo/window/toggle_maximize").catch(function () {});
        break;
      case "toggleFixed":
        state.fixed = !state.fixed;
        applyFixedState();
        snapshotDetachedWindowState().catch(function () {});
        if (state.windowSession && state.windowSession.setFixed) {
          state.windowSession.setFixed(state.fixed).catch(function () {});
        }
        break;
      default:
        break;
    }
  }

  function applyFixedState() {
    renderFixedButton();
    document.body.classList.toggle("is-fixed-window", state.fixed);
    callNativeWindow("__velo/window/set_always_on_top", { onTop: state.fixed }).catch(function () {});
  }

  function forgetDetachedWindowOpenState() {
    if (typeof invoke !== "function" || !state.windowName) return Promise.resolve(null);
    return invoke("/api/window/opened/forget?name=" + encodeURIComponent(state.windowName), { method: "GET" }).catch(function () {});
  }

  function detachedWindowSessionState() {
    return {
      commentEditPreviewVisible: state.commentEditPreviewVisible,
      commentExpandedIds: Array.from(state.commentExpandedIds || []),
      commentPreviewVisible: state.commentPreviewVisible,
      memoId: state.memo && state.memo.id,
      scrollTop: els.content ? els.content.scrollTop : 0,
    };
  }

  function restoreDetachedWindowSessionState(sessionState) {
    const saved = sessionState && typeof sessionState === "object" ? sessionState : {};
    state.commentPreviewVisible = saved.commentPreviewVisible === true;
    state.commentEditPreviewVisible = saved.commentEditPreviewVisible === true;
    if (Array.isArray(saved.commentExpandedIds)) {
      state.commentExpandedIds = new Set(saved.commentExpandedIds.map(String).filter(Boolean));
    }
    window.setTimeout(function () {
      if (els.content && Number.isFinite(Number(saved.scrollTop))) {
        els.content.scrollTop = Number(saved.scrollTop);
      }
    }, 0);
  }

  function scheduleDetachedWindowSessionSnapshot() {
    if (state.windowSession && state.windowSession.scheduleSnapshot) {
      state.windowSession.scheduleSnapshot();
    }
  }

  function renderFixedButton() {
    if (!els.fixedButton) return;
    els.fixedButton.classList.toggle("is-active", state.fixed);
    els.fixedButton.setAttribute("aria-pressed", state.fixed ? "true" : "false");
    els.fixedButton.setAttribute("title", state.fixed ? "取消悬浮" : "悬浮在所有窗口上方");
  }

  function renderDetachedMemo() {
    const memo = state.memo;
    if (!memo) {
      renderDetachedState("找不到 memo");
      return;
    }

    destroyDetachedCommentEditEditor();
    const context = detachedMemoRenderContext(state, memo.id, { readonly: true });
    document.title = memoTitle(memo);
    renderDetachedVisibility(memo);
    els.content.innerHTML = detachedMemoCardTemplate(memo, context, {
      comments: commentsForDetachedMemo(memo.id),
      editingCommentId: state.commentEditingId,
      expandedCommentIds: state.commentExpandedIds,
    });
    if (state.commentEditingId) {
      detachedCommentEditEditor = createDetachedCommentEditEditor(state.commentEditDraft);
      renderDetachedCommentEditPreview();
      if (detachedCommentEditEditor) detachedCommentEditEditor.focus();
    }
    syncDetachedCommentExpandControls();
    syncDetachedCommentForm();
    renderDetachedCommentPreview();
  }

  function renderDetachedState(message) {
    document.title = "Memo";
    renderDetachedVisibility(null);
    els.content.innerHTML = `<div class="memo-window-empty">${escapeHTML(message || "")}</div>`;
    state.commentDraft = "";
    state.commentEditingId = "";
    state.commentEditDraft = "";
    state.commentExpandedIds = new Set();
    syncDetachedCommentForm();
  }

  function renderDetachedVisibility(memo) {
    if (!els.visibility) return;
    if (!memo) {
      els.visibility.hidden = true;
      els.visibility.innerHTML = "";
      return;
    }

    const visibility = VISIBILITY[memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
    els.visibility.innerHTML = `${SVG[visibility.icon]} ${escapeHTML(visibility.label)}`;
    els.visibility.hidden = false;
  }

  function focusDetachedMemo(memoId) {
    const target = state.memos.find((memo) => memo && memo.id === memoId);
    if (!target) {
      showToast("找不到引用的 memo");
      return;
    }
    state.memo = target;
    state.comments = [];
    state.commentDraft = "";
    state.commentEditingId = "";
    state.commentEditDraft = "";
    state.commentExpandedIds = new Set();
    recreateDetachedCommentEditor();
    renderDetachedMemo();
    loadDetachedComments(target.id);
  }

  function copyDetachedMemo() {
    if (!state.memo) return;
    copyText(state.memo.content).then(
      function () {
        showToast("已复制");
      },
      function () {
        showToast("复制失败");
      },
    );
  }

  function copyDetachedMemoRef() {
    if (!state.memo) return;
    copyText(`[[memo:${state.memo.id}|${memoReferenceAlias(memoTitle(state.memo))}]]`).then(
      function () {
        showToast("已复制 memo 引用");
      },
      function () {
        showToast("复制失败");
      },
    );
  }

  function openFileInSelectedEditor(button) {
    const file = button.dataset.editorFile || "";
    const label = button.dataset.editorLabel || button.dataset.editorAppName || "编辑器";
    if (!file) {
      showToast("没有可打开的本地文件");
      return;
    }
    if (typeof invoke !== "function") {
      showToast("当前环境不支持打开 " + label);
      return;
    }

    const line = button.dataset.editorLine || "1";
    const col = button.dataset.editorCol || "1";
    const appId = button.dataset.editorAppId || "";
    const appName = button.dataset.editorAppName || "";
    const appPath = button.dataset.editorAppPath || "";
    let url =
      "/api/editor/open?file=" +
      encodeURIComponent(file) +
      "&line=" +
      encodeURIComponent(line) +
      "&col=" +
      encodeURIComponent(col);
    if (appId) url += "&app=" + encodeURIComponent(appId);
    if (appName) url += "&appName=" + encodeURIComponent(appName);
    if (appPath) url += "&appPath=" + encodeURIComponent(appPath);
    button.disabled = true;
    invoke(url, { method: "GET" }).then(
      function (resp) {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || ("打开 " + label + " 失败"));
          return;
        }
        showToast("已在 " + label + " 中打开");
      },
      function (err) {
        showToast("打开 " + label + " 失败: " + err);
      },
    ).finally(function () {
      button.disabled = false;
    });
  }

  function openExternalLinkInDefaultBrowser(url) {
    if (typeof invoke !== "function") {
      window.open(url, "_blank", "noopener");
      return;
    }

    invoke("/api/external/open?url=" + encodeURIComponent(url), { method: "GET" }).then(
      function (resp) {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || "打开链接失败");
        }
      },
      function (err) {
        showToast("打开链接失败: " + err);
      },
    );
  }

  function openDetachedImagePreview(element) {
    openImagePreviewFromElement(element).catch(function (err) {
      showToast("打开图片预览失败: " + errorMessage(err));
    });
  }

  function callNativeWindow(method, args) {
    if (typeof invoke !== "function") {
      return Promise.reject(new Error("go bridge not available"));
    }
    return invoke(method, { args: args || {} });
  }

  function showToast(message) {
    els.toast.textContent = message;
    els.toast.classList.add("is-visible");
    if (state.toastTimer) window.clearTimeout(state.toastTimer);
    state.toastTimer = window.setTimeout(function () {
      els.toast.classList.remove("is-visible");
    }, 1800);
  }
}
