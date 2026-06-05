(function () {
  const state = {
    config: null,
    messages: new Map(),
    order: [],
    peerOnline: false,
    peerName: "",
    sending: false,
    windowName: "demo-im",
  };

  const els = {};

  function $(id) {
    return document.getElementById(id);
  }

  function setComposeState(text, kind) {
    els.composeState.textContent = text || "";
    els.composeState.classList.toggle("error", kind === "error");
  }

  function invokeGo(path, args) {
    if (typeof window.invoke !== "function") {
      return Promise.reject(new Error("go bridge not available"));
    }
    return window.invoke(path, { args: args || {} }).then(function (response) {
      if (!response || response.code !== 0) {
        throw new Error((response && response.msg) || "request failed");
      }
      return response.data;
    });
  }

  function renderIdentity() {
    const cfg = state.config;
    if (!cfg) return;

    state.windowName = "demo-im-" + cfg.role;
    els.identity.textContent = cfg.display_name + " · " + cfg.listen_port;
    els.localPort.textContent = String(cfg.listen_port);
    els.peerPort.textContent = String(cfg.peer_port);
    els.role.textContent = cfg.role.toUpperCase();
    els.instanceId.textContent = cfg.instance_id;
    els.version.textContent = "Demo IM " + (cfg.started_at || "");
    els.roomSubtitle.textContent =
      cfg.display_name + " on " + cfg.listen_url.replace("http://", "");
  }

  function renderPeer() {
    els.peerDot.classList.toggle("online", state.peerOnline);
    els.peerState.textContent = state.peerOnline ? "online" : "offline";
    els.peerName.textContent = state.peerName || "Peer";
  }

  function formatTime(value) {
    const date = value ? new Date(value) : new Date();
    if (Number.isNaN(date.getTime())) return "";
    return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
  }

  function deliveryText(message) {
    if (message.direction !== "outgoing") return "received";
    if (message.delivery === "delivered") return "delivered";
    if (message.delivery === "failed") return "failed";
    return "sending";
  }

  function upsertMessage(message) {
    if (!message || !message.id) return;
    if (!state.messages.has(message.id)) {
      state.order.push(message.id);
    }
    state.messages.set(message.id, message);
    renderMessages();
  }

  function setMessages(messages) {
    state.messages.clear();
    state.order = [];
    (messages || []).forEach(function (message) {
      if (message && message.id) {
        state.order.push(message.id);
        state.messages.set(message.id, message);
      }
    });
    renderMessages();
  }

  function renderMessages() {
    const list = els.messageList;
    list.innerHTML = "";

    if (state.order.length === 0) {
      const empty = document.createElement("div");
      empty.className = "empty-state";
      empty.textContent = "No messages";
      list.appendChild(empty);
    } else {
      state.order.forEach(function (id) {
        const message = state.messages.get(id);
        if (!message) return;

        const row = document.createElement("div");
        row.className = "message-row " + (message.direction || "incoming");

        const bubble = document.createElement("div");
        bubble.className = "bubble";

        const meta = document.createElement("div");
        meta.className = "message-meta";

        const sender = document.createElement("span");
        sender.className = "sender";
        sender.textContent =
          message.direction === "outgoing"
            ? (state.config && state.config.display_name) || "Me"
            : message.from_name || state.peerName || "Peer";

        const time = document.createElement("span");
        time.className = "time";
        time.textContent = formatTime(message.created_at);

        const text = document.createElement("div");
        text.className = "message-text";
        text.textContent = message.text || "";

        const delivery = document.createElement("div");
        delivery.className =
          "delivery" + (message.delivery === "failed" ? " failed" : "");
        delivery.textContent =
          deliveryText(message) + (message.error ? " · " + message.error : "");

        meta.appendChild(sender);
        meta.appendChild(time);
        bubble.appendChild(meta);
        bubble.appendChild(text);
        bubble.appendChild(delivery);
        row.appendChild(bubble);
        list.appendChild(row);
      });
    }

    const count = state.order.length;
    els.messageCount.textContent = count + (count === 1 ? " message" : " messages");
    requestAnimationFrame(function () {
      list.scrollTop = list.scrollHeight;
    });
  }

  function applyGoMessage(payload) {
    if (!payload || typeof payload !== "object") return;
    if (
      payload.type === "message_sent" ||
      payload.type === "message_received" ||
      payload.type === "message_delivery"
    ) {
      upsertMessage(payload.message);
      return;
    }
    if (payload.type === "peer_status") {
      state.peerOnline = Boolean(payload.online);
      if (payload.name) state.peerName = payload.name;
      renderPeer();
    }
  }

  function resizeInput() {
    els.input.style.height = "auto";
    const next = Math.min(120, Math.max(46, els.input.scrollHeight));
    els.input.style.height = next + "px";
  }

  async function sendMessage() {
    const text = els.input.value.trim();
    if (!text || state.sending) return;

    state.sending = true;
    els.sendButton.disabled = true;
    setComposeState("", "");

    try {
      await invokeGo("/api/message/send", { text: text });
      els.input.value = "";
      resizeInput();
      els.input.focus();
    } catch (error) {
      setComposeState(error.message || String(error), "error");
    } finally {
      state.sending = false;
      els.sendButton.disabled = false;
    }
  }

  async function loadApp() {
    try {
      const data = await invokeGo("/api/app", {});
      state.config = data.config;
      state.peerOnline = Boolean(data.peer_online);
      state.peerName = data.peer_name || "";
      renderIdentity();
      renderPeer();
      setMessages(data.messages || []);
    } catch (error) {
      setComposeState(error.message || String(error), "error");
    }
  }

  async function refreshPeerStatus() {
    try {
      const data = await invokeGo("/api/peer/status", {});
      state.peerOnline = Boolean(data.online);
      if (data.name) state.peerName = data.name;
      renderPeer();
    } catch (_error) {
      state.peerOnline = false;
      renderPeer();
    }
  }

  function bindEvents() {
    els.compose.addEventListener("submit", function (event) {
      event.preventDefault();
      sendMessage();
    });
    els.input.addEventListener("input", resizeInput);
    els.input.addEventListener("keydown", function (event) {
      if (event.key === "Enter" && !event.shiftKey) {
        event.preventDefault();
        sendMessage();
      }
    });

    if (typeof window.onGoMessage === "function") {
      window.onGoMessage(applyGoMessage);
    }

    setInterval(refreshPeerStatus, 4000);

    window.addEventListener("beforeunload", function () {
      const xhr = new XMLHttpRequest();
      xhr.open(
        "GET",
        "/api/window/state/snapshot?name=" + encodeURIComponent(state.windowName),
        false,
      );
      xhr.send();
    });
  }

  document.addEventListener("DOMContentLoaded", function () {
    els.identity = $("identity");
    els.localPort = $("local-port");
    els.peerPort = $("peer-port");
    els.peerName = $("peer-name");
    els.role = $("role");
    els.instanceId = $("instance-id");
    els.version = $("version");
    els.peerDot = $("peer-dot");
    els.peerState = $("peer-state");
    els.messageList = $("message-list");
    els.messageCount = $("message-count");
    els.roomSubtitle = $("room-subtitle");
    els.compose = $("compose");
    els.input = $("message-input");
    els.sendButton = $("send-button");
    els.composeState = $("compose-state");

    bindEvents();
    resizeInput();
    loadApp().then(function () {
      refreshPeerStatus();
      const ready = document.fonts && document.fonts.ready;
      const showWindow = function () {
        invokeGo("/api/window/show", {}).catch(function () {});
      };
      if (ready) {
        ready.then(showWindow);
      } else {
        showWindow();
      }
    });
  });
})();
