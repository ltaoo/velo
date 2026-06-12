import {
  DEFAULT_VISIBILITY,
  VISIBILITY,
  collectTags,
  isMemoFenceClosingLine,
  isMemoFenceLine,
  maskMemoInlineCode,
  memoReferenceAlias,
  memoTitle,
  parseMemoFenceLine,
} from "../../domain/memos.js";
import {
  CLOUD_STORAGE_KEY,
  activeCloudStorageConfig,
  assetReference,
  cloudStorageById as findCloudStorageById,
  missingCloudStorageFields,
  normalizeCloudStorageSettings,
  resolveAssetUrl as resolveAssetUrlWithSettings,
} from "../../domain/storage.js";
import { formatShortDate } from "./memo-date.js";
import { closestElement, escapeHTML } from "./memo-utils.js";

const EDITOR_SETTINGS_STORAGE_KEY = "demo-desktop:settings:editor:v1";
const EDITOR_SETTINGS_API = "/api/settings/editor";
const EDITOR_SETTINGS_SAVE_API = "/api/settings/editor/save";

let cloudStorageSettingsCache = null;

function defaultEditorSettings() {
  return {
    calendarWeekStart: "monday",
    fileEditor: defaultFileEditor(),
    fileEditorRules: defaultFileEditorRules(),
    vimMode: false,
  };
}

function normalizeEditorSettings(value) {
  const raw = value && typeof value === "object" ? value : {};
  const rules = Array.isArray(raw.fileEditorRules) ? normalizeFileEditorRules(raw.fileEditorRules) : defaultFileEditorRules();
  return {
    calendarWeekStart: raw.calendarWeekStart === "sunday" ? "sunday" : "monday",
    fileEditor: normalizeFileEditor(raw.fileEditor),
    fileEditorRules: rules,
    vimMode: raw.vimMode === true,
  };
}

function defaultFileEditor() {
  return {
    id: "code",
    name: "VS Code",
    path: "",
  };
}

function noneFileEditor() {
  return {
    id: "none",
    name: "不打开",
    path: "",
  };
}

function defaultFileEditorRules() {
  const code = defaultFileEditor();
  const none = noneFileEditor();
  return [
    { extension: ".js", editor: code },
    { extension: ".ts", editor: code },
    { extension: ".tsx", editor: code },
    { extension: ".jsx", editor: code },
    { extension: ".mp4", editor: none },
    { extension: ".mp3", editor: none },
  ];
}

function normalizeFileEditor(value) {
  const raw = value && typeof value === "object" ? value : {};
  const defaults = defaultFileEditor();
  const path = String(raw.path || "").trim();
  let id = String(raw.id || "").trim();
  if (!id && path) id = "app:" + path;
  if (!id && !path && !String(raw.name || "").trim()) return defaults;
  const name = String(raw.name || "").trim() || fileEditorNameFromPath(path) || editorNameFromID(id) || defaults.name;
  return {
    id: id || defaults.id,
    name,
    path,
  };
}

function normalizeFileEditorRules(value) {
  const seen = new Set();
  const rules = Array.isArray(value) ? value : [];
  return rules
    .map((rule) => {
      const extension = normalizeFileExtension(rule && rule.extension);
      if (!extension || seen.has(extension)) return null;
      seen.add(extension);
      return {
        extension,
        editor: normalizeFileEditor(rule && rule.editor),
      };
    })
    .filter(Boolean);
}

function normalizeFileExtension(value) {
  const text = String(value || "").trim().toLowerCase().replace(/^\.+/, "");
  if (!text || !/^[a-z0-9_-]+$/.test(text)) return "";
  return "." + text;
}

function editorNameFromID(id) {
  switch (String(id || "").trim()) {
    case "code":
      return "VS Code";
    case "code-insiders":
      return "VS Code Insiders";
    case "cursor":
      return "Cursor";
    case "trae":
      return "Trae";
    case "webstorm":
      return "WebStorm";
    case "idea":
      return "IntelliJ IDEA";
    case "nvim":
      return "Neovim";
    case "vim":
      return "Vim";
    case "emacs":
      return "Emacs";
    case "none":
      return "不打开";
    case "system":
      return "系统默认应用";
    default:
      return "";
  }
}

