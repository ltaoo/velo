function copyText(text) {
  if (navigator.clipboard && navigator.clipboard.writeText) {
    return navigator.clipboard.writeText(text);
  }
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.select();
  const ok = document.execCommand("copy");
  textarea.remove();
  return ok ? Promise.resolve() : Promise.reject(new Error("copy failed"));
}

function closestAnchor(target) {
  return closestElement(target, "a[href]");
}

function closestElement(target, selector) {
  let node = target;
  if (node && node.nodeType === 3) node = node.parentElement || node.parentNode;
  while (node && node !== document) {
    if (node.nodeType === 1) {
      if (typeof node.matches === "function" && node.matches(selector)) return node;
      if (typeof node.webkitMatchesSelector === "function" && node.webkitMatchesSelector(selector)) return node;
    }
    node = node.parentElement || node.parentNode;
  }
  return null;
}

function externalBrowserURLFromAnchor(anchor) {
  const href = String(anchor.getAttribute("href") || anchor.href || "").trim();
  if (!/^https?:\/\//i.test(href)) return "";

  try {
    const url = new URL(href);
    if ((url.protocol === "http:" || url.protocol === "https:") && url.host) {
      return url.href;
    }
  } catch (_) {}
  return "";
}

function escapeHTML(value) {
  return String(value || "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function escapeAttr(value) {
  return escapeHTML(value).replace(/`/g, "&#96;");
}

function escapeCSSIdent(value) {
  if (window.CSS && window.CSS.escape) return window.CSS.escape(String(value || ""));
  return String(value || "").replace(/\\/g, "\\\\").replace(/"/g, '\\"');
}

export {
  closestAnchor,
  closestElement,
  copyText,
  escapeAttr,
  escapeCSSIdent,
  escapeHTML,
  externalBrowserURLFromAnchor,
};
