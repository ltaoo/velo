;(function () {
  var SELECTOR = "input, textarea, [contenteditable='true'], [contenteditable='plaintext-only']";

  function applyInputPolicy(node) {
    if (!node || node.nodeType !== 1) return;

    var tag = node.tagName ? node.tagName.toLowerCase() : "";
    var editable = node.getAttribute && node.getAttribute("contenteditable");
    var isTextTarget =
      tag === "textarea" ||
      tag === "input" ||
      editable === "true" ||
      editable === "plaintext-only";
    if (!isTextTarget) return;

    node.setAttribute("autocomplete", "off");
    node.setAttribute("autocorrect", "off");
    node.setAttribute("autocapitalize", "off");
    node.setAttribute("spellcheck", "false");
    node.setAttribute("writingsuggestions", "false");
    node.setAttribute("data-gramm", "false");
    node.setAttribute("data-gramm_editor", "false");
    node.setAttribute("data-enable-grammarly", "false");

    if ("autocomplete" in node) node.autocomplete = "off";
    if ("autocorrect" in node) node.autocorrect = "off";
    if ("autocapitalize" in node) node.autocapitalize = "off";
    if ("spellcheck" in node) node.spellcheck = false;
    if ("writingSuggestions" in node) node.writingSuggestions = "false";
  }

  function applyInputPolicyTree(root) {
    if (!root || root.nodeType !== 1 && root.nodeType !== 9) return;
    if (root.matches && root.matches(SELECTOR)) applyInputPolicy(root);
    if (!root.querySelectorAll) return;
    root.querySelectorAll(SELECTOR).forEach(applyInputPolicy);
  }

  function start() {
    applyInputPolicyTree(document);
    new MutationObserver(function (mutations) {
      mutations.forEach(function (mutation) {
        if (mutation.type === "attributes") {
          applyInputPolicy(mutation.target);
          return;
        }
        mutation.addedNodes.forEach(applyInputPolicyTree);
      });
    }).observe(document.documentElement, {
      attributeFilter: ["contenteditable"],
      attributes: true,
      childList: true,
      subtree: true,
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", start, { once: true });
  } else {
    start();
  }
})();
