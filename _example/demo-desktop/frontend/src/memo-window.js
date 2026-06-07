import { mountDetachedMemoWindow } from "./pages/home/memos.js";

document.addEventListener("DOMContentLoaded", function () {
  const root = document.querySelector("#root");
  if (!root) {
    console.error("[MemoWindow] Root element not found");
    return;
  }
  mountDetachedMemoWindow(root);
});
