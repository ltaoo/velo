import { DEFAULT_VISIBILITY } from "./memos.js";
import { normalizeProjectID } from "./projects.js";

const memoryDrafts = [];

export const COMPOSER_DRAFT_ID = "draft_composer";

export function memoEditDraftId(memoId) {
  return "draft_memo_" + String(memoId || "").trim();
}

export function normalizeMemoDraftPayload(draft) {
  if (!draft || typeof draft !== "object") return null;
  const id = String(draft.id || "").trim();
  const kind = normalizeDraftKind(draft.kind);
  if (!id || !kind) return null;
  const memoId = String(draft.memoId || "").trim();
  if (kind === "memo-edit" && !memoId) return null;

  return {
    baseUpdatedAt: String(draft.baseUpdatedAt || "").trim(),
    content: String(draft.content || ""),
    id,
    kind,
    memoId,
    projectId: normalizeProjectID(draft.projectId),
    updatedAt: draft.updatedAt || "",
    visibility: draft.visibility || DEFAULT_VISIBILITY,
  };
}

export function loadMemoDraftsFromVault() {
  if (typeof globalThis.invoke !== "function") {
    return Promise.resolve(memoryDrafts.slice());
  }
  return globalThis.invoke("/api/memo-drafts", { method: "GET" }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "读取草稿失败");
    }
    const data = resp.data || {};
    return Array.isArray(data.drafts)
      ? data.drafts.map(normalizeMemoDraftPayload).filter(Boolean)
      : [];
  });
}

export function upsertMemoDraftInVault(draft) {
  const normalized = normalizeMemoDraftPayload(draft);
  if (!normalized) return Promise.reject(new Error("草稿无效"));

  if (typeof globalThis.invoke !== "function") {
    const next = {
      ...normalized,
      updatedAt: new Date().toISOString(),
    };
    const index = memoryDrafts.findIndex((item) => item.id === next.id);
    if (index >= 0) {
      memoryDrafts[index] = next;
    } else {
      memoryDrafts.push(next);
    }
    return Promise.resolve(next);
  }

  return globalThis.invoke("/api/memo-drafts/upsert", {
    method: "POST",
    args: normalized,
  }).then(function (resp) {
    if (!resp || resp.code !== 0 || !resp.data || !resp.data.draft) {
      throw new Error((resp && resp.msg) || "保存草稿失败");
    }
    return normalizeMemoDraftPayload(resp.data.draft);
  });
}

export function deleteMemoDraftInVault(id) {
  const draftId = String(id || "").trim();
  if (!draftId) return Promise.resolve({ success: true });

  if (typeof globalThis.invoke !== "function") {
    const index = memoryDrafts.findIndex((item) => item.id === draftId);
    if (index >= 0) memoryDrafts.splice(index, 1);
    return Promise.resolve({ success: true });
  }

  return globalThis.invoke("/api/memo-drafts/delete", {
    method: "POST",
    args: { id: draftId },
  }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "删除草稿失败");
    }
    return resp.data || { success: true };
  });
}

function normalizeDraftKind(kind) {
  const value = String(kind || "").trim();
  return value === "composer" || value === "memo-edit" ? value : "";
}
