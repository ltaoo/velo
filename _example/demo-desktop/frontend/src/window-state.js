const WINDOW_STATE_POLL_INTERVAL = 250;
const WINDOW_STATE_SNAPSHOT_DEBOUNCE = 800;

let controller = null;

export function startWindowStatePersistence(options = {}) {
  return registerWindowSession(options);
}

export function registerWindowSession(options = {}) {
  if (controller) {
    controller.update(options);
    return controller;
  }

  const state = {
    debounceTimer: null,
    entryPage: String(options.entryPage || "").trim(),
    fixed: readInitialFixed(options),
    getState: typeof options.getState === "function" ? options.getState : null,
    inFlight: false,
    kind: String(options.kind || "").trim() || "open_window",
    lastWindowState: null,
    loadedSession: null,
    name: readWindowName(options),
    pollTimer: null,
    restoreApplied: false,
    restoreState: typeof options.restoreState === "function" ? options.restoreState : null,
    title: String(options.title || "").trim(),
  };

  controller = {
    forget,
    name() {
      return state.name;
    },
    scheduleSnapshot,
    setFixed,
    snapshot,
    update,
  };

  if (!state.name) return controller;

  window.addEventListener("beforeunload", handleBeforeUnload);
  window.addEventListener("hashchange", scheduleSnapshot);
  window.addEventListener("popstate", scheduleSnapshot);
  window.addEventListener("resize", scheduleSnapshot);
  loadPersistedSession();
  startPolling();
  applyFixedState();

  return controller;

  function update(nextOptions = {}) {
    if (Object.prototype.hasOwnProperty.call(nextOptions, "entryPage")) {
      state.entryPage = String(nextOptions.entryPage || "").trim();
    }
    if (Object.prototype.hasOwnProperty.call(nextOptions, "fixed")) {
      state.fixed = Boolean(nextOptions.fixed);
      applyFixedState();
    }
    if (typeof nextOptions.getState === "function") {
      state.getState = nextOptions.getState;
    }
    if (Object.prototype.hasOwnProperty.call(nextOptions, "kind")) {
      state.kind = String(nextOptions.kind || "").trim() || state.kind;
    }
    if (typeof nextOptions.restoreState === "function") {
      state.restoreState = nextOptions.restoreState;
      applyRestoredState();
    }
    if (Object.prototype.hasOwnProperty.call(nextOptions, "title")) {
      state.title = String(nextOptions.title || "").trim();
    }
    scheduleSnapshot();
  }

  function setFixed(fixed) {
    state.fixed = Boolean(fixed);
    applyFixedState();
    return snapshot();
  }

  function applyFixedState() {
    if (typeof invoke !== "function" || !state.name) return;
    invoke("__velo/window/set_always_on_top", { args: { onTop: state.fixed } }).catch(function () {});
  }

  function loadPersistedSession() {
    if (typeof invoke !== "function" || !state.name) return;
    invoke("/api/window/session/get?name=" + encodeURIComponent(state.name), { method: "GET" }).then(
      function (resp) {
        const session = resp && resp.code === 0 && resp.data && resp.data.found ? resp.data.session : null;
        if (!session || typeof session !== "object") return;
        state.loadedSession = session;
        if (typeof session.fixed === "boolean") {
          state.fixed = session.fixed;
          applyFixedState();
        }
        applyRestoredState();
      },
      function () {},
    );
  }

  function applyRestoredState() {
    if (state.restoreApplied || !state.restoreState || !state.loadedSession) return;
    state.restoreApplied = true;
    try {
      state.restoreState(state.loadedSession.state || {}, state.loadedSession);
    } catch (_) {}
  }

  function startPolling() {
    if (typeof invoke !== "function" || state.pollTimer) return;
    snapshotIfChanged();
    state.pollTimer = window.setInterval(snapshotIfChanged, WINDOW_STATE_POLL_INTERVAL);
  }

  function scheduleSnapshot() {
    if (typeof invoke !== "function" || !state.name) return;
    if (state.debounceTimer) {
      window.clearTimeout(state.debounceTimer);
    }
    state.debounceTimer = window.setTimeout(function () {
      state.debounceTimer = null;
      snapshot();
    }, WINDOW_STATE_SNAPSHOT_DEBOUNCE);
  }

  function snapshot() {
    if (typeof invoke !== "function" || !state.name) return Promise.resolve(null);
    return readWindowState().then(function (nextWindowState) {
      if (!nextWindowState) return null;
      state.lastWindowState = nextWindowState;
      return saveWindowSession(nextWindowState);
    });
  }

  function snapshotIfChanged() {
    if (typeof invoke !== "function" || state.inFlight || !state.name) return;
    state.inFlight = true;
    readWindowState().then(
      function (nextWindowState) {
        if (!nextWindowState) return null;
        if (isSameWindowState(state.lastWindowState, nextWindowState)) {
          return null;
        }
        state.lastWindowState = nextWindowState;
        return saveWindowSession(nextWindowState);
      },
      function () {},
    ).finally(function () {
      state.inFlight = false;
    });
  }

  function handleBeforeUnload() {
    const payload = windowSessionPayload(state.lastWindowState || readWindowStateHint());
    if (!payload) return;
    try {
      const xhr = new XMLHttpRequest();
      xhr.open("POST", "/api/window/session/save", false);
      xhr.setRequestHeader("Content-Type", "application/json");
      xhr.send(JSON.stringify(payload));
    } catch (_) {}
  }

  function forget() {
    if (typeof invoke !== "function" || !state.name) return Promise.resolve(null);
    return snapshot().catch(function () {}).then(function () {
      return invoke("/api/window/session/forget?name=" + encodeURIComponent(state.name), { method: "GET" }).catch(function () {});
    });
  }

  function saveWindowSession(windowState) {
    const payload = windowSessionPayload(windowState);
    if (!payload || typeof invoke !== "function") return Promise.resolve(null);
    return invoke("/api/window/session/save", { method: "POST", args: payload }).catch(function () {});
  }

  function windowSessionPayload(windowState) {
    if (!state.name || !windowState || windowState.width <= 0 || windowState.height <= 0) return null;
    return {
      entryPage: state.entryPage,
      fixed: state.fixed,
      height: windowState.height,
      kind: state.kind,
      name: state.name,
      pathname: currentAppPathname(),
      state: readPageState(),
      title: state.title || document.title || "",
      width: windowState.width,
      x: windowState.x,
      y: windowState.y,
    };
  }

  function readPageState() {
    if (!state.getState) return {};
    try {
      const value = state.getState();
      return value && typeof value === "object" ? value : {};
    } catch (_) {
      return {};
    }
  }
}

