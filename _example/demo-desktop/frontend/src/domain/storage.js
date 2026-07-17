export const CLOUD_STORAGE_KEY = "demo-desktop:settings:cloud-storage:v1";

export function normalizeCloudStorageSettings(value) {
  const settings = value && typeof value === "object" ? value : {};
  let storages = [];
  if (Array.isArray(settings.storages)) {
    storages = settings.storages.map(normalizeCloudStorageProfile).filter(function (storage) {
      return storage.id;
    });
  } else if (hasLegacyCloudStorageProfile(settings)) {
    storages = [normalizeCloudStorageProfile(Object.assign({ id: "default", name: "默认存储" }, settings))];
  }

  const seen = Object.create(null);
  storages = storages.map(function (storage, index) {
    let id = sanitizeStorageId(storage.id || storage.name || storage.provider || storage.bucket || "storage-" + (index + 1));
    if (!id) id = "storage-" + (index + 1);
    seen[id] = (seen[id] || 0) + 1;
    if (seen[id] > 1) id = id + "-" + seen[id];
    return Object.assign({}, storage, {
      id,
      name: storage.name || storage.bucket || storage.provider || id,
    });
  });

  let activeStorageId = sanitizeStorageId(settings.activeStorageId || "");
  if (!storages.some(function (storage) { return storage.id === activeStorageId; })) {
    const enabled = storages.find(function (storage) { return storage.enabled; });
    activeStorageId = enabled ? enabled.id : (storages[0] && storages[0].id) || "";
  }

  return {
    activeStorageId,
    defaultsInitialized: Boolean(settings.defaultsInitialized),
    storages,
  };
}

export function normalizeCloudStorageProfile(profile) {
  const value = profile && typeof profile === "object" ? profile : {};
  return {
    accessKeyId: String(value.accessKeyId || "").trim(),
    bucket: String(value.bucket || "").trim(),
    enabled: Boolean(value.enabled),
    endpoint: String(value.endpoint || "").trim(),
    forcePathStyle: value.forcePathStyle !== false,
    id: sanitizeStorageId(value.id || ""),
    local: normalizeLocalStorageSettings(value.local),
    name: String(value.name || "").trim(),
    pathPrefix: String(value.pathPrefix || "").trim(),
    provider: String(value.provider || "s3").trim() || "s3",
    publicBaseUrl: String(value.publicBaseUrl || "").trim(),
    region: String(value.region || "").trim(),
    secretAccessKey: String(value.secretAccessKey || ""),
    sessionToken: String(value.sessionToken || "").trim(),
    useSSL: value.useSSL !== false,
  };
}

export function hasLegacyCloudStorageProfile(value) {
  return Boolean(value && typeof value === "object" && (
    value.enabled ||
    value.endpoint ||
    value.local ||
    value.bucket ||
    value.accessKeyId ||
    value.secretAccessKey ||
    value.sessionToken ||
    value.region ||
    value.pathPrefix ||
    value.publicBaseUrl
  ));
}

export function activeCloudStorageConfig(settings) {
  const normalized = normalizeCloudStorageSettings(settings);
  if (!normalized.activeStorageId) return null;
  return normalized.storages.find(function (storage) {
    return storage.id === normalized.activeStorageId;
  }) || null;
}

export function cloudStorageById(settings, storageId) {
  const normalized = normalizeCloudStorageSettings(settings);
  const id = sanitizeStorageId(storageId);
  if (!id) return null;
  return normalized.storages.find(function (storage) {
    return storage.id === id;
  }) || null;
}

export function sanitizeStorageId(value) {
  return String(value || "")
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_-]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export function assetReference(storageId, key) {
  const id = sanitizeStorageId(storageId);
  const cleanKey = String(key || "").replace(/^\/+/, "");
  if (!id || !cleanKey) return "";
  return "@assets/" + id + "/" + cleanKey;
}

export function parseAssetReference(value) {
  const cleanValue = String(value || "").trim().replace(/\?.*$/, "");
  const match = cleanValue.match(/^@assets\/([a-z0-9_-]+)\/(.+)$/i);
  if (!match) return null;
  return {
    key: decodeAssetReferenceKey(match[2]),
    storageId: sanitizeStorageId(match[1]),
  };
}

