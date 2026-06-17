const SEARCH_DEBOUNCE_MS = 80;
const MAX_RESULTS = 18;
const WINDOW_WIDTH = 720;
const COLLAPSED_HEIGHT = 60;
const MAX_WINDOW_HEIGHT = 430;
const MAX_RESULTS_HEIGHT = 344;
const STATUS_HEIGHT = 26;

const state = {
  activeIndex: 0,
  expanded: null,
  items: [],
  loading: false,
  query: "",
  requestId: 0,
  status: "",
  windowHeight: 0,
};

const els = {
  close: document.querySelector('[data-action="close"]'),
  input: document.querySelector("[data-snippet-input]"),
  launcher: document.querySelector("[data-snippet-launcher]"),
  results: document.querySelector("[data-snippet-results]"),
  status: document.querySelector("[data-snippet-status]"),
};

let searchTimer = null;
let isClosing = false;

init();

function init() {
  bindEvents();
  keepWindowOnTop();
  resizeWindowTo(COLLAPSED_HEIGHT, false).then(render, render);
  focusInput();
}

function bindEvents() {
  if (els.close) {
    els.close.addEventListener("click", closeWindow);
  }
  if (els.input) {
    els.input.addEventListener("input", function () {
      state.query = els.input.value || "";
      scheduleSearch();
    });
    els.input.addEventListener("keydown", handleInputKeydown);
  }
  if (els.results) {
    els.results.addEventListener("mousemove", function (event) {
      const item = event.target.closest("[data-snippet-index]");
      if (!item) return;
      const index = Number(item.dataset.snippetIndex);
      if (!Number.isInteger(index) || index === state.activeIndex) return;
      state.activeIndex = index;
      render();
    });
    els.results.addEventListener("click", function (event) {
      const item = event.target.closest("[data-snippet-index]");
      if (!item) return;
      const index = Number(item.dataset.snippetIndex);
      if (!Number.isInteger(index)) return;
      state.activeIndex = index;
      activateActiveItem(event.shiftKey);
    });
  }
  window.addEventListener("focus", function () {
    isClosing = false;
    focusInput();
  });
  window.addEventListener("blur", hideWindow);
  document.addEventListener("keydown", handleGlobalKeydown);
}

function handleInputKeydown(event) {
  if (event.key === "Escape") {
    event.preventDefault();
    event.stopPropagation();
    handleEscape();
    return;
  }
  if (event.key === "ArrowDown") {
    event.preventDefault();
    moveActive(1);
    return;
  }
  if (event.key === "ArrowUp") {
    event.preventDefault();
    moveActive(-1);
    return;
  }
  if (event.key === "Enter") {
    event.preventDefault();
    activateActiveItem(event.shiftKey);
  }
}

function handleGlobalKeydown(event) {
  if (event.key !== "Escape") return;
  event.preventDefault();
  handleEscape();
}

function handleEscape() {
  closeWindow();
}

function keepWindowOnTop() {
  if (typeof invoke !== "function") return;
  invoke("__velo/window/set_always_on_top", {
    args: { onTop: true },
  }).catch(function () {});
}

function focusInput() {
  window.setTimeout(function () {
    if (!els.input) return;
    els.input.focus();
    els.input.select();
  }, 20);
}

function scheduleSearch() {
  if (searchTimer) window.clearTimeout(searchTimer);
  searchTimer = window.setTimeout(function () {
    searchTimer = null;
    runSearch(state.query);
  }, SEARCH_DEBOUNCE_MS);
}

