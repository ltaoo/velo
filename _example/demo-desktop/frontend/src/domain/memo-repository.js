import { DEFAULT_VISIBILITY, normalizeMemoPayload } from "./memos.js";
import { normalizeProjectID, normalizeProjectPayload } from "./projects.js";

export const MEMOS_STORAGE_KEY = "demo-desktop:memos:items:v1";
export const PROJECTS_STORAGE_KEY = "demo-desktop:memos:projects:v1";

export function loadMemos() {
  if (typeof globalThis.invoke === "function") return [];
  const saved = loadJSON(MEMOS_STORAGE_KEY, null);
  if (Array.isArray(saved)) return saved;
  const memos = seedMemos();
  saveMemos(memos);
  return memos;
}

export function loadProjects() {
  if (typeof globalThis.invoke === "function") return [];
  const saved = loadJSON(PROJECTS_STORAGE_KEY, null);
  return Array.isArray(saved) ? saved.map(normalizeProjectPayload).filter(Boolean) : [];
}

export function saveProjects(projects) {
  if (typeof globalThis.invoke === "function") return;
  localStorage.setItem(PROJECTS_STORAGE_KEY, JSON.stringify(projects));
}

export function loadProjectsFromVault() {
  if (typeof globalThis.invoke !== "function") {
    return Promise.resolve({ activeProjectId: "", projects: loadProjects() });
  }
  return globalThis.invoke("/api/projects", { method: "GET" }).then(function (resp) {
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

export function createProjectInVault(name, color) {
  if (typeof globalThis.invoke !== "function") {
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
  return globalThis.invoke("/api/projects/create", {
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

export function saveMemos(memos) {
  if (typeof globalThis.invoke === "function") return;
  localStorage.setItem(MEMOS_STORAGE_KEY, JSON.stringify(memos));
}

export function loadMemosFromVault() {
  if (typeof globalThis.invoke !== "function") {
    return Promise.resolve(loadMemos());
  }
  return globalThis.invoke("/api/memos", { method: "GET" }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "读取 memo 失败");
    }
    const data = resp.data || {};
    return Array.isArray(data.memos) ? data.memos : [];
  });
}

export function createMemoInVault(content, visibility, projectId, isPrivate, meta) {
  meta = meta || {};
  if (typeof globalThis.invoke !== "function") {
    const now = new Date().toISOString();
    return Promise.resolve(Object.assign({
      archived: false,
      content,
      createdAt: now,
      id: createId(),
      pinned: false,
      private: Boolean(isPrivate),
      projectId: normalizeProjectID(projectId),
      updatedAt: "",
      visibility,
    }, meta));
  }
  return globalThis.invoke("/api/memos/create", {
    method: "POST",
    args: Object.assign({
      content,
      private: Boolean(isPrivate),
      projectId: normalizeProjectID(projectId),
      visibility,
    }, meta),
  }).then(function (resp) {
    if (!resp || resp.code !== 0 || !resp.data || !resp.data.memo) {
      throw new Error((resp && resp.msg) || "发布失败");
    }
    return resp.data.memo;
  });
}

export function updateMemoInVault(id, patch) {
  if (typeof globalThis.invoke !== "function") {
    const memos = loadMemos();
    const index = memos.findIndex((memo) => memo && memo.id === id);
    const next = Object.assign({}, index >= 0 ? memos[index] : { id }, patch);
    if (index >= 0) {
      memos[index] = next;
      saveMemos(memos);
    }
    return Promise.resolve(next);
  }
  const args = { id };
  if (Object.prototype.hasOwnProperty.call(patch, "content")) args.content = patch.content;
  if (Object.prototype.hasOwnProperty.call(patch, "createdAt")) args.createdAt = patch.createdAt;
  if (Object.prototype.hasOwnProperty.call(patch, "projectId")) args.projectId = normalizeProjectID(patch.projectId);
  if (Object.prototype.hasOwnProperty.call(patch, "visibility")) args.visibility = patch.visibility;
  if (Object.prototype.hasOwnProperty.call(patch, "private")) args.private = Boolean(patch.private);
  if (Object.prototype.hasOwnProperty.call(patch, "pinned")) args.pinned = patch.pinned;
  if (Object.prototype.hasOwnProperty.call(patch, "archived")) args.archived = patch.archived;
  if (Object.prototype.hasOwnProperty.call(patch, "kind")) args.kind = patch.kind;
  if (Object.prototype.hasOwnProperty.call(patch, "taskId")) args.taskId = patch.taskId;
  if (Object.prototype.hasOwnProperty.call(patch, "updatedAt")) args.updatedAt = patch.updatedAt;
  if (Object.prototype.hasOwnProperty.call(patch, "alias")) args.alias = patch.alias;
  return globalThis.invoke("/api/memos/update", {
    method: "POST",
    args,
  }).then(function (resp) {
    if (!resp || resp.code !== 0 || !resp.data || !resp.data.memo) {
      throw new Error((resp && resp.msg) || "保存失败");
    }
    return resp.data.memo;
  });
}

export function deleteMemoInVault(id, options) {
  if (typeof globalThis.invoke !== "function") {
    return Promise.resolve({ success: true });
  }
  const args = { id };
  if (options && Object.prototype.hasOwnProperty.call(options, "cleanupAssets")) {
    args.cleanupAssets = Boolean(options.cleanupAssets);
  }
  if (options && Object.prototype.hasOwnProperty.call(options, "deleteTasks")) {
    args.deleteTasks = Boolean(options.deleteTasks);
  }
  return globalThis.invoke("/api/memos/delete", {
    method: "POST",
    args,
  }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "删除失败");
    }
    return resp.data || { success: true };
  });
}

export function loadMemoFromLocal(memoId) {
  const saved = loadJSON(MEMOS_STORAGE_KEY, []);
  const memos = Array.isArray(saved) ? saved : [];
  return {
    memo: memos.find((item) => item && item.id === memoId) || null,
    memos,
  };
}

export function errorMessage(err) {
  return err && err.message ? err.message : String(err || "unknown error");
}

export function loadJSON(key, fallback) {
  try {
    const raw = localStorage.getItem(key);
    return raw ? JSON.parse(raw) : fallback;
  } catch (_) {
    return fallback;
  }
}

export function seedMemos() {
  const now = Date.now();
  return [
    normalizeMemoPayload({
      archived: false,
      content: "把 #velo 的桌面示例做成 memo 工作台。\n- [x] 左侧过滤\n- [ ] ProseMirror mini editor\n\n本地优先，适合快速捕捉。",
      createdAt: new Date(now - 1000 * 60 * 35).toISOString(),
      id: createId(),
      pinned: true,
      updatedAt: "",
      visibility: DEFAULT_VISIBILITY,
    }),
    normalizeMemoPayload({
      archived: false,
      content: "#idea Memos 风格的首页应该先看到编辑器，再看到时间线。\n\n支持 #inbox、置顶、归档和全文搜索。",
      createdAt: new Date(now - 1000 * 60 * 60 * 5).toISOString(),
      id: createId(),
      pinned: false,
      updatedAt: "",
      visibility: "PROTECTED",
    }),
    normalizeMemoPayload({
      archived: false,
      content: "发布前检查：\n1. mini editor 可输入\n2. 标签可筛选\n3. 任务可以勾选\n\n[usememos](https://github.com/usememos/memos)",
      createdAt: new Date(now - 1000 * 60 * 60 * 24).toISOString(),
      id: createId(),
      pinned: false,
      updatedAt: "",
      visibility: "PUBLIC",
    }),
  ].filter(Boolean);
}

export function createId() {
  return `memo_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`;
}
