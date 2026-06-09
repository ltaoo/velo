import {
  DEFAULT_VISIBILITY,
  VISIBILITY,
  buildMemoReferenceIndex,
  collectTags,
  collectTodos,
  compactText,
  extractTags,
  getTodoStats,
  isMemoFenceLine,
  memoReferenceAlias,
  memoTitle,
  normalizeMemoPayload,
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
  collectLinks,
  collectResources,
  getResourceStats,
  sortMemoReference,
} from "../../domain/memo-resources.js";
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
  detachedMemoCardTemplate,
  detachedMemoRenderContext,
  detachedMemoWindowTemplate,
  emptyFeedTemplate,
  emptyFilesTemplate,
  emptyLinksTemplate,
  emptyTasksTemplate,
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
  taskGroupTemplate,
  taskWorkspaceTemplate,
} from "./memo-templates.js";
import {
  createMiniEditor,
  filesToMarkdown,
  insertPlainTextIntoEditor,
  refreshCloudStorageSettings,
  uploadErrorMessage,
} from "./memo-editor.js";
import { renderMemoMarkdown } from "./memo-markdown.js";
import { addMonths, dateFromKey, formatDateKey, formatShortDate, memoDateKey, startOfMonth } from "./memo-date.js";
import {
  closestAnchor,
  closestElement,
  copyText,
  escapeAttr,
  escapeCSSIdent,
  escapeHTML,
  externalBrowserURLFromAnchor,
} from "./memo-utils.js";

const LAST_PROJECT_STORAGE_KEY = "demo-desktop:memos:last-project:v1";
const SHORTCUTS_STORAGE_KEY = "demo-desktop:settings:shortcuts:v1";
const TASK_FILTER_STORAGE_KEY = "demo-desktop:gtd:task-filter:v1";
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

