import { mountMemoEditDialog } from "./pages/home/memo-dialog-edit.js";
import { registerWindowSession } from "./window-state.js";
import { loadEditorSettingsFromVault } from "./pages/home/memo-editor.js";
import { collectTags } from "./domain/memos.js";
import { createProjectInVault, errorMessage } from "./domain/memo-repository.js";
import { normalizeProjectPayload } from "./domain/projects.js";

document.addEventListener("DOMContentLoaded", function () {
  var root = document.querySelector("#root");
  if (!root) {
    console.error("[EditMemoWindow] Root element not found");
    return;
  }

  var params = new URLSearchParams(window.location.search);
  var memoId = (params.get("id") || "").trim();
  if (!memoId) {
    root.innerHTML = '<div class="memo-dialog"><section class="memo-dialog-panel"><p>缺少 memo id</p></section></div>';
    return;
  }

  var state = {
    editorSettings: null,
    memo: null,
    memos: [],
    projects: [],
    toastTimer: null,
  };

  root.innerHTML = '<div class="memo-editor-loading">加载中...</div>';

  var windowSession = registerWindowSession({
    entryPage: "edit-memo-window.html",
    kind: "edit_memo_window",
    title: "编辑 Memo",
  });

  loadEditData();

  function loadEditData() {
    if (typeof invoke !== "function") {
      root.innerHTML = '<div class="memo-dialog"><section class="memo-dialog-panel"><p>请在 velo 桌面应用中打开</p></section></div>';
      return;
    }

    var editDataPromise = invoke("/api/memo-window/edit?id=" + encodeURIComponent(memoId), { method: "GET" }).then(
      function (resp) {
        var data = resp && resp.code === 0 ? resp.data || {} : {};
        if (!data.found || !data.memo) {
          throw new Error("找不到 memo");
        }
        state.memo = typeof data.memo === "string" ? JSON.parse(data.memo) : data.memo;
        state.memos = normalizeMemosPayload(data.memos);
        return data;
      }
    );

    var settingsPromise = loadEditorSettingsFromVault().then(
      function (settings) {
        state.editorSettings = settings;
      },
      function () {
        state.editorSettings = {};
      }
    );

    // Load projects from vault so the project dropdown is populated even
    // when the detached window didn't pass any projects.
    var projectsPromise = invoke("/api/projects", { method: "GET" }).then(
      function (resp) {
        if (resp && resp.code === 0 && resp.data) {
          var vaultProjects = resp.data.projects;
          if (Array.isArray(vaultProjects)) {
            state.projects = vaultProjects.map(function (p) {
              return typeof p === "string" ? JSON.parse(p) : p;
            });
          }
        }
      },
      function () {}
    );

    Promise.all([editDataPromise, settingsPromise, projectsPromise]).then(
      function () {
        if (!state.memo) {
          root.innerHTML = '<div class="memo-dialog"><section class="memo-dialog-panel"><p>找不到 memo</p></section></div>';
          return;
        }
        mountEditor();
      },
      function (err) {
        if (err && err.message === "找不到 memo") {
          root.innerHTML = '<div class="memo-dialog"><section class="memo-dialog-panel"><p>找不到 memo</p></section></div>';
          return;
        }
        root.innerHTML = '<div class="memo-dialog"><section class="memo-dialog-panel"><p>加载失败</p></section></div>';
      },
    );
  }

  function normalizeMemosPayload(raw) {
    if (!raw) return [];
    var list = typeof raw === "string" ? JSON.parse(raw) : raw;
    return Array.isArray(list) ? list : [];
  }

  function normalizeProjectsPayload(raw) {
    if (!raw) return [];
    var list = typeof raw === "string" ? JSON.parse(raw) : raw;
    return Array.isArray(list) ? list : [];
  }

  function mountEditor() {
    root.innerHTML = "";
    mountMemoEditDialog(root, {
      memo: state.memo,
      initialDraft: null,
      memos: state.memos,
      projects: state.projects,
      editorSettings: state.editorSettings,
      tagItems: function () {
        return collectTags(state.memos.filter(function (m) { return m && !m.archived; }));
      },
      onSaveComplete: function (savedMemoId) {
        if (typeof invoke === "function") {
          invoke("/api/memo-window/memo-saved", { method: "POST", args: { memoId: savedMemoId } }).catch(function () {});
          invoke("__velo/window/close").catch(function () {});
        }
      },
      onClose: function () {
        if (typeof invoke === "function") {
          invoke("__velo/window/close").catch(function () {});
        }
      },
      onDraftUpsert: function () {},
      onDraftDelete: function () {},
      showToast: showToast,
      resolveOrCreateProject: function (name) {
        return createProjectInVault(name).then(function (project) {
          var normalized = normalizeProjectPayload(project);
          if (normalized) {
            state.projects = state.projects.concat(normalized);
          }
          return normalized ? normalized.id : "";
        });
      },
    });
  }

  function showToast(message) {
    var toast = document.querySelector("[data-toast]");
    if (!toast) {
      toast = document.createElement("div");
      toast.className = "memo-toast";
      toast.dataset.toast = "";
      toast.setAttribute("role", "status");
      root.appendChild(toast);
    }
    toast.textContent = message;
    toast.classList.add("is-visible");
    if (state.toastTimer) window.clearTimeout(state.toastTimer);
    state.toastTimer = window.setTimeout(function () {
      toast.classList.remove("is-visible");
    }, 1800);
  }
});
