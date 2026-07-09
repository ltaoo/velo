import { callNativeAPI } from "./native.js";
import { normalizeProjectID } from "./projects.js";

export const TASK_STATUS = {
  ARCHIVED: "archived",
  CANCELLED: "cancelled",
  COMPLETED: "completed",
  OPEN: "open",
};

export const TASK_PRIORITY = {
  HIGH: "high",
  LOW: "low",
  MEDIUM: "medium",
  NONE: "none",
};

export function normalizeTaskPayload(task) {
  if (!task || typeof task !== "object") return null;
  const id = normalizeTaskID(task.id);
  const title = String(task.title || "").trim();
  if (!id || !title) return null;
  return {
    cancelledAt: String(task.cancelledAt || ""),
    completedAt: String(task.completedAt || ""),
    contexts: normalizeStringList(task.contexts),
    createdAt: task.createdAt || new Date().toISOString(),
    dueAt: String(task.dueAt || ""),
    id,
    links: Array.isArray(task.links) ? task.links.map(normalizeTaskLink).filter(Boolean) : [],
    listId: normalizeTaskListID(task.listId),
    notes: String(task.notes || ""),
    noteRefs: Array.isArray(task.noteRefs) ? task.noteRefs.map(normalizeTaskNoteRef).filter(Boolean) : [],
    parentId: normalizeTaskID(task.parentId),
    path: String(task.path || ""),
    priority: normalizeTaskPriority(task.priority),
    projectId: normalizeProjectID(task.projectId),
    reminders: Array.isArray(task.reminders) ? task.reminders.map(normalizeTaskReminder).filter(Boolean) : [],
    repeat: normalizeTaskRepeat(task.repeat),
    source: normalizeTaskSource(task.source),
    startAt: String(task.startAt || ""),
    status: normalizeTaskStatus(task.status),
    subtaskIds: normalizeStringList(task.subtaskIds).map(normalizeTaskID).filter(Boolean),
    tags: normalizeStringList(task.tags),
    timezone: String(task.timezone || "UTC").trim() || "UTC",
    title,
    updatedAt: task.updatedAt || "",
  };
}

export function normalizeTaskSummary(task) {
  const normalized = normalizeTaskPayload(Object.assign({ notes: "" }, task));
  if (!normalized) return null;
  return {
    contexts: normalized.contexts,
    completedAt: normalized.completedAt,
    createdAt: normalized.createdAt,
    dueAt: normalized.dueAt,
    id: normalized.id,
    listId: normalized.listId,
    noteCount: Number.isFinite(Number(task.noteCount)) ? Number(task.noteCount) : normalized.noteRefs.length,
    parentId: normalized.parentId,
    path: normalized.path,
    priority: normalized.priority,
    projectId: normalized.projectId,
    reminders: normalized.reminders,
    source: normalized.source,
    startAt: normalized.startAt,
    status: normalized.status,
    subtaskCount: Number.isFinite(Number(task.subtaskCount)) ? Number(task.subtaskCount) : normalized.subtaskIds.length,
    tags: normalized.tags,
    title: normalized.title,
    updatedAt: normalized.updatedAt,
  };
}

export function normalizeTaskID(value) {
  return String(value || "").trim().replace(/[^A-Za-z0-9_-]+/g, "-").replace(/^-+|-+$/g, "");
}

export function normalizeTaskListID(value) {
  return normalizeTaskID(value) || "inbox";
}

export function normalizeTaskStatus(value) {
  const status = String(value || "").trim().toLowerCase();
  if (status === TASK_STATUS.COMPLETED) return TASK_STATUS.COMPLETED;
  if (status === TASK_STATUS.CANCELLED) return TASK_STATUS.CANCELLED;
  if (status === TASK_STATUS.ARCHIVED) return TASK_STATUS.ARCHIVED;
  return TASK_STATUS.OPEN;
}

export function normalizeTaskPriority(value) {
  const priority = String(value || "").trim().toLowerCase();
  if (priority === TASK_PRIORITY.LOW) return TASK_PRIORITY.LOW;
  if (priority === TASK_PRIORITY.MEDIUM) return TASK_PRIORITY.MEDIUM;
  if (priority === TASK_PRIORITY.HIGH) return TASK_PRIORITY.HIGH;
  return TASK_PRIORITY.NONE;
}

export function normalizeStringList(values) {
  const seen = new Set();
  const list = Array.isArray(values) ? values : [];
  return list.map((value) => String(value || "").trim()).filter(function (value) {
    if (!value || seen.has(value)) return false;
    seen.add(value);
    return true;
  });
}

export function normalizeTaskReminder(reminder) {
  if (!reminder || typeof reminder !== "object") return null;
  const type = String(reminder.type || "").trim().toLowerCase();
  if (!type) return null;
  return {
    at: String(reminder.at || ""),
    base: String(reminder.base || "").trim(),
    offsetMinutes: Number.isFinite(Number(reminder.offsetMinutes)) ? Number(reminder.offsetMinutes) : 0,
    type,
  };
}

export function normalizeTaskRepeat(repeat) {
  const value = repeat && typeof repeat === "object" ? repeat : {};
  return {
    end: normalizeTaskRepeatEnd(value.end),
    frequency: String(value.frequency || "none").trim().toLowerCase() || "none",
    interval: Number.isFinite(Number(value.interval)) ? Math.max(0, Number(value.interval)) : 0,
    weekdays: normalizeStringList(value.weekdays),
  };
}

