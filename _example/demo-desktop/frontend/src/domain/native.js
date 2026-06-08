export function canUseNativeBridge() {
  return typeof globalThis.invoke === "function";
}

export function callNativeAPI(url, options) {
  if (!canUseNativeBridge()) {
    return Promise.reject(new Error("go bridge not available"));
  }
  return globalThis.invoke(url, options || {}).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "request failed");
    }
    return resp.data || {};
  });
}

export function errorText(err) {
  return err && err.message ? err.message : String(err || "unknown error");
}
