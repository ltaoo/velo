import { errorText } from "../../domain/native.js";
import { loadVaultStatus, normalizeVaultPath, openVault, selectVaultDirectory } from "../../domain/vaults.js";

const FOLDER_ICON = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6.5A2.5 2.5 0 0 1 5.5 4H10l2 2h6.5A2.5 2.5 0 0 1 21 8.5v8A2.5 2.5 0 0 1 18.5 19h-13A2.5 2.5 0 0 1 3 16.5z"></path></svg>';
const PLUS_ICON = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 5v14"></path><path d="M5 12h14"></path></svg>';
const CHECK_ICON = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M20 6 9 17l-5-5"></path></svg>';

export function VaultPickerPageView() {
  let picker = null;

  return View(
    {
      class: "page vault-picker-page w-full h-full",
      onMounted(el) {
        picker = mountVaultPicker(el);
      },
      onUnmounted() {
        if (picker) picker.destroy();
        picker = null;
      },
    },
    [],
  );
}

function mountVaultPicker(root) {
  const state = {
    active: null,
    dataPath: "",
    dataFileExists: false,
    loading: false,
    message: "",
    messageType: "",
    vaults: [],
  };

  root.innerHTML = template();
  const els = {
    dataPath: root.querySelector("[data-vault-data-path]"),
    list: root.querySelector("[data-vault-list]"),
    message: root.querySelector("[data-vault-message]"),
    pathInput: root.querySelector("[data-vault-path]"),
    status: root.querySelector("[data-vault-status]"),
  };

  root.addEventListener("click", handleClick);
  root.addEventListener("submit", handleSubmit);
  loadStatus();

  return {
    destroy() {
      root.removeEventListener("click", handleClick);
      root.removeEventListener("submit", handleSubmit);
      root.innerHTML = "";
    },
  };

  function handleClick(event) {
    const action = event.target.closest("[data-action]");
    if (!action || !root.contains(action)) return;

    if (action.dataset.action === "chooseVault") {
      chooseVault();
      return;
    }

    if (action.dataset.action === "openVault") {
      openSelectedVault(action.dataset.vaultPath || "");
    }
  }

  function handleSubmit(event) {
    const form = event.target.closest("[data-vault-form]");
    if (!form || !root.contains(form)) return;
    event.preventDefault();
    openSelectedVault(els.pathInput.value);
  }

  function loadStatus() {
    setLoading(true);
    loadVaultStatus().then(
      function (status) {
        state.active = status.active;
        state.dataPath = status.dataPath;
        state.dataFileExists = status.dataFileExists;
        state.vaults = status.vaults;
        render();
      },
      function (err) {
        setMessage("读取 vault 状态失败: " + errorText(err), "error");
      },
    ).finally(function () {
      setLoading(false);
    });
  }

  function chooseVault() {
    setLoading(true);
    selectVaultDirectory().then(
      function (path) {
        if (!path) {
          setMessage("没有选择目录", "warning");
          return;
        }
        els.pathInput.value = path;
        return openSelectedVault(path);
      },
      function (err) {
        const message = errorText(err);
        setMessage(message === "cancelled" ? "已取消选择" : "选择目录失败: " + message, "warning");
      },
    ).finally(function () {
      setLoading(false);
    });
  }

  function openSelectedVault(path) {
    const value = normalizeVaultPath(path);
    if (!value) {
      setMessage("请输入或选择 vault 目录", "warning");
      return Promise.resolve();
    }
    setLoading(true);
    return openVault(value).then(
      function (data) {
        const created = data && data.created;
        setMessage(created ? "已创建 vault" : "已加载 vault", "success");
        window.setTimeout(function () {
          window.location.replace("/desktop");
        }, 180);
      },
      function (err) {
        setMessage("打开 vault 失败: " + errorText(err), "error");
      },
    ).finally(function () {
      setLoading(false);
    });
  }

  function setLoading(loading) {
    state.loading = Boolean(loading);
    root.classList.toggle("is-loading", state.loading);
    root.querySelectorAll("button, input").forEach(function (node) {
      node.disabled = state.loading;
    });
  }

  function setMessage(message, type) {
    state.message = message || "";
    state.messageType = type || "";
    renderMessage();
  }

  function render() {
    els.status.textContent = state.dataFileExists ? "已有本机 vault 记录" : "首次打开";
    els.dataPath.textContent = state.dataPath || "-";
    els.list.innerHTML = state.vaults.length
      ? state.vaults.map(vaultItemTemplate).join("")
      : '<div class="vault-picker-empty">暂无 vault</div>';
    renderMessage();
  }

  function renderMessage() {
    els.message.textContent = state.message || "";
    els.message.className = "vault-picker-message" + (state.messageType ? " is-" + state.messageType : "");
  }
}

function template() {
  return `
    <main class="vault-picker-shell">
      <section class="vault-picker-panel">
        <header class="vault-picker-header">
          <div class="vault-picker-mark">${FOLDER_ICON}</div>
          <div>
            <h1>选择 Vault</h1>
            <p data-vault-status>正在检查</p>
          </div>
        </header>

        <div class="vault-picker-meta">
          <span>本机记录</span>
          <strong data-vault-data-path>-</strong>
        </div>

        <form class="vault-picker-form" data-vault-form>
          <input data-vault-path type="text" placeholder="/Users/litao/Documents/memo-vault" autocomplete="off" />
          <button class="vault-picker-button is-primary" type="submit">${CHECK_ICON}<span>打开</span></button>
        </form>

        <div class="vault-picker-actions">
          <button class="vault-picker-button" type="button" data-action="chooseVault">${PLUS_ICON}<span>选择目录</span></button>
        </div>

        <section class="vault-picker-section">
          <div class="vault-picker-section-title">最近 Vault</div>
          <div class="vault-picker-list" data-vault-list></div>
        </section>

        <div class="vault-picker-message" data-vault-message role="status"></div>
      </section>
    </main>
  `;
}

function vaultItemTemplate(vault) {
  const path = String(vault.path || "");
  return `
    <button class="vault-picker-item" type="button" data-action="openVault" data-vault-path="${escapeAttr(path)}">
      <span class="vault-picker-item-icon">${FOLDER_ICON}</span>
      <span class="vault-picker-item-copy">
        <strong>${escapeHTML(vault.name || "Vault")}</strong>
        <small>${escapeHTML(path)}</small>
      </span>
    </button>
  `;
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
