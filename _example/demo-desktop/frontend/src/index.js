import { app, history, client, views } from "./store/index.js";
import { storage } from "./store/storage.js";
import { RouterSubViews } from "./components/sub-views.js";

const WINDOW_STATE_POLL_INTERVAL = 250;
const WINDOW_STATE_SNAPSHOT_DEBOUNCE = 800;

let snapshotPollTimer = null;
let snapshotDebounceTimer = null;
let lastWindowState = null;

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
  if (typeof invoke !== "function" || snapshotPollTimer) return;
  lastWindowState = readWindowStateHint();
  window.addEventListener("resize", scheduleWindowStateSnapshot);
  snapshotPollTimer = window.setInterval(function () {
    const nextWindowState = readWindowStateHint();
    if (!isSameWindowStateHint(lastWindowState, nextWindowState)) {
      lastWindowState = nextWindowState;
      scheduleWindowStateSnapshot();
    }
  }, WINDOW_STATE_POLL_INTERVAL);
}

function stopWindowStateSnapshots() {
  if (snapshotPollTimer) {
    window.clearInterval(snapshotPollTimer);
    snapshotPollTimer = null;
  }
  if (snapshotDebounceTimer) {
    window.clearTimeout(snapshotDebounceTimer);
    snapshotDebounceTimer = null;
  }
  window.removeEventListener("resize", scheduleWindowStateSnapshot);
}

function snapshotWindowStateSync() {
  try {
    const xhr = new XMLHttpRequest();
    xhr.open("GET", "/api/window/state/snapshot?name=" + encodeURIComponent(windowStateName()), false);
    xhr.send();
  } catch (_) {}
}

function scheduleWindowStateSnapshot() {
  if (typeof invoke !== "function") return;
  if (snapshotDebounceTimer) {
    window.clearTimeout(snapshotDebounceTimer);
  }
  snapshotDebounceTimer = window.setTimeout(function () {
    snapshotDebounceTimer = null;
    invoke("/api/window/state/snapshot?name=" + encodeURIComponent(windowStateName()), { method: "GET" }).catch(function () {});
  }, WINDOW_STATE_SNAPSHOT_DEBOUNCE);
}

function readWindowStateHint() {
  return {
    x: Math.round(Number(window.screenX ?? window.screenLeft ?? 0)),
    y: Math.round(Number(window.screenY ?? window.screenTop ?? 0)),
    width: Math.round(Number(window.outerWidth ?? window.innerWidth ?? 0)),
    height: Math.round(Number(window.outerHeight ?? window.innerHeight ?? 0)),
  };
}

function isSameWindowStateHint(a, b) {
  return Boolean(a && b && a.x === b.x && a.y === b.y && a.width === b.width && a.height === b.height);
}
