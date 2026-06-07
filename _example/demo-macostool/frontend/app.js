const platform = document.getElementById("platform");
const result = document.getElementById("result");
const refreshButton = document.getElementById("refresh");
const applyButton = document.getElementById("apply");
const acStatus = document.getElementById("ac-status");
const batteryStatus = document.getElementById("battery-status");
const keyboardStatusText = document.getElementById("keyboard-status");
const keyboardPermission = document.getElementById("keyboard-permission");
const keyboardEventTap = document.getElementById("keyboard-eventtap");
const keyboardDisableButton = document.getElementById("keyboard-disable");
const keyboardEnableButton = document.getElementById("keyboard-enable");
const keyboardMessage = document.getElementById("keyboard-message");

let presets = {};
let currentKeyboardState = null;
let keyboardBusy = false;

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

function setKeyboardMessage(kind, message) {
  keyboardMessage.className = `keyboard-message ${kind || ""}`.trim();
  keyboardMessage.textContent = message || "";
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

function hasSetting(values, key) {
  return values && Object.prototype.hasOwnProperty.call(values, key);
}

function settingNumber(values, key) {
  if (!hasSetting(values, key)) {
    return null;
  }
  const value = Number.parseInt(values[key], 10);
  return Number.isFinite(value) ? value : null;
}

function minuteText(minutes) {
  if (minutes === 0) {
    return "永不";
  }
  if (minutes > 0 && minutes % 60 === 0) {
    return `${minutes / 60} 小时`;
  }
  return `${minutes} 分钟`;
}

function onOffText(value) {
  return value === 1 ? "开启" : "关闭";
}

function renderMinutePolicy(values, key, label, zeroText, timedText) {
  const minutes = settingNumber(values, key);
  if (minutes === null) {
    return null;
  }
  return minutes === 0 ? `${label}：${zeroText}` : `${label}：${timedText(minuteText(minutes))}`;
}

function renderBooleanPolicy(values, key, label, enabledText, disabledText) {
  const value = settingNumber(values, key);
  if (value === null) {
    return null;
  }
  return `${label}：${onOffText(value)}，${value === 1 ? enabledText : disabledText}`;
}

function renderHibernateMode(values) {
  const mode = settingNumber(values, "hibernatemode");
  if (mode === null) {
    return null;
  }

  const descriptions = {
    0: "普通睡眠，唤醒快，但断电后可能无法恢复会话",
    3: "安全睡眠，保存内存镜像，通常从内存快速唤醒",
    25: "深度休眠，更省电，唤醒更慢",
  };
  return `休眠模式：${descriptions[mode] || "自定义模式"}（${mode}）`;
}

function renderStatus(title, values) {
  const lines = [
    renderMinutePolicy(values, "displaysleep", "显示器", "不会因空闲自动关闭", (time) => `空闲 ${time} 后关闭，仅屏幕变黑`),
    renderMinutePolicy(values, "sleep", "系统睡眠", "不会因空闲自动睡眠，后台服务可继续运行", (time) => `空闲 ${time} 后睡眠，后台服务会暂停`),
    renderMinutePolicy(values, "disksleep", "磁盘", "不会因空闲休眠", (time) => `空闲 ${time} 后休眠`),
    renderBooleanPolicy(values, "powernap", "Power Nap", "睡眠中允许系统执行邮件、iCloud 等维护", "睡眠中不执行这类系统维护"),
    renderHibernateMode(values),
    renderBooleanPolicy(values, "standby", "Standby", "系统睡眠一段时间后可进入更省电状态", "系统睡眠后不主动进入 standby"),
    renderBooleanPolicy(values, "autopoweroff", "Auto Power Off", "系统睡眠较久后可进入更低功耗状态", "系统睡眠较久后不进入 autopoweroff"),
  ].filter(Boolean);

  return lines.length ? lines.join("\n") : `${title}：暂无可读配置`;
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

keyboardDisableButton.addEventListener("click", () => setKeyboardDisabled(true));
keyboardEnableButton.addEventListener("click", () => setKeyboardDisabled(false));

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

function renderKeyboardStatus(data) {
  currentKeyboardState = data || currentKeyboardState || {
    supported: false,
    os: "unknown",
    disabled: false,
    permission_granted: false,
    event_tap_ready: false,
  };

  const state = currentKeyboardState;
  if (!state.supported) {
    keyboardStatusText.textContent = `当前系统为 ${state.os}，键盘控制不可用`;
    keyboardPermission.textContent = "macOS 专用";
    keyboardEventTap.textContent = "不可用";
  } else {
    keyboardStatusText.textContent = state.disabled ? "键盘已禁用" : "键盘可输入";
    keyboardPermission.textContent = state.permission_granted ? "辅助功能权限已允许" : "辅助功能权限未允许";
    keyboardEventTap.textContent = state.event_tap_ready ? "事件拦截已就绪" : "事件拦截待初始化";
  }

  keyboardDisableButton.disabled = keyboardBusy || !state.supported || state.disabled;
  keyboardEnableButton.disabled = keyboardBusy || !state.supported || !state.disabled;
}

async function refreshKeyboardStatus() {
  try {
    const response = await invokeGo("/api/keyboard/status", {});
    if (!response || response.code !== 0) {
      throw new Error(response && response.msg ? response.msg : "读取键盘状态失败");
    }
    renderKeyboardStatus(response.data);
  } catch (error) {
    setKeyboardMessage("error", error.message || String(error));
  }
}

async function setKeyboardDisabled(disable) {
  keyboardBusy = true;
  renderKeyboardStatus(currentKeyboardState);
  setKeyboardMessage("", disable ? "正在禁用键盘..." : "正在启用键盘...");

  try {
    const response = await invokeGo(disable ? "/api/keyboard/disable" : "/api/keyboard/enable", {});
    if (!response || response.code !== 0) {
      throw new Error(response && response.msg ? response.msg : "键盘控制失败");
    }
    renderKeyboardStatus(response.data);
    setKeyboardMessage("ok", disable ? "键盘已禁用。" : "键盘已启用。");
  } catch (error) {
    setKeyboardMessage("error", error.message || String(error));
    await refreshKeyboardStatus();
  } finally {
    keyboardBusy = false;
    renderKeyboardStatus(currentKeyboardState);
  }
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
refreshKeyboardStatus();
