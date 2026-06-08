import {
  DEFAULT_VISIBILITY,
  VISIBILITY,
  buildMemoReferenceIndex,
  collectTags,
  collectTodos,
  compactText,
  extractTags,
  getTodoStats,
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

const DRAFT_STORAGE_KEY = "demo-desktop:memos:draft:v1";
const LAST_PROJECT_STORAGE_KEY = "demo-desktop:memos:last-project:v1";
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
    activeProjectFilter: "all",
    composerProjectId: localStorage.getItem(LAST_PROJECT_STORAGE_KEY) || "",
    lastComposerProjectId: localStorage.getItem(LAST_PROJECT_STORAGE_KEY) || "",
    memoRefIndex: null,
    memos: loadMemos(),
    projects: loadProjects(),
    query: "",
    selectedCalendarDate: "",
    sortDesc: true,
    saving: false,
    taskDetails: new Map(),
    taskFilter: "today",
    tasks: [],
    tasksLoading: false,
    toastTimer: null,
    visibility: DEFAULT_VISIBILITY,
  };

  let composerEditor = null;
  let editEditor = null;

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
    linkNavCount: root.querySelector("[data-link-nav-count]"),
    mainSubtitle: root.querySelector("[data-main-subtitle]"),
    mainTitle: root.querySelector("[data-main-title]"),
    memoList: root.querySelector("[data-memo-list]"),
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
  };

  composerEditor = createMiniEditor(els.composerHost, {
    memoItems() {
      return state.memos;
    },
    onChange(value) {
      localStorage.setItem(DRAFT_STORAGE_KEY, value);
      renderComposerStatus(value);
    },
    onSubmit() {
      createMemo();
    },
    placeholder: "记录想法、任务或链接...",
    value: localStorage.getItem(DRAFT_STORAGE_KEY) || "",
    vimStatusHost: els.composerVimStatus,
  });

  renderAll();
  renderComposerStatus(composerEditor.getText());
  bindGoMessages();
  refreshProjectsFromVault();
  refreshMemosFromVault();
  refreshTasksFromVault();
  refreshStorageForRender();
  window.addEventListener("focus", refreshStorageForRender);

  window.addEventListener("click", handleExternalLinkClick, true);
  root.addEventListener("click", handleClick);
  root.addEventListener("input", handleInput);
  root.addEventListener("change", handleChange);
  root.addEventListener("submit", handleSubmit);

  return {
    destroy() {
      window.removeEventListener("click", handleExternalLinkClick, true);
      root.removeEventListener("click", handleClick);
      root.removeEventListener("input", handleInput);
      root.removeEventListener("change", handleChange);
      root.removeEventListener("submit", handleSubmit);
      window.removeEventListener("focus", refreshStorageForRender);
      if (state.toastTimer) window.clearTimeout(state.toastTimer);
      if (state.highlightTimer) window.clearTimeout(state.highlightTimer);
      if (composerEditor) composerEditor.destroy();
      if (editEditor) editEditor.destroy();
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
      state.taskFilter = taskFilter.dataset.taskFilter || "today";
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
    if (event.target.matches("[data-search-input]")) {
      state.query = event.target.value.trim();
      renderAll();
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

  function createMemo() {
    if (state.saving) return;
    const content = composerEditor.getText();
    if (!content.trim()) {
      showToast("先写点内容");
      composerEditor.focus();
      return;
    }

    state.saving = true;
    renderComposerStatus(content);
    createMemoInVault(content, state.visibility, state.composerProjectId).then(
      function (memo) {
        const normalized = normalizeMemoPayload(memo);
        state.memos = [normalized].filter(Boolean).concat(state.memos);
        saveMemos(state.memos);
        rememberComposerProject(state.composerProjectId);
        composerEditor.setText("");
        localStorage.removeItem(DRAFT_STORAGE_KEY);
        state.activeView = "memos";
        state.activeFilter = "all";
        state.activeTag = "";
        state.selectedCalendarDate = "";
        renderAll();
        renderComposerStatus("");
        refreshTasksFromVault();
        showToast("已发布到 " + projectLabel(normalized && normalized.projectId));
        window.requestAnimationFrame(() => {
          if (composerEditor && els.composerHost.isConnected) composerEditor.focus();
        });
      },
      function (err) {
        showToast("发布失败: " + errorMessage(err));
      },
    ).finally(function () {
      state.saving = false;
      renderComposerStatus(composerEditor.getText());
    });
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
    state.editingId = memoId;
    state.editDraft = memo.content;
    state.editProjectId = memo.projectId || "";
    state.editVisibility = memo.visibility || DEFAULT_VISIBILITY;
    renderFeed();
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
    state.editingId = "";
    state.editDraft = "";
    renderFeed();
  }

  function saveEdit(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    const content = editEditor ? editEditor.getText() : state.editDraft;
    if (!content.trim()) {
      showToast("内容不能为空");
      return;
    }
    state.editingId = "";
    state.editDraft = "";
    updateMemo(memoId, {
      content,
      projectId: state.editProjectId,
      updatedAt: new Date().toISOString(),
      visibility: state.editVisibility,
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
      updateMemoInVault(memoId, patch).then(
        function (memo) {
          const normalized = normalizeMemoPayload(memo);
          if (!normalized) return;
          state.memos = state.memos.map((item) => item.id === memoId ? normalized : item);
          saveMemos(state.memos);
          renderAll();
          if (Object.prototype.hasOwnProperty.call(patch, "content")) {
            refreshTasksFromVault();
          }
        },
        function (err) {
          showToast("保存失败: " + errorMessage(err));
          refreshMemosFromVault();
        },
      );
    }
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
    const todoLines = lines.filter((line) => parseTaskLine(line));
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
    els.memoList.classList.toggle("is-todo-list", state.activeView === "todos");
    els.memoList.classList.toggle("is-resource-list", state.activeView === "links" || state.activeView === "files");
  }

  function renderMainContent() {
    switch (state.activeView) {
      case "todos":
        renderTodos();
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
    els.todoNavCount.textContent = todoStats.open ? String(todoStats.open) : "";
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
    const linkCount = collectLinks(memos).length;
    const resourceStats = getResourceStats(memos);

    els.stats.innerHTML = [
      statTemplate("全部", active.length),
      statTemplate("公开", publicCount),
      statTemplate("标签", tags.length),
      statTemplate("任务", taskStats.total),
      statTemplate("未完成", taskStats.open),
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
      editEditor.destroy();
      editEditor = null;
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
          onSubmit() {
            saveEdit(memo.id);
          },
          placeholder: "编辑 memo...",
          sourceMemoId: memo.id,
          value: memo.content,
          vimStatusHost: statusHost,
        });
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
      editEditor.destroy();
      editEditor = null;
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

  function renderLinks() {
    if (editEditor) {
      editEditor.destroy();
      editEditor = null;
    }

    const links = visibleLinks();
    els.feedCount.textContent = links.length ? `${links.length} 个链接` : "0 个链接";
    els.memoList.innerHTML = links.length ? links.map(linkTemplate).join("") : emptyLinksTemplate();
  }

  function renderFiles() {
    if (editEditor) {
      editEditor.destroy();
      editEditor = null;
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
        if (payload.activeProjectId && state.activeProjectFilter === "all") {
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

  function taskMatchesFilter(task, filter) {
    switch (filter) {
      case "all":
        return true;
      case "completed":
        return task.status === "completed";
      case "inbox":
        return task.status !== "completed" && (task.listId === "inbox" || !task.listId);
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
    return [task.startAt, task.dueAt].some((value) => value && dateKey(new Date(value)) === today);
  }

  function isTaskOverdue(task) {
    if (!task.dueAt || task.status === "completed") return false;
    const due = new Date(task.dueAt);
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
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? Number.MAX_SAFE_INTEGER : date.getTime();
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
