const MEMOS_STORAGE_KEY = "demo-desktop:memos:items:v1";
const DRAFT_STORAGE_KEY = "demo-desktop:memos:draft:v1";
const CLOUD_STORAGE_KEY = "demo-desktop:settings:cloud-storage:v1";
const DEFAULT_VISIBILITY = "PRIVATE";

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

function mountMemosHome(root, options = {}) {
  const state = {
    activeFilter: "all",
    activeTag: "",
    editingId: "",
    editDraft: "",
    editVisibility: DEFAULT_VISIBILITY,
    memos: loadMemos(),
    query: "",
    sortDesc: true,
    toastTimer: null,
    updateInfo: null,
    updateMessage: "",
    updateProgress: 0,
    updateStatus: "idle",
    visibility: DEFAULT_VISIBILITY,
  };

  let composerEditor = null;
  let editEditor = null;

  root.innerHTML = shellTemplate();

  const els = {
    attachInput: root.querySelector("[data-attach-input]"),
    composerHost: root.querySelector("[data-composer-host]"),
    composerStatus: root.querySelector("[data-composer-status]"),
    composerVimStatus: root.querySelector("[data-composer-vim-status]"),
    createButton: root.querySelector('[data-action="createMemo"]'),
    feedCount: root.querySelector("[data-feed-count]"),
    memoList: root.querySelector("[data-memo-list]"),
    pinnedList: root.querySelector("[data-pinned-list]"),
    searchInput: root.querySelector("[data-search-input]"),
    stats: root.querySelector("[data-stats]"),
    tagList: root.querySelector("[data-tag-list]"),
    tagSummary: root.querySelector("[data-tag-summary]"),
    toast: root.querySelector("[data-toast]"),
    updateActions: root.querySelector("[data-update-actions]"),
    updateMessage: root.querySelector("[data-update-message]"),
    updateProgress: root.querySelector("[data-update-progress]"),
    updateProgressBar: root.querySelector("[data-update-progress-bar]"),
    updateRelease: root.querySelector("[data-update-release]"),
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
    vimStatusHost: els.composerVimStatus,
  });

  renderAll();
  renderUpdate();
  renderComposerStatus(composerEditor.getText());
  bindGoMessages();
  refreshStorageForRender();
  window.addEventListener("focus", refreshStorageForRender);

  root.addEventListener("click", handleClick);
  root.addEventListener("input", handleInput);
  root.addEventListener("change", handleChange);

  return {
    destroy() {
      root.removeEventListener("click", handleClick);
      root.removeEventListener("input", handleInput);
      root.removeEventListener("change", handleChange);
      window.removeEventListener("focus", refreshStorageForRender);
      if (state.toastTimer) window.clearTimeout(state.toastTimer);
      if (composerEditor) composerEditor.destroy();
      if (editEditor) editEditor.destroy();
      root.innerHTML = "";
    },
    setVersion(version) {
      updateVersion(els.version, version);
    },
  };

  function refreshStorageForRender() {
    refreshCloudStorageSettings().then(function () {
      renderAll();
    }, function () {});
  }

  function handleClick(event) {
    const command = event.target.closest("[data-command]");
    if (command && root.contains(command)) {
      runComposerCommand(command.dataset.command);
      return;
    }

    const filter = event.target.closest("[data-filter]");
    if (filter && root.contains(filter)) {
      state.activeFilter = filter.dataset.filter;
      state.activeTag = "";
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
      case "downloadUpdate":
        downloadUpdate();
        break;
      case "editMemo":
        startEdit(memoId);
        break;
      case "checkUpdate":
        checkUpdate();
        break;
      case "openSettings":
        openSettings();
        break;
      case "applyUpdate":
        applyUpdate();
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
        return;
      }
      if (payload.type !== "download_progress") return;
      state.updateProgress = Math.round(payload.percentage || 0);
      var speed = payload.speed || 0;
      var speedText =
        speed > 1048576
          ? (speed / 1048576).toFixed(1) + " MB/s"
          : (speed / 1024).toFixed(0) + " KB/s";
      state.updateMessage =
        "正在下载更新... " + state.updateProgress + "% (" + speedText + ")";
      renderUpdate();
    });
  }

  function callApi(path) {
    return invoke(path, { method: "GET" });
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

  async function checkUpdate() {
    state.updateStatus = "checking";
    state.updateMessage = "正在检查更新...";
    state.updateInfo = null;
    state.updateProgress = 0;
    renderUpdate();

    try {
      var resp = await callApi("/api/update/check");
      if (!resp || resp.code !== 0) {
        state.updateStatus = "idle";
        state.updateMessage = (resp && resp.msg) || "检查失败";
        showToast(state.updateMessage);
        renderUpdate();
        return;
      }
      var data = resp.data || {};
      if (data.hasUpdate) {
        state.updateStatus = "checked";
        state.updateInfo = data;
        state.updateMessage = "发现新版本: " + data.version;
      } else {
        state.updateStatus = "idle";
        state.updateMessage = "当前已是最新版本 (" + data.currentVersion + ")";
      }
    } catch (err) {
      state.updateStatus = "idle";
      state.updateMessage = "检查失败: " + err;
    }
    renderUpdate();
  }

  async function downloadUpdate() {
    state.updateStatus = "downloading";
    state.updateProgress = 0;
    state.updateMessage = "正在下载更新... 0%";
    renderUpdate();

    try {
      var resp = await callApi("/api/update/download");
      if (!resp || resp.code !== 0) {
        state.updateStatus = "checked";
        state.updateMessage = (resp && resp.msg) || "下载失败";
        showToast(state.updateMessage);
        renderUpdate();
        return;
      }
      var data = resp.data || {};
      if (data.success) {
        state.updateProgress = 100;
        state.updateStatus = "downloaded";
        state.updateMessage = "下载完成，可以应用更新";
      } else {
        state.updateStatus = "checked";
        state.updateMessage = "下载失败: " + (data.error || "未知错误");
      }
    } catch (err) {
      state.updateStatus = "checked";
      state.updateMessage = "下载失败: " + err;
    }
    renderUpdate();
  }

  async function applyUpdate() {
    state.updateStatus = "applying";
    state.updateMessage = "正在应用更新并重启...";
    renderUpdate();

    try {
      var resp = await callApi("/api/update/restart");
      if (!resp || resp.code !== 0) {
        state.updateStatus = "downloaded";
        state.updateMessage = (resp && resp.msg) || "应用失败";
        showToast(state.updateMessage);
        renderUpdate();
        return;
      }
      var data = resp.data || {};
      if (!data.success) {
        state.updateStatus = "downloaded";
        state.updateMessage = "应用失败: " + (data.error || "未知错误");
        renderUpdate();
      }
    } catch (err) {
      state.updateStatus = "downloaded";
      state.updateMessage = "应用失败: " + err;
      renderUpdate();
    }
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
    lines[lineIndex] = lines[lineIndex].replace(/^- \[[ xX]\]/, checked ? "- [x]" : "- [ ]");
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
    renderFilterButtons();
    renderStats();
    renderTags();
    renderPinned();
    renderFeed();
  }

  function renderFilterButtons() {
    root.querySelectorAll("[data-filter]").forEach((button) => {
      button.classList.toggle("is-active", button.dataset.filter === state.activeFilter && !state.activeTag);
    });
  }

  function renderStats() {
    const active = state.memos.filter((memo) => !memo.archived);
    const archived = state.memos.length - active.length;
    const publicCount = active.filter((memo) => memo.visibility === "PUBLIC").length;
    const tags = collectTags(active);

    els.stats.innerHTML = [
      statTemplate("全部", active.length),
      statTemplate("公开", publicCount),
      statTemplate("标签", tags.length),
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
              <article class="memo-pinned-item">
                <div class="memo-pinned-content memo-content">${renderMemoMarkdown(memo.content)}</div>
                <small>${formatShortDate(memo.createdAt)}</small>
              </article>
            `,
          )
          .join("")
      : '<div class="memo-empty-mini">暂无置顶</div>';
  }

  function renderUpdate() {
    els.updateActions.innerHTML = updateActionsTemplate(state.updateStatus);
    els.updateMessage.textContent = state.updateMessage || "可手动检查新版本";
    els.updateProgress.hidden =
      state.updateStatus !== "downloading" && state.updateProgress <= 0;
    els.updateProgressBar.style.width = state.updateProgress + "%";
    els.updateRelease.innerHTML = state.updateInfo
      ? updateReleaseTemplate(state.updateInfo)
      : "";
  }

  function renderFeed() {
    if (editEditor) {
      editEditor.destroy();
      editEditor = null;
    }

    const memos = visibleMemos();
    els.feedCount.textContent = `${memos.length} 条`;
    els.memoList.innerHTML = memos.length
      ? memos.map((memo) => memoTemplate(memo, state.editingId)).join("")
      : emptyFeedTemplate();

    if (state.editingId) {
      const memo = findMemo(state.editingId);
      const host = els.memoList.querySelector("[data-edit-host]");
      const statusHost = els.memoList.querySelector("[data-edit-vim-status]");
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
          vimStatusHost: statusHost,
        });
        editEditor.focus();
      }
    }
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

  function findMemo(memoId) {
    return state.memos.find((memo) => memo.id === memoId);
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
            <h1>Inbox</h1>
            <p>捕捉、整理、回看</p>
          </div>
          <button class="memo-icon-text-button" type="button" data-action="sortMemos" title="排序">
            ${SVG.sort}
            <span>排序</span>
          </button>
        </header>

        <section class="memo-composer" aria-label="Create memo">
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
          <div class="memo-inspector-title">概览</div>
          <div class="memo-stats" data-stats></div>
        </section>
        <section class="memo-inspector-section">
          <div class="memo-inspector-title">置顶</div>
          <div class="memo-pinned-list" data-pinned-list></div>
        </section>
        <section class="memo-inspector-section">
          <div class="memo-inspector-title">版本与更新</div>
          <div class="memo-version" data-version></div>
          <div class="memo-update-panel">
            <div class="memo-update-actions" data-update-actions></div>
            <div class="memo-update-progress" data-update-progress hidden>
              <div class="memo-update-progress-bar" data-update-progress-bar></div>
            </div>
            <div class="memo-update-message" data-update-message></div>
            <div class="memo-update-release" data-update-release></div>
          </div>
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

function updateActionsTemplate(status) {
  if (status === "checking") {
    return '<button class="memo-secondary-button" type="button" disabled>检查中...</button>';
  }
  if (status === "checked") {
    return `
      <button class="memo-secondary-button" type="button" data-action="checkUpdate">重新检查</button>
      <button class="memo-primary-button" type="button" data-action="downloadUpdate">下载更新</button>
    `;
  }
  if (status === "downloading") {
    return '<button class="memo-secondary-button" type="button" disabled>下载中...</button>';
  }
  if (status === "downloaded") {
    return `
      <button class="memo-secondary-button" type="button" data-action="checkUpdate">重新检查</button>
      <button class="memo-primary-button" type="button" data-action="applyUpdate">应用更新并重启</button>
    `;
  }
  if (status === "applying") {
    return '<button class="memo-secondary-button" type="button" disabled>应用中...</button>';
  }
  return '<button class="memo-secondary-button" type="button" data-action="checkUpdate">检查更新</button>';
}

function updateReleaseTemplate(info) {
  return `
    <div class="memo-update-version">新版本 ${escapeHTML(info.version || "")}</div>
    <div class="memo-update-current">当前版本 ${escapeHTML(info.currentVersion || "")}</div>
    <div class="memo-update-notes">${escapeHTML(info.releaseNotes || "暂无更新说明")}</div>
  `;
}

function memoTemplate(memo, editingId) {
  const visibility = VISIBILITY[memo.visibility] || VISIBILITY[DEFAULT_VISIBILITY];
  const tags = extractTags(memo.content);
  const archived = memo.archived;
  const editing = memo.id === editingId;

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

function editTemplate(memo) {
  return `
    <div class="memo-inline-editor">
      <div class="memo-editor-host is-inline" data-edit-host></div>
      <div class="memo-inline-actions">
        <div class="memo-inline-status-line" data-edit-vim-status></div>
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
    if (!files.length) return;
    event.preventDefault();
    event.stopPropagation();
    insertFilesIntoEditor(editor, files);
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
          const option = event.target.closest("[data-slash-command-index]");
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
          const option = event.target.closest("[data-time-picker-index]");
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
  if (!String(config.endpoint || "").trim()) missing.push(isLocalCloudStorage(config) ? "本地根目录" : "Endpoint");
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
  return {
    destroy() {
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

    const taskMatch = line.match(/^- \[([ xX])\]\s+(.*)$/);
    if (taskMatch) {
      closeList();
      const checked = taskMatch[1].toLowerCase() === "x";
      html += `
        <label class="memo-task-line">
          <input type="checkbox" data-task-line="${index}" ${checked ? "checked" : ""} />
          <span>${inlineMarkdown(taskMatch[2])}</span>
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
  const body = `
    <span class="memo-file-block-icon">${SVG.paperclip}</span>
    <span class="memo-file-block-text">
      <span class="memo-file-block-name">${escapeHTML(name)}</span>
      <span class="memo-file-block-url">${escapeHTML(compactFileURL(displayUrl))}</span>
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

document.addEventListener("DOMContentLoaded", function () {
  var root = document.getElementById("root");
  if (!root) {
    console.error("[Render] Root element not found");
    return;
  }

  var memoApp = mountMemosHome(root);

  invoke("/api/app", { method: "GET" }).then(function (res) {
    if (res && res.code === 0 && res.data) {
      memoApp.setVersion(res.data.version);
    }
  });

  invoke("/api/window/state/restore?name=desktop", { method: "GET" });

  var snapshotTimer = setInterval(function () {
    invoke("/api/window/state/snapshot?name=desktop", { method: "GET" });
  }, 3000);

  window.addEventListener("beforeunload", function () {
    clearInterval(snapshotTimer);
    var xhr = new XMLHttpRequest();
    xhr.open("GET", "/api/window/state/snapshot?name=desktop", false);
    xhr.send();
    memoApp.destroy();
  });
});
