import { DEFAULT_VISIBILITY, normalizeMemoPayload } from "./domain/memos.js";
import {
  createMemoInVault,
  errorMessage,
  loadMemosFromVault,
  saveMemos,
} from "./domain/memo-repository.js";
import { forgetPersistedWindow, registerWindowSession, setPersistedWindowFixed } from "./window-state.js";

const SVG = {
  pin:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 4 5 5-4 4v5l-2 2-5-5-5-5 2-2h5z"></path><path d="m9 15-5 5"></path></svg>',
  send:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m22 2-7 20-4-9-9-4z"></path><path d="M22 2 11 13"></path></svg>',
};

document.addEventListener("DOMContentLoaded", function () {
  const root = document.querySelector("#root");
  if (!root) {
    console.error("[MemoSlim] Root element not found");
    return;
  }
  mountMemoSlim(root);
});

function mountMemoSlim(root) {
  const params = new URLSearchParams(window.location.search);
  const state = {
    error: "",
    fixed: params.get("fixed") === "1",
    loading: false,
    memos: [],
    saving: false,
    toastTimer: null,
  };
  registerWindowSession({
    entryPage: "memo-slim.html",
    fixed: state.fixed,
    title: "Memos",
  });

  root.innerHTML = slimTemplate();

  const els = {
    fixedButton: root.querySelector('[data-window-control="toggleFixed"]'),
    form: root.querySelector("[data-slim-form]"),
    input: root.querySelector("[data-slim-input]"),
    list: root.querySelector("[data-slim-list]"),
    submit: root.querySelector("[data-slim-submit]"),
    toast: root.querySelector("[data-toast]"),
  };

  root.addEventListener("click", function (event) {
    const control = closestElement(event.target, "[data-window-control]");
    if (!control || !root.contains(control)) return;
    runWindowControl(control.dataset.windowControl);
  });
  els.form.addEventListener("submit", function (event) {
    event.preventDefault();
    createMemo();
  });
  els.input.addEventListener("keydown", function (event) {
    if (event.key !== "Enter" || (!event.metaKey && !event.ctrlKey)) return;
    event.preventDefault();
    createMemo();
  });
  window.addEventListener("focus", refreshMemos);

  applyFixedState();
  refreshMemos();
  window.requestAnimationFrame(function () {
    els.input.focus();
  });

  function refreshMemos() {
    state.loading = true;
    renderList();
    loadMemosFromVault().then(
      function (memos) {
        state.error = "";
        state.memos = memos.map(normalizeMemoPayload).filter(Boolean);
        renderList();
      },
      function (err) {
        state.error = "读取 memo 失败: " + errorMessage(err);
      },
    ).finally(function () {
      state.loading = false;
      renderList();
    });
  }

  function createMemo() {
    if (state.saving) return;
    const content = els.input.value;
    if (!content.trim()) {
      els.input.focus();
      return;
    }

    state.saving = true;
    els.submit.disabled = true;
    createMemoInVault(content, DEFAULT_VISIBILITY).then(
      function (memo) {
        state.error = "";
        const normalized = normalizeMemoPayload(memo);
        if (normalized) {
          state.memos = [normalized].concat(state.memos);
          saveMemos(state.memos);
        }
        els.input.value = "";
        renderList();
        window.requestAnimationFrame(function () {
          els.input.focus();
        });
      },
      function (err) {
        state.error = "发布失败: " + errorMessage(err);
        renderList();
      },
    ).finally(function () {
      state.saving = false;
      els.submit.disabled = false;
    });
  }

  function renderList() {
    if (state.error) {
      renderState(state.error);
      return;
    }

    if (state.loading && state.memos.length === 0) {
      renderState("正在加载 memo...");
      return;
    }

    const memos = state.memos
      .filter(function (memo) {
        return memo && !memo.archived;
      })
      .sort(sortSlimMemos);
    if (memos.length === 0) {
      renderState("还没有 memo");
      return;
    }

    els.list.innerHTML = memos.map(memoItemTemplate).join("");
  }

  function renderState(message) {
    els.list.innerHTML = `<div class="memo-slim-state">${escapeHTML(message || "")}</div>`;
  }

  function runWindowControl(control) {
    switch (control) {
      case "openFull":
        openFullMemos();
        break;
      case "toggleFixed":
        state.fixed = !state.fixed;
        applyFixedState();
        setPersistedWindowFixed(state.fixed).catch(function () {});
        break;
      default:
        break;
    }
  }

  function openFullMemos() {
    if (typeof invoke !== "function") {
      window.open("/desktop", "_blank", "noopener");
      return;
    }

    invoke("/api/open_window?pathname=%2Fdesktop", { method: "GET" }).then(
      function (resp) {
        if (!resp || resp.code !== 0) {
          showToast((resp && resp.msg) || "打开完整版失败");
          return;
        }
        forgetPersistedWindow().finally(function () {
          callNativeWindow("__velo/window/close").catch(function () {
            window.close();
          });
        });
      },
      function (err) {
        showToast("打开完整版失败: " + err);
      },
    );
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
    els.fixedButton.setAttribute("aria-label", state.fixed ? "取消固定" : "固定在所有窗口上方");
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

function sortSlimMemos(a, b) {
  const left = slimMemoTime(a);
  const right = slimMemoTime(b);
  if (left !== right) return right - left;
  return String(b.id || "").localeCompare(String(a.id || ""));
}

function slimMemoTime(memo) {
  const time = new Date((memo && (memo.updatedAt || memo.createdAt)) || "").getTime();
  return Number.isFinite(time) ? time : 0;
}

function slimTemplate() {
  return `
    <div class="memo-window-shell memo-slim-shell velo-drag" data-velo-drag>
      <header class="memo-window-titlebar memo-slim-titlebar velo-drag" data-velo-drag>
        <div class="memo-window-native-controls" aria-hidden="true"></div>
        <div class="memo-window-drag-region" aria-hidden="true"></div>
        <div class="memo-window-title-actions">
          <button class="memo-window-text-button velo-no-drag" type="button" data-window-control="openFull">完整版</button>
          <button class="memo-window-icon-button velo-no-drag" type="button" data-window-control="toggleFixed" title="固定在所有窗口上方" aria-label="固定在所有窗口上方" aria-pressed="false">
            ${SVG.pin}
          </button>
        </div>
      </header>
      <main class="memo-window-body memo-slim-body velo-no-drag">
        <form class="memo-slim-form" data-slim-form>
          <textarea class="memo-slim-input" data-slim-input placeholder="记录 memo..." rows="4"></textarea>
          <button class="memo-slim-submit" type="submit" data-slim-submit title="发布" aria-label="发布">
            ${SVG.send}
          </button>
        </form>
        <section class="memo-slim-list" data-slim-list aria-label="Memo list"></section>
      </main>
      <div class="memo-toast" data-toast role="status"></div>
    </div>
  `;
}

function memoItemTemplate(memo) {
  return `
    <article class="memo-slim-item ${memo.pinned ? "is-pinned" : ""}" data-memo-id="${escapeAttr(memo.id)}">
      <div class="memo-slim-meta">
        ${memo.pinned ? '<span class="memo-slim-pin">置顶</span>' : ""}
        <time class="memo-slim-time" datetime="${escapeAttr(memo.updatedAt || memo.createdAt)}">${escapeHTML(formatDate(memo.updatedAt || memo.createdAt))}</time>
      </div>
      <div class="memo-slim-content">${escapeHTML(memo.content)}</div>
    </article>
  `;
}

function formatDate(value) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return String(value);
  return date.toLocaleString([], {
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    month: "2-digit",
  });
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