function runSearch(query) {
  const command = parseLauncherCommand(query);
  if (!command) {
    state.requestId += 1;
    state.loading = false;
    state.items = [];
    state.activeIndex = 0;
    state.status = "";
    resizeWindowTo(COLLAPSED_HEIGHT, false).catch(function () {});
    render();
    return;
  }

  state.loading = true;
  state.status = "";
  const requestId = ++state.requestId;
  render();

  if (typeof invoke !== "function") {
    state.loading = false;
    state.items = [];
    state.status = "当前环境不可用";
    render();
    return;
  }

  const url = command.endpoint + "?q=" + encodeURIComponent(command.raw) + "&limit=" + MAX_RESULTS;
  invoke(url, { method: "GET" }).then(function (resp) {
    if (requestId !== state.requestId) return;
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "搜索失败");
    }
    const data = resp.data || {};
    state.items = Array.isArray(data.items) ? data.items : [];
    state.activeIndex = Math.max(0, Math.min(state.activeIndex, state.items.length - 1));
    state.loading = false;
    state.status = "";
    render();
  }, function (err) {
    if (requestId !== state.requestId) return;
    state.items = [];
    state.activeIndex = 0;
    state.loading = false;
    state.status = errorMessage(err);
    render();
  });
}

function parseLauncherCommand(value) {
  const raw = String(value || "").trim();
  const match = raw.match(/^(snippet|snip|link|links|url|链接)(?:\s+(.*))?$/i);
  if (!match) return null;
  const name = match[1].toLowerCase();
  const term = String(match[2] || "").trim();
  if (name === "link" || name === "links" || name === "url" || name === "链接") {
    return {
      endpoint: "/api/links/search",
      emptyLabel: "link",
      name,
      raw: term ? "link " + term : "link",
      term,
      type: "link",
    };
  }
  return {
    endpoint: "/api/snippets/search",
    emptyLabel: "snippet",
    name,
    raw: term ? "snippet " + term : "snippet",
    term,
    type: "snippet",
  };
}

function moveActive(delta) {
  if (!state.items.length) return;
  state.activeIndex = (state.activeIndex + delta + state.items.length) % state.items.length;
  render();
  scrollActiveIntoView();
}

function activateActiveItem(copyOnly) {
  const item = state.items[state.activeIndex];
  if (!item) return;
  if (item.url) {
    if (copyOnly) {
      copyActiveLink();
    } else {
      openActiveLink();
    }
    return;
  }
  copyActiveSnippet();
}

function copyActiveSnippet() {
  const item = state.items[state.activeIndex];
  if (!item) return;
  copyText(item.code || "").then(function () {
    state.status = "已复制 " + (item.command || item.title || "代码块");
    renderStatus();
    window.setTimeout(closeWindow, 160);
  }, function (err) {
    state.status = "复制失败: " + errorMessage(err);
    renderStatus();
  });
}

function openActiveLink() {
  const item = state.items[state.activeIndex];
  if (!item || !item.url) return;
  openLinkInDefaultBrowser(item.url).then(function () {
    state.status = "已打开 " + (item.label || item.url);
    renderStatus();
    window.setTimeout(closeWindow, 120);
  }, function (err) {
    state.status = "打开失败: " + errorMessage(err);
    renderStatus();
  });
}

function copyActiveLink() {
  const item = state.items[state.activeIndex];
  if (!item || !item.url) return;
  copyText(item.url).then(function () {
    state.status = "已复制 " + (item.label || item.url);
    renderStatus();
    window.setTimeout(closeWindow, 160);
  }, function (err) {
    state.status = "复制失败: " + errorMessage(err);
    renderStatus();
  });
}

function openLinkInDefaultBrowser(url) {
  if (typeof invoke !== "function") {
    window.open(url, "_blank", "noopener");
    return Promise.resolve();
  }
  return invoke("/api/external/open?confirm=false&url=" + encodeURIComponent(url), { method: "GET" }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "打开链接失败");
    }
    return resp;
  });
}

function closeWindow() {
  if (isClosing) return;
  isClosing = true;
  resetLauncherState();
  if (typeof invoke === "function") {
    invoke("__velo/window/hide", { args: {} }).catch(function () {
      try {
        window.close();
      } catch (_) {}
    });
    return;
  }
  try {
    window.close();
  } catch (_) {}
}

function hideWindow() {
  if (isClosing) return;
  resetLauncherState();
  if (typeof invoke !== "function") return;
  invoke("__velo/window/hide", { args: {} }).catch(function () {});
}

