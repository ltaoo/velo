const platform = document.getElementById("platform");
const result = document.getElementById("result");
const refreshButton = document.getElementById("refresh");
const applyButton = document.getElementById("apply");
const acStatus = document.getElementById("ac-status");
const batteryStatus = document.getElementById("battery-status");

let presets = {};

const fields = {
  ac: {
    display_sleep: document.getElementById("ac-display"),
    system_sleep: document.getElementById("ac-system"),
    disk_sleep: document.getElementById("ac-disk"),
    power_nap: document.getElementById("ac-powernap"),
  },
  battery: {
    display_sleep: document.getElementById("battery-display"),
    system_sleep: document.getElementById("battery-system"),
    disk_sleep: document.getElementById("battery-disk"),
    power_nap: document.getElementById("battery-powernap"),
  },
  hibernate_mode: document.getElementById("hibernate-mode"),
  standby: document.getElementById("standby"),
  auto_power_off: document.getElementById("autopoweroff"),
};

async function invokeGo(path, args) {
  if (typeof window.invoke !== "function") {
    throw new Error("Go bridge is not ready");
  }
  return window.invoke(path, { args });
}

function setResult(kind, message) {
  result.className = `result ${kind || ""}`.trim();
  result.textContent = message || "";
}

function readNumber(input) {
  const value = Number.parseInt(input.value, 10);
  return Number.isFinite(value) ? value : 0;
}

function writeProfile(name, profile) {
  const group = fields[name];
  group.display_sleep.value = profile.display_sleep;
  group.system_sleep.value = profile.system_sleep;
  group.disk_sleep.value = profile.disk_sleep;
  group.power_nap.checked = Boolean(profile.power_nap);
}

function readProfile(name) {
  const group = fields[name];
  return {
    display_sleep: readNumber(group.display_sleep),
    system_sleep: readNumber(group.system_sleep),
    disk_sleep: readNumber(group.disk_sleep),
    power_nap: group.power_nap.checked,
  };
}

function writeRequest(req) {
  writeProfile("ac", req.ac);
  writeProfile("battery", req.battery);
  fields.hibernate_mode.value = String(req.hibernate_mode);
  fields.standby.checked = Boolean(req.standby);
  fields.auto_power_off.checked = Boolean(req.auto_power_off);
}

function readRequest() {
  return {
    ac: readProfile("ac"),
    battery: readProfile("battery"),
    hibernate_mode: readNumber(fields.hibernate_mode),
    standby: fields.standby.checked,
    auto_power_off: fields.auto_power_off.checked,
  };
}

function renderStatus(title, values) {
  const keys = ["displaysleep", "sleep", "disksleep", "powernap", "hibernatemode", "standby", "autopoweroff"];
  const lines = keys
    .filter((key) => values && Object.prototype.hasOwnProperty.call(values, key))
    .map((key) => `${key}: ${values[key]}`);
  return lines.length ? lines.join("\n") : `${title}: -`;
}

function applyPreset(name) {
  const preset = presets[name];
  if (preset) {
    writeRequest(preset);
  }
}

document.querySelectorAll("input[name='preset']").forEach((input) => {
  input.addEventListener("change", () => applyPreset(input.value));
});

refreshButton.addEventListener("click", () => refreshStatus(true));

applyButton.addEventListener("click", async () => {
  applyButton.disabled = true;
  setResult("", "正在请求管理员授权...");
  try {
    const response = await invokeGo("/api/power/apply", readRequest());
    if (!response || response.code !== 0) {
      throw new Error(response && response.msg ? response.msg : "应用失败");
    }
    setResult("ok", "设置已应用。");
    if (response.data && response.data.status) {
      renderFullStatus(response.data.status);
    } else {
      refreshStatus(false);
    }
  } catch (error) {
    setResult("error", error.message || String(error));
  } finally {
    applyButton.disabled = false;
  }
});

function renderFullStatus(data) {
  platform.textContent = data.supported ? "macOS pmset 已连接" : `当前系统为 ${data.os}，只能查看示例界面`;
  presets = data.presets || presets;
  acStatus.textContent = renderStatus("AC", data.ac || {});
  batteryStatus.textContent = renderStatus("Battery", data.battery || {});
}

async function refreshStatus(showResult) {
  refreshButton.disabled = true;
  try {
    const response = await invokeGo("/api/power/status", {});
    if (!response || response.code !== 0) {
      throw new Error(response && response.msg ? response.msg : "读取失败");
    }
    renderFullStatus(response.data);
    if (showResult) {
      setResult("ok", "状态已刷新。");
    }
    if (!Object.keys(presets).length) {
      return;
    }
    const selected = document.querySelector("input[name='preset']:checked");
    applyPreset(selected ? selected.value : "pluggedNeverBatteryTimed");
  } catch (error) {
    platform.textContent = "无法读取系统状态";
    setResult("error", error.message || String(error));
  } finally {
    refreshButton.disabled = false;
  }
}

refreshStatus(false);