export function setPersistedWindowFixed(fixed) {
  return registerWindowSession({ fixed }).setFixed(fixed);
}

export function forgetPersistedWindow() {
  return registerWindowSession().forget();
}

export function persistedWindowFixedFromURL() {
  return readFixedFromURL();
}

function installGlobalAPI() {
  if (!Object.prototype.hasOwnProperty.call(window, "registerWindowSession")) {
    Object.defineProperty(window, "registerWindowSession", {
      value: registerWindowSession,
      writable: false,
      configurable: true,
    });
  }
  if (!Object.prototype.hasOwnProperty.call(window, "forgetPersistedWindow")) {
    Object.defineProperty(window, "forgetPersistedWindow", {
      value: forgetPersistedWindow,
      writable: false,
      configurable: true,
    });
  }
  if (!Object.prototype.hasOwnProperty.call(window, "setPersistedWindowFixed")) {
    Object.defineProperty(window, "setPersistedWindowFixed", {
      value: setPersistedWindowFixed,
      writable: false,
      configurable: true,
    });
  }
}

function readInitialFixed(options) {
  return Object.prototype.hasOwnProperty.call(options, "fixed") ? Boolean(options.fixed) : readFixedFromURL();
}

function readFixedFromURL() {
  try {
    return new URLSearchParams(window.location.search || "").get("fixed") === "1";
  } catch (_) {
    return false;
  }
}

function readWindowName(options) {
  const explicit = String((options && options.name) || "").trim();
  if (explicit) return explicit;
  try {
    const runtime = window.__VELO__ || {};
    const name = runtime.window && runtime.window.name;
    return String(name || "").trim();
  } catch (_) {
    return "";
  }
}

function currentAppPathname() {
  try {
    return window.location.pathname + window.location.search + window.location.hash;
  } catch (_) {
    return "/";
  }
}

function readWindowState() {
  if (typeof invoke !== "function") {
    return Promise.resolve(readWindowStateHint());
  }
  return invoke("__velo/window/state", { args: {} }).then(
    function (resp) {
      if (!resp || resp.success === false || resp.width <= 0 || resp.height <= 0) {
        return readWindowStateHint();
      }
      return {
        height: Math.round(Number(resp.height || 0)),
        width: Math.round(Number(resp.width || 0)),
        x: Math.round(Number(resp.x || 0)),
        y: Math.round(Number(resp.y || 0)),
      };
    },
    function () {
      return readWindowStateHint();
    },
  );
}

function readWindowStateHint() {
  return {
    height: Math.round(Number(window.outerHeight ?? window.innerHeight ?? 0)),
    width: Math.round(Number(window.outerWidth ?? window.innerWidth ?? 0)),
    x: Math.round(Number(window.screenX ?? window.screenLeft ?? 0)),
    y: Math.round(Number(window.screenY ?? window.screenTop ?? 0)),
  };
}

function isSameWindowState(a, b) {
  return Boolean(a && b && a.x === b.x && a.y === b.y && a.width === b.width && a.height === b.height);
}

installGlobalAPI();
registerWindowSession();
