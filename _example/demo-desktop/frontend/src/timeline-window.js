import { buildMemoReferenceIndex, memoTitle, normalizeMemoPayload } from "./domain/memos.js";
import {
  loadMemoCommentsFromVault,
  normalizeMemoCommentPayload,
} from "./domain/memo-comments.js";
import {
  errorMessage,
  loadMemosFromVault,
} from "./domain/memo-repository.js";
import { renderMemoMarkdown } from "./pages/home/memo-markdown.js";
import { relativeTimeTemplate } from "./pages/home/memo-date.js";
import { loadTasks } from "./domain/tasks.js";
import { registerWindowSession } from "./window-state.js";

const SVG = {
  clock:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="12" cy="12" r="10"></circle><polyline points="12 6 12 12 16 14"></polyline></svg>',
  memo:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline></svg>',
  comment:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path></svg>',
  check:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><polyline points="20 6 9 17 4 12"></polyline></svg>',
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
  const state = {
    comments: [],
    error: "",
    expandedItemIds: new Set(),
    loading: false,
    memoRefIndex: null,
    memos: [],
    page: 1,
    tasks: [],
    query: "",
    selectedDate: todayKey(),
    toastTimer: null,
  };
  registerWindowSession({
    entryPage: "timeline-window.html",
    title: "时间线",
  });

  root.innerHTML = timelineShellTemplate();

  const els = {
    dateLabel: root.querySelector("[data-timeline-date]"),
    list: root.querySelector("[data-timeline-list]"),
    searchInput: root.querySelector("[data-timeline-search]"),
    toast: root.querySelector("[data-toast]"),
  };

  root.addEventListener("click", function (event) {
    const dateNav = closestElement(event.target, "[data-date-nav]");
    if (dateNav && root.contains(dateNav)) {
      navigateDate(dateNav.dataset.dateNav);
      return;
    }
    const expandBtn = closestElement(event.target, "[data-action=\"toggleExpand\"]");
    if (expandBtn && root.contains(expandBtn)) {
      toggleItemExpand(expandBtn.dataset.expandId);
      return;
    }
    const memoRef = closestElement(event.target, "[data-memo-ref-target]");
    if (memoRef && root.contains(memoRef)) {
      event.preventDefault();
      openMemoInWindow(memoRef.dataset.memoRefTarget);
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
    fragment.innerHTML = pageItems.map(function (item) { return timelineItemTemplate(item, state.expandedItemIds, state.memoRefIndex); }).join("")
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
      loadTasks(),
    ]).then(
      function (results) {
        state.error = "";
        state.memos = results[0].map(normalizeMemoPayload).filter(Boolean);
        state.comments = results[1].map(normalizeMemoCommentPayload).filter(Boolean);
        state.tasks = results[2].tasks || [];
        state.memoRefIndex = buildMemoReferenceIndex(state.memos);
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

  function toggleItemExpand(id) {
    var isExpanded = state.expandedItemIds.has(id);
    if (isExpanded) {
      state.expandedItemIds.delete(id);
    } else {
      state.expandedItemIds.add(id);
    }

    var safeId = CSS.escape(id);
    var button = root.querySelector(`[data-expand-id="${safeId}"]`);
    var collapse = button ? button.closest(".memo-list-collapse") : null;
    if (!collapse) return;

    var content = collapse.querySelector(".memo-content");
    if (!content) return;

    var lineH = parseFloat(getComputedStyle(content).lineHeight);
    var collapsedHeight = Math.round(lineH * 36);

    // Cancel any in-progress transition
    content.style.transition = "none";
    content.offsetHeight; // flush
    content.style.transition = "";

    collapse.classList.remove("is-short");

    if (isExpanded) {
      // COLLAPSE: expanded → collapsed
      content.style.maxHeight = content.scrollHeight + "px";
      content.offsetHeight;
      collapse.classList.remove("is-expanded");
      collapse.classList.add("is-collapsed");
      content.style.maxHeight = collapsedHeight + "px";

      if (button) {
        button.setAttribute("aria-expanded", "false");
        var span = button.querySelector("span");
        if (span) span.textContent = "展开";
      }
    } else {
      // EXPAND: collapsed → expanded
      content.style.maxHeight = collapsedHeight + "px";
      content.offsetHeight;
      collapse.classList.remove("is-collapsed");
      collapse.classList.add("is-expanded");
      content.style.maxHeight = content.scrollHeight + "px";

      if (button) {
        button.setAttribute("aria-expanded", "true");
        var span2 = button.querySelector("span");
        if (span2) span2.textContent = "收起";
      }
    }

    content.addEventListener("transitionend", function handler() {
      content.removeEventListener("transitionend", handler);
      if (!collapse.classList.contains("is-collapsed")) {
        content.style.maxHeight = "";
      }
      if (collapse.classList.contains("is-collapsed")) {
        var lines = parseInt(collapse.dataset.memoLines, 10);
        if (lines <= 36) {
          collapse.classList.add("is-short");
        }
      }
    });
  }

  function syncExpandControls() {
    var items = root.querySelectorAll("[data-memo-collapse]");
    items.forEach(function (item) {
      var content = item.querySelector(".memo-content");
      if (!content) return;
      var lines = parseInt(item.dataset.memoLines, 10);
      item.classList.remove("is-short");
      if (item.classList.contains("is-collapsed")) {
        if (lines <= 36) {
          item.classList.add("is-short");
          content.style.maxHeight = "";
        } else {
          var lineHeight = parseFloat(getComputedStyle(content).lineHeight);
          content.style.maxHeight = Math.round(lineHeight * 36) + "px";
        }
      } else {
        content.style.maxHeight = "";
      }
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
    els.list.innerHTML = visible.map(function (item) { return timelineItemTemplate(item, state.expandedItemIds, state.memoRefIndex); }).join("")
      + (hasMore ? '<div class="timeline-load-more">加载更多...</div>' : "");
    setTimeout(syncExpandControls, 0);
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

    state.tasks.forEach(function (task) {
      if (task.status !== "completed") return;
      if (!task.completedAt) return;
      if (memoDateKey(task.completedAt) !== dateKey) return;
      items.push({
        content: task.title,
        createdAt: task.completedAt,
        id: task.id,
        parentMemoId: task.id,
        parentTitle: "",
        type: "task",
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

  function callNativeWindow(method, args) {
    if (typeof invoke !== "function") {
      return Promise.reject(new Error("go bridge not available"));
    }
    return invoke(method, { args: args || {} });
  }

  function openMemoInWindow(memoId) {
    var target = state.memos.find(function (m) { return m && m.id === memoId; });
    if (!target) {
      showToast("找不到引用的 memo");
      return;
    }
    if (typeof invoke !== "function") {
      window.open("memo-window.html?id=" + encodeURIComponent(memoId), "_blank", "noopener");
      return;
    }
    invoke("/api/memo-window/open", {
      method: "POST",
      args: {
        memo: target,
        memos: state.memos,
      },
    }).catch(function (err) {
      showToast("打开 memo 失败: " + (err && err.message ? err.message : err));
    });
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

function timelineItemTemplate(item, expandedItemIds, memoRefIndex) {
  const timeHTML = relativeTimeTemplate(item.createdAt);
  var typeIcon, typeLabel;
  if (item.type === "comment") {
    typeIcon = SVG.comment;
    typeLabel = "评论";
  } else if (item.type === "task") {
    typeIcon = SVG.check;
    typeLabel = "完成任务";
  } else {
    typeIcon = SVG.memo;
    typeLabel = "memo";
  }
  const parentLabel = item.parentTitle && item.parentTitle.length > 20
    ? item.parentTitle.slice(0, 20) + "..."
    : item.parentTitle;
  const parentInfo = item.type === "comment" && item.parentTitle
    ? `<span class="timeline-item-parent">回复: ${escapeHTML(parentLabel)}</span>`
    : "";

  var cardContent;
  if (item.type === "task") {
    cardContent = escapeHTML(item.content);
  } else {
    cardContent = renderMemoMarkdown(item.content, {
      index: memoRefIndex || undefined,
      readonly: true,
      showLineNumbers: false,
      sourceId: item.id,
      sourceMemoId: item.type === "comment" ? item.parentMemoId : item.id,
      sourceCommentId: item.type === "comment" ? item.id : "",
      sourceType: item.type === "comment" ? "comment" : "memo",
    });
  }

  var cardBody;
  if (item.type === "task") {
    cardBody = `<div class="timeline-item-card">
      <div class="memo-content">${cardContent}</div>
    </div>`;
  } else {
    const isExpanded = expandedItemIds && expandedItemIds.has(item.id);
    const textLines = (item.content || "").split("\n").length;
    cardBody = `<div class="timeline-item-card">
      <div class="memo-list-collapse ${isExpanded ? "is-expanded" : "is-collapsed"}${!isExpanded && textLines <= 36 ? " is-short" : ""}" data-memo-collapse data-memo-lines="${textLines}">
        <div class="memo-content">${cardContent}</div>
        <button class="memo-expand-button" type="button" data-action="toggleExpand" data-expand-id="${escapeAttr(item.id)}" aria-expanded="${isExpanded ? "true" : "false"}">
          <span>${isExpanded ? "收起" : "展开"}</span>
          <svg viewBox="0 0 24 24" aria-hidden="true"><polyline points="6 9 12 15 18 9"></polyline></svg>
        </button>
      </div>
    </div>`;
  }

  return `
    <article class="timeline-item ${item.type === "comment" ? "is-comment" : ""}${item.type === "task" ? " is-task" : ""}">
      <div class="timeline-item-dot"></div>
      <div class="timeline-item-body">
        <div class="timeline-item-head">
          <span class="timeline-item-type">${typeIcon}<span>${typeLabel}</span></span>
          ${timeHTML}
        </div>
        ${parentInfo}
        ${cardBody}
      </div>
    </article>
  `;
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
