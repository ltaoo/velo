export const MEMO_COMMENTS_STORAGE_KEY = "demo-desktop:memos:comments:v1";

export function normalizeMemoCommentPayload(comment) {
  if (!comment || typeof comment !== "object") return null;
  const id = String(comment.id || "").trim();
  const memoId = String(comment.memoId || "").trim();
  if (!id || !memoId) return null;
  return {
    content: String(comment.content || ""),
    createdAt: comment.createdAt || new Date().toISOString(),
    id,
    memoId,
    path: String(comment.path || ""),
    private: Boolean(comment.private),
    reactions: Array.isArray(comment.reactions) ? comment.reactions.filter(String) : [],
    references: Array.isArray(comment.references) ? comment.references.map(String).filter(Boolean) : [],
    tags: Array.isArray(comment.tags) ? comment.tags.map(String).filter(Boolean) : [],
    updatedAt: comment.updatedAt || "",
    visibility: comment.visibility || "PRIVATE",
  };
}

export function loadMemoCommentsFromVault(memoId = "") {
  const targetMemoId = String(memoId || "").trim();
  if (typeof globalThis.invoke !== "function") {
    const comments = loadLocalMemoComments();
    return Promise.resolve(targetMemoId ? comments.filter((comment) => comment.memoId === targetMemoId) : comments);
  }
  const query = targetMemoId ? "?memoId=" + encodeURIComponent(targetMemoId) : "";
  return globalThis.invoke("/api/memo-comments" + query, { method: "GET" }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "读取评论失败");
    }
    const data = resp.data || {};
    return Array.isArray(data.comments)
      ? data.comments.map(normalizeMemoCommentPayload).filter(Boolean)
      : [];
  });
}

export function createMemoCommentInVault(memoId, content, visibility, isPrivate) {
  const targetMemoId = String(memoId || "").trim();
  const text = String(content || "");
  const vis = visibility || "PRIVATE";
  const priv = Boolean(isPrivate);
  if (!targetMemoId) return Promise.reject(new Error("memo id is required"));
  if (!text.trim()) return Promise.reject(new Error("comment content is required"));

  if (typeof globalThis.invoke !== "function") {
    const now = new Date().toISOString();
    const comment = normalizeMemoCommentPayload({
      content: text,
      createdAt: now,
      id: createMemoCommentId(),
      memoId: targetMemoId,
      private: priv,
      updatedAt: "",
      visibility: vis,
    });
    const comments = [comment].filter(Boolean).concat(loadLocalMemoComments());
    saveLocalMemoComments(comments);
    return Promise.resolve(comment);
  }

  return globalThis.invoke("/api/memo-comments/create", {
    method: "POST",
    args: {
      content: text,
      memoId: targetMemoId,
      private: priv,
      visibility: vis,
    },
  }).then(function (resp) {
    if (!resp || resp.code !== 0 || !resp.data || !resp.data.comment) {
      throw new Error((resp && resp.msg) || "评论失败");
    }
    return normalizeMemoCommentPayload(resp.data.comment);
  });
}

export function updateMemoCommentInVault(id, patch) {
  const commentId = String(id || "").trim();
  if (!commentId) return Promise.reject(new Error("comment id is required"));

  if (typeof globalThis.invoke !== "function") {
    let updated = null;
    const comments = loadLocalMemoComments().map(function (comment) {
      if (comment.id !== commentId) return comment;
      updated = normalizeMemoCommentPayload({
        ...comment,
        ...patch,
        updatedAt: new Date().toISOString(),
      });
      return updated;
    });
    saveLocalMemoComments(comments);
    return updated ? Promise.resolve(updated) : Promise.reject(new Error("comment not found"));
  }

  const args = { id: commentId };
  if (Object.prototype.hasOwnProperty.call(patch, "content")) args.content = patch.content;
  if (Object.prototype.hasOwnProperty.call(patch, "private")) args.private = Boolean(patch.private);
  if (Object.prototype.hasOwnProperty.call(patch, "reactions")) args.reactions = patch.reactions;
  return globalThis.invoke("/api/memo-comments/update", {
    method: "POST",
    args,
  }).then(function (resp) {
    if (!resp || resp.code !== 0 || !resp.data || !resp.data.comment) {
      throw new Error((resp && resp.msg) || "保存评论失败");
    }
    return normalizeMemoCommentPayload(resp.data.comment);
  });
}

export function deleteMemoCommentInVault(id, options) {
  const commentId = String(id || "").trim();
  if (!commentId) return Promise.resolve({ success: true });

  if (typeof globalThis.invoke !== "function") {
    saveLocalMemoComments(loadLocalMemoComments().filter((comment) => comment.id !== commentId));
    return Promise.resolve({ success: true });
  }

  const args = { id: commentId };
  if (options && Object.prototype.hasOwnProperty.call(options, "cleanupAssets")) {
    args.cleanupAssets = Boolean(options.cleanupAssets);
  }
  return globalThis.invoke("/api/memo-comments/delete", {
    method: "POST",
    args,
  }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "删除评论失败");
    }
    return resp.data || { success: true };
  });
}

function loadLocalMemoComments() {
  try {
    const raw = localStorage.getItem(MEMO_COMMENTS_STORAGE_KEY);
    const comments = raw ? JSON.parse(raw) : [];
    return Array.isArray(comments) ? comments.map(normalizeMemoCommentPayload).filter(Boolean) : [];
  } catch (_) {
    return [];
  }
}

function saveLocalMemoComments(comments) {
  localStorage.setItem(MEMO_COMMENTS_STORAGE_KEY, JSON.stringify(comments));
}

function createMemoCommentId() {
  return `comment_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`;
}
