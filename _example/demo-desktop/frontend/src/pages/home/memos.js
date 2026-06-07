const MEMOS_STORAGE_KEY = "demo-desktop:memos:items:v1";
const DRAFT_STORAGE_KEY = "demo-desktop:memos:draft:v1";
const CLOUD_STORAGE_KEY = "demo-desktop:settings:cloud-storage:v1";
const DEFAULT_VISIBILITY = "PRIVATE";
const TASK_LINE_REGEX = /^(\s*[-*]\s+\[)([ xX])(\]\s+)(.*)$/;

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
  clock:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="9"></circle><path d="M12 7v5l3 2"></path></svg>',
  code:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m9 18-6-6 6-6"></path><path d="m15 6 6 6-6 6"></path></svg>',
  copy:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="9" y="9" width="11" height="11" rx="2"></rect><rect x="4" y="4" width="11" height="11" rx="2"></rect></svg>',
  edit:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 20h9"></path><path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z"></path></svg>',
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

export function mountMemosHome(root, options = {}) {
  const state = {
    activeFilter: "all",
    activeTag: "",
    activeView: "memos",
    editingId: "",
    editDraft: "",
    editVisibility: DEFAULT_VISIBILITY,
    highlightMemoId: "",
    highlightTimer: null,
    memos: loadMemos(),
    query: "",
    sortDesc: true,
    toastTimer: null,
    visibility: DEFAULT_VISIBILITY,
  };

  let composerEditor = null;
  let editEditor = null;

  root.innerHTML = shellTemplate();

  const els = {
    attachInput: root.querySelector("[data-attach-input]"),
    composer: root.querySelector("[data-composer]"),
    composerHost: root.querySelector("[data-composer-host]"),
    composerStatus: root.querySelector("[data-composer-status]"),
    createButton: root.querySelector('[data-action="createMemo"]'),
    feedCount: root.querySelector("[data-feed-count]"),
    mainSubtitle: root.querySelector("[data-main-subtitle]"),
    mainTitle: root.querySelector("[data-main-title]"),
    memoList: root.querySelector("[data-memo-list]"),
    pinnedList: root.querySelector("[data-pinned-list]"),
    searchInput: root.querySelector("[data-search-input]"),
    stats: root.querySelector("[data-stats]"),
    tagList: root.querySelector("[data-tag-list]"),
    tagSummary: root.querySelector("[data-tag-summary]"),
    todoNavCount: root.querySelector("[data-todo-nav-count]"),
    toast: root.querySelector("[data-toast]"),
    version: root.querySelector("[data-version]"),
  };

  updateVersion(els.version, options.version);

  composerEditor = createMiniEditor(els.composerHost, {
    onChange(value) {
      localStorage.setItem(DRAFT_STORAGE_KEY, value);
      renderComposerStatus(value);
    },
    onSubmit() {
      createMemo();
    },
    placeholder: "记录想法、任务或链接...",
    value: localStorage.getItem(DRAFT_STORAGE_KEY) || "",
  });

  renderAll();
  renderComposerStatus(composerEditor.getText());
  bindGoMessages();

  root.addEventListener("click", handleClick);
  root.addEventListener("input", handleInput);
  root.addEventListener("change", handleChange);

  return {
    destroy() {
      root.removeEventListener("click", handleClick);
      root.removeEventListener("input", handleInput);
      root.removeEventListener("change", handleChange);
      if (state.toastTimer) window.clearTimeout(state.toastTimer);
      if (state.highlightTimer) window.clearTimeout(state.highlightTimer);
      if (composerEditor) composerEditor.destroy();
      if (editEditor) editEditor.destroy();
      root.innerHTML = "";
    },
  };

  function bindGoMessages() {
    if (!window.onGoMessage) return;
    window.onGoMessage((payload) => {
      if (!payload || payload.type !== "memo_file_drop") return;
      insertDroppedFiles(payload.files);
    });
  }

  function handleClick(event) {
    const command = event.target.closest("[data-command]");
    if (command && root.contains(command)) {
      runComposerCommand(command.dataset.command);
      return;
    }

    const filter = event.target.closest("[data-filter]");
    if (filter && root.contains(filter)) {
      state.activeView = "memos";
      state.activeFilter = filter.dataset.filter;
      state.activeTag = "";
      renderAll();
      return;
    }

    const view = event.target.closest("[data-view]");
    if (view && root.contains(view)) {
      state.activeView = view.dataset.view;
      state.activeFilter = "all";
      state.activeTag = "";
      state.editingId = "";
      state.query = "";
      els.searchInput.value = "";
      renderAll();
      return;
    }

    const tag = event.target.closest("[data-tag]");
    if (tag && root.contains(tag)) {
      state.activeTag = state.activeTag === tag.dataset.tag ? "" : tag.dataset.tag;
      state.activeFilter = "all";
      renderAll();
      return;
    }

    const action = event.target.closest("[data-action]");
    if (!action || !root.contains(action)) return;

    const memoNode = action.closest("[data-memo-id]");
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
        state.query = "";
        els.searchInput.value = "";
        renderAll();
        break;
      case "copyMemo":
        copyMemo(memoId);
        break;
      case "createMemo":
        createMemo();
        break;
      case "deleteMemo":
        deleteMemo(memoId);
        break;
      case "editMemo":
        startEdit(memoId);
        break;
      case "openSourceMemo":
        openSourceMemo(memoId);
        break;
      case "openSettings":
        openSettings();
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

  function handleInput(event) {
    if (event.target.matches("[data-search-input]")) {
      state.query = event.target.value.trim();
      renderAll();
    }
  }

  function openSettings() {
    if (typeof invoke !== "function") {
      window.open("settings.html");
      return;
    }
    invoke("/api/open_window?pathname=%2Fsettings", { method: "GET" }).then(
      (resp) => {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || "打开设置失败");
        }
      },
      (err) => {
        showToast("打开设置失败: " + err);
      },
    );
  }

  function handleChange(event) {
    if (event.target.matches("[data-visibility-select]")) {
      state.visibility = event.target.value;
      return;
    }

    if (event.target.matches("[data-edit-visibility]")) {
      state.editVisibility = event.target.value;
      return;
    }

    if (event.target.matches("[data-task-line]")) {
      const memoNode = event.target.closest("[data-memo-id]");
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
    const content = composerEditor.getText().trim();
    if (!content) {
      showToast("先写点内容");
      composerEditor.focus();
      return;
    }

    const now = new Date().toISOString();
    state.memos.unshift({
      archived: false,
      content,
      createdAt: now,
      id: createId(),
      pinned: false,
      updatedAt: "",
      visibility: state.visibility,
    });
    saveMemos(state.memos);
    composerEditor.setText("");
    localStorage.removeItem(DRAFT_STORAGE_KEY);
    state.activeView = "memos";
    state.activeFilter = "all";
    state.activeTag = "";
    renderAll();
    renderComposerStatus("");
    window.requestAnimationFrame(() => {
      if (composerEditor && els.composerHost.isConnected) composerEditor.focus();
    });
  }

  function startEdit(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    state.editingId = memoId;
    state.editDraft = memo.content;
    state.editVisibility = memo.visibility || DEFAULT_VISIBILITY;
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
    const content = (editEditor ? editEditor.getText() : state.editDraft).trim();
    if (!content) {
      showToast("内容不能为空");
      return;
    }
    state.editingId = "";
    state.editDraft = "";
    updateMemo(memoId, {
      content,
      updatedAt: new Date().toISOString(),
      visibility: state.editVisibility,
    });
  }

  function updateMemo(memoId, patch) {
    state.memos = state.memos.map((memo) => {
      if (memo.id !== memoId) return memo;
      return {
        ...memo,
        ...patch,
        updatedAt: patch.updatedAt || memo.updatedAt,
      };
    });
    saveMemos(state.memos);
    renderAll();
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
      (_, prefix, _marker, suffix, text) => `${prefix}${checked ? "x" : " "}${suffix}${text}`,
    );
    updateMemo(memoId, {
      content: lines.join("\n"),
      updatedAt: new Date().toISOString(),
    });
  }

  function deleteMemo(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    if (!window.confirm("删除这条 memo？")) return;
    state.memos = state.memos.filter((item) => item.id !== memoId);
    saveMemos(state.memos);
    renderAll();
  }

  function copyMemo(memoId) {
    const memo = findMemo(memoId);
    if (!memo) return;
    copyText(memo.content).then(
      () => showToast("已复制"),
      () => showToast("复制失败"),
    );
  }

  function insertFiles(files) {
    if (!files || files.length === 0) return;
    if (composerEditor.insertFiles) {
      composerEditor.insertFiles(files);
    } else {
      filesToMarkdown(files).then((markdown) => {
        if (markdown) composerEditor.insertBlock(markdown);
      }).catch((err) => {
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
    droppedFilesToMarkdown(files).then((markdown) => {
      if (!markdown || !composerEditor) return;
      composerEditor.insertBlock(markdown);
      composerEditor.focus();
      showToast("已插入拖拽文件");
    }).catch((err) => {
      showToast(uploadErrorMessage(err));
    });
  }

  function renderAll() {
    renderMainChrome();
    renderViewButtons();
    renderFilterButtons();
    renderStats();
    renderTags();
    renderPinned();
    renderMainContent();
  }

  function renderMainChrome() {
    const isTodos = state.activeView === "todos";
    els.mainTitle.textContent = isTodos ? "代办" : "Inbox";
    els.mainSubtitle.textContent = isTodos ? "从所有 memo 中汇总任务" : "捕捉、整理、回看";
    els.composer.classList.toggle("hidden", isTodos);
    els.searchInput.placeholder = isTodos ? "搜索代办或来源 memo" : "搜索 memos";
    els.memoList.classList.toggle("is-todo-list", isTodos);
  }

  function renderMainContent() {
    if (state.activeView === "todos") {
      renderTodos();
      return;
    }
    renderFeed();
  }

  function renderViewButtons() {
    root.querySelectorAll("[data-view]").forEach((button) => {
      button.classList.toggle("is-active", button.dataset.view === state.activeView);
    });
    const todoStats = getTodoStats(state.memos);
    els.todoNavCount.textContent = todoStats.open ? String(todoStats.open) : "";
  }

  function renderFilterButtons() {
    root.querySelectorAll("[data-filter]").forEach((button) => {
      button.classList.toggle(
        "is-active",
        state.activeView === "memos" && button.dataset.filter === state.activeFilter && !state.activeTag,
      );
    });
  }

  function renderStats() {
    const active = state.memos.filter((memo) => !memo.archived);
    const archived = state.memos.length - active.length;
    const publicCount = active.filter((memo) => memo.visibility === "PUBLIC").length;
    const todoStats = getTodoStats(state.memos);

    els.stats.innerHTML = [
      statTemplate("全部", active.length),
      statTemplate("公开", publicCount),
      statTemplate("代办", todoStats.total),
      statTemplate("未完成", todoStats.open),
      statTemplate("完成", todoStats.done),
      statTemplate("归档", archived),
    ].join("");
  }

  function renderTags() {
    const tags = collectTags(state.memos.filter((memo) => !memo.archived));
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
    const pinned = state.memos.filter((memo) => memo.pinned && !memo.archived).slice(0, 3);
    els.pinnedList.innerHTML = pinned.length
      ? pinned
          .map(
            (memo) => `
              <button class="memo-pinned-item" type="button" data-tag="${escapeAttr(extractTags(memo.content)[0] || "")}">
                <span>${escapeHTML(compactText(memo.content, 54))}</span>
                <small>${formatShortDate(memo.createdAt)}</small>
              </button>
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
      ? memos.map((memo) => memoTemplate(memo, state.editingId, state.highlightMemoId)).join("")
      : emptyFeedTemplate();

    if (state.editingId) {
      const memo = findMemo(state.editingId);
      const host = els.memoList.querySelector("[data-edit-host]");
      if (memo && host) {
        editEditor = createMiniEditor(host, {
          onChange(value) {
            state.editDraft = value;
          },
          onSubmit() {
            saveEdit(memo.id);
          },
          placeholder: "编辑 memo...",
          value: memo.content,
        });
        editEditor.focus();
      }
    }

    scrollHighlightedMemoIntoView();
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
          todoGroupTemplate("未完成", openTodos),
          todoGroupTemplate("已完成", doneTodos),
        ].join("")
      : emptyTodosTemplate();
  }

  function renderComposerStatus(value) {
    const text = String(value || "");
    const tagCount = extractTags(text).length;
    const chars = text.trim().length;
    els.composerStatus.textContent = `${chars} 字符 / ${tagCount} 标签`;
    els.createButton.disabled = chars === 0;
  }

  function visibleMemos() {
    const query = state.query.toLowerCase();
    return state.memos
      .filter((memo) => {
        if (state.activeFilter === "archive") return memo.archived;
        if (memo.archived) return false;
        if (state.activeFilter === "pinned" && !memo.pinned) return false;
        if (state.activeFilter === "public" && memo.visibility !== "PUBLIC") return false;
        if (state.activeFilter === "private" && memo.visibility !== "PRIVATE") return false;
        if (state.activeTag && !extractTags(memo.content).includes(state.activeTag)) return false;
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
    return collectTodos(state.memos)
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

  function findMemo(memoId) {
    return state.memos.find((memo) => memo.id === memoId);
  }

  function openSourceMemo(memoId) {
    if (!findMemo(memoId)) return;
    state.activeView = "memos";
    state.activeFilter = "all";
    state.activeTag = "";
    state.query = "";
    state.editingId = "";
    state.highlightMemoId = memoId;
    els.searchInput.value = "";
    renderAll();
  }

  function scrollHighlightedMemoIntoView() {
    if (!state.highlightMemoId) return;
    const node = Array.from(els.memoList.querySelectorAll("[data-memo-id]")).find(
      (item) => item.dataset.memoId === state.highlightMemoId,
    );
    if (!node) return;
    window.requestAnimationFrame(() => {
      node.scrollIntoView({ block: "center", behavior: "smooth" });
    });
    if (state.highlightTimer) window.clearTimeout(state.highlightTimer);
    state.highlightTimer = window.setTimeout(() => {
      node.classList.remove("is-highlighted");
      state.highlightMemoId = "";
    }, 1600);
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
          ${todoNavButtonTemplate()}
          ${filterButtonTemplate("pinned", "置顶", SVG.pin)}
          ${filterButtonTemplate("private", "仅自己", SVG.lock)}
          ${filterButtonTemplate("public", "公开", SVG.globe)}
          ${filterButtonTemplate("archive", "归档", SVG.archive)}
        </nav>
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
          <button class="memo-icon-text-button" type="button" data-action="sortMemos" title="排序">
            ${SVG.sort}
            <span>排序</span>
          </button>
        </header>

        <section class="memo-composer" aria-label="Create memo" data-composer>
          <div class="memo-composer-head">
            <div class="memo-composer-title">今天有什么想法？</div>
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
            <div class="memo-tool-group">
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
            <div class="memo-composer-actions">
              <span data-composer-status></span>
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
          <div class="memo-inspector-title">概览</div>
          <div class="memo-stats" data-stats></div>
        </section>
        <section class="memo-inspector-section">
          <div class="memo-inspector-title">置顶</div>
          <div class="memo-pinned-list" data-pinned-list></div>
        </section>
        <section class="memo-inspector-section">
          <div class="memo-inspector-title">版本</div>
          <div class="memo-version" data-version></div>
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

function todoNavButtonTemplate() {
  return `
    <button class="memo-nav-button" type="button" data-view="todos">
      ${SVG.check}
      <span>代办</span>
      <strong data-todo-nav-count></strong>
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

function memoTemplate(memo, editingId, highlightId) {
  const visibility = VISIBILITY[memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
  const tags = extractTags(memo.content);
  const archived = memo.archived;
  const editing = memo.id === editingId;
  const highlighted = memo.id === highlightId;

  return `
    <article class="memo-card ${memo.pinned ? "is-pinned" : ""} ${archived ? "is-archived" : ""} ${highlighted ? "is-highlighted" : ""}" data-memo-id="${escapeAttr(memo.id)}">
      <header class="memo-card-head">
        <div class="memo-author">
          <div class="memo-avatar">U</div>
          <div>
            <div class="memo-author-name">You</div>
            <time datetime="${escapeAttr(memo.createdAt)}">${formatRelativeDate(memo.createdAt)}</time>
          </div>
        </div>
        <div class="memo-card-meta">
          <span class="memo-visibility">${SVG[visibility.icon]} ${visibility.label}</span>
          ${memo.pinned ? '<span class="memo-pin-label">置顶</span>' : ""}
        </div>
      </header>
      ${
        editing
          ? editTemplate(memo)
          : `
            <div class="memo-content">${renderMemoMarkdown(memo.content)}</div>
            ${tags.length ? `<div class="memo-card-tags">${tags.map((tag) => `<button type="button" data-tag="${escapeAttr(tag)}">#${escapeHTML(tag)}</button>`).join("")}</div>` : ""}
          `
      }
      <footer class="memo-card-actions">
        <button class="memo-action-button" type="button" data-action="togglePin" title="${memo.pinned ? "取消置顶" : "置顶"}">${SVG.pin}</button>
        <button class="memo-action-button" type="button" data-action="copyMemo" title="复制">${SVG.copy}</button>
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

function todoGroupTemplate(label, todos) {
  if (!todos.length) return "";
  return `
    <section class="memo-todo-group" aria-label="${escapeAttr(label)}">
      <div class="memo-todo-group-head">
        <span>${escapeHTML(label)}</span>
        <strong>${todos.length}</strong>
      </div>
      ${todos.map(todoTemplate).join("")}
    </section>
  `;
}

function todoTemplate(todo) {
  const visibility = VISIBILITY[todo.memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
  const tags = extractTags(todo.memo.content);
  return `
    <article class="memo-todo-card ${todo.checked ? "is-complete" : ""}" data-memo-id="${escapeAttr(todo.memoId)}">
      <label class="memo-todo-check">
        <input type="checkbox" data-task-line="${todo.lineIndex}" ${todo.checked ? "checked" : ""} />
        <span>${inlineMarkdown(todo.text)}</span>
      </label>
      <div class="memo-todo-source">
        <button class="memo-todo-source-button" type="button" data-action="openSourceMemo" title="查看来源 memo">
          <span>来源 memo</span>
          <strong>${escapeHTML(todo.sourceText)}</strong>
        </button>
        <div class="memo-todo-meta">
          <time datetime="${escapeAttr(todo.memo.createdAt)}">${formatRelativeDate(todo.memo.createdAt)}</time>
          <span>${SVG[visibility.icon]} ${visibility.label}</span>
          ${tags.slice(0, 3).map((tag) => `<span>#${escapeHTML(tag)}</span>`).join("")}
        </div>
      </div>
    </article>
  `;
}

function editTemplate(memo) {
  return `
    <div class="memo-inline-editor">
      <div class="memo-editor-host is-inline" data-edit-host></div>
      <div class="memo-inline-actions">
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

function visibilityOptionsTemplate(selected) {
  return Object.entries(VISIBILITY)
    .map(([value, item]) => `<option value="${value}" ${value === selected ? "selected" : ""}>${item.label}</option>`)
    .join("");
}

function createMiniEditor(host, options) {
  const PM = window.ProsemirrorMod;
  if (!PM) return createFallbackEditor(host, options);

  const plugins = [
    PM.history(),
    PM.keymap({
      "Mod-Enter": () => {
        if (options.onSubmit) options.onSubmit();
        return true;
      },
    }),
    PM.keymap(PM.baseKeymap),
  ];

  host.dataset.placeholder = options.placeholder || "";

  let view = new PM.EditorView(host, {
    dispatchTransaction(transaction) {
      const nextState = view.state.apply(transaction);
      view.updateState(nextState);
      syncEmptyState();
      if (options.onChange) options.onChange(getText());
    },
    state: createState(options.value || ""),
  });

  const removeDrop = installFileDropHandler(host, {
    focus() {
      view.focus();
    },
    insertFiles,
  });

  syncEmptyState();

  return {
    destroy() {
      removeDrop();
      view.destroy();
    },
    focus() {
      view.focus();
    },
    getText,
    insertBlock(text) {
      const current = getText();
      const prefix = current && !current.endsWith("\n") ? "\n" : "";
      insertText(prefix + text);
    },
    insertFiles,
    insertText,
    requestFiles(accept) {
      requestFilesForEditor({ focus: () => view.focus(), getText, insertFiles, insertText }, accept || "");
    },
    setText(value) {
      view.updateState(createState(value || ""));
      syncEmptyState();
      if (options.onChange) options.onChange(getText());
    },
    wrap(prefix, suffix, placeholder) {
      const { from, to, empty } = view.state.selection;
      const selected = empty ? placeholder : view.state.doc.textBetween(from, to, "\n");
      const text = `${prefix}${selected}${suffix}`;
      const transaction = view.state.tr.insertText(text, from, to);
      const cursor = empty ? from + prefix.length + selected.length : from + text.length;
      transaction.setSelection(PM.TextSelection.create(transaction.doc, cursor));
      view.dispatch(transaction.scrollIntoView());
    },
  };

  function createState(value) {
    const doc = textToDoc(PM, value);
    return PM.EditorState.create({
      doc,
      selection: PM.Selection.near(doc.resolve(Math.min(1, doc.content.size)), 1),
      plugins,
    });
  }

  function getText() {
    return view.state.doc.textBetween(0, view.state.doc.content.size, "\n");
  }

  function insertText(text) {
    view.dispatch(view.state.tr.insertText(text).scrollIntoView());
  }

  function insertFiles(files) {
    filesToMarkdown(files).then((markdown) => {
      if (!markdown) return;
      const current = getText();
      const prefix = current && !current.endsWith("\n") ? "\n" : "";
      insertText(prefix + markdown);
      view.focus();
    }).catch((err) => {
      console.error(uploadErrorMessage(err));
    });
  }

  function syncEmptyState() {
    host.classList.toggle("is-empty", getText().trim().length === 0);
  }
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
  const removeDrop = installFileDropHandler(host, {
    focus() {
      textarea.focus();
    },
    insertFiles,
  });
  return {
    destroy() {
      removeDrop();
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
    insertFiles,
    insertText(text) {
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;
      textarea.value = textarea.value.slice(0, start) + text + textarea.value.slice(end);
      textarea.selectionStart = textarea.selectionEnd = start + text.length;
      if (options.onChange) options.onChange(textarea.value);
    },
    requestFiles(accept) {
      requestFilesForEditor(
        {
          focus() {
            textarea.focus();
          },
          getText() {
            return textarea.value;
          },
          insertFiles,
          insertText(text) {
            const start = textarea.selectionStart;
            const end = textarea.selectionEnd;
            textarea.value = textarea.value.slice(0, start) + text + textarea.value.slice(end);
            textarea.selectionStart = textarea.selectionEnd = start + text.length;
            if (options.onChange) options.onChange(textarea.value);
          },
        },
        accept || "",
      );
    },
    setText(value) {
      textarea.value = value || "";
      textarea.selectionStart = textarea.selectionEnd = 0;
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

  function insertFiles(files) {
    filesToMarkdown(files).then((markdown) => {
      if (!markdown) return;
      textarea.value += `${textarea.value && !textarea.value.endsWith("\n") ? "\n" : ""}${markdown}`;
      if (options.onChange) options.onChange(textarea.value);
      textarea.focus();
    }).catch((err) => {
      console.error(uploadErrorMessage(err));
    });
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
    editor.insertFiles(files);
    editor.focus();
  }

  function onPaste(event) {
    const files = filesFromClipboard(event.clipboardData);
    if (!files.length) return;
    event.preventDefault();
    event.stopPropagation();
    editor.insertFiles(files);
    editor.focus();
  }

  host.addEventListener("dragenter", onDragOver, true);
  host.addEventListener("dragover", onDragOver, true);
  host.addEventListener("drop", onDrop, true);
  host.addEventListener("paste", onPaste, true);
  return function removeFileDropHandler() {
    host.removeEventListener("dragenter", onDragOver, true);
    host.removeEventListener("dragover", onDragOver, true);
    host.removeEventListener("drop", onDrop, true);
    host.removeEventListener("paste", onPaste, true);
  };
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
  input.addEventListener("change", () => {
    editor.insertFiles(input.files);
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
    (resp) => {
      if (!resp || resp.code !== 0 || !resp.data || !resp.data.file) return;
      droppedFilesToMarkdown([resp.data.file]).then((markdown) => {
        if (!markdown) return;
        const current = editor.getText();
        editor.insertText((current && !current.endsWith("\n") ? "\n" : "") + markdown);
        editor.focus();
      }).catch((err) => {
        console.error(uploadErrorMessage(err));
      });
    },
    () => {},
  );
}

function filesToMarkdown(files) {
  const list = Array.from(files || []);
  return Promise.all(
    list.map((file) =>
      readFileAsDataURL(file).then((url) =>
        fileInfoToMarkdownAsync({
          name: file.name || "file",
          type: file.type || "",
          url,
        }),
      ),
    ),
  ).then((items) => items.join("\n"));
}

function droppedFilesToMarkdown(files) {
  const list = Array.from(files || []).filter((file) => file && (file.dataURL || file.url || file.path));
  return Promise.all(
    list
      .map((file) =>
        fileInfoToMarkdownAsync({
          name: file.name || "file",
          type: file.type || "",
          url: file.dataURL || file.url || file.path,
        }),
      )
  ).then((items) => items.join("\n"));
}

function fileInfoToMarkdownAsync(file) {
  return fileInfoToUploadURL(file).then((uploaded) =>
    fileInfoToMarkdown({
      name: uploaded.name || file.name,
      type: uploaded.type || file.type,
      url: uploaded.url || file.url,
    }),
  );
}

function fileInfoToUploadURL(file) {
  return loadCloudStorageConfig().then((config) => {
    if (!config || !config.enabled) return file;
    if (typeof invoke !== "function") throw new Error("当前环境不支持云存储上传");

    const missing = missingCloudStorageFields(config);
    if (missing.length) {
      throw new Error("云存储配置缺少: " + missing.join(", "));
    }

    const contentBase64 = dataURLToBase64(file.url || file.dataURL || "");
    if (!contentBase64) throw new Error("无法读取文件内容");

    return invoke("/api/oss/upload", {
      args: {
        config,
        content_base64: contentBase64,
        name: file.name || "file",
        type: file.type || "",
      },
    }).then((resp) => {
      if (!resp || resp.code !== 0 || !resp.data) {
        throw new Error((resp && resp.msg) || "上传失败");
      }
      return {
        name: resp.data.name || file.name,
        type: resp.data.type || file.type,
        url: resp.data.url || file.url,
      };
    });
  });
}

function loadCloudStorageConfig() {
  if (typeof invoke === "function") {
    return invoke("/api/settings/cloud-storage", { method: "GET" }).then(
      (resp) => {
        if (resp && resp.code === 0 && resp.data && resp.data.found && resp.data.config) {
          return resp.data.config;
        }
        if (resp && resp.code === 0) return loadLocalCloudStorageConfig();
        throw new Error((resp && resp.msg) || "读取云存储配置失败");
      },
      (err) => {
        const localConfig = loadLocalCloudStorageConfig();
        if (localConfig) return localConfig;
        throw err || new Error("读取云存储配置失败");
      },
    );
  }
  return Promise.resolve(loadLocalCloudStorageConfig());
}

function loadLocalCloudStorageConfig() {
  try {
    const saved = JSON.parse(localStorage.getItem(CLOUD_STORAGE_KEY) || "null");
    return saved && typeof saved === "object" ? saved : null;
  } catch (_) {
    return null;
  }
}

function missingCloudStorageFields(config) {
  const missing = [];
  if (!String(config.endpoint || "").trim()) missing.push("Endpoint");
  if (!String(config.bucket || "").trim()) missing.push("Bucket");
  if (!String(config.accessKeyId || "").trim()) missing.push("Access Key ID");
  if (!String(config.secretAccessKey || "").trim()) missing.push("Secret Access Key");
  return missing;
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
  if (isImageFile(file)) return `![${name}](${url})`;
  return `[${name}](${url})`;
}

function isImageFile(file) {
  const type = String((file && file.type) || "");
  if (type.startsWith("image/")) return true;
  return /\.(avif|bmp|gif|jpe?g|png|svg|webp)$/i.test(String((file && file.name) || ""));
}

function readFileAsDataURL(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      resolve(String(reader.result || ""));
    };
    reader.onerror = () => {
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

function textToDoc(PM, value) {
  const lines = String(value || "").replace(/\r\n/g, "\n").split("\n");
  const blocks = lines.length ? lines : [""];
  return PM.schema.node(
    "doc",
    null,
    blocks.map((line) => PM.schema.node("paragraph", null, line ? [PM.schema.text(line)] : null)),
  );
}

function loadMemos() {
  const saved = loadJSON(MEMOS_STORAGE_KEY, null);
  if (Array.isArray(saved)) return saved;
  const memos = seedMemos();
  saveMemos(memos);
  return memos;
}

function saveMemos(memos) {
  localStorage.setItem(MEMOS_STORAGE_KEY, JSON.stringify(memos));
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
    if (memo.archived) return;
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

function parseTaskLine(line) {
  const match = String(line || "").match(TASK_LINE_REGEX);
  if (!match) return null;
  return {
    checked: match[2].toLowerCase() === "x",
    text: match[4].trim(),
  };
}

function memoSourceText(lines, lineIndex) {
  const before = lines
    .slice(0, lineIndex)
    .reverse()
    .find((line) => line.trim() && !parseTaskLine(line));
  const fallback = lines.find((line) => line.trim() && !parseTaskLine(line));
  return compactText(cleanMemoLine(before || fallback || "") || "仅包含任务的 memo", 84);
}

function cleanMemoLine(line) {
  return String(line || "")
    .replace(/^#{1,6}\s+/, "")
    .replace(/^>\s?/, "")
    .replace(/^\s*[-*+]\s+/, "")
    .replace(/^\s*\d+\.\s+/, "")
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

function renderMemoMarkdown(content) {
  const lines = String(content || "").replace(/\r\n/g, "\n").split("\n");
  let html = "";
  let code = [];
  let inCode = false;
  let listType = "";

  const closeList = () => {
    if (!listType) return;
    html += `</${listType}>`;
    listType = "";
  };

  lines.forEach((line, index) => {
    if (line.trim().startsWith("```")) {
      if (inCode) {
        html += `<pre><code>${escapeHTML(code.join("\n"))}</code></pre>`;
        code = [];
        inCode = false;
      } else {
        closeList();
        inCode = true;
      }
      return;
    }

    if (inCode) {
      code.push(line);
      return;
    }

    if (!line.trim()) {
      closeList();
      html += '<div class="memo-markdown-gap"></div>';
      return;
    }

    const standaloneResource = parseStandaloneMarkdownResource(line);
    if (standaloneResource) {
      closeList();
      html += standaloneResource.type === "image"
        ? renderMemoImageBlock(standaloneResource)
        : renderMemoFileBlock(standaloneResource);
      return;
    }

    const task = parseTaskLine(line);
    if (task) {
      closeList();
      html += `
        <label class="memo-task-line">
          <input type="checkbox" data-task-line="${index}" ${task.checked ? "checked" : ""} />
          <span>${inlineMarkdown(task.text)}</span>
        </label>
      `;
      return;
    }

    const unorderedMatch = line.match(/^\s*[-*+]\s+(.*)$/);
    if (unorderedMatch) {
      if (listType !== "ul") {
        closeList();
        listType = "ul";
        html += "<ul>";
      }
      html += `<li>${inlineMarkdown(unorderedMatch[1])}</li>`;
      return;
    }

    const orderedMatch = line.match(/^\s*\d+\.\s+(.*)$/);
    if (orderedMatch) {
      if (listType !== "ol") {
        closeList();
        listType = "ol";
        html += "<ol>";
      }
      html += `<li>${inlineMarkdown(orderedMatch[1])}</li>`;
      return;
    }

    closeList();

    const heading = line.match(/^(#{1,3})\s+(.*)$/);
    if (heading) {
      const level = heading[1].length;
      html += `<h${level + 2}>${inlineMarkdown(heading[2])}</h${level + 2}>`;
      return;
    }

    const quote = line.match(/^>\s?(.*)$/);
    if (quote) {
      html += `<blockquote>${inlineMarkdown(quote[1])}</blockquote>`;
      return;
    }

    html += `<p>${inlineMarkdown(line)}</p>`;
  });

  closeList();
  if (inCode) html += `<pre><code>${escapeHTML(code.join("\n"))}</code></pre>`;
  return html;
}

function inlineMarkdown(value) {
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
  return html;
}

function parseStandaloneMarkdownResource(line) {
  const match = String(line || "").match(/^\s*(!?)\[([^\]]*)\]\(([^)]+)\)\s*$/);
  if (!match) return null;

  const url = match[3].trim();
  if (!url) return null;

  const type = match[1] === "!" ? "image" : "file";
  const label = (match[2] || "").trim() || fileDisplayName("", url);
  if (type === "file" && !isFileAttachment(label, url)) return null;

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
  const body = `
    <span class="memo-file-block-icon">${SVG.paperclip}</span>
    <span class="memo-file-block-text">
      <span class="memo-file-block-name">${escapeHTML(name)}</span>
      <span class="memo-file-block-url">${escapeHTML(compactFileURL(resource.url))}</span>
    </span>
  `;

  if (href === "#") {
    return `<div class="memo-file-block">${body}</div>`;
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
  const body = `${SVG.paperclip}<span>${escapeHTML(name)}</span>`;

  if (href === "#") {
    return `<span class="memo-file-token">${body}</span>`;
  }
  return `<a class="memo-file-token" href="${escapeAttr(href)}" target="_blank" rel="noreferrer">${body}</a>`;
}

function isFileAttachment(label, url) {
  const pattern = /\.(?:7z|aac|apk|avi|csv|dmg|docx?|flac|gz|heic|ics|json|key|log|m4a|mkv|mov|mp3|mp4|numbers|pages|pdf|pptx?|rar|rtf|tar|txt|wav|webm|xlsx?|xml|yaml|yml|zip)(?:[?#].*)?$/i;
  if (/^(local:\/\/|blob:|data:)/i.test(String(url || ""))) return true;
  return pattern.test(String(label || "")) || pattern.test(String(url || ""));
}

function fileDisplayName(label, url) {
  const raw = String(label || "").trim() || String(url || "").trim();
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
  const url = String(value || "").trim();
  if (/^(https?:|local:\/\/|blob:)/i.test(url)) return url;
  if (/^data:image\//i.test(url)) return url;
  return "";
}

function safeUrl(value) {
  const url = String(value || "").trim();
  if (/^(https?:|mailto:|local:\/\/|blob:)/i.test(url)) return url;
  return "#";
}

function compactText(value, length) {
  const text = String(value || "").replace(/\s+/g, " ").trim();
  return text.length > length ? `${text.slice(0, length - 1)}...` : text;
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

function createId() {
  return `memo_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`;
}

function updateVersion(el, versionRef) {
  const setValue = (value) => {
    el.textContent = value ? `版本 ${value}` : "版本读取中";
  };
  if (versionRef && versionRef.__isRef) {
    setValue(versionRef.value);
    versionRef._subscribe({
      onChange(value) {
        setValue(value);
      },
      onPatch() {
        setValue(versionRef.value);
      },
    });
    return;
  }
  setValue(versionRef || "");
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
