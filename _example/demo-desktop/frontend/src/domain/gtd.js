import { callNativeAPI } from "./native.js";
import { normalizeProjectID } from "./projects.js";
import { normalizeTaskID, normalizeStringList } from "./tasks.js";

export const GTD_ITEM_STATUS = {
  CLOSED: "closed",
  OPEN: "open",
  RESOLVED: "resolved",
  TRIAGED: "triaged",
  WAITING: "waiting",
};

export const GTD_ITEM_TYPE = {
  BUG: "bug",
  CHORE: "chore",
  FEATURE: "feature",
  IDEA: "idea",
  QUESTION: "question",
};

export const GTD_MILESTONE_STATUS = {
  ACTIVE: "active",
  CANCELLED: "cancelled",
  COMPLETED: "completed",
  PLANNED: "planned",
};

export function normalizeGTDItem(item) {
  if (!item || typeof item !== "object") return null;
  const id = normalizeGTDID(item.id);
  const title = String(item.title || "").trim();
  if (!id || !title) return null;
  return {
    closedAt: String(item.closedAt || ""),
    createdAt: item.createdAt || new Date().toISOString(),
    decision: String(item.decision || ""),
    id,
    labels: normalizeStringList(item.labels),
    linkedMemoIds: normalizeStringList(item.linkedMemoIds),
    linkedTaskIds: normalizeStringList(item.linkedTaskIds).map(normalizeTaskID).filter(Boolean),
    milestoneId: normalizeGTDID(item.milestoneId),
    projectId: normalizeProjectID(item.projectId),
    status: normalizeGTDItemStatus(item.status),
    title,
    type: normalizeGTDItemType(item.type),
    updatedAt: item.updatedAt || "",
  };
}

export function normalizeGTDMilestone(milestone) {
  if (!milestone || typeof milestone !== "object") return null;
  const id = normalizeGTDID(milestone.id);
  const title = String(milestone.title || "").trim();
  if (!id || !title) return null;
  return {
    completedAt: String(milestone.completedAt || ""),
    createdAt: milestone.createdAt || new Date().toISOString(),
    id,
    itemIds: normalizeStringList(milestone.itemIds).map(normalizeGTDID).filter(Boolean),
    projectIds: normalizeStringList(milestone.projectIds).map(normalizeProjectID).filter(Boolean),
    reviewMemoId: String(milestone.reviewMemoId || "").trim(),
    status: normalizeGTDMilestoneStatus(milestone.status),
    targetAt: String(milestone.targetAt || ""),
    taskIds: normalizeStringList(milestone.taskIds).map(normalizeTaskID).filter(Boolean),
    title,
    updatedAt: milestone.updatedAt || "",
  };
}

export function normalizeGTDID(value) {
  return String(value || "").trim().replace(/[^A-Za-z0-9_-]+/g, "-").replace(/^-+|-+$/g, "");
}

export function normalizeGTDItemStatus(value) {
  const status = String(value || "").trim().toLowerCase();
  if (status === GTD_ITEM_STATUS.TRIAGED) return GTD_ITEM_STATUS.TRIAGED;
  if (status === GTD_ITEM_STATUS.WAITING) return GTD_ITEM_STATUS.WAITING;
  if (status === GTD_ITEM_STATUS.RESOLVED) return GTD_ITEM_STATUS.RESOLVED;
  if (status === GTD_ITEM_STATUS.CLOSED) return GTD_ITEM_STATUS.CLOSED;
  return GTD_ITEM_STATUS.OPEN;
}

export function normalizeGTDItemType(value) {
  const type = String(value || "").trim().toLowerCase();
  if (type === GTD_ITEM_TYPE.BUG) return GTD_ITEM_TYPE.BUG;
  if (type === GTD_ITEM_TYPE.QUESTION) return GTD_ITEM_TYPE.QUESTION;
  if (type === GTD_ITEM_TYPE.FEATURE) return GTD_ITEM_TYPE.FEATURE;
  if (type === GTD_ITEM_TYPE.CHORE) return GTD_ITEM_TYPE.CHORE;
  return GTD_ITEM_TYPE.IDEA;
}

export function normalizeGTDMilestoneStatus(value) {
  const status = String(value || "").trim().toLowerCase();
  if (status === GTD_MILESTONE_STATUS.ACTIVE) return GTD_MILESTONE_STATUS.ACTIVE;
  if (status === GTD_MILESTONE_STATUS.COMPLETED) return GTD_MILESTONE_STATUS.COMPLETED;
  if (status === GTD_MILESTONE_STATUS.CANCELLED) return GTD_MILESTONE_STATUS.CANCELLED;
  return GTD_MILESTONE_STATUS.PLANNED;
}

export function loadGTDItems() {
  return callNativeAPI("/api/gtd/items", { method: "GET" }).then(function (data) {
    return Array.isArray(data.items) ? data.items.map(normalizeGTDItem).filter(Boolean) : [];
  });
}

export function createGTDItem(input) {
  return callNativeAPI("/api/gtd/items/create", {
    method: "POST",
    args: input && typeof input === "object" ? input : {},
  }).then(function (data) {
    const item = normalizeGTDItem(data.item);
    if (!item) throw new Error("create item failed");
    return item;
  });
}

export function updateGTDItem(id, patch) {
  const itemId = normalizeGTDID(id);
  if (!itemId) return Promise.reject(new Error("item id is required"));
  return callNativeAPI("/api/gtd/items/update", {
    method: "POST",
    args: Object.assign({}, patch || {}, { id: itemId }),
  }).then(function (data) {
    const item = normalizeGTDItem(data.item);
    if (!item) throw new Error("update item failed");
    return item;
  });
}

export function closeGTDItem(id) {
  const itemId = normalizeGTDID(id);
  if (!itemId) return Promise.reject(new Error("item id is required"));
  return callNativeAPI("/api/gtd/items/close", {
    method: "POST",
    args: { id: itemId },
  }).then(function (data) {
    const item = normalizeGTDItem(data.item);
    if (!item) throw new Error("close item failed");
    return item;
  });
}

export function loadGTDMilestones() {
  return callNativeAPI("/api/gtd/milestones", { method: "GET" }).then(function (data) {
    return Array.isArray(data.milestones) ? data.milestones.map(normalizeGTDMilestone).filter(Boolean) : [];
  });
}

export function createGTDMilestone(input) {
  return callNativeAPI("/api/gtd/milestones/create", {
    method: "POST",
    args: input && typeof input === "object" ? input : {},
  }).then(function (data) {
    const milestone = normalizeGTDMilestone(data.milestone);
    if (!milestone) throw new Error("create milestone failed");
    return milestone;
  });
}

export function updateGTDMilestone(id, patch) {
  const milestoneId = normalizeGTDID(id);
  if (!milestoneId) return Promise.reject(new Error("milestone id is required"));
  return callNativeAPI("/api/gtd/milestones/update", {
    method: "POST",
    args: Object.assign({}, patch || {}, { id: milestoneId }),
  }).then(function (data) {
    const milestone = normalizeGTDMilestone(data.milestone);
    if (!milestone) throw new Error("update milestone failed");
    return milestone;
  });
}
