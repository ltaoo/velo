const MEMOS_STORAGE_KEY = "demo-desktop:memos:items:v1";
const DRAFT_STORAGE_KEY = "demo-desktop:memos:draft:v1";
const CLOUD_STORAGE_KEY = "demo-desktop:settings:cloud-storage:v1";
const PROJECTS_STORAGE_KEY = "demo-desktop:memos:projects:v1";
const LAST_PROJECT_STORAGE_KEY = "demo-desktop:memos:last-project:v1";
const DEFAULT_VISIBILITY = "PRIVATE";
const TASK_LINE_REGEX = /^(\s*[-*]\s+\[)([ xX])(\]\s+)(.*)$/;
const CALENDAR_WEEKDAYS = ["日", "一", "二", "三", "四", "五", "六"];

let cloudStorageSettingsCache = null;

const VISIBILITY = {
  PRIVATE: { label: "仅自己", icon: "lock" },
  PROTECTED: { label: "工作区", icon: "shield" },
  PUBLIC: { label: "公开", icon: "globe" },
};

const SVG = {
  archive:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="4" width="18" height="4" rx="1"></rect><path d="M5 8v11a2 2 0 0 0 2 2h10a2 2 0 0 0 2-2V8"></path><path d="M10 12h4"></path></svg>',
  bold:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M7 5h6.2a3.3 3.3 0 0 1 0 6.6H7z"></path><path d="M7 11.6h7.2a3.7 3.7 0 0 1 0 7.4H7z"></path></svg>',
  check:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 6 9 17l-5-5"></path></svg>',
  chevronDown:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m6 9 6 6 6-6"></path></svg>',
  chevronLeft:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 18-6-6 6-6"></path></svg>',
  chevronRight:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 18 6-6-6-6"></path></svg>',
  clock:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9"></circle><path d="M12 7v5l3 2"></path></svg>',
  code:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 18-6-6 6-6"></path><path d="m15 6 6 6-6 6"></path></svg>',
  copy:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="9" y="9" width="11" height="11" rx="2"></rect><rect x="4" y="4" width="11" height="11" rx="2"></rect></svg>',
  edit:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 20h9"></path><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"></path></svg>',
  external:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M15 3h6v6"></path><path d="M10 14 21 3"></path><path d="M21 14v5a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5"></path></svg>',
  globe:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9"></circle><path d="M3 12h18"></path><path d="M12 3a14 14 0 0 1 0 18"></path><path d="M12 3a14 14 0 0 0 0 18"></path></svg>',
  hash:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 9h16"></path><path d="M4 15h16"></path><path d="M10 3 8 21"></path><path d="m16 3-2 18"></path></svg>',
  image:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="3" y="5" width="18" height="14" rx="2"></rect><circle cx="8.5" cy="10" r="1.5"></circle><path d="m21 15-4-4L8 19"></path></svg>',
  italic:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M19 4h-9"></path><path d="M14 20H5"></path><path d="m15 4-6 16"></path></svg>',
  link:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M10 13a5 5 0 0 0 7.1 0l2-2a5 5 0 0 0-7.1-7.1l-1.1 1.1"></path><path d="M14 11a5 5 0 0 0-7.1 0l-2 2A5 5 0 0 0 12 20.1l1.1-1.1"></path></svg>',
  list:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M8 6h13"></path><path d="M8 12h13"></path><path d="M8 18h13"></path><path d="M3 6h.01"></path><path d="M3 12h.01"></path><path d="M3 18h.01"></path></svg>',
  lock:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="4" y="10" width="16" height="10" rx="2"></rect><path d="M8 10V7a4 4 0 0 1 8 0v3"></path></svg>',
  paperclip:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m21.4 11.1-9.2 9.2a6 6 0 1 1-8.5-8.5l9.2-9.2a4 4 0 0 1 5.7 5.7l-9.2 9.2a2 2 0 0 1-2.8-2.8l8.5-8.5"></path></svg>',
  pin:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 4 5 5-4 4v5l-2 2-5-5-5-5 2-2h5z"></path><path d="m9 15-5 5"></path></svg>',
  plus:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14"></path><path d="M5 12h14"></path></svg>',
  restore:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 12a9 9 0 1 0 3-6.7"></path><path d="M3 4v6h6"></path></svg>',
  search:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7"></circle><path d="m21 21-4.3-4.3"></path></svg>',
  send:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m22 2-7 20-4-9-9-4z"></path><path d="M22 2 11 13"></path></svg>',
  settings:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V22a2 2 0 1 1-4 0v-.2a1.7 1.7 0 0 0-1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H2a2 2 0 1 1 0-4h.2a1.7 1.7 0 0 0 1.5-1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3 1.7 1.7 0 0 0 1-1.5V2a2 2 0 1 1 4 0v.2a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8 1.7 1.7 0 0 0 1.5 1H22a2 2 0 1 1 0 4h-.2a1.7 1.7 0 0 0-1.5 1z"></path></svg>',
  shield:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10"></path></svg>',
  sort:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 9h12"></path><path d="M9 15h6"></path><path d="M3 3h18"></path><path d="M10 21h4"></path></svg>',
  trash:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6h18"></path><path d="M8 6V4h8v2"></path><path d="M19 6l-1 14H6L5 6"></path></svg>',
  x:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 6 6 18"></path><path d="m6 6 12 12"></path></svg>',
};

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
  refreshStorageForRender();
  window.addEventListener("focus", refreshStorageForRender);

  window.addEventListener("click", handleExternalLinkClick, true);
  root.addEventListener("click", handleClick);
  root.addEventListener("input", handleInput);
  root.addEventListener("change", handleChange);

  return {
    destroy() {
      window.removeEventListener("click", handleExternalLinkClick, true);
      root.removeEventListener("click", handleClick);
      root.removeEventListener("input", handleInput);
      root.removeEventListener("change", handleChange);
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

    switch (action.dataset.action) {
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
    lines[lineIndex] = lines[lineIndex].replace(
      TASK_LINE_REGEX,
      function (_match, prefix, _marker, suffix, text) {
        return `${prefix}${checked ? "x" : " "}${suffix}${text}`;
      },
    );
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
      deleteMemoInVault(memoId, { cleanupAssets: options.deleteFiles }).then(
        function (result) {
          state.memos = state.memos.filter((item) => item.id !== memoId);
          if (preservedMemo) {
            state.memos = [preservedMemo].concat(state.memos);
          }
          saveMemos(state.memos);
          renderAll();
          if (result && Array.isArray(result.assetErrors) && result.assetErrors.length) {
            showToast("已删除 memo，部分文件删除失败");
          } else if (preservedMemo) {
            showToast("已删除 memo，todo 已保留");
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
    const todoStats = getTodoStats(memos);
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
    const todoStats = getTodoStats(memos);
    const linkCount = collectLinks(memos).length;
    const resourceStats = getResourceStats(memos);

    els.stats.innerHTML = [
      statTemplate("全部", active.length),
      statTemplate("公开", publicCount),
      statTemplate("标签", tags.length),
      statTemplate("代办", todoStats.total),
      statTemplate("未完成", todoStats.open),
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

    const todos = visibleTodos();
    const openTodos = todos.filter((todo) => !todo.checked);
    const doneTodos = todos.filter((todo) => todo.checked);
    els.feedCount.textContent = todos.length ? `${openTodos.length} 未完成 / ${todos.length} 项` : "0 项";
    els.memoList.innerHTML = todos.length
      ? [
          todoGroupTemplate("未完成", openTodos, (todo) => memoRenderContext(todo.memoId, { readonly: true }), state.projects),
          todoGroupTemplate("已完成", doneTodos, (todo) => memoRenderContext(todo.memoId, { readonly: true }), state.projects),
        ].join("")
      : emptyTodosTemplate();
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
    const saved = loadJSON(MEMOS_STORAGE_KEY, []);
    const memos = Array.isArray(saved) ? saved : [];
    const memo = memos.find((item) => item && item.id === memoId);
    if (!memo) {
      renderDetachedState("找不到 memo");
      return;
    }
    setDetachedPayload(memo, memos);
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
      <div class="memo-toast" data-toast role="status"></div>
    </div>
  `;
}

function detachedMemoCardTemplate(memo, renderContext) {
  const tags = extractTags(memo.content);
  const backlinks = memoBacklinkCount(renderContext, memo.id);

  return `
    <article class="memo-card memo-window-card" data-memo-id="${escapeAttr(memo.id)}">
      <header class="memo-card-head">
        <div class="memo-author">
          <div class="memo-avatar">U</div>
          <div>
            <div class="memo-author-name">You</div>
            <time datetime="${escapeAttr(memo.createdAt)}">${formatRelativeDate(memo.createdAt)}</time>
          </div>
        </div>
        <div class="memo-card-meta">
          ${memo.pinned ? '<span class="memo-pin-label">置顶</span>' : ""}
          ${backlinks ? `<span class="memo-backlink-label">${backlinks} 引用</span>` : ""}
        </div>
      </header>
      <div class="memo-content">${renderMemoMarkdown(memo.content, renderContext)}</div>
      ${tags.length ? `<div class="memo-card-tags">${tags.map((tag) => `<span>#${escapeHTML(tag)}</span>`).join("")}</div>` : ""}
      <footer class="memo-card-actions">
        <button class="memo-action-button" type="button" data-action="copyMemo" title="复制">${SVG.copy}</button>
        <button class="memo-action-button" type="button" data-action="copyMemoRef" title="复制引用">${SVG.link}</button>
      </footer>
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
    sourceId: sourceId || "",
    stack: options.stack || (sourceId ? [sourceId] : []),
  };
}

function normalizeMemoPayload(memo) {
  if (!memo || typeof memo !== "object") return null;
  const id = String(memo.id || "").trim();
  if (!id) return null;
  return {
    archived: Boolean(memo.archived),
    content: String(memo.content || ""),
    createdAt: memo.createdAt || new Date().toISOString(),
    id,
    pinned: Boolean(memo.pinned),
    projectId: normalizeProjectID(memo.projectId),
    updatedAt: memo.updatedAt || "",
    visibility: memo.visibility || DEFAULT_VISIBILITY,
  };
}

function normalizeProjectPayload(project) {
  if (!project || typeof project !== "object") return null;
  const id = normalizeProjectID(project.id);
  const name = String(project.name || "").trim();
  if (!id || !name) return null;
  return {
    archived: Boolean(project.archived),
    color: normalizeProjectColor(project.color),
    createdAt: project.createdAt || new Date().toISOString(),
    id,
    name,
    sortOrder: Number.isFinite(Number(project.sortOrder)) ? Number(project.sortOrder) : 0,
    updatedAt: project.updatedAt || "",
  };
}

function normalizeProjectID(value) {
  return String(value || "").trim().replace(/[^A-Za-z0-9_-]+/g, "-").replace(/^-+|-+$/g, "");
}

function normalizeProjectFilter(value) {
  const raw = String(value || "").trim();
  if (raw === "all" || raw === "unassigned") return raw;
  return normalizeProjectID(raw) || "all";
}

function normalizeProjectColor(value) {
  const color = String(value || "").trim();
  return /^#[0-9a-fA-F]{6}$/.test(color) ? color.toLowerCase() : "#2563eb";
}

function activeViewMeta(view) {
  const metas = {
    files: {
      hideComposer: true,
      searchPlaceholder: "搜索文件、图片或来源 memo",
      subtitle: "从所有 memo 中汇总文件和图片",
      title: "文件",
    },
    links: {
      hideComposer: true,
      searchPlaceholder: "搜索链接或来源 memo",
      subtitle: "从所有 memo 中汇总超链接",
      title: "超链接",
    },
    memos: {
      hideComposer: false,
      searchPlaceholder: "搜索 memos",
      subtitle: "捕捉、整理、回看",
      title: "Inbox",
    },
    todos: {
      hideComposer: true,
      searchPlaceholder: "搜索代办或来源 memo",
      subtitle: "从所有 memo 中汇总任务",
      title: "代办",
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
            <span>标签</span>
            <span data-tag-summary></span>
          </div>
          <div class="memo-tag-list" data-tag-list></div>
        </div>
        <div class="memo-sidebar-section">
          <div class="memo-sidebar-heading">
            <span>聚合</span>
          </div>
          <nav class="memo-nav memo-collection-nav" aria-label="Memo collections">
            ${viewNavButtonTemplate("todos", "代办", SVG.check, "data-todo-nav-count")}
            ${viewNavButtonTemplate("links", "超链接", SVG.link, "data-link-nav-count")}
            ${viewNavButtonTemplate("files", "文件", SVG.paperclip, "data-file-nav-count")}
          </nav>
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
          <div class="memo-editor-host" data-composer-host></div>
          <div class="memo-composer-toolbar">
            <div class="memo-composer-status-line">
              <span data-composer-vim-status></span>
              <span data-composer-status></span>
            </div>
            <div class="memo-composer-actions">
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
          <div class="memo-inspector-title">概览</div>
          <div class="memo-stats" data-stats></div>
        </section>
        <section class="memo-inspector-section">
          <div class="memo-inspector-title">置顶</div>
          <div class="memo-pinned-list" data-pinned-list></div>
        </section>
      </aside>
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

function calendarTemplate(monthDate, memos, selectedDate) {
  const month = startOfMonth(monthDate);
  const counts = memoDateCounts(memos);
  const days = generateCalendarDays(month);
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
      ${CALENDAR_WEEKDAYS.map((day) => `<span>${day}</span>`).join("")}
    </div>
    <div class="memo-calendar-grid">
      ${days
        .map((day) => {
          const count = counts.get(day.key) || 0;
          const classes = [
            "memo-calendar-day",
            day.inMonth ? "" : "is-outside",
            day.key === todayKey ? "is-today" : "",
            day.key === selectedDate ? "is-selected" : "",
            count ? "has-memo" : "",
          ]
            .filter(Boolean)
            .join(" ");
          return `
            <button
              class="${classes}"
              type="button"
              data-calendar-date="${escapeAttr(day.key)}"
              aria-label="${escapeAttr(day.key)}"
            >
              <span>${day.date.getDate()}</span>
              ${count ? `<strong>${count}</strong>` : ""}
            </button>
          `;
        })
        .join("")}
    </div>
  `;
}

function memoTemplate(memo, editingId, renderContext, expanded = false, projects = []) {
  const visibility = VISIBILITY[memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
  const tags = extractTags(memo.content);
  const archived = memo.archived;
  const editing = memo.id === editingId;
  const backlinks = memoBacklinkCount(renderContext, memo.id);
  const expandLabel = expanded ? "收起" : "展开";
  const projectBadge = projectBadgeTemplate(memo.projectId, projects);

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
          : `
            <div class="memo-list-collapse ${expanded ? "is-expanded" : "is-collapsed"}" data-memo-collapse>
              <div class="memo-content">${renderMemoMarkdown(memo.content, renderContext)}</div>
              <button class="memo-expand-button" type="button" data-action="toggleMemoExpand" aria-expanded="${expanded ? "true" : "false"}" title="${expandLabel}">
                <span>${expandLabel}</span>
                ${SVG.chevronDown}
              </button>
            </div>
            ${tags.length ? `<div class="memo-card-tags">${tags.map((tag) => `<button type="button" data-tag="${escapeAttr(tag)}">#${escapeHTML(tag)}</button>`).join("")}</div>` : ""}
          `
      }
      <footer class="memo-card-actions">
        <button class="memo-action-button" type="button" data-action="copyMemoRef" title="复制引用">${SVG.link}</button>
        <button class="memo-action-button" type="button" data-action="editMemo" title="编辑">${SVG.edit}</button>
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
      <label class="memo-todo-check">
        <input type="checkbox" data-task-line="${todo.lineIndex}" ${todo.checked ? "checked" : ""} />
        <span>${inlineMarkdown(todo.text, renderContext)}</span>
      </label>
      <div class="memo-todo-source">
        <button class="memo-todo-source-button" type="button" data-action="openSourceMemo" title="查看来源 memo">
          <span>来源 memo</span>
          <strong>${escapeHTML(todo.sourceText)}</strong>
        </button>
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

function linkTemplate(link) {
  const visibility = VISIBILITY[link.memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
  const tags = extractTags(link.memo.content);
  const href = safeUrl(link.url);
  return `
    <article class="memo-resource-card is-link" data-memo-id="${escapeAttr(link.memoId)}">
      <a class="memo-resource-target" href="${escapeAttr(href)}" target="_blank" rel="noreferrer">
        <span class="memo-resource-icon">${SVG.link}</span>
        <span class="memo-resource-body">
          <span class="memo-resource-title">${escapeHTML(link.label || link.url)}</span>
          <span class="memo-resource-url">${escapeHTML(compactFileURL(link.url))}</span>
        </span>
      </a>
      <div class="memo-resource-source">
        <button class="memo-todo-source-button" type="button" data-action="openSourceMemo" title="查看来源 memo">
          <span>来源 memo</span>
          <strong>${escapeHTML(link.sourceText)}</strong>
        </button>
        <div class="memo-todo-meta">
          <time datetime="${escapeAttr(link.memo.createdAt)}">${formatRelativeDate(link.memo.createdAt)}</time>
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
        <button class="memo-todo-source-button" type="button" data-action="openSourceMemo" title="查看来源 memo">
          <span>来源 memo</span>
          <strong>${escapeHTML(resource.sourceText)}</strong>
        </button>
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

function editTemplate(memo, projects = []) {
  return `
    <div class="memo-inline-editor">
      <div class="memo-editor-host is-inline" data-edit-host></div>
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

function createMiniEditor(host, options) {
  const editorOptions = options || {};
  if (!window.ProsemirrorEditor) return createFallbackEditor(host, editorOptions);

  host.dataset.placeholder = editorOptions.placeholder || "";

  let editor = null;
  editor = new window.ProsemirrorEditor({
    $el: host,
    mode: "mini",
    value: editorOptions.value || "",
    vim: editorOptions.vim !== false,
    fileItems: editorOptions.fileItems || defaultEditorFileItems(),
    onChange(instance) {
      syncEmptyState();
      if (editorOptions.onChange) editorOptions.onChange(instance.getText());
    },
    onFileQuery(query) {
      if (!editor) return;
      const source = editorOptions.fileItems || defaultEditorFileItems();
      editor.setFileItems(filterEditorFileItems(source, query), query);
    },
    onRemoveFile(file) {
      if (editorOptions.onRemoveFile) editorOptions.onRemoveFile(file);
    },
    onSave(instance) {
      if (editorOptions.onSave) editorOptions.onSave(instance);
    },
    onSelectFile(file) {
      if (editorOptions.onSelectFile) editorOptions.onSelectFile(file);
    },
    onUploadImage(image) {
      return readFileAsDataURL(image.file).then(function (url) {
        return fileInfoToUploadURL({
          name: image.fileName || (image.file && image.file.name) || "image",
          type: (image.file && image.file.type) || "",
          url,
        });
      });
    },
  });

  const removePlugins = installMemoEditorPlugins(editor, editorOptions);
  const removeStatus = installVimStatus(host, editor, editorOptions.vimStatusHost);
  const removeVimFocus = installVimEditingMode(host, editor);
  const removeSubmit = installSubmitShortcut(host, editorOptions);
  const removeDrop = installFileDropHandler(host, editor);

  setEditorVimMode(editor, "insert");
  syncEmptyState();

  return {
    destroy() {
      removeSubmit();
      removeDrop();
      removeVimFocus();
      removeStatus();
      removePlugins();
      editor.destroy();
    },
    focus() {
      editor.focus();
      setEditorVimMode(editor, "insert");
    },
    getText() {
      return editor.getText();
    },
    insertBlock(text) {
      const current = editor.getText();
      const prefix = current && !current.endsWith("\n") ? "\n" : "";
      insertText(prefix + text);
    },
    insertText,
    insertFiles(files) {
      insertFilesIntoEditor(editor, files);
    },
    requestFiles(accept) {
      requestFilesForEditor(editor, accept || "");
    },
    setText(value) {
      editor.setText(value || "");
      resetEditorSelection(editor);
      syncEmptyState();
    },
    wrap(prefix, suffix, placeholder) {
      const view = editor.view;
      const { from, to, empty } = view.state.selection;
      const selected = empty ? placeholder : view.state.doc.textBetween(from, to, "\n");
      const text = `${prefix}${selected}${suffix}`;
      const transaction = view.state.tr.insertText(text, from, to);
      const cursor = empty ? from + prefix.length + selected.length : from + text.length;
      transaction.setSelection(window.ProsemirrorMod.TextSelection.create(transaction.doc, cursor));
      view.dispatch(transaction.scrollIntoView());
    },
  };

  function insertText(text) {
    const view = editor.view;
    view.dispatch(view.state.tr.insertText(text).scrollIntoView());
  }

  function syncEmptyState() {
    host.classList.toggle("is-empty", editor.getText().trim().length === 0);
  }
}

function installVimEditingMode(host, editor) {
  function enterInsertModeNow() {
    setEditorVimMode(editor, "insert");
  }

  function enterInsertModeSoon() {
    window.setTimeout(function () {
      enterInsertModeNow();
    }, 0);
  }

  host.addEventListener("mousedown", enterInsertModeNow, true);
  host.addEventListener("touchstart", enterInsertModeNow, true);
  host.addEventListener("focusin", enterInsertModeSoon);
  host.addEventListener("mouseup", enterInsertModeSoon);
  host.addEventListener("touchend", enterInsertModeSoon);

  return function () {
    host.removeEventListener("mousedown", enterInsertModeNow, true);
    host.removeEventListener("touchstart", enterInsertModeNow, true);
    host.removeEventListener("focusin", enterInsertModeSoon);
    host.removeEventListener("mouseup", enterInsertModeSoon);
    host.removeEventListener("touchend", enterInsertModeSoon);
  };
}

function setEditorVimMode(editor, mode) {
  if (!editor || !editor.view || !window.vimPluginKey) return false;
  const state = window.vimPluginKey.getState(editor.view.state);
  if (!state || state.mode === mode) return false;

  editor.view.dispatch(
    editor.view.state.tr.setMeta(window.vimPluginKey, {
      count: "",
      mode,
      pending: null,
      visualAnchor: null,
      visualLine: false,
    }),
  );
  return true;
}

function resetEditorSelection(editor) {
  const PM = window.ProsemirrorMod;
  const view = editor && editor.view;
  if (!PM || !view || !view.state || !view.state.doc) return;

  const doc = view.state.doc;
  const pos = Math.min(1, doc.content.size);
  try {
    view.dispatch(
      view.state.tr
        .setSelection(PM.Selection.near(doc.resolve(pos), 1))
        .scrollIntoView(),
    );
  } catch (_) {
  }
}

function installFileDropHandler(host, editor) {
  function filesFromDataTransfer(dataTransfer) {
    return dataTransfer && dataTransfer.files && dataTransfer.files.length
      ? dataTransfer.files
      : [];
  }

  function filesFromClipboard(clipboardData) {
    return clipboardData && clipboardData.files && clipboardData.files.length
      ? clipboardData.files
      : [];
  }

  function onDragOver(event) {
    if (!filesFromDataTransfer(event.dataTransfer).length) return;
    event.preventDefault();
    event.stopPropagation();
    event.dataTransfer.dropEffect = "copy";
  }

  function onDrop(event) {
    const files = filesFromDataTransfer(event.dataTransfer);
    if (!files.length) return;
    event.preventDefault();
    event.stopPropagation();
    insertFilesIntoEditor(editor, files);
    editor.focus();
  }

  function onPaste(event) {
    const files = filesFromClipboard(event.clipboardData);
    if (files.length) {
      event.preventDefault();
      event.stopPropagation();
      insertFilesIntoEditor(editor, files);
      editor.focus();
      return;
    }

    const url = clipboardPlainURL(event.clipboardData);
    if (!url) return;
    event.preventDefault();
    event.stopPropagation();
    if (!insertMarkdownLinkIntoEditor(editor, url)) {
      insertPlainTextIntoEditor(editor, markdownLinkText(markdownLinkLabel(url), url));
    }
    editor.focus();
  }

  host.addEventListener("dragenter", onDragOver, true);
  host.addEventListener("dragover", onDragOver, true);
  host.addEventListener("drop", onDrop, true);
  host.addEventListener("paste", onPaste, true);
  return function () {
    host.removeEventListener("dragenter", onDragOver, true);
    host.removeEventListener("dragover", onDragOver, true);
    host.removeEventListener("drop", onDrop, true);
    host.removeEventListener("paste", onPaste, true);
  };
}

function installSubmitShortcut(host, options) {
  function onKeyDown(event) {
    if ((event.metaKey || event.ctrlKey) && event.key === "Enter" && options.onSubmit) {
      event.preventDefault();
      options.onSubmit();
    }
  }
  host.addEventListener("keydown", onKeyDown, true);
  return function () {
    host.removeEventListener("keydown", onKeyDown, true);
  };
}

function installVimStatus(host, editor, statusHost) {
  const mount = statusHost || host;
  if (!mount || typeof mount.appendChild !== "function") return function () {};

  const status = document.createElement("span");
  status.className = "memo-vim-status";
  mount.appendChild(status);

  function update() {
    const mode = editor.view.dom.getAttribute("data-vim-mode") || "insert";
    status.dataset.mode = mode;
    status.textContent = mode.toUpperCase();
  }

  const observer = new MutationObserver(update);
  observer.observe(editor.view.dom, {
    attributes: true,
    attributeFilter: ["class", "data-vim-mode"],
  });
  update();

  return function () {
    observer.disconnect();
    status.remove();
  };
}

function installMemoEditorPlugins(editor, options) {
  const PM = window.ProsemirrorMod;
  if (!PM || !editor.view || !editor.view.state.reconfigure) {
    return function () {};
  }

  const plugins = [
    createMemoReferencePlugin(editor, options || {}),
    createMemoSlashCommandPlugin(editor, options || {}),
    createMemoTimePickerPlugin(editor),
  ];
  editor.view.updateState(
    editor.view.state.reconfigure({
      plugins: plugins.concat(editor.view.state.plugins),
    }),
  );

  return function () {};
}

function defaultEditorFileItems() {
  return [
    { label: "index.html", href: "frontend/index.html", detail: "前端入口" },
    { label: "index.js", href: "frontend/index.js", detail: "memo 单入口逻辑" },
    { label: "index.css", href: "frontend/public/index.css", detail: "界面与编辑器样式" },
    { label: "main.go", href: "main.go", detail: "Velo 后端入口" },
    { label: "app-config.json", href: "app-config.json", detail: "应用配置" },
    { label: "vim.js", href: "frontend/public/vim.js", detail: "Vim 模式插件" },
    {
      label: "prosemirror-editor.umd.js",
      href: "frontend/public/prosemirror-editor.umd.js",
      detail: "Mini editor UMD",
    },
  ];
}

function filterEditorFileItems(items, query) {
  const keyword = String(query || "").trim().toLowerCase();
  return (items || defaultEditorFileItems())
    .filter(function (item) {
      if (!keyword) return true;
      return [item.label, item.name, item.href, item.detail]
        .join("\n")
        .toLowerCase()
        .includes(keyword);
    })
    .slice(0, 8);
}

function createMemoReferencePlugin(editor, options) {
  const PM = window.ProsemirrorMod;
  const key = new PM.PluginKey(editor.id + "-memoReference");

  function empty(dismissedKey) {
    return {
      active: false,
      dismissedKey: dismissedKey || null,
      embed: false,
      from: null,
      items: [],
      query: "",
      selectedIndex: 0,
      to: null,
    };
  }

  function findTrigger(state) {
    const selection = state.selection;
    if (!selection.empty) return null;

    const $from = selection.$from;
    if (!$from.parent.isTextblock) return null;

    const before = $from.parent.textBetween(0, $from.parentOffset, "\ufffc", "\ufffc");
    const openIndex = before.lastIndexOf("[[");
    if (openIndex < 0) return null;

    const query = before.slice(openIndex + 2);
    if (/[\[\]\n]/.test(query)) return null;

    const embed = openIndex > 0 && before.charAt(openIndex - 1) === "!";
    const markerStart = embed ? openIndex - 1 : openIndex;
    const prev = markerStart > 0 ? before.charAt(markerStart - 1) : "";
    if (prev && !/[\s([{>]/.test(prev)) return null;

    const from = selection.from - query.length - 2 - (embed ? 1 : 0);
    return {
      embed,
      from,
      key: from + ":" + selection.from + ":" + (embed ? "embed" : "link") + ":" + query,
      query,
      to: selection.from,
    };
  }

  function selectItem(view, item) {
    const state = key.getState(view.state);
    if (!state || !state.active || !item) return false;

    const text = memoReferenceInsertText(item, state.embed);
    view.dispatch(
      view.state.tr
        .insertText(text, state.from, state.to)
        .setMeta(key, { type: "close" })
        .scrollIntoView(),
    );
    view.focus();
    return true;
  }

  return new PM.Plugin({
    key,
    state: {
      init: empty,
      apply(transaction, value, oldState, newState) {
        const meta = transaction.getMeta(key);
        if (meta && meta.type === "close") {
          const trigger = findTrigger(newState);
          return empty(trigger ? trigger.key : null);
        }

        const trigger = findTrigger(newState);
        if (!trigger) return empty();
        if (trigger.key === value.dismissedKey && !transaction.docChanged) {
          return {
            ...empty(value.dismissedKey),
            embed: trigger.embed,
            from: trigger.from,
            query: trigger.query,
            to: trigger.to,
          };
        }

        const items = memoReferenceItems(options, trigger.query);
        let selectedIndex = value.selectedIndex || 0;
        if (meta && meta.type === "setSelectedIndex") selectedIndex = meta.selectedIndex || 0;
        selectedIndex = items.length ? Math.max(0, Math.min(selectedIndex, items.length - 1)) : 0;

        return {
          active: true,
          dismissedKey: null,
          embed: trigger.embed,
          from: trigger.from,
          items,
          query: trigger.query,
          selectedIndex,
          to: trigger.to,
        };
      },
    },
    props: {
      decorations(state) {
        const pluginState = key.getState(state);
        if (!pluginState || !pluginState.active || pluginState.from >= pluginState.to) {
          return PM.DecorationSet.empty;
        }
        return PM.DecorationSet.create(state.doc, [
          PM.Decoration.inline(pluginState.from, pluginState.to, {
            class: "memo-ref-query-range",
          }),
        ]);
      },
      handleKeyDown(view, event) {
        const pluginState = key.getState(view.state);
        if (!pluginState || !pluginState.active) return false;

        if (event.key === "ArrowDown" || event.key === "ArrowUp") {
          const direction = event.key === "ArrowDown" ? 1 : -1;
          const length = pluginState.items.length;
          const selectedIndex = length
            ? (pluginState.selectedIndex + direction + length) % length
            : 0;
          event.preventDefault();
          view.dispatch(
            view.state.tr.setMeta(key, { type: "setSelectedIndex", selectedIndex }),
          );
          return true;
        }

        if (event.key === "Enter" || event.key === "Tab") {
          if (!pluginState.items.length) return false;
          event.preventDefault();
          return selectItem(view, pluginState.items[pluginState.selectedIndex]);
        }

        if (event.key === "Escape") {
          event.preventDefault();
          view.dispatch(view.state.tr.setMeta(key, { type: "close" }));
          return true;
        }

        return false;
      },
    },
    view(view) {
      return createFloatingMenuView({
        className: "memo-ref-menu hidden",
        key,
        render(menu, pluginState) {
          renderMemoReferenceMenu(menu, pluginState);
        },
        onMouseDown(event, pluginState) {
          const option = closestElement(event.target, "[data-memo-ref-index]");
          if (!option) return false;
          event.preventDefault();
          const item = pluginState.items[Number(option.dataset.memoRefIndex)];
          return selectItem(view, item);
        },
        view,
      });
    },
  });
}

function createMemoSlashCommandPlugin(editor, options) {
  const PM = window.ProsemirrorMod;
  const key = new PM.PluginKey(editor.id + "-memoSlashCommand");
  const commands = memoSlashCommands();

  function empty(dismissedKey) {
    return {
      active: false,
      dismissedKey: dismissedKey || null,
      from: null,
      items: [],
      query: "",
      selectedIndex: 0,
      to: null,
    };
  }

  function findTrigger(state) {
    const selection = state.selection;
    if (!selection.empty) return null;

    const $from = selection.$from;
    if (!$from.parent.isTextblock) return null;

    const before = $from.parent.textBetween(0, $from.parentOffset, "\ufffc", "\ufffc");
    const match = /(^|\s)\/([^\s/]*)$/u.exec(before);
    if (!match) return null;

    const query = match[2] || "";
    const from = selection.from - query.length - 1;
    if (from < $from.start()) return null;

    return {
      from,
      key: from + ":" + selection.from + ":" + query,
      query,
      to: selection.from,
    };
  }

  function itemsFor(query) {
    const keyword = String(query || "").trim().toLowerCase();
    if (!keyword) return commands;
    return commands.filter(function (item) {
      return [item.label, item.detail, item.keywords].join("\n").toLowerCase().includes(keyword);
    });
  }

  function selectItem(view, item) {
    const state = key.getState(view.state);
    if (!state || !state.active || !item) return false;

    const text = typeof item.text === "function" ? item.text() : item.text || "";
    let transaction = view.state.tr.insertText(text, state.from, state.to);
    transaction = transaction.setMeta(key, { type: "close" }).scrollIntoView();
    view.dispatch(transaction);

    if (item.action === "files" && options.onRequestFiles) {
      options.onRequestFiles(item.accept || "");
    } else if (item.action === "files") {
      requestFilesForEditor(editor, item.accept || "");
    }
    view.focus();
    return true;
  }

  return new PM.Plugin({
    key,
    state: {
      init: empty,
      apply(transaction, value, oldState, newState) {
        const meta = transaction.getMeta(key);
        if (meta && meta.type === "close") {
          const trigger = findTrigger(newState);
          return empty(trigger ? trigger.key : null);
        }

        const trigger = findTrigger(newState);
        if (!trigger) return empty();
        if (trigger.key === value.dismissedKey && !transaction.docChanged) {
          return {
            ...empty(value.dismissedKey),
            from: trigger.from,
            query: trigger.query,
            to: trigger.to,
          };
        }

        const items = itemsFor(trigger.query);
        let selectedIndex = value.selectedIndex || 0;
        if (meta && meta.type === "setSelectedIndex") selectedIndex = meta.selectedIndex || 0;
        selectedIndex = items.length ? Math.max(0, Math.min(selectedIndex, items.length - 1)) : 0;

        return {
          active: true,
          dismissedKey: null,
          from: trigger.from,
          items,
          query: trigger.query,
          selectedIndex,
          to: trigger.to,
        };
      },
    },
    props: {
      decorations(state) {
        const pluginState = key.getState(state);
        if (!pluginState || !pluginState.active || pluginState.from >= pluginState.to) {
          return PM.DecorationSet.empty;
        }
        return PM.DecorationSet.create(state.doc, [
          PM.Decoration.inline(pluginState.from, pluginState.to, {
            class: "slash-command-range",
          }),
        ]);
      },
      handleKeyDown(view, event) {
        const pluginState = key.getState(view.state);
        if (!pluginState || !pluginState.active) return false;

        if (event.key === "ArrowDown" || event.key === "ArrowUp") {
          const direction = event.key === "ArrowDown" ? 1 : -1;
          const length = pluginState.items.length;
          const selectedIndex = length
            ? (pluginState.selectedIndex + direction + length) % length
            : 0;
          event.preventDefault();
          view.dispatch(
            view.state.tr.setMeta(key, { type: "setSelectedIndex", selectedIndex }),
          );
          return true;
        }

        if (event.key === "Enter" || event.key === "Tab") {
          if (!pluginState.items.length) return false;
          event.preventDefault();
          return selectItem(view, pluginState.items[pluginState.selectedIndex]);
        }

        if (event.key === "Escape") {
          event.preventDefault();
          view.dispatch(view.state.tr.setMeta(key, { type: "close" }));
          return true;
        }

        return false;
      },
    },
    view(view) {
      return createFloatingMenuView({
        className: "slash-command-menu hidden",
        key,
        render(menu, pluginState) {
          renderSlashCommandMenu(menu, pluginState);
        },
        onMouseDown(event, pluginState) {
          const option = closestElement(event.target, "[data-slash-command-index]");
          if (!option) return false;
          event.preventDefault();
          const item = pluginState.items[Number(option.dataset.slashCommandIndex)];
          return selectItem(view, item);
        },
        view,
      });
    },
  });
}

function createMemoTimePickerPlugin(editor) {
  const PM = window.ProsemirrorMod;
  const key = new PM.PluginKey(editor.id + "-memoTimePicker");

  function empty(dismissedKey) {
    return {
      active: false,
      dismissedKey: dismissedKey || null,
      from: null,
      items: [],
      query: "",
      selectedIndex: 0,
      to: null,
      trigger: "::",
    };
  }

  function findTrigger(state) {
    const selection = state.selection;
    if (!selection.empty) return null;

    const $from = selection.$from;
    if (!$from.parent.isTextblock) return null;

    const before = $from.parent.textBetween(0, $from.parentOffset, "\ufffc", "\ufffc");
    const indexes = [
      { trigger: "::", index: before.lastIndexOf("::") },
      { trigger: "：：", index: before.lastIndexOf("：：") },
    ].filter(function (item) {
      return item.index >= 0;
    });
    if (!indexes.length) return null;

    indexes.sort(function (a, b) {
      return b.index - a.index;
    });
    const found = indexes[0];
    const prev = found.index > 0 ? before.charAt(found.index - 1) : "";
    if (prev && !/\s/.test(prev)) return null;

    const query = before.slice(found.index + found.trigger.length);
    const from = selection.from - query.length - found.trigger.length;
    return {
      from,
      key: from + ":" + selection.from + ":" + query,
      query,
      to: selection.from,
      trigger: found.trigger,
    };
  }

  function selectItem(view, item) {
    const state = key.getState(view.state);
    if (!state || !state.active || !item) return false;

    const text = state.trigger + item.value + " ";
    view.dispatch(
      view.state.tr
        .insertText(text, state.from, state.to)
        .setMeta(key, { type: "close" })
        .scrollIntoView(),
    );
    view.focus();
    return true;
  }

  return new PM.Plugin({
    key,
    state: {
      init: empty,
      apply(transaction, value, oldState, newState) {
        const meta = transaction.getMeta(key);
        if (meta && meta.type === "close") {
          const trigger = findTrigger(newState);
          return empty(trigger ? trigger.key : null);
        }

        const trigger = findTrigger(newState);
        if (!trigger) return empty();
        if (trigger.key === value.dismissedKey && !transaction.docChanged) {
          return {
            ...empty(value.dismissedKey),
            from: trigger.from,
            query: trigger.query,
            to: trigger.to,
            trigger: trigger.trigger,
          };
        }

        const items = memoTimeItems(trigger.query);
        let selectedIndex = value.selectedIndex || 0;
        if (meta && meta.type === "setSelectedIndex") selectedIndex = meta.selectedIndex || 0;
        selectedIndex = items.length ? Math.max(0, Math.min(selectedIndex, items.length - 1)) : 0;

        return {
          active: true,
          dismissedKey: null,
          from: trigger.from,
          items,
          query: trigger.query,
          selectedIndex,
          to: trigger.to,
          trigger: trigger.trigger,
        };
      },
    },
    props: {
      decorations(state) {
        const pluginState = key.getState(state);
        if (!pluginState || !pluginState.active || pluginState.from >= pluginState.to) {
          return PM.DecorationSet.empty;
        }
        return PM.DecorationSet.create(state.doc, [
          PM.Decoration.inline(pluginState.from, pluginState.to, {
            class: "time-query-range",
          }),
        ]);
      },
      handleKeyDown(view, event) {
        const pluginState = key.getState(view.state);
        if (!pluginState || !pluginState.active) return false;

        if (event.key === "ArrowDown" || event.key === "ArrowUp") {
          const direction = event.key === "ArrowDown" ? 1 : -1;
          const length = pluginState.items.length;
          const selectedIndex = length
            ? (pluginState.selectedIndex + direction + length) % length
            : 0;
          event.preventDefault();
          view.dispatch(
            view.state.tr.setMeta(key, { type: "setSelectedIndex", selectedIndex }),
          );
          return true;
        }

        if (event.key === "Enter" || event.key === "Tab") {
          if (!pluginState.items.length) return false;
          event.preventDefault();
          return selectItem(view, pluginState.items[pluginState.selectedIndex]);
        }

        if (event.key === "Escape") {
          event.preventDefault();
          view.dispatch(view.state.tr.setMeta(key, { type: "close" }));
          return true;
        }

        return false;
      },
    },
    view(view) {
      return createFloatingMenuView({
        className: "time-picker-menu hidden",
        key,
        render(menu, pluginState) {
          renderTimePickerMenu(menu, pluginState);
        },
        onMouseDown(event, pluginState) {
          const option = closestElement(event.target, "[data-time-picker-index]");
          if (!option) return false;
          event.preventDefault();
          const item = pluginState.items[Number(option.dataset.timePickerIndex)];
          return selectItem(view, item);
        },
        view,
      });
    },
  });
}

function createFloatingMenuView(config) {
  const menu = document.createElement("div");
  let frame = 0;
  menu.className = config.className;
  document.body.appendChild(menu);

  function position(pluginState) {
    window.cancelAnimationFrame(frame);
    frame = window.requestAnimationFrame(function () {
      if (!pluginState || !pluginState.active) return;
      try {
        const coords = config.view.coordsAtPos(pluginState.to);
        const rect = menu.getBoundingClientRect();
        const margin = 8;
        const maxLeft = Math.max(margin, window.innerWidth - rect.width - margin);
        const left = Math.min(Math.max(coords.left, margin), maxLeft);
        let top = coords.bottom + 6;
        if (top + rect.height > window.innerHeight - margin) {
          top = Math.max(margin, coords.top - rect.height - 6);
        }
        menu.style.left = left + "px";
        menu.style.top = top + "px";
      } catch (_) {
        menu.classList.add("hidden");
      }
    });
  }

  function update(view) {
    const pluginState = config.key.getState(view.state);
    if (!pluginState || !pluginState.active) {
      menu.classList.add("hidden");
      return;
    }
    config.render(menu, pluginState);
    menu.classList.remove("hidden");
    position(pluginState);
  }

  function onMouseDown(event) {
    const pluginState = config.key.getState(config.view.state);
    if (!pluginState || !pluginState.active) return;
    config.onMouseDown(event, pluginState);
  }

  function onReposition() {
    const pluginState = config.key.getState(config.view.state);
    position(pluginState);
  }

  menu.addEventListener("mousedown", onMouseDown);
  window.addEventListener("resize", onReposition);
  window.addEventListener("scroll", onReposition, true);
  update(config.view);

  return {
    update,
    destroy() {
      window.cancelAnimationFrame(frame);
      window.removeEventListener("resize", onReposition);
      window.removeEventListener("scroll", onReposition, true);
      menu.removeEventListener("mousedown", onMouseDown);
      menu.remove();
    },
  };
}

function memoSlashCommands() {
  return [
    { icon: "H1", label: "标题", detail: "插入 # 标题", keywords: "heading title", text: "# " },
    { icon: "TODO", label: "任务", detail: "插入待办项", keywords: "todo task", text: "- [ ] " },
    { icon: "UL", label: "无序列表", detail: "插入 - 列表", keywords: "list bullet", text: "- " },
    { icon: "OL", label: "有序列表", detail: "插入 1. 列表", keywords: "list ordered", text: "1. " },
    { icon: ">", label: "引用", detail: "插入引用块", keywords: "quote", text: "> " },
    { icon: "<>", label: "代码块", detail: "插入 fenced code", keywords: "code pre", text: "```\n\n```" },
    {
      icon: "TBL",
      label: "表格",
      detail: "插入 Markdown 表格",
      keywords: "table",
      text: "| 列 1 | 列 2 |\n| --- | --- |\n|  |  |",
    },
    {
      icon: "TIME",
      label: "时间",
      detail: "插入 :: 时间语法",
      keywords: "time date",
      text: function () {
        return "::" + formatMemoDateTime(new Date(), true) + " ";
      },
    },
    {
      action: "files",
      accept: "",
      icon: "FILE",
      label: "上传文件",
      detail: "选择文件并插入 Markdown 链接",
      keywords: "file upload attach",
      text: "",
    },
    {
      action: "files",
      accept: "image/*",
      icon: "IMG",
      label: "上传图片",
      detail: "选择图片并插入 Markdown 图片",
      keywords: "image upload picture",
      text: "",
    },
  ];
}

function renderSlashCommandMenu(menu, pluginState) {
  if (!pluginState.items.length) {
    menu.innerHTML = '<div class="slash-command-empty">没有匹配的命令</div>';
    return;
  }

  menu.innerHTML = pluginState.items
    .map(function (item, index) {
      return `
        <div class="slash-command-option ${index === pluginState.selectedIndex ? "active" : ""}" data-slash-command-index="${index}">
          <span class="slash-command-icon">${escapeHTML(item.icon)}</span>
          <span class="slash-command-copy">
            <span class="slash-command-label">${escapeHTML(item.label)}</span>
            <span class="slash-command-detail">${escapeHTML(item.detail)}</span>
          </span>
        </div>
      `;
    })
    .join("");
}

function memoReferenceItems(options, query) {
  const source = typeof options.memoItems === "function"
    ? options.memoItems()
    : options.memoItems;
  const sourceMemoId = String(options.sourceMemoId || "");
  const keyword = String(query || "")
    .trim()
    .replace(/^memo:/i, "")
    .toLowerCase();

  return (Array.isArray(source) ? source : [])
    .filter(function (memo) {
      return memo && memo.id && memo.id !== sourceMemoId;
    })
    .map(function (memo) {
      const title = memoTitle(memo);
      const detail = memoReferenceDetail(memo);
      return {
        alias: memoReferenceAlias(title),
        content: memo.content || "",
        detail,
        id: memo.id,
        label: title,
        pinned: Boolean(memo.pinned),
        time: new Date(memo.updatedAt || memo.createdAt || 0).getTime() || 0,
      };
    })
    .filter(function (item) {
      if (!keyword) return true;
      return [item.label, item.detail, item.id, item.content]
        .join("\n")
        .toLowerCase()
        .includes(keyword);
    })
    .sort(function (a, b) {
      if (a.pinned !== b.pinned) return a.pinned ? -1 : 1;
      return b.time - a.time;
    })
    .slice(0, 8);
}

function memoReferenceDetail(memo) {
  const visibility = VISIBILITY[memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
  const date = memo.updatedAt || memo.createdAt;
  const parts = [
    visibility.label,
    date ? formatShortDate(date) : "",
    "memo:" + memo.id,
  ].filter(Boolean);
  return parts.join(" · ");
}

function memoReferenceInsertText(item, embed) {
  const alias = memoReferenceAlias(item.alias || item.label || "");
  const target = "memo:" + item.id + (alias ? "|" + alias : "");
  return (embed ? "![[" : "[[") + target + "]]";
}

function renderMemoReferenceMenu(menu, pluginState) {
  if (!pluginState.items.length) {
    menu.innerHTML = '<div class="memo-ref-empty">没有匹配的 memo</div>';
    return;
  }

  menu.innerHTML = pluginState.items
    .map(function (item, index) {
      return `
        <div class="memo-ref-option ${index === pluginState.selectedIndex ? "active" : ""}" data-memo-ref-index="${index}">
          <span class="memo-ref-option-kind">${pluginState.embed ? "EMBED" : "LINK"}</span>
          <span class="memo-ref-option-copy">
            <span class="memo-ref-option-label">${escapeHTML(item.label)}</span>
            <span class="memo-ref-option-detail">${escapeHTML(item.detail)}</span>
          </span>
        </div>
      `;
    })
    .join("");
}

function memoTimeItems(query) {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const tomorrow = new Date(today);
  tomorrow.setDate(tomorrow.getDate() + 1);
  const value = String(query || "").trim();
  const items = [
    { label: "当前时间", detail: formatMemoDateTime(now, true), value: formatMemoDateTime(now, true) },
    { label: "今天", detail: formatMemoDate(today), value: formatMemoDate(today) },
    { label: "当前时分", detail: formatMemoTime(now, false), value: formatMemoTime(now, false) },
    { label: "明天", detail: formatMemoDate(tomorrow), value: formatMemoDate(tomorrow) },
  ];
  if (value) {
    items.unshift({ label: value, detail: "使用输入值", value });
  }
  return items;
}

function renderTimePickerMenu(menu, pluginState) {
  if (!pluginState.items.length) {
    menu.innerHTML = '<div class="time-picker-empty">输入时间，例如 23:21 或 2026-05-01</div>';
    return;
  }

  menu.innerHTML = pluginState.items
    .map(function (item, index) {
      return `
        <div class="time-picker-option ${index === pluginState.selectedIndex ? "active" : ""}" data-time-picker-index="${index}">
          <span class="time-picker-label">${escapeHTML(item.label)}</span>
          <span class="time-picker-detail">${escapeHTML(item.detail)}</span>
        </div>
      `;
    })
    .join("");
}

function formatMemoDate(date) {
  return (
    date.getFullYear() +
    "-" +
    padMemoNumber(date.getMonth() + 1) +
    "-" +
    padMemoNumber(date.getDate())
  );
}

function formatMemoTime(date, withSeconds) {
  const value = padMemoNumber(date.getHours()) + ":" + padMemoNumber(date.getMinutes());
  return withSeconds ? value + ":" + padMemoNumber(date.getSeconds()) : value;
}

function formatMemoDateTime(date, withSeconds) {
  return formatMemoDate(date) + " " + formatMemoTime(date, withSeconds);
}

function padMemoNumber(value) {
  return String(value).padStart(2, "0");
}

function insertFilesIntoEditor(editor, files) {
  filesToMarkdown(files).then(function (markdown) {
    if (!markdown) return;
    const current = editor.getText();
    insertPlainTextIntoEditor(editor, (current && !current.endsWith("\n") ? "\n" : "") + markdown);
    editor.focus();
  }).catch(function (err) {
    console.error(uploadErrorMessage(err));
  });
}

function requestFilesForEditor(editor, accept) {
  if (canUseNativeFilePicker()) {
    requestNativeFileForEditor(editor, accept);
    return;
  }

  const input = document.createElement("input");
  input.type = "file";
  input.multiple = true;
  if (accept) input.accept = accept;
  input.className = "hidden";
  document.body.appendChild(input);
  input.addEventListener("change", function () {
    insertFilesIntoEditor(editor, input.files);
    input.remove();
  });
  input.click();
}

function canUseNativeFilePicker() {
  return typeof invoke === "function";
}

function requestNativeFileForEditor(editor, accept) {
  const mode = String(accept || "").startsWith("image/") ? "image" : "";
  invoke("/api/file/select-data-url?accept=" + encodeURIComponent(mode), { method: "GET" }).then(
    function (resp) {
      if (!resp || resp.code !== 0 || !resp.data || !resp.data.file) return;
      droppedFilesToMarkdown([resp.data.file]).then(function (markdown) {
        if (!markdown) return;
        const current = editor.getText();
        insertPlainTextIntoEditor(editor, (current && !current.endsWith("\n") ? "\n" : "") + markdown);
        editor.focus();
      }).catch(function (err) {
        console.error(uploadErrorMessage(err));
      });
    },
    function () {},
  );
}

function insertPlainTextIntoEditor(editor, text) {
  const view = editor.view;
  view.dispatch(view.state.tr.insertText(text).scrollIntoView());
}

function insertMarkdownLinkIntoEditor(editor, url) {
  const view = editor && editor.view;
  const PM = window.ProsemirrorMod;
  if (!view || !view.state || !PM || !PM.TextSelection) return false;

  const { from, to, empty } = view.state.selection;
  const selected = empty ? "" : view.state.doc.textBetween(from, to, "\n");
  const label = markdownLinkLabel(selected || url);
  const text = markdownLinkText(label, url);
  const transaction = view.state.tr.insertText(text, from, to);
  transaction.setSelection(PM.TextSelection.create(transaction.doc, from + 1, from + 1 + label.length));
  view.dispatch(transaction.scrollIntoView());
  return true;
}

function clipboardPlainURL(clipboardData) {
  if (!clipboardData || typeof clipboardData.getData !== "function") return "";
  return singlePlainURL(clipboardData.getData("text/plain"));
}

function singlePlainURL(value) {
  const text = String(value || "").trim();
  if (!text || /\s/.test(text) || !/^https?:\/\//i.test(text)) return "";

  try {
    const url = new window.URL(text);
    return url.protocol === "http:" || url.protocol === "https:" ? text : "";
  } catch (_) {
    return "";
  }
}

function markdownLinkText(label, url) {
  return "[" + label + "](" + markdownUrl(url) + ")";
}

function markdownLinkLabel(value) {
  return markdownLabel(String(value || "").replace(/\s+/g, " ").trim());
}

function filesToMarkdown(files) {
  const list = Array.from(files || []);
  return Promise.all(
    list.map(function (file) {
      return readFileAsDataURL(file).then(function (url) {
        return fileInfoToMarkdownAsync({
          name: file.name || "file",
          type: file.type || "",
          url,
        });
      });
    }),
  ).then(function (items) {
    return items.join("\n");
  });
}

function droppedFilesToMarkdown(files) {
  const list = Array.from(files || []).filter(function (file) {
    return file && (file.dataURL || file.url || file.path);
  });
  return Promise.all(
    list
      .map(function (file) {
        return fileInfoToMarkdownAsync({
          name: file.name || "file",
          type: file.type || "",
          url: file.dataURL || file.url || file.path,
        });
      })
  ).then(function (items) {
    return items.join("\n");
  });
}

function fileInfoToMarkdownAsync(file) {
  return fileInfoToUploadURL(file).then(function (uploaded) {
    return fileInfoToMarkdown({
      name: uploaded.name || file.name,
      type: uploaded.type || file.type,
      url: uploaded.ref || uploaded.url || file.url,
    });
  });
}

function fileInfoToUploadURL(file) {
  return loadCloudStorageConfig().then(function (settings) {
    cloudStorageSettingsCache = normalizeCloudStorageSettings(settings);
    const storage = activeCloudStorageConfig(settings);
    if (!storage || !storage.enabled) {
      return file;
    }
    if (typeof invoke !== "function") {
      throw new Error("当前环境不支持云存储上传");
    }
    const missing = missingCloudStorageFields(storage);
    if (missing.length) {
      throw new Error("云存储配置缺少: " + missing.join(", "));
    }
    const contentBase64 = dataURLToBase64(file.url || file.dataURL || "");
    if (!contentBase64) {
      throw new Error("无法读取文件内容");
    }
    return invoke("/api/oss/upload", {
      args: {
        content_base64: contentBase64,
        name: file.name || "file",
        storageId: storage.id,
        type: file.type || "",
      },
    }).then(function (resp) {
      if (!resp || resp.code !== 0 || !resp.data) {
        throw new Error((resp && resp.msg) || "上传失败");
      }
      const storageId = resp.data.storageId || storage.id;
      const ref = markdownUrl(resp.data.ref || assetReference(storageId, resp.data.key || ""));
      return {
        key: resp.data.key || "",
        name: resp.data.name || file.name,
        publicUrl: resp.data.url || "",
        ref,
        storageId,
        type: resp.data.type || file.type,
        url: ref || resp.data.url || file.url,
      };
    });
  });
}

function refreshCloudStorageSettings() {
  return loadCloudStorageConfig().then(function (settings) {
    cloudStorageSettingsCache = normalizeCloudStorageSettings(settings);
    return cloudStorageSettingsCache;
  });
}

function loadCloudStorageConfig() {
  if (typeof invoke === "function") {
    return invoke("/api/settings/cloud-storage", { method: "GET" }).then(
      function (resp) {
        if (resp && resp.code === 0 && resp.data && resp.data.found && resp.data.config) {
          return normalizeCloudStorageSettings(resp.data.config);
        }
        if (resp && resp.code === 0) {
          return normalizeCloudStorageSettings(loadLocalCloudStorageConfig());
        }
        throw new Error((resp && resp.msg) || "读取云存储配置失败");
      },
      function (err) {
        const localConfig = loadLocalCloudStorageConfig();
        if (localConfig) return normalizeCloudStorageSettings(localConfig);
        throw err || new Error("读取云存储配置失败");
      },
    );
  }
  return Promise.resolve(normalizeCloudStorageSettings(loadLocalCloudStorageConfig()));
}

function loadLocalCloudStorageConfig() {
  try {
    const saved = JSON.parse(localStorage.getItem(CLOUD_STORAGE_KEY) || "null");
    return saved && typeof saved === "object" ? saved : null;
  } catch (_) {
    return null;
  }
}

function currentCloudStorageSettings() {
  if (!cloudStorageSettingsCache) {
    cloudStorageSettingsCache = normalizeCloudStorageSettings(loadLocalCloudStorageConfig());
  }
  return cloudStorageSettingsCache;
}

function normalizeCloudStorageSettings(value) {
  const settings = value && typeof value === "object" ? value : {};
  let storages = [];
  if (Array.isArray(settings.storages)) {
    storages = settings.storages.map(normalizeCloudStorageProfile).filter(function (storage) {
      return storage.id;
    });
  } else if (hasLegacyCloudStorageProfile(settings)) {
    storages = [normalizeCloudStorageProfile(Object.assign({ id: "default", name: "默认存储" }, settings))];
  }

  const seen = Object.create(null);
  storages = storages.map(function (storage, index) {
    let id = sanitizeStorageId(storage.id || storage.name || storage.provider || storage.bucket || "storage-" + (index + 1));
    if (!id) id = "storage-" + (index + 1);
    seen[id] = (seen[id] || 0) + 1;
    if (seen[id] > 1) id = id + "-" + seen[id];
    return Object.assign({}, storage, {
      id,
      name: storage.name || storage.bucket || storage.provider || id,
    });
  });

  let activeStorageId = sanitizeStorageId(settings.activeStorageId || "");
  if (!storages.some(function (storage) { return storage.id === activeStorageId; })) {
    const enabled = storages.find(function (storage) { return storage.enabled; });
    activeStorageId = enabled ? enabled.id : (storages[0] && storages[0].id) || "";
  }

  return {
    activeStorageId,
    defaultsInitialized: Boolean(settings.defaultsInitialized),
    storages,
  };
}

function normalizeCloudStorageProfile(profile) {
  const value = profile && typeof profile === "object" ? profile : {};
  return {
    accessKeyId: String(value.accessKeyId || "").trim(),
    bucket: String(value.bucket || "").trim(),
    enabled: Boolean(value.enabled),
    endpoint: String(value.endpoint || "").trim(),
    forcePathStyle: value.forcePathStyle !== false,
    id: sanitizeStorageId(value.id || ""),
    local: normalizeLocalStorageSettings(value.local),
    name: String(value.name || "").trim(),
    pathPrefix: String(value.pathPrefix || "").trim(),
    provider: String(value.provider || "s3").trim() || "s3",
    publicBaseUrl: String(value.publicBaseUrl || "").trim(),
    region: String(value.region || "").trim(),
    secretAccessKey: String(value.secretAccessKey || ""),
    sessionToken: String(value.sessionToken || "").trim(),
    useSSL: value.useSSL !== false,
  };
}

function hasLegacyCloudStorageProfile(value) {
  return Boolean(value && typeof value === "object" && (
    value.enabled ||
    value.endpoint ||
    value.local ||
    value.bucket ||
    value.accessKeyId ||
    value.secretAccessKey ||
    value.sessionToken ||
    value.region ||
    value.pathPrefix ||
    value.publicBaseUrl
  ));
}

function activeCloudStorageConfig(settings) {
  const normalized = normalizeCloudStorageSettings(settings);
  if (!normalized.activeStorageId) return null;
  return normalized.storages.find(function (storage) {
    return storage.id === normalized.activeStorageId;
  }) || null;
}

function cloudStorageById(storageId) {
  const id = sanitizeStorageId(storageId);
  if (!id) return null;
  return currentCloudStorageSettings().storages.find(function (storage) {
    return storage.id === id;
  }) || null;
}

function sanitizeStorageId(value) {
  return String(value || "")
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_-]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

function assetReference(storageId, key) {
  const id = sanitizeStorageId(storageId);
  const cleanKey = String(key || "").replace(/^\/+/, "");
  if (!id || !cleanKey) return "";
  return "@assets/" + id + "/" + cleanKey;
}

function parseAssetReference(value) {
  const match = String(value || "").trim().match(/^@assets\/([a-z0-9_-]+)\/(.+)$/i);
  if (!match) return null;
  return {
    key: decodeAssetReferenceKey(match[2]),
    storageId: sanitizeStorageId(match[1]),
  };
}

function decodeAssetReferenceKey(value) {
  return String(value || "").replace(/%29/gi, ")").replace(/%28/gi, "(");
}

function resolveAssetUrl(value) {
  const asset = parseAssetReference(value);
  if (!asset) return String(value || "").trim();
  const storage = cloudStorageById(asset.storageId);
  if (!storage) return "";
  return publicCloudStorageObjectUrl(storage, asset.key);
}

function publicCloudStorageObjectUrl(storage, key) {
  const encodedKey = encodeObjectKey(key);
  if (!encodedKey) return "";
  if (isLocalCloudStorage(storage)) {
    return "/api/oss/assets?storageId=" + encodeURIComponent(sanitizeStorageId(storage.id)) + "&path=" + encodeURIComponent(String(key || "").replace(/^\/+/, ""));
  }
  const publicBaseUrl = String(storage.publicBaseUrl || "").trim().replace(/\/+$/, "");
  if (publicBaseUrl) return publicBaseUrl + "/" + encodedKey;

  const endpoint = normalizeOSSEndpoint(storage.endpoint, storage.useSSL);
  if (!endpoint) return "";
  if (storage.forcePathStyle) {
    return endpoint.replace(/\/+$/, "") + "/" + encodeURIComponent(String(storage.bucket || "").replace(/^\/+|\/+$/g, "")) + "/" + encodedKey;
  }
  try {
    const url = new URL(endpoint);
    url.hostname = String(storage.bucket || "").replace(/\.+$/g, "") + "." + url.hostname;
    url.pathname = "/" + encodedKey;
    url.search = "";
    url.hash = "";
    return url.toString();
  } catch (_) {
    return endpoint.replace(/\/+$/, "") + "/" + encodedKey;
  }
}

function normalizeOSSEndpoint(endpoint, useSSL) {
  const value = String(endpoint || "").trim().replace(/\/+$/, "");
  if (!value) return "";
  if (/^https?:\/\//i.test(value)) return value;
  return (useSSL === false ? "http://" : "https://") + value;
}

function encodeObjectKey(key) {
  return String(key || "")
    .replace(/^\/+/, "")
    .split("/")
    .map(function (part) {
      return encodeURIComponent(part);
    })
    .join("/");
}

function missingCloudStorageFields(config) {
  const missing = [];
  if (!String(config.endpoint || "").trim() && !(isLocalCloudStorage(config) && normalizeLocalStorageSettings(config.local))) missing.push(isLocalCloudStorage(config) ? "本地根目录" : "Endpoint");
  if (!String(config.bucket || "").trim()) missing.push("Bucket");
  if (!isLocalCloudStorage(config)) {
    if (!String(config.accessKeyId || "").trim()) missing.push("Access Key ID");
    if (!String(config.secretAccessKey || "").trim()) missing.push("Secret Access Key");
  }
  return missing;
}

function isLocalCloudStorage(config) {
  const provider = String((config && config.provider) || "").trim().toLowerCase();
  return provider === "local" || provider === "local-oss";
}

function normalizeLocalStorageSettings(value) {
  const raw = value && typeof value === "object" ? value : {};
  let root = String(raw.root || "").trim();
  let rootMode = String(raw.rootMode || "").trim().toLowerCase();
  if (!rootMode) {
    if (!root) return null;
    rootMode = root.charAt(0) === "/" || root.indexOf(":\\") === 1 || root.indexOf("~/") === 0 ? "absolute" : "vault";
  }
  if (rootMode === "vault") {
    root = root.replace(/\\/g, "/").replace(/^\/+|\/+$/g, "") || "storage";
    return { root, rootMode: "vault" };
  }
  if (rootMode === "absolute" && root) {
    return { root, rootMode: "absolute" };
  }
  return null;
}

function dataURLToBase64(value) {
  const text = String(value || "");
  const comma = text.indexOf(",");
  if (text.startsWith("data:") && comma >= 0) return text.slice(comma + 1);
  return text;
}

function uploadErrorMessage(err) {
  return "上传失败: " + ((err && err.message) || err || "未知错误");
}

function fileInfoToMarkdown(file) {
  const name = markdownLabel(file.name || "file");
  const url = markdownUrl(file.url || "");
  if (isImageFile(file)) {
    return "![" + name + "](" + url + ")";
  }
  return "[" + name + "](" + url + ")";
}

function isImageFile(file) {
  const type = String((file && file.type) || "");
  if (type.startsWith("image/")) return true;
  return /\.(avif|bmp|gif|jpe?g|png|svg|webp)$/i.test(String((file && file.name) || ""));
}

function readFileAsDataURL(file) {
  return new Promise(function (resolve, reject) {
    const reader = new FileReader();
    reader.onload = function () {
      resolve(String(reader.result || ""));
    };
    reader.onerror = function () {
      reject(reader.error || new Error("failed to read file"));
    };
    reader.readAsDataURL(file);
  });
}

function markdownLabel(value) {
  return String(value || "")
    .replace(/\\/g, "\\\\")
    .replace(/\[/g, "\\[")
    .replace(/\]/g, "\\]");
}

function markdownUrl(value) {
  return String(value || "").replace(/\)/g, "%29");
}

function createFallbackEditor(host, options) {
  const textarea = document.createElement("textarea");
  textarea.className = "memo-fallback-editor";
  textarea.placeholder = options.placeholder || "";
  textarea.value = options.value || "";
  host.appendChild(textarea);
  textarea.addEventListener("input", () => {
    if (options.onChange) options.onChange(textarea.value);
  });
  textarea.addEventListener("keydown", (event) => {
    if ((event.metaKey || event.ctrlKey) && event.key === "Enter" && options.onSubmit) {
      event.preventDefault();
      options.onSubmit();
    }
  });
  function onPaste(event) {
    const url = clipboardPlainURL(event.clipboardData);
    if (!url) return;
    event.preventDefault();
    event.stopPropagation();
    insertMarkdownLinkIntoTextarea(textarea, url, options.onChange);
  }
  textarea.addEventListener("paste", onPaste);
  return {
    destroy() {
      textarea.removeEventListener("paste", onPaste);
      textarea.remove();
    },
    focus() {
      textarea.focus();
    },
    getText() {
      return textarea.value;
    },
    insertBlock(text) {
      textarea.value += `${textarea.value && !textarea.value.endsWith("\n") ? "\n" : ""}${text}`;
      if (options.onChange) options.onChange(textarea.value);
    },
    insertText(text) {
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;
      textarea.value = textarea.value.slice(0, start) + text + textarea.value.slice(end);
      textarea.selectionStart = textarea.selectionEnd = start + text.length;
      if (options.onChange) options.onChange(textarea.value);
    },
    insertFiles(files) {
      filesToMarkdown(files).then((markdown) => {
        if (!markdown) return;
        textarea.value += `${textarea.value && !textarea.value.endsWith("\n") ? "\n" : ""}${markdown}`;
        if (options.onChange) options.onChange(textarea.value);
      }).catch((err) => {
        console.error(uploadErrorMessage(err));
      });
    },
    setText(value) {
      textarea.value = value || "";
      if (options.onChange) options.onChange(textarea.value);
    },
    wrap(prefix, suffix, placeholder) {
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;
      const selected = textarea.value.slice(start, end) || placeholder;
      const text = `${prefix}${selected}${suffix}`;
      textarea.value = textarea.value.slice(0, start) + text + textarea.value.slice(end);
      textarea.selectionStart = textarea.selectionEnd = start + prefix.length + selected.length;
      if (options.onChange) options.onChange(textarea.value);
    },
  };
}

function insertMarkdownLinkIntoTextarea(textarea, url, onChange) {
  const start = textarea.selectionStart;
  const end = textarea.selectionEnd;
  const selected = textarea.value.slice(start, end);
  const label = markdownLinkLabel(selected || url);
  const text = markdownLinkText(label, url);
  textarea.value = textarea.value.slice(0, start) + text + textarea.value.slice(end);
  textarea.selectionStart = start + 1;
  textarea.selectionEnd = start + 1 + label.length;
  if (onChange) onChange(textarea.value);
}

function loadMemos() {
  if (typeof invoke === "function") return [];
  const saved = loadJSON(MEMOS_STORAGE_KEY, null);
  if (Array.isArray(saved)) return saved;
  const memos = seedMemos();
  saveMemos(memos);
  return memos;
}

function loadProjects() {
  if (typeof invoke === "function") return [];
  const saved = loadJSON(PROJECTS_STORAGE_KEY, null);
  return Array.isArray(saved) ? saved.map(normalizeProjectPayload).filter(Boolean) : [];
}

function saveProjects(projects) {
  if (typeof invoke === "function") return;
  localStorage.setItem(PROJECTS_STORAGE_KEY, JSON.stringify(projects));
}

function loadProjectsFromVault() {
  if (typeof invoke !== "function") {
    return Promise.resolve({ activeProjectId: "", projects: loadProjects() });
  }
  return invoke("/api/projects", { method: "GET" }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "读取 project 失败");
    }
    const data = resp.data || {};
    return {
      activeProjectId: normalizeProjectID(data.activeProjectId),
      projects: Array.isArray(data.projects) ? data.projects : [],
    };
  });
}

function createProjectInVault(name, color) {
  if (typeof invoke !== "function") {
    const now = new Date().toISOString();
    return Promise.resolve({
      archived: false,
      color: color || "#2563eb",
      createdAt: now,
      id: "project_" + Date.now().toString(36),
      name,
      sortOrder: 0,
      updatedAt: now,
    });
  }
  return invoke("/api/projects/create", {
    method: "POST",
    args: {
      color: color || "",
      name,
    },
  }).then(function (resp) {
    if (!resp || resp.code !== 0 || !resp.data || !resp.data.project) {
      throw new Error((resp && resp.msg) || "创建 project 失败");
    }
    return resp.data.project;
  });
}

function saveMemos(memos) {
  if (typeof invoke === "function") return;
  localStorage.setItem(MEMOS_STORAGE_KEY, JSON.stringify(memos));
}

function loadMemosFromVault() {
  if (typeof invoke !== "function") {
    return Promise.resolve(loadMemos());
  }
  return invoke("/api/memos", { method: "GET" }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "读取 memo 失败");
    }
    const data = resp.data || {};
    return Array.isArray(data.memos) ? data.memos : [];
  });
}

function createMemoInVault(content, visibility, projectId) {
  if (typeof invoke !== "function") {
    const now = new Date().toISOString();
    return Promise.resolve({
      archived: false,
      content,
      createdAt: now,
      id: createId(),
      pinned: false,
      projectId: normalizeProjectID(projectId),
      updatedAt: "",
      visibility,
    });
  }
  return invoke("/api/memos/create", {
    method: "POST",
    args: {
      content,
      projectId: normalizeProjectID(projectId),
      visibility,
    },
  }).then(function (resp) {
    if (!resp || resp.code !== 0 || !resp.data || !resp.data.memo) {
      throw new Error((resp && resp.msg) || "发布失败");
    }
    return resp.data.memo;
  });
}

function updateMemoInVault(id, patch) {
  if (typeof invoke !== "function") {
    return Promise.resolve(Object.assign({ id }, patch));
  }
  const args = { id };
  if (Object.prototype.hasOwnProperty.call(patch, "content")) args.content = patch.content;
  if (Object.prototype.hasOwnProperty.call(patch, "projectId")) args.projectId = normalizeProjectID(patch.projectId);
  if (Object.prototype.hasOwnProperty.call(patch, "visibility")) args.visibility = patch.visibility;
  if (Object.prototype.hasOwnProperty.call(patch, "pinned")) args.pinned = patch.pinned;
  if (Object.prototype.hasOwnProperty.call(patch, "archived")) args.archived = patch.archived;
  return invoke("/api/memos/update", {
    method: "POST",
    args,
  }).then(function (resp) {
    if (!resp || resp.code !== 0 || !resp.data || !resp.data.memo) {
      throw new Error((resp && resp.msg) || "保存失败");
    }
    return resp.data.memo;
  });
}

function deleteMemoInVault(id, options) {
  if (typeof invoke !== "function") {
    return Promise.resolve({ success: true });
  }
  const args = { id };
  if (options && Object.prototype.hasOwnProperty.call(options, "cleanupAssets")) {
    args.cleanupAssets = Boolean(options.cleanupAssets);
  }
  return invoke("/api/memos/delete", {
    method: "POST",
    args,
  }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "删除失败");
    }
    return resp.data || { success: true };
  });
}

function errorMessage(err) {
  return err && err.message ? err.message : String(err || "unknown error");
}

function loadJSON(key, fallback) {
  try {
    const raw = localStorage.getItem(key);
    return raw ? JSON.parse(raw) : fallback;
  } catch (_) {
    return fallback;
  }
}

function seedMemos() {
  const now = Date.now();
  return [
    {
      archived: false,
      content: "把 #velo 的桌面示例做成 memo 工作台。\n- [x] 左侧过滤\n- [ ] ProseMirror mini editor\n\n本地优先，适合快速捕捉。",
      createdAt: new Date(now - 1000 * 60 * 35).toISOString(),
      id: createId(),
      pinned: true,
      updatedAt: "",
      visibility: "PRIVATE",
    },
    {
      archived: false,
      content: "#idea Memos 风格的首页应该先看到编辑器，再看到时间线。\n\n支持 #inbox、置顶、归档和全文搜索。",
      createdAt: new Date(now - 1000 * 60 * 60 * 5).toISOString(),
      id: createId(),
      pinned: false,
      updatedAt: "",
      visibility: "PROTECTED",
    },
    {
      archived: false,
      content: "发布前检查：\n1. mini editor 可输入\n2. 标签可筛选\n3. 任务可以勾选\n\n[usememos](https://github.com/usememos/memos)",
      createdAt: new Date(now - 1000 * 60 * 60 * 24).toISOString(),
      id: createId(),
      pinned: false,
      updatedAt: "",
      visibility: "PUBLIC",
    },
  ];
}

function collectTags(memos) {
  const counts = new Map();
  memos.forEach((memo) => {
    extractTags(memo.content).forEach((tag) => {
      counts.set(tag, (counts.get(tag) || 0) + 1);
    });
  });
  return Array.from(counts.entries()).sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
}

function collectTodos(memos) {
  const todos = [];
  memos.forEach((memo) => {
    const lines = String(memo.content || "").replace(/\r\n/g, "\n").split("\n");
    lines.forEach((line, lineIndex) => {
      const task = parseTaskLine(line);
      if (!task) return;
      todos.push({
        checked: task.checked,
        id: `${memo.id}:${lineIndex}`,
        lineIndex,
        memo,
        memoId: memo.id,
        projectId: memo.projectId || "",
        sourceText: memoSourceText(lines, lineIndex),
        text: task.text,
      });
    });
  });
  return todos;
}

function getTodoStats(memos) {
  const todos = collectTodos(memos);
  const done = todos.filter((todo) => todo.checked).length;
  return {
    done,
    open: todos.length - done,
    total: todos.length,
  };
}

function collectLinks(memos) {
  return collectMemoReferences(memos).filter((reference) => reference.type === "link");
}

function collectResources(memos) {
  return collectMemoReferences(memos).filter((reference) => reference.type === "file" || reference.type === "image");
}

function getResourceStats(memos) {
  const resources = collectResources(memos);
  return {
    files: resources.filter((resource) => resource.type === "file").length,
    images: resources.filter((resource) => resource.type === "image").length,
    total: resources.length,
  };
}

function collectMemoReferences(memos) {
  const references = [];
  memos.forEach((memo) => {
    const lines = String(memo.content || "").replace(/\r\n/g, "\n").split("\n");
    lines.forEach((line, lineIndex) => {
      references.push(...collectLineReferences(memo, lines, line, lineIndex));
    });
  });
  return references;
}

function collectLineReferences(memo, lines, line, lineIndex) {
  const references = [];
  const markdownRanges = [];
  const markdownLinkRegex = /(!?)\[([^\]]*)\]\(([^)]+)\)/g;
  let match;

  while ((match = markdownLinkRegex.exec(line))) {
    markdownRanges.push([match.index, match.index + match[0].length]);

    const url = markdownReferenceURL(match[3]);
    if (!url) continue;

    const label = (match[2] || "").trim();
    const type = markdownReferenceType(match[1], label, url);
    if (!type) continue;

    references.push(
      referenceView(memo, lines, lineIndex, references.length, {
        label: referenceLabel(type, label, url),
        syntax: match[1] === "!" ? "image" : "markdown",
        type,
        url,
      }),
    );
  }

  const rawURLRegex = /\bhttps?:\/\/[^\s<>"`]+/gi;
  while ((match = rawURLRegex.exec(line))) {
    if (rangeIncludes(markdownRanges, match.index)) continue;

    const url = cleanRawURL(match[0]);
    if (!url) continue;

    const type = rawURLReferenceType(url);
    references.push(
      referenceView(memo, lines, lineIndex, references.length, {
        label: referenceLabel(type, "", url),
        syntax: "raw",
        type,
        url,
      }),
    );
  }

  return references;
}

function referenceView(memo, lines, lineIndex, index, reference) {
  return {
    id: `${memo.id}:${lineIndex}:${index}:${reference.type}`,
    label: reference.label,
    lineIndex,
    memo,
    memoId: memo.id,
    sourceText: memoSourceText(lines, lineIndex, "仅包含资源的 memo"),
    syntax: reference.syntax,
    type: reference.type,
    url: reference.url,
  };
}

function markdownReferenceURL(value) {
  return String(value || "").trim();
}

function markdownReferenceType(marker, label, url) {
  if (marker === "!" || isImageAttachment(label, url)) return "image";
  if (isFileAttachment(label, url)) return "file";
  if (isHyperlinkURL(url)) return "link";
  return "";
}

function rawURLReferenceType(url) {
  if (isImageAttachment("", url)) return "image";
  if (isFileAttachment("", url)) return "file";
  return "link";
}

function referenceLabel(type, label, url) {
  const text = cleanMemoLine(label || "");
  if (text) return compactText(text, 120);
  if (type === "file" || type === "image") return fileDisplayName("", url);
  return linkDisplayName(url);
}

function linkDisplayName(url) {
  try {
    const parsed = new URL(url, window.location.origin);
    return parsed.pathname && parsed.pathname !== "/" ? `${parsed.host}${parsed.pathname}` : parsed.host || url;
  } catch (_) {
    return url;
  }
}

function cleanRawURL(value) {
  let url = String(value || "").trim();
  while (/[),.;:!?，。；：！？]$/.test(url)) {
    url = url.slice(0, -1);
  }
  return url;
}

function rangeIncludes(ranges, index) {
  return ranges.some(([start, end]) => index >= start && index < end);
}

function isHyperlinkURL(url) {
  return /^(https?:|mailto:)/i.test(String(url || "")) || /^\/(?!\/)/.test(String(url || ""));
}

function sortMemoReference(a, b, sortDesc = true) {
  const created = new Date(a.memo.createdAt).getTime() - new Date(b.memo.createdAt).getTime();
  if (created !== 0) return sortDesc ? -created : created;
  if (a.lineIndex !== b.lineIndex) return a.lineIndex - b.lineIndex;
  return a.id.localeCompare(b.id);
}

function parseTaskLine(line) {
  const match = String(line || "").match(TASK_LINE_REGEX);
  if (!match) return null;
  return {
    checked: match[2].toLowerCase() === "x",
    text: match[4].trim(),
  };
}

function memoSourceText(lines, lineIndex, fallbackText = "仅包含任务的 memo") {
  const before = lines
    .slice(0, lineIndex)
    .reverse()
    .find((line) => line.trim() && !parseTaskLine(line));
  const fallback = lines.find((line) => line.trim() && !parseTaskLine(line));
  return compactText(cleanMemoLine(before || fallback || "") || fallbackText, 84);
}

function cleanMemoLine(line) {
  return String(line || "")
    .replace(/^#{1,6}\s+/, "")
    .replace(/^>\s?/, "")
    .replace(/^\s*[-*+]\s+/, "")
    .replace(/^\s*\d+\.\s+/, "")
    .replace(/!\[\[([^\]]+)\]\]/g, "$1")
    .replace(/\[\[([^\]]+)\]\]/g, "$1")
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/[`*_~]/g, "")
    .trim();
}

function extractTags(text) {
  const tags = new Set();
  const matches = String(text || "").match(/(^|\s)#([\w\u4e00-\u9fa5-]+)/g) || [];
  matches.forEach((match) => tags.add(match.trim().slice(1)));
  return Array.from(tags);
}

function buildMemoReferenceIndex(memos) {
  const list = Array.isArray(memos) ? memos : [];
  const incoming = new Map();
  const memoById = new Map();
  const memoByTitleKey = new Map();
  const outgoing = new Map();
  const unresolved = [];

  list.forEach(function (memo) {
    if (!memo || !memo.id) return;
    memoById.set(memo.id, memo);
  });

  list.forEach(function (memo) {
    if (!memo || !memo.id) return;
    const key = memoTitleKey(memoTitle(memo));
    if (key && !memoByTitleKey.has(key)) memoByTitleKey.set(key, memo.id);
  });

  list.forEach(function (memo) {
    if (!memo || !memo.id) return;
    const refs = parseMemoReferences(memo.content).map(function (ref) {
      const target = resolveMemoReferenceTarget(ref, { index: { memoById, memoByTitleKey } });
      const edge = {
        ...ref,
        sourceId: memo.id,
        targetId: target ? target.id : "",
      };
      if (edge.targetId) {
        if (!incoming.has(edge.targetId)) incoming.set(edge.targetId, []);
        incoming.get(edge.targetId).push(edge);
      } else {
        unresolved.push(edge);
      }
      return edge;
    });
    outgoing.set(memo.id, refs);
  });

  return {
    incoming,
    memoById,
    memoByTitleKey,
    outgoing,
    unresolved,
  };
}

function parseMemoReferences(content) {
  const refs = [];
  const lines = memoLines(content);
  let inCode = false;

  lines.forEach(function (line, index) {
    if (isMemoFenceLine(line)) {
      inCode = !inCode;
      return;
    }
    if (inCode) return;

    const pattern = /(!?)\[\[([^\]\n]+)\]\]/g;
    let match = pattern.exec(line);
    while (match) {
      const ref = parseMemoReferenceInner(match[2], match[1] === "!");
      if (ref) {
        refs.push({
          ...ref,
          line: index + 1,
          raw: match[0],
        });
      }
      match = pattern.exec(line);
    }
  });

  return refs;
}

function parseStandaloneMemoEmbed(line) {
  const match = String(line || "").match(/^\s*!\[\[([^\]\n]+)\]\]\s*$/);
  return match ? parseMemoReferenceInner(match[1], true) : null;
}

function parseMemoReferenceInner(value, embed) {
  const raw = String(value || "").trim();
  if (!raw) return null;

  const aliasIndex = raw.indexOf("|");
  const targetExpr = (aliasIndex >= 0 ? raw.slice(0, aliasIndex) : raw).trim();
  const alias = aliasIndex >= 0 ? raw.slice(aliasIndex + 1).trim() : "";
  const selectorIndex = targetExpr.indexOf("#");
  const target = (selectorIndex >= 0 ? targetExpr.slice(0, selectorIndex) : targetExpr).trim();
  const selectorRaw = selectorIndex >= 0 ? targetExpr.slice(selectorIndex + 1).trim() : "";

  if (!target) return null;

  return {
    alias,
    embed: Boolean(embed),
    selector: parseMemoSelector(selectorRaw),
    target,
  };
}

function parseMemoSelector(value) {
  const raw = String(value || "").trim();
  if (!raw) return null;

  const line = raw.match(/^L([1-9]\d*)(?:-(?:L)?([1-9]\d*))?$/i);
  if (line) {
    const start = Number(line[1]);
    const end = Number(line[2] || line[1]);
    return {
      end,
      raw,
      start,
      type: end >= start ? "line" : "invalid",
    };
  }

  return {
    raw,
    type: "unsupported",
  };
}

function renderMemoMarkdown(content, context = {}) {
  const lines = memoLines(content);
  let html = "";
  let inCode = false;

  lines.forEach((line, index) => {
    const lineNumber = index + 1;
    if (line.trim().startsWith("```")) {
      const fenceClass = inCode ? "is-code-end" : "is-code-start";
      html += memoLineTemplate(
        lineNumber,
        `<pre class="memo-code-line is-fence"><code>${escapeHTML(line)}</code></pre>`,
        `is-code is-code-fence ${fenceClass}`,
      );
      inCode = !inCode;
      return;
    }

    if (inCode) {
      html += memoLineTemplate(
        lineNumber,
        `<pre class="memo-code-line"><code>${escapeHTML(line)}</code></pre>`,
        "is-code is-code-body",
      );
      return;
    }

    if (!line.trim()) {
      html += memoLineTemplate(lineNumber, '<div class="memo-markdown-gap"></div>', "is-empty");
      return;
    }

    const memoEmbed = parseStandaloneMemoEmbed(line);
    if (memoEmbed) {
      html += memoLineTemplate(lineNumber, renderMemoEmbedCard(memoEmbed, context), "is-embed");
      return;
    }

    const standaloneResource = parseStandaloneMarkdownResource(line);
    if (standaloneResource) {
      html += memoLineTemplate(
        lineNumber,
        standaloneResource.type === "image"
          ? renderMemoImageBlock(standaloneResource)
          : renderMemoFileBlock(standaloneResource),
        "is-resource",
      );
      return;
    }

    const taskMatch = line.match(TASK_LINE_REGEX);
    if (taskMatch) {
      const checked = taskMatch[2].toLowerCase() === "x";
      html += memoLineTemplate(
        lineNumber,
        `
        <label class="memo-task-line">
          <input type="checkbox" ${context.readonly ? "disabled" : `data-task-line="${index}"`} ${checked ? "checked" : ""} />
          <span>${inlineMarkdown(taskMatch[4], context)}</span>
        </label>
      `,
        "is-task",
      );
      return;
    }

    const unorderedMatch = line.match(/^(\s*)[-*+]\s+(.*)$/);
    if (unorderedMatch) {
      html += memoLineTemplate(
        lineNumber,
        `<div class="memo-line-list-item is-ul" style="--memo-list-indent: ${listIndentWidth(unorderedMatch[1])}px"><span class="memo-line-list-content">${inlineMarkdown(unorderedMatch[2], context)}</span></div>`,
        "is-list",
      );
      return;
    }

    const orderedMatch = line.match(/^(\s*)(\d+\.)\s+(.*)$/);
    if (orderedMatch) {
      html += memoLineTemplate(
        lineNumber,
        `<div class="memo-line-list-item is-ol" style="--memo-list-indent: ${listIndentWidth(orderedMatch[1])}px"><span class="memo-line-list-marker">${escapeHTML(orderedMatch[2])}</span><span class="memo-line-list-content">${inlineMarkdown(orderedMatch[3], context)}</span></div>`,
        "is-list",
      );
      return;
    }

    const heading = parseMemoHeadingLine(line);
    if (heading) {
      html += memoLineTemplate(
        lineNumber,
        `<h${heading.level} class="memo-heading memo-heading-${heading.level}">${inlineMarkdown(heading.text, context)}</h${heading.level}>`,
        `is-heading is-heading-${heading.level}`,
      );
      return;
    }

    const quote = line.match(/^>\s?(.*)$/);
    if (quote) {
      html += memoLineTemplate(lineNumber, `<blockquote>${inlineMarkdown(quote[1], context)}</blockquote>`, "is-quote");
      return;
    }

    html += memoLineTemplate(lineNumber, `<p>${inlineMarkdown(line, context)}</p>`);
  });

  return `<div class="memo-line-list">${html}</div>`;
}

function memoLineTemplate(lineNumber, body, className = "") {
  return `
    <div class="memo-source-line ${className}">
      <span class="memo-line-number" aria-hidden="true">${lineNumber}</span>
      <div class="memo-line-body">${body}</div>
    </div>
  `;
}

function listIndentWidth(whitespace) {
  const level = Math.min(6, Math.floor(String(whitespace || "").replace(/\t/g, "  ").length / 2));
  return level * 18;
}

function inlineMarkdown(value, context = {}) {
  const text = String(value || "");
  const pattern = /(!?)\[\[([^\]\n]+)\]\]/g;
  let html = "";
  let lastIndex = 0;
  let match = pattern.exec(text);

  while (match) {
    html += inlineMarkdownBase(text.slice(lastIndex, match.index));
    const ref = parseMemoReferenceInner(match[2], match[1] === "!");
    html += ref ? renderMemoRefChip(ref, context) : inlineMarkdownBase(match[0]);
    lastIndex = match.index + match[0].length;
    match = pattern.exec(text);
  }

  html += inlineMarkdownBase(text.slice(lastIndex));
  return html;
}

function inlineMarkdownBase(value) {
  let html = escapeHTML(value);
  html = html.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_, alt, src) => {
    return renderMemoImageToken({ label: alt, url: src });
  });
  html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, label, url) => {
    if (isFileAttachment(label, url)) {
      return renderMemoFileToken({ label, url });
    }
    const href = safeUrl(url);
    return `<a href="${escapeAttr(href)}" target="_blank" rel="noreferrer">${escapeHTML(label)}</a>`;
  });
  html = html.replace(/`([^`]+)`/g, "<code>$1</code>");
  html = html.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  html = html.replace(/(^|[^*])\*([^*]+)\*/g, "$1<em>$2</em>");
  html = html.replace(/(^|\s)#([\w\u4e00-\u9fa5-]+)/g, '$1<span class="memo-hashtag">#$2</span>');
  html = replaceMemoTimeSyntax(html);
  return html;
}

function replaceMemoTimeSyntax(html) {
  return String(html || "")
    .split(/(<code\b[^>]*>.*?<\/code>|<[^>]+>)/gi)
    .map(function (part) {
      if (!part || part.charAt(0) === "<") return part;
      return part.replace(
        /(^|[\s([{（【「『])(::|：：)((?:\d{4}(?:-\d{1,2}(?:-\d{1,2}(?:[ T]\d{1,2}:\d{2}(?::\d{2})?)?)?)?)|(?:\d{1,2}:\d{2}(?::\d{2})?)|(?:[^\s<>()\[\]{}，。！？、；;,.]{1,32}))/g,
        function (_, prefix, trigger, value) {
          return prefix + renderMemoTimeToken(trigger, value);
        },
      );
    })
    .join("");
}

function renderMemoTimeToken(trigger, value) {
  const label = String(value || "");
  return `
    <span class="memo-time-token" title="${escapeAttr(trigger + label)}" aria-label="时间 ${escapeAttr(label)}">
      ${SVG.clock}
      <span>${label}</span>
    </span>
  `;
}

function renderMemoRefChip(ref, context) {
  const target = resolveMemoReferenceTarget(ref, context);
  const label = target ? memoRefTitle(ref, target) : ref.alias || ref.target;
  const range = target && ref.selector ? memoSelectorLabel(ref.selector) : "";

  if (!target) {
    return `<span class="memo-ref-chip is-missing" title="找不到 ${escapeAttr(ref.target)}">[[${escapeHTML(label)}]]</span>`;
  }

  return `
    <button class="memo-ref-chip ${ref.embed ? "is-embed" : ""}" type="button" data-memo-ref-target="${escapeAttr(target.id)}" title="打开 memo">
      <span>${escapeHTML(label)}</span>
      ${range ? `<small>${escapeHTML(range)}</small>` : ""}
    </button>
  `;
}

function renderMemoEmbedCard(ref, context) {
  const target = resolveMemoReferenceTarget(ref, context);
  if (!target) {
    return renderMemoRefStateCard("is-missing", "引用不可用", "找不到 " + ref.target);
  }

  const stack = Array.isArray(context.stack) ? context.stack : [];
  if (stack.includes(target.id)) {
    return renderMemoRefStateCard("is-cycle", memoRefTitle(ref, target), "循环引用已停止");
  }

  const depth = Number(context.depth || 0);
  const maxDepth = Number(context.maxDepth || 2);
  if (depth >= maxDepth) {
    return renderMemoRefStateCard("is-collapsed", memoRefTitle(ref, target), "嵌套引用已折叠");
  }

  const excerpt = memoReferenceExcerpt(target.content, ref.selector);
  if (excerpt.error) {
    return renderMemoRefStateCard("is-missing", memoRefTitle(ref, target), excerpt.error);
  }

  const childContext = {
    ...context,
    depth: depth + 1,
    readonly: true,
    sourceId: target.id,
    stack: stack.concat(target.id),
  };
  const updatedAt = target.updatedAt || target.createdAt;
  const meta = [
    excerpt.label ? `<span>${escapeHTML(excerpt.label)}</span>` : "",
    updatedAt ? `<time datetime="${escapeAttr(updatedAt)}">${formatRelativeDate(updatedAt)}</time>` : "",
  ].filter(Boolean).join("");

  return `
    <aside class="memo-ref-card" data-memo-ref-card="${escapeAttr(target.id)}">
      ${meta ? `<div class="memo-ref-meta-line">${meta}</div>` : ""}
      <div class="memo-ref-body memo-content">${renderMemoMarkdown(excerpt.content, childContext)}</div>
    </aside>
  `;
}

function renderMemoRefStateCard(className, title, message) {
  return `
    <aside class="memo-ref-card ${className}">
      <header class="memo-ref-head">
        <span class="memo-ref-title is-static">${escapeHTML(title || "引用")}</span>
      </header>
      <div class="memo-ref-state">${escapeHTML(message || "")}</div>
    </aside>
  `;
}

function resolveMemoReferenceTarget(ref, context = {}) {
  const index = context.index || {};
  const memoById = index.memoById || new Map();
  const memoByTitleKey = index.memoByTitleKey || new Map();
  const raw = String((ref && ref.target) || "").trim();
  if (!raw) return null;

  const id = raw.toLowerCase().startsWith("memo:") ? raw.slice(5).trim() : raw;
  if (memoById.has(id)) return memoById.get(id);

  const titleId = memoByTitleKey.get(memoTitleKey(raw));
  return titleId ? memoById.get(titleId) || null : null;
}

function memoReferenceExcerpt(content, selector) {
  const text = String(content || "").replace(/\r\n/g, "\n");
  if (!selector) {
    return {
      content: text,
      label: "",
    };
  }

  if (selector.type === "invalid") {
    return { error: "行范围无效" };
  }
  if (selector.type !== "line") {
    return { error: "暂不支持选择器 #" + selector.raw };
  }

  const lines = memoLines(text);
  if (selector.start > lines.length) {
    return { error: "目标 memo 没有第 " + selector.start + " 行" };
  }

  const actualEnd = Math.min(selector.end, lines.length);
  const selected = lines.slice(selector.start - 1, actualEnd);
  const wrapped = wrapPartialCodeFence(lines, selector.start - 1, actualEnd, selected);
  return {
    content: wrapped.join("\n"),
    label: selector.start === actualEnd ? "L" + selector.start : "L" + selector.start + "-L" + actualEnd,
  };
}

function wrapPartialCodeFence(lines, startIndex, endIndex, selected) {
  const output = selected.slice();
  let inCode = false;

  for (let i = 0; i < startIndex; i += 1) {
    if (isMemoFenceLine(lines[i])) inCode = !inCode;
  }

  const startedInsideCode = inCode;
  for (let i = startIndex; i < endIndex; i += 1) {
    if (isMemoFenceLine(lines[i])) inCode = !inCode;
  }

  if (startedInsideCode) output.unshift("```");
  if (inCode) output.push("```");
  return output;
}