function resetLauncherState() {
  if (searchTimer) {
    window.clearTimeout(searchTimer);
    searchTimer = null;
  }
  state.requestId += 1;
  state.loading = false;
  state.items = [];
  state.activeIndex = 0;
  state.status = "";
  state.query = "";
  if (els.input) els.input.value = "";
  resizeWindowTo(COLLAPSED_HEIGHT, false).catch(function () {});
  render();
}

function resizeWindowTo(height, expanded) {
  const nextHeight = Math.max(COLLAPSED_HEIGHT, Math.min(MAX_WINDOW_HEIGHT, Math.round(height || COLLAPSED_HEIGHT)));
  if (state.expanded === expanded && state.windowHeight === nextHeight) return Promise.resolve();

  if (!expanded) {
    state.expanded = false;
    state.windowHeight = nextHeight;
    if (els.launcher) els.launcher.classList.remove("is-expanded");
    if (els.results) els.results.hidden = true;
    if (els.status) els.status.hidden = true;
  }

  const applyExpanded = function () {
    state.expanded = expanded;
    state.windowHeight = nextHeight;
    if (els.launcher) {
      els.launcher.classList.toggle("is-expanded", expanded);
    }
  };

  if (typeof invoke !== "function") {
    applyExpanded();
    return Promise.resolve();
  }

  return invoke("__velo/window/set_size", {
    args: {
      width: WINDOW_WIDTH,
      height: nextHeight,
    },
  }).then(function () {
    applyExpanded();
  }, function (err) {
    applyExpanded();
    throw err;
  });
}

function copyText(value) {
  const text = String(value || "");
  if (navigator.clipboard && navigator.clipboard.writeText) {
    return navigator.clipboard.writeText(text).catch(function () {
      return copyTextFallback(text);
    });
  }
  return copyTextFallback(text);
}

function copyTextFallback(value) {
  return new Promise(function (resolve, reject) {
    const textarea = document.createElement("textarea");
    textarea.value = value;
    textarea.setAttribute("readonly", "readonly");
    textarea.style.position = "fixed";
    textarea.style.left = "-9999px";
    textarea.style.top = "0";
    document.body.appendChild(textarea);
    textarea.focus();
    textarea.select();
    try {
      if (!document.execCommand("copy")) {
        throw new Error("copy command failed");
      }
      resolve();
    } catch (err) {
      reject(err);
    } finally {
      document.body.removeChild(textarea);
      focusInput();
    }
  });
}

function render() {
  renderResults();
  renderStatus();
}

function renderResults() {
  if (!els.results) return;
  const command = parseLauncherCommand(state.query);
  if (!command) {
    els.results.hidden = true;
    if (els.status) els.status.hidden = true;
    els.results.innerHTML = "";
    return;
  }

  const wasHidden = els.results.hidden;
  els.results.hidden = false;
  if (els.status) els.status.hidden = false;

  if (state.loading && !state.items.length) {
    els.results.innerHTML = '<div class="snippet-empty">搜索中</div>';
  } else if (!state.items.length) {
    els.results.innerHTML = '<div class="snippet-empty">没有匹配的 ' + escapeHTML(command.emptyLabel || "结果") + '</div>';
  } else {
    els.results.innerHTML = state.items.map(function (item, index) {
      if (item.url) return renderLinkResult(item, index);
      return renderSnippetResult(item, index);
    }).join("");
  }

  const desiredHeight = desiredExpandedWindowHeight();
  const shouldDeferReveal = wasHidden || !state.expanded;
  if (shouldDeferReveal) {
    els.results.style.visibility = "hidden";
    if (els.status) els.status.style.visibility = "hidden";
  }

  resizeWindowTo(desiredHeight, true).then(function () {
    els.results.style.visibility = "";
    if (els.status) els.status.style.visibility = "";
    scrollActiveIntoView();
  }, function () {
    els.results.style.visibility = "";
    if (els.status) els.status.style.visibility = "";
    scrollActiveIntoView();
  });
}

