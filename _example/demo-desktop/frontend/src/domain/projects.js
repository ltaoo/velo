export function normalizeProjectPayload(project) {
  if (!project || typeof project !== "object") return null;
  const id = normalizeProjectID(project.id);
  const name = String(project.name || "").trim();
  if (!id || !name) return null;
  return {
    archived: Boolean(project.archived),
    color: normalizeProjectColor(project.color),
    createdAt: project.createdAt || new Date().toISOString(),
    id,
    name,
    sortOrder: Number.isFinite(Number(project.sortOrder)) ? Number(project.sortOrder) : 0,
    updatedAt: project.updatedAt || "",
  };
}

export function normalizeProjectID(value) {
  return String(value || "").trim().replace(/[^A-Za-z0-9_-]+/g, "-").replace(/^-+|-+$/g, "");
}

export function normalizeProjectFilter(value) {
  const raw = String(value || "").trim();
  if (raw === "all" || raw === "unassigned") return raw;
  return normalizeProjectID(raw) || "all";
}

export function normalizeProjectColor(value) {
  const color = String(value || "").trim();
  return /^#[0-9a-fA-F]{6}$/.test(color) ? color.toLowerCase() : "#2563eb";
}