function fileEditorNameFromPath(path) {
  const value = String(path || "").trim();
  if (!value) return "";
  const parts = value.split(/[\\/]/);
  return (parts[parts.length - 1] || value).replace(/\.app$/i, "");
}

function loadEditorSettings() {
  return loadLocalEditorSettings();
}

function loadLocalEditorSettings() {
  try {
    return normalizeEditorSettings(JSON.parse(localStorage.getItem(EDITOR_SETTINGS_STORAGE_KEY) || "null"));
  } catch (_) {
    return defaultEditorSettings();
  }
}

function saveLocalEditorSettings(value) {
  const settings = normalizeEditorSettings(value);
  localStorage.setItem(EDITOR_SETTINGS_STORAGE_KEY, JSON.stringify(settings));
  return settings;
}

function loadEditorSettingsFromVault() {
  if (typeof invoke !== "function") {
    return Promise.resolve(loadLocalEditorSettings());
  }
  return invoke(EDITOR_SETTINGS_API, { method: "GET" }).then(
    function (resp) {
      if (resp && resp.code === 0 && resp.data && resp.data.config) {
        return saveLocalEditorSettings(resp.data.config);
      }
      if (resp && resp.code === 0) return loadLocalEditorSettings();
      throw new Error((resp && resp.msg) || "读取编辑器设置失败");
    },
    function (err) {
      const localSettings = loadLocalEditorSettings();
      if (localSettings) return localSettings;
      throw err || new Error("读取编辑器设置失败");
    },
  );
}

function saveEditorSettingsToVault(value) {
  const settings = normalizeEditorSettings(value);
  if (typeof invoke !== "function") {
    return Promise.resolve(saveLocalEditorSettings(settings));
  }
  return invoke(EDITOR_SETTINGS_SAVE_API, {
    method: "POST",
    args: settings,
  }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "保存编辑器设置失败");
    }
    const saved = normalizeEditorSettings(resp.data && resp.data.config ? resp.data.config : settings);
    saveLocalEditorSettings(saved);
    return saved;
  });
}

