import { memoTitle, normalizeMemoPayload } from "./domain/memos.js";
import {
  loadMemoCommentsFromVault,
  normalizeMemoCommentPayload,
} from "./domain/memo-comments.js";
import {
  errorMessage,
  loadMemosFromVault,
} from "./domain/memo-repository.js";
import { renderMemoMarkdown } from "./pages/home/memo-markdown.js";
import { forgetPersistedWindow, registerWindowSession, setPersistedWindowFixed } from "./window-state.js";

const SVG = {
  pin:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 4 5 5-4 4v5l-2 2-5-5-5-5 2-2h5z"></path><path d="m9 15-5 5"></path></svg>',
  clock:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>',
  memo:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>',
  comment:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path></svg>',
};

document.addEventListener("DOMContentLoaded", function () {
  const root = document.querySelector("#root");
  if (!root) {
    console.error("[TimelineWindow] Root element not found");
    return;
  }
  mountTimelineWindow(root);
});

function mountTimelineWindow(root) {
  const PAGE_SIZE = 10;
  const params = new URLSearchParams(window.location.search);
  const state = {
    comments: [],
    error: "",
    fixed: params.get("fixed") === "1",
    loading: false,
    memos: [],
    page: 1,
    query: "",
    selectedDate: todayKey(),
    toastTimer: null,
  };
  registerWindowSession({
    entryPage: "timeline-window.html",
    fixed: state.fixed,
    title: "时间线",
  });

  root.innerHTML = timelineShellTemplate();

  const els = {
    dateLabel: root.querySelector("[data-timeline-date]"),
    fixedButton: root.querySelector('[data-window-control="toggleFixed"]'),
    list: root.querySelector("[data-timeline-list]"),
    searchInput: root.querySelector("[data-timeline-search]"),
    toast: root.querySelector("[data-toast]"),
  };

  root.addEventListener("click", function (event) {
    const control = closestElement(event.target, "[data-window-control]");
    if (control && root.contains(control)) {
      runWindowControl(control.dataset.windowControl);
      return;
    }
    const dateNav = closestElement(event.target, "[data-date-nav]");
    if (dateNav && root.contains(dateNav)) {
      navigateDate(dateNav.dataset.dateNav);
      return;
    }
  });

  if (els.searchInput) {
    els.searchInput.addEventListener("input", function () {
      state.query = els.searchInput.value.trim().toLowerCase();
      state.page = 1;
      renderTimeline();
    });
  }

  els.list.addEventListener("scroll", handleScroll);

  window.addEventListener("focus", refreshData);

  applyFixedState();
  renderDateLabel();
  refreshData();

  function handleScroll() {
    const list = els.list;
    if (!list) return;
    const threshold = 60;
    if (list.scrollTop + list.clientHeight >= list.scrollHeight - threshold) {
      loadMore();
    }
  }

  function loadMore() {
    const allItems = buildAllTimelineItems();
    const maxPage = Math.ceil(allItems.length / PAGE_SIZE);
    if (state.page >= maxPage) return;
    state.page++;
    appendPage();
  }

  function appendPage() {
    const allItems = buildAllTimelineItems();
    const start = (state.page - 1) * PAGE_SIZE;
    const end = state.page * PAGE_SIZE;
    const pageItems = allItems.slice(start, end);
    if (pageItems.length === 0) return;

    const hasMore = end < allItems.length;
    const existingLoader = els.list.querySelector(".timeline-load-more");
    if (existingLoader) existingLoader.remove();

    const fragment = document.createElement("div");
    fragment.innerHTML = pageItems.map(timelineItemTemplate).join("")
      + (hasMore ? '<div class="timeline-load-more">加载更多...</div>' : "");
    while (fragment.firstChild) {
      els.list.appendChild(fragment.firstChild);
    }
  }

  function refreshData() {
    state.loading = true;
    renderTimeline();
    Promise.all([
      loadMemosFromVault(),
      loadMemoCommentsFromVault(),
    ]).then(
      function (results) {
        state.error = "";
        state.memos = results[0].map(normalizeMemoPayload).filter(Boolean);
        state.comments = results[1].map(normalizeMemoCommentPayload).filter(Boolean);
        renderTimeline();
      },
      function (err) {
        state.error = "加载失败: " + errorMessage(err);
        renderTimeline();
      },
    ).finally(function () {
      state.loading = false;
      renderTimeline();
    });
  }

  function navigateDate(direction) {
    const current = dateFromKey(state.selectedDate);
    if (direction === "prev") {
      current.setDate(current.getDate() - 1);
    } else if (direction === "next") {
      current.setDate(current.getDate() + 1);
    } else if (direction === "today") {
      const now = new Date();
      current.setFullYear(now.getFullYear(), now.getMonth(), now.getDate());
    }
    state.selectedDate = formatDateKey(current);
    state.page = 1;
    renderDateLabel();
    renderTimeline();
  }

  function renderDateLabel() {
    if (!els.dateLabel) return;
    const date = dateFromKey(state.selectedDate);
    const today = todayKey();
    const label = state.selectedDate === today
      ? "今天"
      : date.toLocaleDateString("zh-CN", { month: "long", day: "numeric", weekday: "short" });
    els.dateLabel.textContent = label;
  }

  function renderTimeline() {
    if (state.error) {
      els.list.innerHTML = `<div class="timeline-state">${escapeHTML(state.error)}</div>`;
      return;
    }
    if (state.loading && state.memos.length === 0) {
      els.list.innerHTML = '<div class="timeline-state">正在加载...</div>';
      return;
    }

    const allItems = buildAllTimelineItems();
    if (allItems.length === 0) {
      els.list.innerHTML = '<div class="timeline-state">该日暂无内容</div>';
      return;
    }

    const visible = allItems.slice(0, state.page * PAGE_SIZE);
    const hasMore = visible.length < allItems.length;
    els.list.innerHTML = visible.map(timelineItemTemplate).join("")
      + (hasMore ? '<div class="timeline-load-more">加载更多...</div>' : "");
  }

  function buildAllTimelineItems() {
    const dateKey = state.selectedDate;
    const memoById = new Map(state.memos.map(function (m) { return [m.id, m]; }));
    const items = [];

    state.memos.forEach(function (memo) {
      if (memo.archived) return;
      if (memoDateKey(memo.createdAt) !== dateKey) return;
      items.push({
        content: memo.content,
        createdAt: memo.createdAt,
        id: memo.id,
        parentMemoId: memo.id,
        parentTitle: "",
        type: "memo",
      });
    });

    state.comments.forEach(function (comment) {
      if (!comment || !comment.memoId) return;
      if (memoDateKey(comment.createdAt) !== dateKey) return;
      const parent = memoById.get(comment.memoId);
      if (!parent) return;
      items.push({
        content: comment.content,
        createdAt: comment.createdAt,
        id: comment.id,
        parentMemoId: comment.memoId,
        parentTitle: memoTitle(parent),
        type: "comment",
      });
    });

    items.sort(function (a, b) {
      const ta = new Date(a.createdAt).getTime() || 0;
      const tb = new Date(b.createdAt).getTime() || 0;
      return tb - ta;
    });

    if (state.query) {
      const q = state.query;
      return items.filter(function (item) {
        return (item.content || "").toLowerCase().indexOf(q) !== -1
          || (item.parentTitle || "").toLowerCase().indexOf(q) !== -1;
      });
    }

    return items;
  }

  function runWindowControl(control) {
    switch (control) {
      case "toggleFixed":
        state.fixed = !state.fixed;
        applyFixedState();
        setPersistedWindowFixed(state.fixed).catch(function () {});
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
    els.fixedButton.setAttribute("title", state.fixed ? "取消固定" : "固定在所有窗口上方");
  }

  function callNativeWindow(method, args) {
    if (typeof invoke !== "function") {
      return Promise.reject(new Error("go bridge not available"));
    }
    return invoke(method, { args: args || {} });
  }

  function showToast(message) {
    if (!els.toast) return;
    els.toast.textContent = message;
    els.toast.classList.add("is-visible");
    if (state.toastTimer) window.clearTimeout(state.toastTimer);
    state.toastTimer = window.setTimeout(function () {
      els.toast.classList.remove("is-visible");
    }, 1800);
  }
}

function timelineShellTemplate() {
  return `
    <div class="memo-window-shell timeline-shell velo-drag" data-velo-drag>
      <header class="memo-window-titlebar timeline-titlebar velo-drag" data-velo-drag>
        <div class="memo-window-native-controls" aria-hidden="true"></div>
        <div class="memo-window-drag-region" aria-hidden="true"></div>
        <div class="memo-window-title-actions">
          <button class="memo-window-icon-button velo-no-drag" type="button" data-window-control="toggleFixed" title="固定在所有窗口上方" aria-label="固定在所有窗口上方" aria-pressed="false">
            ${SVG.pin}
          </button>
        </div>
      </header>
      <main class="memo-window-body timeline-body velo-no-drag">
        <div class="timeline-header">
          <div class="timeline-date-nav">
            <button class="timeline-date-btn" type="button" data-date-nav="prev" title="前一天" aria-label="前一天">&lsaquo;</button>
            <span class="timeline-date-label" data-timeline-date>今天</span>
            <button class="timeline-date-btn" type="button" data-date-nav="next" title="后一天" aria-label="后一天">&rsaquo;</button>
            <button class="timeline-date-btn timeline-today-btn" type="button" data-date-nav="today" title="回到今天" aria-label="回到今天">${SVG.clock}</button>
          </div>
          <input class="timeline-search" type="search" placeholder="搜索..." data-timeline-search />
        </div>
        <section class="timeline-list" data-timeline-list aria-label="时间线"></section>
      </main>
      <div class="memo-toast" data-toast role="status"></div>
    </div>
  `;
}

function timelineItemTemplate(item) {
  const time = formatTime(item.createdAt);
  const typeIcon = item.type === "comment" ? SVG.comment : SVG.memo;
  const typeLabel = item.type === "comment" ? "评论" : "memo";
  const parentLabel = item.parentTitle && item.parentTitle.length > 20
    ? item.parentTitle.slice(0, 20) + "..."
    : item.parentTitle;
  const parentInfo = item.type === "comment" && item.parentTitle
    ? `<span class="timeline-item-parent">回复: ${escapeHTML(parentLabel)}</span>`
    : "";
  const renderedContent = renderMemoMarkdown(item.content, {
    readonly: true,
    showLineNumbers: false,
    sourceId: item.id,
    sourceMemoId: item.type === "comment" ? item.parentMemoId : item.id,
    sourceCommentId: item.type === "comment" ? item.id : "",
    sourceType: item.type === "comment" ? "comment" : "memo",
  });

  return `
    <article class="timeline-item ${item.type === "comment" ? "is-comment" : ""}">
      <div class="timeline-item-dot"></div>
      <div class="timeline-item-body">
        <div class="timeline-item-head">
          <span class="timeline-item-type">${typeIcon}<span>${typeLabel}</span></span>
          <time class="timeline-item-time" datetime="${escapeAttr(item.createdAt)}">${time}</time>
        </div>
        ${parentInfo}
        <div class="timeline-item-card">
          <div class="memo-content">${renderedContent}</div>
        </div>
      </div>
    </article>
  `;
}

function formatTime(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function todayKey() {
  return formatDateKey(new Date());
}

function formatDateKey(date) {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function dateFromKey(value) {
  const match = String(value || "").match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if (!match) return new Date();
  return new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]));
}

function memoDateKey(createdAt) {
  if (!createdAt) return "";
  const date = new Date(createdAt);
  if (Number.isNaN(date.getTime())) return "";
  return formatDateKey(date);
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
  return escapeHTML(value);
}

function closestElement(target, selector) {
  if (!target || typeof target.closest !== "function") return null;
  return target.closest(selector);
}
