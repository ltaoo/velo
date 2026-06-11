import { mountImagePreview } from "./components/image-preview.js";

document.addEventListener("DOMContentLoaded", function () {
  const root = document.querySelector("#root");
  if (!root) {
    console.error("[ImagePreview] Root element not found");
    return;
  }
  mountImagePreview(root);
});