export function parseImageQueryParams(value) {
  var text = String(value || "").trim();
  var qIndex = text.indexOf("?");
  if (qIndex < 0) return null;
  var query = text.slice(qIndex + 1);
  var params = {};
  query.split("&").forEach(function (part) {
    var eq = part.indexOf("=");
    if (eq < 0) return;
    params[decodeURIComponent(part.slice(0, eq))] = decodeURIComponent(part.slice(eq + 1));
  });
  var w = parseInt(params.w, 10);
  var h = parseInt(params.h, 10);
  if (!w || !h || w <= 0 || h <= 0) return null;
  return {
    width: w,
    height: h,
    size: params.size || "",
    type: params.type || "",
  };
}

export function decodeAssetReferenceKey(value) {
  return String(value || "").replace(/%29/gi, ")").replace(/%28/gi, "(");
}

export function resolveAssetUrl(value, settings) {
  const asset = parseAssetReference(value);
  if (!asset) return String(value || "").trim();
  const storage = cloudStorageById(settings, asset.storageId);
  if (!storage) {
    return "/api/oss/assets?storageId=" + encodeURIComponent(asset.storageId) + "&path=" + encodeURIComponent(String(asset.key || "").replace(/^\/+/, ""));
  }
  return publicCloudStorageObjectUrl(storage, asset.key);
}

export function publicCloudStorageObjectUrl(storage, key) {
  const encodedKey = encodeObjectKey(key);
  if (!encodedKey) return "";
  if (isLocalCloudStorage(storage)) {
    return "/api/oss/assets?storageId=" + encodeURIComponent(sanitizeStorageId(storage.id)) + "&path=" + encodeURIComponent(String(key || "").replace(/^\/+/, ""));
  }
  const publicBaseUrl = String(storage.publicBaseUrl || "").trim().replace(/\/+$/, "");
  if (publicBaseUrl) return publicBaseUrl + "/" + encodedKey;

  const endpoint = normalizeOSSEndpoint(storage.endpoint, storage.useSSL);
  if (!endpoint) return "";
  if (storage.forcePathStyle) {
    return endpoint.replace(/\/+$/, "") + "/" + encodeURIComponent(String(storage.bucket || "").replace(/^\/+|\/+$/g, "")) + "/" + encodedKey;
  }
  try {
    const url = new URL(endpoint);
    url.hostname = String(storage.bucket || "").replace(/\.+$/g, "") + "." + url.hostname;
    url.pathname = "/" + encodedKey;
    url.search = "";
    url.hash = "";
    return url.toString();
  } catch (_) {
    return endpoint.replace(/\/+$/, "") + "/" + encodedKey;
  }
}

export function normalizeOSSEndpoint(endpoint, useSSL) {
  const value = String(endpoint || "").trim().replace(/\/+$/, "");
  if (!value) return "";
  if (/^https?:\/\//i.test(value)) return value;
  return (useSSL === false ? "http://" : "https://") + value;
}

export function encodeObjectKey(key) {
  return String(key || "")
    .replace(/^\/+/, "")
    .split("/")
    .map(function (part) {
      return encodeURIComponent(part);
    })
    .join("/");
}

export function missingCloudStorageFields(config) {
  const missing = [];
  if (!String(config.endpoint || "").trim() && !(isLocalCloudStorage(config) && normalizeLocalStorageSettings(config.local))) missing.push(isLocalCloudStorage(config) ? "本地根目录" : "Endpoint");
  if (!String(config.bucket || "").trim()) missing.push("Bucket");
  if (!isLocalCloudStorage(config)) {
    if (!String(config.accessKeyId || "").trim()) missing.push("Access Key ID");
    if (!String(config.secretAccessKey || "").trim()) missing.push("Secret Access Key");
  }
  return missing;
}

export function isLocalCloudStorage(config) {
  const provider = String((config && config.provider) || "").trim().toLowerCase();
  return provider === "local" || provider === "local-oss";
}

export function normalizeLocalStorageSettings(value) {
  const raw = value && typeof value === "object" ? value : {};
  let root = String(raw.root || "").trim();
  let rootMode = String(raw.rootMode || "").trim().toLowerCase();
  if (!rootMode) {
    if (!root) return null;
    rootMode = root.charAt(0) === "/" || root.indexOf(":\\") === 1 || root.indexOf("~/") === 0 ? "absolute" : "vault";
  }
  if (rootMode === "vault") {
    root = root.replace(/\\/g, "/").replace(/^\/+|\/+$/g, "") || "storage";
    return { root, rootMode: "vault" };
  }
  if (rootMode === "absolute" && root) {
    return { root, rootMode: "absolute" };
  }
  return null;
}