function renderSnippetResult(item, index) {
  const line = item.endLine && item.endLine !== item.startLine
    ? "L" + item.startLine + "-L" + item.endLine
    : "L" + item.startLine;
  const meta = [
    item.command || "",
    item.language || "",
    item.memoTitle || item.memoId || "",
    line,
  ].filter(Boolean).join(" · ");
  return [
    '<button id="snippet-result-' + index + '" class="snippet-result ' + (index === state.activeIndex ? "is-active" : "") + '" type="button" role="option" aria-selected="' + (index === state.activeIndex ? "true" : "false") + '" data-snippet-index="' + index + '">',
    '<span class="snippet-result-main">',
    '<span class="snippet-result-title">',
    '<span class="snippet-kind ' + (item.marked ? "is-snippet" : "is-code") + '">' + (item.marked ? "SNIP" : "CODE") + '</span>',
    '<span class="snippet-name">' + escapeHTML(item.title || item.command || "代码片段") + '</span>',
    '</span>',
    '<span class="snippet-meta">' + escapeHTML(meta) + '</span>',
    '</span>',
    '<pre class="snippet-code"><code>' + escapeHTML(compactCode(item.code || "")) + '</code></pre>',
    '</button>',
  ].join("");
}

function renderLinkResult(item, index) {
  const meta = [
    item.memoTitle || item.memoId || "",
    item.line ? "L" + item.line : "",
    item.syntax || "",
  ].filter(Boolean).join(" · ");
  return [
    '<button id="snippet-result-' + index + '" class="snippet-result ' + (index === state.activeIndex ? "is-active" : "") + '" type="button" role="option" aria-selected="' + (index === state.activeIndex ? "true" : "false") + '" data-snippet-index="' + index + '">',
    '<span class="snippet-result-main">',
    '<span class="snippet-result-title">',
    '<span class="snippet-kind is-link">LINK</span>',
    '<span class="snippet-name">' + escapeHTML(item.label || item.url || "超链接") + '</span>',
    '</span>',
    '<span class="snippet-meta">' + escapeHTML(meta) + '</span>',
    '</span>',
    '<pre class="snippet-code is-link"><code>' + escapeHTML(compactCode(item.url || "")) + '</code></pre>',
    '</button>',
  ].join("");
}

function desiredExpandedWindowHeight() {
  if (!els.results) return COLLAPSED_HEIGHT;
  const resultsHeight = Math.min(resultsContentHeight(), MAX_RESULTS_HEIGHT);
  return COLLAPSED_HEIGHT + resultsHeight + STATUS_HEIGHT;
}

function resultsContentHeight() {
  if (!els.results) return 0;
  const styles = window.getComputedStyle ? window.getComputedStyle(els.results) : null;
  const padding = styles
    ? (Number.parseFloat(styles.paddingTop) || 0) + (Number.parseFloat(styles.paddingBottom) || 0)
    : 0;
  const children = Array.prototype.slice.call(els.results.children || []);
  if (!children.length) return Math.max(els.results.scrollHeight || 0, 88);
  const contentHeight = children.reduce(function (total, child) {
    const childStyles = window.getComputedStyle ? window.getComputedStyle(child) : null;
    const margin = childStyles
      ? (Number.parseFloat(childStyles.marginTop) || 0) + (Number.parseFloat(childStyles.marginBottom) || 0)
      : 0;
    return total + child.getBoundingClientRect().height + margin;
  }, padding);
  return Math.ceil(Math.max(contentHeight, els.results.scrollHeight || 0, 88));
}

function renderStatus() {
  if (!els.status) return;
  if (state.status) {
    els.status.textContent = state.status;
    return;
  }
  els.status.textContent = state.items.length ? state.items.length + " 个结果" : "";
}

function scrollActiveIntoView() {
  if (!els.results) return;
  const active = els.results.querySelector(".snippet-result.is-active");
  if (active) active.scrollIntoView({ block: "nearest" });
}

function compactCode(value) {
  const text = String(value || "").trim();
  if (!text) return "";
  return text.length > 220 ? text.slice(0, 217) + "..." : text;
}

function escapeHTML(value) {
  return String(value || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

function errorMessage(err) {
  return err && err.message ? err.message : String(err || "unknown error");
}