export function normalizeTaskRepeatEnd(end) {
  const value = end && typeof end === "object" ? end : {};
  return {
    at: String(value.at || ""),
    count: Number.isFinite(Number(value.count)) ? Math.max(0, Number(value.count)) : 0,
    type: String(value.type || "").trim().toLowerCase(),
  };
}

export function normalizeTaskSource(source) {
  const value = source && typeof source === "object" ? source : {};
  return {
    commentId: String(value.commentId || "").trim(),
    commentPath: String(value.commentPath || "").trim(),
    line: Number.isFinite(Number(value.line)) ? Math.max(0, Number(value.line)) : 0,
    memoId: String(value.memoId || "").trim(),
    memoPath: String(value.memoPath || "").trim(),
    text: String(value.text || "").trim(),
    type: String(value.type || "").trim().toLowerCase(),
  };
}

export function normalizeTaskLink(link) {
  if (!link || typeof link !== "object") return null;
  const type = String(link.type || "").trim().toLowerCase();
  const id = String(link.id || "").trim();
  const url = String(link.url || "").trim();
  if (!type || (!id && !url)) return null;
  return {
    id,
    label: String(link.label || "").trim(),
    type,
    url,
  };
}

export function normalizeTaskNoteRef(ref) {
  if (!ref || typeof ref !== "object") return null;
  const memoId = String(ref.memoId || "").trim();
  if (!memoId) return null;
  return {
    createdAt: String(ref.createdAt || ""),
    memoId,
    role: String(ref.role || "note").trim().toLowerCase() || "note",
    sortOrder: Number.isFinite(Number(ref.sortOrder)) ? Number(ref.sortOrder) : 0,
  };
}

export function loadTasks() {
  return callNativeAPI("/api/tasks", { method: "GET" }).then(function (data) {
    return {
      index: data.index || null,
      tasks: Array.isArray(data.tasks) ? data.tasks.map(normalizeTaskSummary).filter(Boolean) : [],
    };
  });
}

export function getTask(id) {
  const taskId = normalizeTaskID(id);
  if (!taskId) return Promise.reject(new Error("task id is required"));
  return callNativeAPI("/api/tasks/get?id=" + encodeURIComponent(taskId), { method: "GET" }).then(function (data) {
    const task = normalizeTaskPayload(data.task);
    if (!task) throw new Error("task not found");
    return task;
  });
}

export function createTask(input) {
  const task = input && typeof input === "object" ? input : {};
  return callNativeAPI("/api/tasks/create", {
    method: "POST",
    args: task,
  }).then(function (data) {
    const normalized = normalizeTaskPayload(data.task);
    if (!normalized) throw new Error("create task failed");
    return normalized;
  });
}

export function updateTask(id, patch) {
  const taskId = normalizeTaskID(id);
  if (!taskId) return Promise.reject(new Error("task id is required"));
  return callNativeAPI("/api/tasks/update", {
    method: "POST",
    args: Object.assign({}, patch || {}, { id: taskId }),
  }).then(function (data) {
    const normalized = normalizeTaskPayload(data.task);
    if (!normalized) throw new Error("update task failed");
    return normalized;
  });
}

export function completeTask(id) {
  const taskId = normalizeTaskID(id);
  if (!taskId) return Promise.reject(new Error("task id is required"));
  return callNativeAPI("/api/tasks/complete", {
    method: "POST",
    args: { id: taskId },
  }).then(function (data) {
    const normalized = normalizeTaskPayload(data.task);
    if (!normalized) throw new Error("complete task failed");
    return normalized;
  });
}

export function deleteTask(id) {
  const taskId = normalizeTaskID(id);
  if (!taskId) return Promise.reject(new Error("task id is required"));
  return callNativeAPI("/api/tasks/delete", {
    method: "POST",
    args: { id: taskId },
  });
}

export function createTaskNote(taskId, input) {
  const id = normalizeTaskID(taskId);
  if (!id) return Promise.reject(new Error("task id is required"));
  return callNativeAPI("/api/tasks/notes/create", {
    method: "POST",
    args: Object.assign({}, input || {}, { taskId: id }),
  }).then(function (data) {
    const task = normalizeTaskPayload(data.task);
    if (!task) throw new Error("create task note failed");
    return {
      memo: data.memo || null,
      task,
    };
  });
}

export function extractTaskFromMemoLine(parentTaskId, memoId, lineIndex, options) {
  const id = normalizeTaskID(parentTaskId);
  if (!id) return Promise.reject(new Error("parent task id is required"));
  const line = Number(lineIndex);
  if (!Number.isFinite(line) || line < 0) return Promise.reject(new Error("line index is required"));
  return callNativeAPI("/api/tasks/extract-from-memo", {
    method: "POST",
    args: {
      lineIndex: line,
      memoId: String(memoId || "").trim(),
      parentTaskId: id,
      replaceWithRef: Boolean(options && options.replaceWithRef),
    },
  }).then(function (data) {
    const parentTask = normalizeTaskPayload(data.parentTask);
    const childTask = normalizeTaskPayload(data.childTask);
    if (!parentTask || !childTask) throw new Error("extract task failed");
    return {
      childTask,
      memo: data.memo || null,
      parentTask,
    };
  });
}

export function rebuildTaskIndex() {
  return callNativeAPI("/api/task-index/rebuild", { method: "GET" }).then(function (data) {
    return {
      index: data.index || null,
      tasks: Array.isArray(data.tasks) ? data.tasks.map(normalizeTaskSummary).filter(Boolean) : [],
    };
  });
}
