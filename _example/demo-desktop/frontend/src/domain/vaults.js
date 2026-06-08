import { callNativeAPI } from "./native.js";

export function normalizeVaultEntry(vault) {
  if (!vault || typeof vault !== "object") return null;
  const path = String(vault.path || "").trim();
  if (!path) return null;
  return {
    id: String(vault.id || "").trim(),
    lastOpenedAt: vault.lastOpenedAt || "",
    name: String(vault.name || "Vault").trim() || "Vault",
    path,
  };
}

export function normalizeVaultStatus(payload) {
  const data = payload && typeof payload === "object" ? payload : {};
  return {
    active: data.active || null,
    dataFileExists: Boolean(data.dataFileExists),
    dataPath: String(data.dataPath || ""),
    vaults: Array.isArray(data.vaults)
      ? data.vaults.map(normalizeVaultEntry).filter(Boolean)
      : [],
  };
}

export function normalizeVaultPath(value) {
  return String(value || "").trim();
}

export function loadVaultStatus() {
  return callNativeAPI("/api/vault/status", { method: "GET" }).then(normalizeVaultStatus);
}

export function selectVaultDirectory() {
  return callNativeAPI("/api/vault/select-directory", { method: "GET" }).then(function (data) {
    return normalizeVaultPath(data && data.path);
  });
}

export function openVault(path) {
  const value = normalizeVaultPath(path);
  if (!value) return Promise.reject(new Error("请输入或选择 vault 目录"));
  return callNativeAPI("/api/vault/open", {
    method: "POST",
    args: { path: value },
  });
}
