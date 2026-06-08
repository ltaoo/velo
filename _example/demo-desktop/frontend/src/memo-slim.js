const MEMOS_STORAGE_KEY = "demo-desktop:memos:items:v1";
const DEFAULT_VISIBILITY = "PRIVATE";

const SVG = {
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
  const state = {
    error: "",
    loading: false,
    memos: [],
    saving: false,
  };

  root.innerHTML = slimTemplate();

  const els = {
    form: root.querySelector("[data-slim-form]"),
    input: root.querySelector("[data-slim-input]"),
    list: root.querySelector("[data-slim-list]"),
    submit: root.querySelector("[data-slim-submit]"),
  };

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

    const memos = state.memos.filter(function (memo) {
      return memo && !memo.archived;
    });
    if (memos.length === 0) {
      renderState("还没有 memo");
      return;
    }

    els.list.innerHTML = memos.map(memoItemTemplate).join("");
  }

  function renderState(message) {
    els.list.innerHTML = `<div class="memo-slim-state">${escapeHTML(message || "")}</div>`;
  }
}

function slimTemplate() {
  return `
    <main class="memo-slim-shell">
      <form class="memo-slim-form" data-slim-form>
        <textarea class="memo-slim-input" data-slim-input placeholder="记录 memo..." rows="4"></textarea>
        <button class="memo-slim-submit" type="submit" data-slim-submit title="发布" aria-label="发布">
          ${SVG.send}
        </button>
      </form>
      <section class="memo-slim-list" data-slim-list aria-label="Memo list"></section>
    </main>
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

function createMemoInVault(content, visibility) {
  if (typeof invoke !== "function") {
    const now = new Date().toISOString();
    return Promise.resolve({
      archived: false,
      content,
      createdAt: now,
      id: createId(),
      pinned: false,
      updatedAt: "",
      visibility,
    });
  }
  return invoke("/api/memos/create", {
    method: "POST",
    args: {
      content,
      visibility,
    },
  }).then(function (resp) {
    if (!resp || resp.code !== 0 || !resp.data || !resp.data.memo) {
      throw new Error((resp && resp.msg) || "发布失败");
    }
    return resp.data.memo;
  });
}

function loadMemos() {
  const saved = loadJSON(MEMOS_STORAGE_KEY, []);
  return Array.isArray(saved) ? saved : [];
}

function saveMemos(memos) {
  if (typeof invoke === "function") return;
  localStorage.setItem(MEMOS_STORAGE_KEY, JSON.stringify(memos));
}

function normalizeMemoPayload(memo) {
  if (!memo || !memo.id) return null;
  return {
    archived: Boolean(memo.archived),
    content: String(memo.content || ""),
    createdAt: memo.createdAt || new Date().toISOString(),
    id: String(memo.id),
    pinned: Boolean(memo.pinned),
    updatedAt: memo.updatedAt || "",
    visibility: memo.visibility || DEFAULT_VISIBILITY,
  };
}

function loadJSON(key, fallback) {
  try {
    const raw = localStorage.getItem(key);
    return raw ? JSON.parse(raw) : fallback;
  } catch (_) {
    return fallback;
  }
}

function createId() {
  return `memo_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`;
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

function errorMessage(err) {
  return err && err.message ? err.message : String(err || "unknown error");
}