export function mountMemosHome(root) {
  const state = {
    activeFilter: "all",
    activeTag: "",
    activeView: "memos",
    calendarMonth: startOfMonth(new Date()),
    editingId: "",
    editDraft: "",
    editProjectId: "",
    editVisibility: DEFAULT_VISIBILITY,
    highlightMemoId: "",
    highlightTimer: null,
    expandedMemoIds: new Set(),
    draftsLoaded: false,
    gtdItems: [],
    gtdLoading: false,
    gtdMilestones: [],
    activeProjectFilter: "all",
    composerProjectId: localStorage.getItem(LAST_PROJECT_STORAGE_KEY) || "",
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
    tasks: [],
    tasksLoading: false,
    toastTimer: null,
    visibility: DEFAULT_VISIBILITY,
  };

  let composerEditor = null;
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

  composerEditor = createMiniEditor(els.composerHost, {
    memoItems() {
      return state.memos;
    },
    onChange(value) {
      renderComposerStatus(value);
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
    value: "",
    vimStatusHost: els.composerVimStatus,
  });

  renderAll();
  renderComposerStatus(composerEditor.getText());
  bindGoMessages();
  refreshProjectsFromVault();
  refreshMemosFromVault();
  refreshMemoDraftsFromVault();
  refreshTasksFromVault();
  refreshGTDFromVault();
  refreshStorageForRender();

  window.addEventListener("click", handleExternalLinkClick, true);
  root.addEventListener("click", handleClick);
  root.addEventListener("input", handleInput);
  root.addEventListener("change", handleChange);
  root.addEventListener("submit", handleSubmit);
  window.addEventListener("keydown", handleKeydown);

  return {
    destroy() {
      window.removeEventListener("click", handleExternalLinkClick, true);
      root.removeEventListener("click", handleClick);
      root.removeEventListener("input", handleInput);
      root.removeEventListener("change", handleChange);
      root.removeEventListener("submit", handleSubmit);
      window.removeEventListener("keydown", handleKeydown);
      if (state.toastTimer) window.clearTimeout(state.toastTimer);
      if (state.highlightTimer) window.clearTimeout(state.highlightTimer);
      if (composerEditor) composerEditor.destroy();
      if (editEditor) editEditor.destroy();
      editEditorMemoId = "";
      root.innerHTML = "";
    },
  };

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
      state.activeView = view.dataset.view;
      state.activeFilter = "all";
      state.activeTag = "";
      state.editingId = "";
      state.query = "";
      state.selectedCalendarDate = "";
      els.searchInput.value = "";
      renderAll();
      return;
    }

    const taskFilter = closestElement(event.target, "[data-task-filter]");
    if (taskFilter && root.contains(taskFilter)) {
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
      openFileInVSCode(editorOpen);
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
      case "clearFilters":
        state.activeFilter = "all";
        state.activeTag = "";
        state.activeProjectFilter = "all";
        state.composerProjectId = state.lastComposerProjectId || "";
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
      if (payload.type === "memo_file_drop") {
        insertDroppedFiles(payload.files);
      }
    });
  }

  function openFileInVSCode(button) {
    const file = button.dataset.editorFile || "";
    if (!file) {
      showToast("没有可打开的本地文件");
      return;
    }
    if (typeof invoke !== "function") {
      showToast("当前环境不支持打开 VS Code");
      return;
    }

    const line = button.dataset.editorLine || "1";
    const col = button.dataset.editorCol || "1";
    button.disabled = true;
    invoke(
      "/api/editor/open?file=" +
        encodeURIComponent(file) +
        "&line=" +
        encodeURIComponent(line) +
        "&col=" +
        encodeURIComponent(col) +
        "&app=code",
      { method: "GET" },
    ).then(
      function (resp) {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || "打开 VS Code 失败");
          return;
        }
        showToast("已在 VS Code 中打开");
      },
      function (err) {
        showToast("打开 VS Code 失败: " + err);
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
    const memo = findMemo(memoId);
    if (!memo) {
      showToast("找不到 memo");
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
      els.memoSearchResults.innerHTML = '<div class="memo-command-empty">没有匹配的 memo</div>';
      return;
    }

    state.memoSearchActiveIndex = Math.max(0, Math.min(state.memoSearchActiveIndex, results.length - 1));
    els.memoSearchResults.innerHTML = results.map(function (memo, index) {
      const title = memoTitle(memo);
      const summary = compactText(memo.content, 112);
      const meta = [
        memo.archived ? "归档" : "",
        memo.pinned ? "置顶" : "",
        projectLabel(memo.projectId),
        formatRelativeDate(memo.createdAt),
      ].filter(Boolean).join(" · ");
      return [
        '<button class="memo-command-result ' + (index === state.memoSearchActiveIndex ? "is-active" : "") + '" type="button" role="option" aria-selected="' + (index === state.memoSearchActiveIndex ? "true" : "false") + '" data-memo-search-result="' + escapeAttr(memo.id) + '">',
        '<span class="memo-command-result-title">' + escapeHTML(title) + '</span>',
        '<span class="memo-command-result-summary">' + escapeHTML(summary) + '</span>',
        '<span class="memo-command-result-meta">' + escapeHTML(meta) + '</span>',
        '</button>',
      ].join("");
    }).join("");

    const active = els.memoSearchResults.querySelector(".memo-command-result.is-active");
    if (active) active.scrollIntoView({ block: "nearest" });
  }

  function memoSearchResults() {
    const query = state.memoSearchQuery.toLowerCase();
    return state.memos
      .filter(function (memo) {
        if (!query) return true;
        return [
          memo.id,
          memoTitle(memo),
          memo.content,
          memo.visibility,
          projectLabel(memo.projectId),
        ].join(" ").toLowerCase().includes(query);
      })
      .sort(function (a, b) {
        if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
        return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
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
    }
  }

  function handleKeydown(event) {
    if (state.memoSearchOpen) {
      handleMemoSearchKeydown(event);
      return;
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
      completeExistingTask(taskNode.dataset.taskId);
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

  function completeExistingTask(taskId) {
    const id = String(taskId || "").trim();
    if (!id) return;
    completeTask(id).then(
      function (task) {
        const summary = normalizeTaskSummary(task);
        state.tasks = state.tasks.map((item) => item.id === id && summary ? summary : item);
        renderAll();
        refreshTasksFromVault();
        showToast("已完成任务");
      },
      function (err) {
        showToast("完成任务失败: " + errorMessage(err));
        refreshTasksFromVault();
      },
    );
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


  function startEdit(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    const draft = findDraft(memoEditDraftId(memoId));
    state.editingId = memoId;
    state.editDraft = draft ? draft.content : memo.content;
    state.editProjectId = draft ? normalizeProjectID(draft.projectId) : memo.projectId || "";
    state.editVisibility = draft ? draft.visibility || DEFAULT_VISIBILITY : memo.visibility || DEFAULT_VISIBILITY;
    renderFeed();
    if (draft) showToast("已恢复编辑草稿");
  }

  function toggleMemoExpand(memoId) {
    if (!memoId) return;
    if (state.expandedMemoIds.has(memoId)) {
      state.expandedMemoIds.delete(memoId);
    } else {
      state.expandedMemoIds.add(memoId);
    }
    renderFeed();
  }

  function cancelEdit() {
    const draftId = state.editingId ? memoEditDraftId(state.editingId) : "";
    state.editingId = "";
    state.editDraft = "";
    renderFeed();
    if (draftId) {
      removeDraftFromState(draftId);
      deleteMemoDraftInVault(draftId).catch(function (err) {
        showToast("删除草稿失败: " + errorMessage(err));
      });
    }
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
    let inCode = false;
    const todoLines = lines.filter(function (line) {
      if (isMemoFenceLine(line)) {
        inCode = !inCode;
        return false;
      }
      if (inCode) return false;
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
    const fileCount = collectManagedResources([memo]).length;
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

  function insertDroppedFiles(files) {
    if (!els.composerHost || !els.composerHost.isConnected) return;
    droppedFilesToMarkdown(files).then(function (markdown) {
      if (!markdown || !composerEditor) return;
      composerEditor.insertBlock(markdown);
      composerEditor.focus();
      showToast("已插入拖拽文件");
    }).catch(function (err) {
      showToast(uploadErrorMessage(err));
    });
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
    els.memoList.classList.toggle("is-resource-list", state.activeView === "links" || state.activeView === "files");
  }

  function renderMainContent() {
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
      case "files":
        renderFiles();
        return;
      default:
        renderFeed();
    }
  }

  function syncEditDraftFromEditor() {
    if (!editEditor || !state.editingId || editEditorMemoId !== state.editingId) return;
    state.editDraft = editEditor.getText();
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
    const resourceCount = collectResources(memos).length;
    const openItemCount = scopedGTDItems().filter((item) => item.status !== "closed" && item.status !== "resolved").length;
    const activeMilestoneCount = scopedGTDMilestones().filter((milestone) => milestone.status === "active" || milestone.status === "planned").length;
    els.todoNavCount.textContent = todoStats.open ? String(todoStats.open) : "";
    if (els.itemNavCount) els.itemNavCount.textContent = openItemCount ? String(openItemCount) : "";
    if (els.milestoneNavCount) els.milestoneNavCount.textContent = activeMilestoneCount ? String(activeMilestoneCount) : "";
    els.linkNavCount.textContent = linkCount ? String(linkCount) : "";
    els.fileNavCount.textContent = resourceCount ? String(resourceCount) : "";
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
    els.calendar.innerHTML = calendarTemplate(state.calendarMonth, scopedMemos(), state.selectedCalendarDate);
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
          .map(
            (memo) => `
              <article class="memo-pinned-item">
                <div class="memo-pinned-content memo-content">${renderMemoMarkdown(memo.content, memoRenderContext(memo.id, { readonly: true }))}</div>
                <small>${formatShortDate(memo.createdAt)}</small>
              </article>
            `,
          )
          .join("")
      : '<div class="memo-empty-mini">暂无置顶</div>';
  }

  function renderFeed() {
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
          .map((memo) => memoTemplate(memo, state.editingId, memoRenderContext(memo.id), state.expandedMemoIds.has(memo.id), state.projects))
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
          onChange(value) {
            state.editDraft = value;
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
          vimStatusHost: statusHost,
        });
        editEditorMemoId = memo.id;
        editEditor.focus();
      }
    }

    syncMemoExpandControls();
  }

  function syncMemoExpandControls() {
    const collapsibleItems = els.memoList.querySelectorAll("[data-memo-collapse]");
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
        return `${memo.content} ${memo.visibility}`.toLowerCase().includes(query);
      })
      .sort((a, b) => {
        if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
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
    return taskTimeValue(b.updatedAt || b.createdAt) - taskTimeValue(a.updatedAt || a.createdAt);
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
    fixed: params.get("fixed") === "1" || Boolean(options.fixed),
    memo: null,
    memoRefIndex: null,
    memos: [],
    toastTimer: null,
  };

  root.innerHTML = detachedMemoWindowTemplate();

  const els = {
    content: root.querySelector("[data-window-content]"),
    fixedButton: root.querySelector('[data-window-control="toggleFixed"]'),
    toast: root.querySelector("[data-toast]"),
    visibility: root.querySelector("[data-window-visibility]"),
  };

  renderFixedButton();
  applyFixedState();
  renderDetachedState("正在加载 memo...");
  loadDetachedMemo();

  window.addEventListener("click", handleExternalLinkClick, true);
  root.addEventListener("click", handleClick);

  return {
    destroy() {
      window.removeEventListener("click", handleExternalLinkClick, true);
      root.removeEventListener("click", handleClick);
      if (state.toastTimer) window.clearTimeout(state.toastTimer);
      root.innerHTML = "";
    },
  };

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
          setDetachedPayload(data.memo, data.memos);
          renderDetachedMemo();
          applyFixedState();
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
  }

  function setDetachedPayload(memo, memos) {
    state.memo = normalizeMemoPayload(memo);
    state.memos = Array.isArray(memos)
      ? memos.map(normalizeMemoPayload).filter(Boolean)
      : [];
    if (state.memo && !state.memos.some((item) => item.id === state.memo.id)) {
      state.memos.unshift(state.memo);
    }
    state.memoRefIndex = null;
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

    const editorOpen = closestElement(event.target, "[data-editor-open]");
    if (editorOpen && root.contains(editorOpen)) {
      event.preventDefault();
      event.stopPropagation();
      openFileInVSCode(editorOpen);
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
      default:
        break;
    }
  }

  function runWindowControl(control) {
    switch (control) {
      case "close":
        callNativeWindow("__velo/window/close").catch(function () {
          window.close();
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

    const context = detachedMemoRenderContext(state, memo.id, { readonly: true });
    document.title = memoTitle(memo);
    renderDetachedVisibility(memo);
    els.content.innerHTML = detachedMemoCardTemplate(memo, context);
  }

  function renderDetachedState(message) {
    document.title = "Memo";
    renderDetachedVisibility(null);
    els.content.innerHTML = `<div class="memo-window-empty">${escapeHTML(message || "")}</div>`;
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
    renderDetachedMemo();
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

  function openFileInVSCode(button) {
    const file = button.dataset.editorFile || "";
    if (!file) {
      showToast("没有可打开的本地文件");
      return;
    }
    if (typeof invoke !== "function") {
      showToast("当前环境不支持打开 VS Code");
      return;
    }

    const line = button.dataset.editorLine || "1";
    const col = button.dataset.editorCol || "1";
    button.disabled = true;
    invoke(
      "/api/editor/open?file=" +
        encodeURIComponent(file) +
        "&line=" +
        encodeURIComponent(line) +
        "&col=" +
        encodeURIComponent(col) +
        "&app=code",
      { method: "GET" },
    ).then(
      function (resp) {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || "打开 VS Code 失败");
          return;
        }
        showToast("已在 VS Code 中打开");
      },
      function (err) {
        showToast("打开 VS Code 失败: " + err);
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
