import { app, history, client, views } from "./store/index.js";
import { storage } from "./store/storage.js";
import { RouterSubViews } from "./components/sub-views.js";

let snapshotTimer = null;

function windowStateName() {
  return window.location.pathname === "/vault-picker" ? "vault-picker" : "desktop";
}

const render = ($root) => {
  const { innerWidth, innerHeight, location } = window;
  history.$router.prepare(location);
  app
    .start({
      width: innerWidth,
      height: innerHeight,
    })
    .then(() => {
      const v$ = RouterSubViews({
        class: classnames("root-view w-full h-full"),
        view: history.$view,
        app,
        views,
        history,
        storage,
        client,
      });
      $root.appendChild(v$.render());
      v$.onMounted();
      restoreWindowState();
      startWindowStateSnapshots();
    });
};

document.addEventListener("DOMContentLoaded", function () {
  const $root = document.querySelector("#root");
  if (!$root) {
    console.error("[Render] Root element not found");
    return;
  }
  render($root);
});

window.addEventListener("beforeunload", function () {
  stopWindowStateSnapshots();
  snapshotWindowStateSync();
});

function restoreWindowState() {
  if (typeof invoke !== "function") return;
  invoke("/api/window/state/restore?name=" + encodeURIComponent(windowStateName()), { method: "GET" }).catch(function () {});
}

function startWindowStateSnapshots() {
  if (typeof invoke !== "function" || snapshotTimer) return;
  snapshotTimer = window.setInterval(function () {
    invoke("/api/window/state/snapshot?name=" + encodeURIComponent(windowStateName()), { method: "GET" }).catch(function () {});
  }, 3000);
}

function stopWindowStateSnapshots() {
  if (!snapshotTimer) return;
  window.clearInterval(snapshotTimer);
  snapshotTimer = null;
}

function snapshotWindowStateSync() {
  try {
    const xhr = new XMLHttpRequest();
    xhr.open("GET", "/api/window/state/snapshot?name=" + encodeURIComponent(windowStateName()), false);
    xhr.send();
  } catch (_) {}
}