function createMiniEditor(host, options) {
  const editorOptions = options || {};
  if (!window.ProsemirrorEditor) return createFallbackEditor(host, editorOptions);
  const vimEnabled = editorOptions.vim === true;

  host.dataset.placeholder = editorOptions.placeholder || "";

  let editor = null;
  let api = null;
  editor = new window.ProsemirrorEditor({
    $el: host,
    mode: "mini",
    value: editorOptions.value || "",
    vim: vimEnabled,
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
    onCommit(instance, detail) {
      if (editorOptions.onCommit) return editorOptions.onCommit(api || instance, detail);
    },
    onDiscard(instance, detail) {
      if (editorOptions.onDiscard) return editorOptions.onDiscard(api || instance, detail);
    },
    onRemoveFile(file) {
      if (editorOptions.onRemoveFile) editorOptions.onRemoveFile(file);
    },
    onQuit(instance, detail) {
      if (editorOptions.onQuit) return editorOptions.onQuit(api || instance, detail);
    },
    onSave(instance, detail) {
      if (editorOptions.onWriteDraft) return editorOptions.onWriteDraft(api || instance, detail);
      if (editorOptions.onSave) return editorOptions.onSave(api || instance, detail);
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
    onWriteDraft(instance, detail) {
      if (editorOptions.onWriteDraft) return editorOptions.onWriteDraft(api || instance, detail);
      if (editorOptions.onSave) return editorOptions.onSave(api || instance, detail);
    },
  });

  const removePlugins = installMemoEditorPlugins(editor, editorOptions);
  const removeStatus = vimEnabled ? installVimStatus(host, editor, editorOptions.vimStatusHost) : function () {};
  const removeVimFocus = vimEnabled ? installVimEditingMode(host, editor) : function () {};
  const removeSubmit = installSubmitShortcut(host, editorOptions);
  const removeDrop = installFileDropHandler(host, editor);

  if (vimEnabled) setEditorVimMode(editor, "insert");
  syncEmptyState();

  api = {
    blur() {
      if (editor.view && editor.view.dom) editor.view.dom.blur();
    },
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
      if (vimEnabled) setEditorVimMode(editor, "insert");
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
  return api;

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
    createMemoClipboardPlugin(editor),
    createMemoReferencePlugin(editor, options || {}),
    createMemoSlashCommandPlugin(editor, options || {}),
    createMemoTagPickerPlugin(editor, options || {}),
    createMemoTimeSyntaxHighlightPlugin(editor),
    createMemoTimePickerPlugin(editor),
  ];
  editor.view.updateState(
    editor.view.state.reconfigure({
      plugins: plugins.concat(editor.view.state.plugins),
    }),
  );

  return function () {};
}

function createMemoClipboardPlugin(editor) {
  const PM = window.ProsemirrorMod;
  const key = new PM.PluginKey(editor.id + "-memoClipboard");

  return new PM.Plugin({
    key,
    props: {
      handleDOMEvents: {
        copy(view, event) {
          if (!event.clipboardData) return false;
          const selection = view.state.selection;
          if (!selection || selection.empty) return false;

          const text = view.state.doc.textBetween(selection.from, selection.to, "\n");
          event.preventDefault();
          event.clipboardData.setData("text/plain", text);
          return true;
        },
      },
    },
  });
}

function createMemoTimeSyntaxHighlightPlugin(editor) {
  const PM = window.ProsemirrorMod;
  const key = new PM.PluginKey(editor.id + "-memoTimeSyntaxHighlight");
  const timePattern = /(^|[\s([{（【「『])(::(?!:)(?:\d{4}(?:[-/]\d{1,2}(?:[-/]\d{1,2}(?:[ T]\d{1,2}:\d{2}(?::\d{2})?)?)?)?|(?:\d{1,2}:\d{2}(?::\d{2})?)|(?:[^\s<>()\[\]{}，。！？、；;,.]{1,32})))/g;

  return new PM.Plugin({
    key,
    props: {
      decorations(state) {
        const decorations = [];
        state.doc.descendants(function (node, pos) {
          if (!node.isText || !node.text) return;
          timePattern.lastIndex = 0;
          let match = null;
          while ((match = timePattern.exec(node.text))) {
            const from = pos + match.index + match[1].length;
            const to = from + match[2].length;
            decorations.push(
              PM.Decoration.inline(from, to, {
                class: "mini-time-token",
              }),
            );
          }
        });
        return decorations.length
          ? PM.DecorationSet.create(state.doc, decorations)
          : PM.DecorationSet.empty;
      },
    },
  });
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
    if (isSelectionInsideMemoCode(state)) return null;

    const before = maskMemoInlineCode($from.parent.textBetween(0, $from.parentOffset, "\ufffc", "\ufffc"));
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
    if (isSelectionInsideMemoCode(state)) return null;

    const before = maskMemoInlineCode($from.parent.textBetween(0, $from.parentOffset, "\ufffc", "\ufffc"));
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

function isSelectionInsideMemoCode(state) {
  return isSelectionInsideMemoCodeNode(state) ||
    isSelectionInsideMemoFence(state) ||
    isSelectionInsideMemoInlineCode(state);
}

function isSelectionInsideMemoCodeNode(state) {
  const selection = state && state.selection;
  if (!selection || !selection.empty) return false;

  const $from = selection.$from;
  for (let depth = $from.depth; depth >= 0; depth -= 1) {
    const node = $from.node(depth);
    const type = node && node.type;
    const typeName = type && type.name;
    if (
      (type && type.spec && type.spec.code) ||
      typeName === "code_block" ||
      typeName === "codeBlock"
    ) {
      return true;
    }
  }
  return false;
}

function isSelectionInsideMemoFence(state) {
  const selection = state && state.selection;
  if (!selection || !selection.empty) return false;

  const textBeforeCursor = state.doc.textBetween(0, selection.from, "\n", "\n");
  const lines = textBeforeCursor.replace(/\r\n/g, "\n").split("\n");
  const previousLines = lines.slice(0, -1);
  const currentLine = lines[lines.length - 1] || "";
  let activeFence = null;

  previousLines.forEach(function (line) {
    const unquoted = unquoteMemoFenceLine(line);
    const fence = parseMemoFenceLine(unquoted);
    if (activeFence) {
      if (fence && isMemoFenceClosingLine(unquoted, activeFence)) activeFence = null;
    } else if (fence) {
      activeFence = fence;
    }
  });
  return Boolean(activeFence) || isMemoFenceLine(unquoteMemoFenceLine(currentLine));
}

function isSelectionInsideMemoInlineCode(state) {
  const selection = state && state.selection;
  if (!selection || !selection.empty) return false;

  const $from = selection.$from;
  if (!$from.parent || !$from.parent.isTextblock) return false;

  const before = $from.parent.textBetween(0, $from.parentOffset, "\ufffc", "\ufffc");
  return hasOpenMemoInlineCode(before);
}

function hasOpenMemoInlineCode(value) {
  const text = String(value || "");
  let activeDelimiter = "";
  let index = 0;

  while (index < text.length) {
    const tickIndex = text.indexOf("`", index);
    if (tickIndex < 0) break;
    if (isEscapedMemoBacktick(text, tickIndex)) {
      index = tickIndex + 1;
      continue;
    }

    let end = tickIndex + 1;
    while (end < text.length && text.charAt(end) === "`") end += 1;
    const delimiter = text.slice(tickIndex, end);
    if (!activeDelimiter) {
      activeDelimiter = delimiter;
    } else if (delimiter === activeDelimiter) {
      activeDelimiter = "";
    }
    index = end;
  }

  return Boolean(activeDelimiter);
}

function isEscapedMemoBacktick(text, index) {
  let slashCount = 0;
  for (let i = index - 1; i >= 0 && text.charAt(i) === "\\"; i -= 1) {
    slashCount += 1;
  }
  return slashCount % 2 === 1;
}

function unquoteMemoFenceLine(line) {
  let value = String(line || "");
  while (/^\s{0,3}>\s?/.test(value)) {
    value = value.replace(/^\s{0,3}>\s?/, "");
  }
  return value;
}

function createMemoTagPickerPlugin(editor, options) {
  const PM = window.ProsemirrorMod;
  const key = new PM.PluginKey(editor.id + "-memoTagPicker");

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
    if (isSelectionInsideMemoCode(state)) return null;

    const before = maskMemoInlineCode($from.parent.textBetween(0, $from.parentOffset, "\ufffc", "\ufffc"));
    const match = /(^|\s)#([\w\u4e00-\u9fa5-]*)$/u.exec(before);
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

  function selectItem(view, item) {
    const state = key.getState(view.state);
    if (!state || !state.active || !item) return false;

    const tag = String(item.tag || item.label || "").replace(/^#/, "").trim();
    if (!tag) return false;

    view.dispatch(
      view.state.tr
        .insertText("#" + tag + " ", state.from, state.to)
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
          };
        }

        const items = memoTagItems(options, trigger.query);
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
            class: "tag-query-range",
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
        className: "tag-picker-menu hidden",
        key,
        render(menu, pluginState) {
          renderTagPickerMenu(menu, pluginState);
        },
        onMouseDown(event, pluginState) {
          const option = closestElement(event.target, "[data-tag-picker-index]");
          if (!option) return false;
          event.preventDefault();
          const item = pluginState.items[Number(option.dataset.tagPickerIndex)];
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
      dismissedFrom: null,
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
    if (isSelectionInsideMemoCode(state)) return null;

    const before = maskMemoInlineCode($from.parent.textBetween(0, $from.parentOffset, "\ufffc", "\ufffc"));
    const indexes = [
      { trigger: "::", index: before.lastIndexOf("::") },
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
    if (before.charAt(found.index + found.trigger.length) === ":") return null;

    const query = before.slice(found.index + found.trigger.length);
    if (!isActiveMemoTimeQuery(query)) return null;

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
        .setMeta(key, { dismissTriggerFrom: true, type: "close" })
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
          if (!trigger) return empty();
          return {
            ...empty(trigger.key),
            dismissedFrom: meta.dismissTriggerFrom ? trigger.from : null,
          };
        }

        const trigger = findTrigger(newState);
        if (!trigger) return empty();
        if (trigger.from === value.dismissedFrom) {
          return {
            ...empty(value.dismissedKey),
            dismissedFrom: value.dismissedFrom,
            from: trigger.from,
            query: trigger.query,
            to: trigger.to,
            trigger: trigger.trigger,
          };
        }
        if (trigger.key === value.dismissedKey && !transaction.docChanged) {
          return {
            ...empty(value.dismissedKey),
            dismissedFrom: value.dismissedFrom,
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
          dismissedFrom: null,
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

function isActiveMemoTimeQuery(query) {
  const value = String(query || "");
  if (!value) return true;
  if (value.charAt(0) === ":") return false;
  if (/[\r\n]/.test(value)) return false;
  if (/[<>()\[\]{}，。！？、；;,.]/u.test(value)) return false;
  if (!/\s/u.test(value)) return true;

  const dateTime = /^(\d{4}[-/]\d{1,2}[-/]\d{1,2})[ T](\d{0,2}(?::\d{0,2}(?::\d{0,2})?)?)$/u.exec(value);
  return Boolean(dateTime);
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
    { icon: ">", label: "引用", detail: "插入引用块", keywords: "quote", text: "> \n> " },
    { icon: "<>", label: "代码块", detail: "插入 fenced code", keywords: "code pre", text: "```\n\n```" },
    { icon: "SNIP", label: "代码片段", detail: "插入可搜索 snippet", keywords: "snippet snip code alias 代码片段", text: "```sh snippet 标题 | alias\n\n```" },
    {
      icon: "TBL",
      label: "表格",
      detail: "插入 Markdown 表格",
      keywords: "table",
      text: "| 列 1 | 列 2 |\n| --- | --- |\n|  |  |",
    },
    {
      icon: "GRID",
      label: "图片布局",
      detail: "插入 grid 图片布局",
      keywords: "image layout gallery grid photos 九宫格 图片 微博",
      text: ":::image-layout grid\n![图片 1]()\n![图片 2]()\n![图片 3]()\n:::",
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

function memoTagItems(options, query) {
  const rawQuery = String(query || "").replace(/^#/, "").trim();
  const keyword = rawQuery.toLowerCase();
  const source = memoTagSource(options);
  const seen = new Set();
  const items = [];

  source.forEach(function (entry) {
    const item = normalizeMemoTagItem(entry);
    if (!item || seen.has(item.key)) return;
    if (keyword && !item.key.includes(keyword)) return;
    seen.add(item.key);
    items.push(item);
  });

  if (rawQuery && isMemoTagName(rawQuery) && !items.length && !seen.has(rawQuery.toLowerCase())) {
    items.push({
      count: 0,
      detail: "新标签",
      key: rawQuery.toLowerCase(),
      label: "#" + rawQuery,
      tag: rawQuery,
    });
  }

  return items.slice(0, 8);
}

function memoTagSource(options) {
  const explicit = typeof options.tagItems === "function"
    ? options.tagItems()
    : options.tagItems;
  if (Array.isArray(explicit)) return explicit;

  const memos = typeof options.memoItems === "function"
    ? options.memoItems()
    : options.memoItems;
  return collectTags(Array.isArray(memos) ? memos : []);
}

function normalizeMemoTagItem(entry) {
  let tag = "";
  let count = 0;
  let detail = "";

  if (Array.isArray(entry)) {
    tag = entry[0];
    count = Number(entry[1] || 0) || 0;
  } else if (entry && typeof entry === "object") {
    tag = entry.tag || entry.label || entry.value || entry.name || "";
    count = Number(entry.count || entry.total || 0) || 0;
    detail = entry.detail || "";
  } else {
    tag = entry;
  }

  tag = String(tag || "").replace(/^#/, "").trim();
  if (!isMemoTagName(tag)) return null;

  return {
    count,
    detail: detail || (count ? count + " 条 memo" : "标签"),
    key: tag.toLowerCase(),
    label: "#" + tag,
    tag,
  };
}

function isMemoTagName(value) {
  return /^[\w\u4e00-\u9fa5-]+$/u.test(String(value || ""));
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

function renderTagPickerMenu(menu, pluginState) {
  if (!pluginState.items.length) {
    menu.innerHTML = '<div class="tag-picker-empty">没有匹配的标签</div>';
    return;
  }

  menu.innerHTML = pluginState.items
    .map(function (item, index) {
      return `
        <div class="tag-picker-option ${index === pluginState.selectedIndex ? "active" : ""}" data-tag-picker-index="${index}">
          <span class="tag-picker-mark">#</span>
          <span class="tag-picker-copy">
            <span class="tag-picker-label">${escapeHTML(item.label)}</span>
            <span class="tag-picker-detail">${escapeHTML(item.detail)}</span>
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
  const in30Minutes = addMemoTime(now, { minutes: 30 });
  const in1Hour = addMemoTime(now, { hours: 1 });
  const in1Day = addMemoTime(now, { days: 1 });
  const in1Week = addMemoTime(now, { days: 7 });
  const value = String(query || "").trim();
  const items = [
    { label: "当前时间", detail: formatMemoDateTime(now, true), value: formatMemoDateTime(now, true) },
    { label: "当前时分", detail: formatMemoTime(now, false), value: formatMemoTime(now, false) },
    { label: "30分钟后", detail: formatMemoDateTime(in30Minutes, false), value: formatMemoDateTime(in30Minutes, false) },
    { label: "1小时后", detail: formatMemoDateTime(in1Hour, false), value: formatMemoDateTime(in1Hour, false) },
    { label: "1天后", detail: formatMemoDateTime(in1Day, false), value: formatMemoDateTime(in1Day, false) },
    { label: "今天", detail: formatMemoDate(today), value: formatMemoDate(today) },
    { label: "明天", detail: formatMemoDate(tomorrow), value: formatMemoDate(tomorrow) },
    { label: "1周后", detail: formatMemoDateTime(in1Week, false), value: formatMemoDateTime(in1Week, false) },
  ];
  if (value) {
    items.unshift({ label: value, detail: "使用输入值", value });
  }
  return items;
}

function addMemoTime(date, delta) {
  const value = new Date(date);
  if (delta.days) value.setDate(value.getDate() + delta.days);
  if (delta.hours) value.setHours(value.getHours() + delta.hours);
  if (delta.minutes) value.setMinutes(value.getMinutes() + delta.minutes);
  return value;
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

function cloudStorageById(storageId) {
  return findCloudStorageById(currentCloudStorageSettings(), storageId);
}

function resolveAssetUrl(value) {
  return resolveAssetUrlWithSettings(value, currentCloudStorageSettings());
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
      return;
    }
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
      const save = options.onWriteDraft || options.onSave;
      if (!save) return;
      event.preventDefault();
      save();
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
    blur() {
      textarea.blur();
    },
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

export {
  EDITOR_SETTINGS_STORAGE_KEY,
  cloudStorageById,
  createMiniEditor,
  defaultEditorSettings,
  fileInfoToUploadURL,
  filesToMarkdown,
  insertPlainTextIntoEditor,
  loadEditorSettingsFromVault,
  loadEditorSettings,
  normalizeFileEditor,
  normalizeFileEditorRules,
  normalizeEditorSettings,
  refreshCloudStorageSettings,
  resolveAssetUrl,
  saveEditorSettingsToVault,
  uploadErrorMessage,
};