function memoLines(content) {
  return String(content || "").replace(/\r\n/g, "\n").split("\n");
}

function isMemoFenceLine(line) {
  return String(line || "").trim().startsWith("```");
}

function parseMemoHeadingLine(line) {
  const match = String(line || "").match(/^\s{0,3}(#{1,6})(?:[ \t]+|$)(.*)$/);
  if (!match) return null;

  return {
    level: match[1].length,
    text: match[2].replace(/[ \t]+#{1,}\s*$/, "").trim(),
  };
}

function memoRefTitle(ref, memo) {
  return String((ref && ref.alias) || "").trim() || memoTitle(memo);
}

function memoReferenceAlias(value) {
  return String(value || "")
    .replace(/\|+/g, " ")
    .replace(/\]\]+/g, "]")
    .replace(/\s+/g, " ")
    .trim();
}

function memoTitle(memo) {
  const lines = memoLines((memo && memo.content) || "");
  const first = lines.find(function (line) {
    return line.trim();
  });
  if (!first) return memo && memo.id ? memo.id : "Untitled memo";

  const heading = parseMemoHeadingLine(first);
  return compactMemoTitle(stripMemoTitleMarkdown(heading ? heading.text : first));
}

function stripMemoTitleMarkdown(value) {
  return String(value || "")
    .replace(/!\[\[([^\]]+)\]\]/g, "$1")
    .replace(/\[\[([^\]]+)\]\]/g, "$1")
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/[`*_>#-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function compactMemoTitle(value) {
  const title = String(value || "").trim();
  if (!title) return "Untitled memo";
  return title.length > 48 ? title.slice(0, 47) + "..." : title;
}

function memoTitleKey(value) {
  return String(value || "").trim().toLowerCase();
}

function memoSelectorLabel(selector) {
  if (!selector) return "";
  if (selector.type === "line") {
    return selector.start === selector.end ? "L" + selector.start : "L" + selector.start + "-L" + selector.end;
  }
  return "#" + selector.raw;
}

function memoBacklinkCount(context, memoId) {
  const incoming = context && context.index && context.index.incoming;
  return incoming && incoming.has(memoId) ? incoming.get(memoId).length : 0;
}

function parseStandaloneMarkdownResource(line) {
  const match = String(line || "").match(/^\s*(!?)\[([^\]]*)\]\(([^)]+)\)\s*$/);
  if (!match) return null;

  const url = match[3].trim();
  if (!url) return null;

  const label = (match[2] || "").trim() || fileDisplayName("", url);
  const type = match[1] === "!" || isImageAttachment(label, url) ? "image" : "file";

  return { type, label, url };
}

function renderMemoImageBlock(resource) {
  const src = safeImageUrl(resource.url);
  if (!src) return `<p>${renderMemoImageToken(resource)}</p>`;

  const label = resource.label || fileDisplayName("", resource.url);
  return `
    <figure class="memo-image-block">
      <img src="${escapeAttr(src)}" alt="${escapeAttr(label)}" loading="lazy" />
      ${label ? `<figcaption>${escapeHTML(label)}</figcaption>` : ""}
    </figure>
  `;
}

function renderMemoFileBlock(resource) {
  const href = safeUrl(resource.url);
  const name = fileDisplayName(resource.label, resource.url);
  const displayUrl = href !== "#" ? href : resource.url;
  const openButton = renderVSCodeOpenButton(resource.url);
  const localClass = openButton ? " has-editor-open" : "";
  const body = `
    <span class="memo-file-block-icon">${SVG.paperclip}</span>
    <span class="memo-file-block-text">
      <span class="memo-file-block-name">${escapeHTML(name)}</span>
      <span class="memo-file-block-url">${escapeHTML(compactFileURL(displayUrl))}</span>
    </span>
  `;

  if (href === "#" || openButton) {
    return `<div class="memo-file-block${localClass}">${body}${openButton}</div>`;
  }
  return `<a class="memo-file-block" href="${escapeAttr(href)}" target="_blank" rel="noreferrer">${body}</a>`;
}

function renderMemoImageToken(resource) {
  const src = safeImageUrl(resource.url);
  const label = resource.label || fileDisplayName("", resource.url);
  return `
    <span class="memo-image-token">
      ${SVG.image}
      <span>${escapeHTML(label || resource.url)}</span>
      ${src ? `<span class="memo-image-token-preview"><img src="${escapeAttr(src)}" alt="${escapeAttr(label)}" loading="lazy" /></span>` : ""}
    </span>
  `;
}

function renderMemoFileToken(resource) {
  const href = safeUrl(resource.url);
  const name = fileDisplayName(resource.label, resource.url);
  const openButton = renderVSCodeOpenButton(resource.url);
  const localClass = openButton ? " has-editor-open" : "";
  const body = `${SVG.paperclip}<span>${escapeHTML(name)}</span>`;

  if (href === "#" || openButton) {
    return `<span class="memo-file-token${localClass}">${body}${openButton}</span>`;
  }
  return `<a class="memo-file-token" href="${escapeAttr(href)}" target="_blank" rel="noreferrer">${body}</a>`;
}

function renderVSCodeOpenButton(url) {
  const target = localEditorTarget(url);
  if (!target) return "";
  return `
    <button
      class="memo-file-open-vscode"
      type="button"
      data-editor-open="vscode"
      data-editor-file="${escapeAttr(target.file)}"
      data-editor-line="${escapeAttr(target.line)}"
      data-editor-col="${escapeAttr(target.col)}"
      title="在 VS Code 中打开"
      aria-label="在 VS Code 中打开"
    >
      ${SVG.code}
      <span>在 VS Code 中打开</span>
    </button>
  `;
}

function localEditorTarget(value) {
  const target = parseEditorTarget(value);
  if (!target || !target.file) return null;
  if (!isLocalEditorResource(target.file)) return null;
  return target;
}

function parseEditorTarget(value) {
  let file = String(value || "").trim();
  if (!file) return null;

  let line = "1";
  let col = "1";
  try {
    const parsed = new URL(file, window.location.origin);
    line = editorPositionValue(parsed.searchParams.get("line"), line);
    col = editorPositionValue(parsed.searchParams.get("col") || parsed.searchParams.get("column"), col);
  } catch (_) {}

  if (!isLocalOSSAssetURL(file)) {
    const suffix = file.match(/^(.*):(\d+)(?::(\d+))?$/);
    if (suffix && !/^[a-zA-Z]:[\\/]/.test(file)) {
      file = suffix[1];
      line = editorPositionValue(suffix[2], line);
      col = editorPositionValue(suffix[3], col);
    }
  }

  return { file, line, col };
}

function editorPositionValue(value, fallback) {
  const text = String(value || "").trim();
  return /^[1-9]\d*$/.test(text) ? text : fallback || "1";
}

function isLocalEditorResource(value) {
  const url = String(value || "").trim();
  if (!url) return false;
  if (isLocalAssetReference(url)) return true;
  if (/^@assets\//i.test(url)) return false;
  if (isLocalOSSAssetURL(url)) return true;
  if (/^(local:\/\/|file:\/\/)/i.test(url)) return true;
  if (/^(https?:|mailto:|blob:|data:)/i.test(url)) return false;
  if (/^[a-z][a-z0-9+.-]*:/i.test(url) && !/^[a-zA-Z]:[\\/]/.test(url)) return false;
  return true;
}

function isLocalAssetReference(value) {
  const asset = typeof parseAssetReference === "function" ? parseAssetReference(value) : null;
  if (!asset) return false;
  const storage = typeof cloudStorageById === "function" ? cloudStorageById(asset.storageId) : null;
  return editorStorageIsLocal(storage);
}

function isLocalOSSAssetURL(value) {
  const raw = String(value || "").trim();
  try {
    const parsed = new URL(raw, window.location.origin);
    return parsed.pathname === "/api/oss/assets" && (parsed.origin === window.location.origin || raw.startsWith("/"));
  } catch (_) {
    return /^\/api\/oss\/assets(?:\?|$)/i.test(raw);
  }
}

function editorStorageIsLocal(storage) {
  const provider = String((storage && storage.provider) || "").trim().toLowerCase();
  return provider === "local" || provider === "local-oss";
}

function isFileAttachment(label, url) {
  const pattern = /\.(?:7z|aac|apk|avi|csv|dmg|docx?|flac|gz|heic|ics|json|key|log|m4a|mkv|mov|mp3|mp4|numbers|pages|pdf|pptx?|rar|rtf|tar|txt|wav|webm|xlsx?|xml|yaml|yml|zip)(?:[?#].*)?$/i;
  if (parseAssetReference(url)) return true;
  if (/^(local:\/\/|blob:|data:)/i.test(String(url || ""))) return true;
  return pattern.test(String(label || "")) || pattern.test(String(url || ""));
}

function isImageAttachment(label, url) {
  const pattern = /\.(?:avif|bmp|gif|jpe?g|png|svg|webp)(?:[?#].*)?$/i;
  const asset = parseAssetReference(url);
  if (/^data:image\//i.test(String(url || ""))) return true;
  return pattern.test(String(label || "")) || pattern.test(String(url || "")) || pattern.test(asset ? asset.key : "");
}

function fileDisplayName(label, url) {
  const asset = parseAssetReference(url);
  const raw = String(label || "").trim() || (asset ? asset.key : String(url || "").trim());
  const clean = raw.split(/[?#]/)[0].replace(/\/+$/, "");
  const last = clean.split("/").pop() || raw;
  try {
    return decodeURIComponent(last) || raw;
  } catch (_) {
    return last || raw;
  }
}

function compactFileURL(url) {
  const value = String(url || "").trim();
  if (value.length <= 72) return value;
  return value.slice(0, 34) + "..." + value.slice(-28);
}

function safeImageUrl(value) {
  const url = resolveAssetUrl(value);
  if (/^\/(?!\/)/.test(url)) return url;
  if (/^(https?:|local:\/\/|blob:)/i.test(url)) return url;
  if (/^data:image\//i.test(url)) return url;
  return "";
}

function safeUrl(value) {
  const url = resolveAssetUrl(value);
  if (/^\/(?!\/)/.test(url)) return url;
  if (/^(https?:|mailto:|local:\/\/|blob:)/i.test(url)) return url;
  return "#";
}

function compactText(value, length) {
  const text = String(value || "").replace(/\s+/g, " ").trim();
  return text.length > length ? text.slice(0, length - 1) + "..." : text;
}

function formatRelativeDate(value) {
  const date = new Date(value);
  const diff = Date.now() - date.getTime();
  const minute = 1000 * 60;
  const hour = minute * 60;
  const day = hour * 24;
  if (diff < minute) return "刚刚";
  if (diff < hour) return `${Math.floor(diff / minute)} 分钟前`;
  if (diff < day) return `${Math.floor(diff / hour)} 小时前`;
  if (diff < day * 7) return `${Math.floor(diff / day)} 天前`;
  return date.toLocaleDateString();
}

function formatShortDate(value) {
  const date = new Date(value);
  return `${date.getMonth() + 1}/${date.getDate()}`;
}

function addMonths(date, delta) {
  const value = new Date(date);
  return startOfMonth(new Date(value.getFullYear(), value.getMonth() + delta, 1));
}

function startOfMonth(date) {
  const value = new Date(date);
  return new Date(value.getFullYear(), value.getMonth(), 1);
}

function generateCalendarDays(monthDate) {
  const month = startOfMonth(monthDate);
  const start = new Date(month);
  start.setDate(start.getDate() - start.getDay());

  return Array.from({ length: 42 }, function (_value, index) {
    const date = new Date(start);
    date.setDate(start.getDate() + index);
    return {
      date,
      inMonth: date.getMonth() === month.getMonth(),
      key: formatDateKey(date),
    };
  });
}

function memoDateCounts(memos) {
  const counts = new Map();
  memos.forEach((memo) => {
    if (memo.archived) return;
    const key = memoDateKey(memo);
    if (!key) return;
    counts.set(key, (counts.get(key) || 0) + 1);
  });
  return counts;
}

function memoDateKey(memo) {
  const value = memo && memo.createdAt;
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return formatDateKey(date);
}

function formatDateKey(date) {
  const value = new Date(date);
  const year = value.getFullYear();
  const month = String(value.getMonth() + 1).padStart(2, "0");
  const day = String(value.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function dateFromKey(value) {
  const match = String(value || "").match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!match) return new Date();
  return new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]));
}

function createId() {
  return `memo_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`;
}

function copyText(text) {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    return navigator.clipboard.writeText(text);
  }
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.select();
  const ok = document.execCommand("copy");
  textarea.remove();
  return ok ? Promise.resolve() : Promise.reject(new Error("copy failed"));
}

function closestAnchor(target) {
  return closestElement(target, "a[href]");
}

function closestElement(target, selector) {
  let node = target;
  if (node && node.nodeType === 3) node = node.parentElement || node.parentNode;
  while (node && node !== document) {
    if (node.nodeType === 1) {
      if (typeof node.matches === "function" && node.matches(selector)) return node;
      if (typeof node.webkitMatchesSelector === "function" && node.webkitMatchesSelector(selector)) return node;
    }
    node = node.parentElement || node.parentNode;
  }
  return null;
}

function externalBrowserURLFromAnchor(anchor) {
  const href = String(anchor.getAttribute("href") || anchor.href || "").trim();
  if (!/^https?:\/\//i.test(href)) return "";

  try {
    const url = new URL(href);
    if ((url.protocol === "http:" || url.protocol === "https:") && url.host) {
      return url.href;
    }
  } catch (_) {}
  return "";
}

function escapeHTML(value) {
  return String(value || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function escapeAttr(value) {
  return escapeHTML(value).replace(/`/g, "&#96;");
}

function escapeCSSIdent(value) {
  if (window.CSS && window.CSS.escape) return window.CSS.escape(String(value || ""));
  return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}
