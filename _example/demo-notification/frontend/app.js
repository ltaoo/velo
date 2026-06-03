const form = document.getElementById("notification-form");
const result = document.getElementById("result");
const statusPanel = document.getElementById("status");
const pushStatusPanel = document.getElementById("push-status");
const sendButton = document.getElementById("send");
const resetButton = document.getElementById("reset");
const cleanupButton = document.getElementById("cleanup");
const registerPushButton = document.getElementById("register-push");
const platform = document.getElementById("platform");

const presets = {
  info: {
    title: "Build started",
    body: "The background task has started and will report when it finishes.",
  },
  success: {
    title: "Upload complete",
    body: "All files were uploaded successfully.",
  },
  warning: {
    title: "Storage almost full",
    body: "Free up disk space before starting the next export.",
  },
  error: {
    title: "Sync failed",
    body: "The remote service rejected the latest sync request.",
  },
};

function selectedType() {
  const input = form.querySelector("input[name='type']:checked");
  return input ? input.value : "info";
}

function applyPreset(type) {
  const preset = presets[type] || presets.info;
  document.getElementById("title").value = preset.title;
  document.getElementById("body").value = preset.body;
}

function setResult(kind, message) {
  result.className = `result ${kind || ""}`.trim();
  result.textContent = message;
}

async function invokeGo(path, args) {
  if (typeof window.invoke !== "function") {
    throw new Error("Go bridge is not ready");
  }
  return window.invoke(path, { args });
}

async function refreshStatus() {
  try {
    const response = await invokeGo("/api/notification/status", {});
    if (response && response.code === 0) {
      const data = response.data || {};
      statusPanel.textContent = [
        `permission: ${data.status || "unknown"}`,
        `supported: ${data.supported}`,
        data.bundle_id ? `bundle id: ${data.bundle_id}` : "",
        data.bundle_path ? `bundle path: ${data.bundle_path}` : "",
      ].filter(Boolean).join("\n");
    }
  } catch (error) {
    statusPanel.textContent = `permission: unavailable (${error.message || error})`;
  }
}

function renderPushState(data) {
  data = data || {};
  pushStatusPanel.textContent = [
    "remote push: APNs",
    data.registered_at ? `registered at: ${data.registered_at}` : "registered at: -",
    data.token ? `device token: ${data.token}` : "device token: -",
    data.error ? `error: ${data.error}` : "",
    data.payload ? `last payload: ${data.payload}` : "",
  ].filter(Boolean).join("\n");
}

async function refreshPushState() {
  try {
    const response = await invokeGo("/api/push/state", {});
    if (response && response.code === 0) {
      renderPushState(response.data);
    }
  } catch (error) {
    pushStatusPanel.textContent = `remote push: unavailable (${error.message || error})`;
  }
}

form.addEventListener("change", (event) => {
  if (event.target.name === "type") {
    applyPreset(event.target.value);
  }
});

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  sendButton.disabled = true;
  setResult("", "Sending notification...");

  const payload = {
    type: selectedType(),
    title: document.getElementById("title").value,
    body: document.getElementById("body").value,
    app_name: document.getElementById("app_name").value,
    icon: document.getElementById("icon").value,
    sound: document.getElementById("sound").checked,
  };

  try {
    const response = await invokeGo("/api/notification/send", payload);
    if (!response || response.code !== 0) {
      throw new Error(response && response.msg ? response.msg : "Notification failed");
    }
    setResult("ok", `${response.data.type} notification sent: ${response.data.title}`);
    refreshStatus();
  } catch (error) {
    setResult("error", error.message || String(error));
  } finally {
    sendButton.disabled = false;
  }
});

resetButton.addEventListener("click", () => {
  form.reset();
  applyPreset("info");
  setResult("", "");
});

cleanupButton.addEventListener("click", async () => {
  cleanupButton.disabled = true;
  setResult("", "Cleaning delivered and pending notifications...");
  try {
    const appName = document.getElementById("app_name").value || "Velo Notification Demo";
    const response = await invokeGo(`/api/notification/cleanup?app_name=${encodeURIComponent(appName)}`, {});
    if (!response || response.code !== 0) {
      throw new Error(response && response.msg ? response.msg : "Cleanup failed");
    }
    setResult("ok", response.data.note);
    refreshStatus();
  } catch (error) {
    setResult("error", error.message || String(error));
  } finally {
    cleanupButton.disabled = false;
  }
});

registerPushButton.addEventListener("click", async () => {
  registerPushButton.disabled = true;
  setResult("", "Registering with APNs...");
  try {
    const response = await invokeGo("/api/push/register", {});
    if (!response || response.code !== 0) {
      throw new Error(response && response.msg ? response.msg : "APNs registration failed");
    }
    renderPushState(response.data);
    setResult("ok", "APNs registration requested. Token or error will update below.");
  } catch (error) {
    setResult("error", error.message || String(error));
  } finally {
    registerPushButton.disabled = false;
  }
});

if (typeof window.onGoMessage === "function") {
  window.onGoMessage((payload) => {
    if (payload && typeof payload === "object" && String(payload.type || "").startsWith("remote_push_")) {
      renderPushState(payload.data);
    }
  });
}

async function loadAppInfo() {
  try {
    const response = await invokeGo("/api/app", {});
    if (response && response.code === 0) {
      platform.textContent = `${response.data.name} on ${response.data.os}`;
    }
  } catch (_error) {
    platform.textContent = "Notification form";
  }
}

applyPreset("info");
loadAppInfo();
refreshStatus();
refreshPushState();
