(function () {
  try {
    function ensure_cbs() {
      if (!window.invoke_cbs) {
        Object.defineProperty(window, "invoke_cbs", {
          value: {},
          writable: true,
          configurable: true,
        });
      }
      if (!window._goCallbacks) {
        Object.defineProperty(window, "_goCallbacks", {
          value: window.invoke_cbs,
          writable: true,
          configurable: true,
        });
      } else if (window._goCallbacks !== window.invoke_cbs) {
        window._goCallbacks = window.invoke_cbs;
      }
    }
    function ensure_go_msg_handlers() {
      if (!window.__goMessageHandlers) {
        Object.defineProperty(window, "__goMessageHandlers", {
          value: [],
          writable: true,
          configurable: true,
        });
      }
      if (!window.__receiveGoMessage) {
        Object.defineProperty(window, "__receiveGoMessage", {
          value: function (payload) {
            ensure_go_msg_handlers();
            var list = window.__goMessageHandlers || [];
            console.log("before invoke handlers", list);
            for (var i = 0; i < list.length; i++) {
              try {
                list[i](payload);
              } catch (_e) {}
            }
          },
          writable: true,
          configurable: true,
        });
      }
      if (!window.onGoMessage) {
        Object.defineProperty(window, "onGoMessage", {
          value: function (handler) {
            if (typeof handler !== "function") {
              return;
            }
            ensure_go_msg_handlers();
            window.__goMessageHandlers.push(handler);
          },
          writable: true,
          configurable: true,
          enumerable: false,
        });
        window.onGoMessage((payload) => {
          console.log(payload);
        });
      }
    }
    function notify_go_ready() {
      try {
        post_message_to_go({
          id:
            "go_ready_" +
            String(Date.now()) +
            Math.random().toString(16).slice(2),
          method: "__bridge_ready__",
          args: [],
        });
      } catch (_e) {}
    }
    function post_message_to_go(payload) {
      if (
        window.webkit &&
        window.webkit.messageHandlers &&
        window.webkit.messageHandlers.go
      ) {
        try {
          window.webkit.messageHandlers.go.postMessage(JSON.stringify(payload));
          return true;
        } catch (e) {}
      }
      if (
        window.chrome &&
        window.chrome.webview &&
        typeof window.chrome.webview.postMessage === "function"
      ) {
        try {
          window.chrome.webview.postMessage(JSON.stringify(payload));
          return true;
        } catch (e) {}
      }
      return false;
    }
    function invoke(url, args) {
      return new Promise(function (resolve, reject) {
        const id = String(Date.now()) + Math.random().toString(16).slice(2);
        const payload = {
          id: id,
          method: url,
          headers: args.headers,
          args: args.args,
        };
        ensure_cbs();
        window.invoke_cbs[id] = function (result) {
          delete window.invoke_cbs[id];
          if (typeof result === "string") {
            try {
              resolve(JSON.parse(result));
              return;
            } catch (e) {}
          }
          resolve(result);
        };
        if (!post_message_to_go(payload)) {
          delete window.invoke_cbs[id];
          reject(new Error("go bridge not available"));
        }
      });
    }
    Object.defineProperty(window, "invoke", {
      value: invoke,
      writable: false,
      configurable: false,
      enumerable: false,
    });
    if (!window.goCall) {
      Object.defineProperty(window, "goCall", {
        value: invoke,
        writable: false,
        configurable: false,
        enumerable: false,
      });
    }
    ensure_go_msg_handlers();
    notify_go_ready();
    Object.defineProperty(invoke, "toString", {
      value: function () {
        return "function invoke() { [native code] }";
      },
      writable: false,
      configurable: false,
      enumerable: false,
    });
    if (
      !Object.prototype.hasOwnProperty.call(
        window,
        "__linkInterceptorInstalled__",
      )
    ) {
      Object.defineProperty(window, "__linkInterceptorInstalled__", {
        value: true,
        writable: false,
        configurable: false,
      });
      function showConfirm(url, onConfirm) {
        var overlay = document.createElement("div");
        overlay.style.position = "fixed";
        overlay.style.left = "0";
        overlay.style.top = "0";
        overlay.style.right = "0";
        overlay.style.bottom = "0";
        overlay.style.background = "rgba(0,0,0,0.4)";
        overlay.style.zIndex = "2147483647";
        overlay.style.display = "flex";
        overlay.style.alignItems = "center";
        overlay.style.justifyContent = "center";

        var dialog = document.createElement("div");
        dialog.style.background = "#fff";
        dialog.style.padding = "16px";
        dialog.style.borderRadius = "8px";
        dialog.style.maxWidth = "80%";
        dialog.style.boxShadow = "0 4px 12px rgba(0,0,0,0.2)";
        dialog.style.fontFamily =
          "system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif";

        var title = document.createElement("div");
        title.textContent = "将打开链接";
        title.style.fontSize = "16px";
        title.style.marginBottom = "8px";
        title.style.fontWeight = "600";

        var urlEl = document.createElement("div");
        urlEl.textContent = url;
        urlEl.style.wordBreak = "break-all";
        urlEl.style.margin = "8px 0";
        urlEl.style.color = "#333";

        var buttons = document.createElement("div");
        buttons.style.display = "flex";
        buttons.style.gap = "8px";
        buttons.style.justifyContent = "flex-end";
        buttons.style.marginTop = "12px";

        var cancelBtn = document.createElement("button");
        cancelBtn.textContent = "取消";
        cancelBtn.style.padding = "6px 12px";
        cancelBtn.style.borderRadius = "4px";
        cancelBtn.style.border = "1px solid #d9d9d9";
        cancelBtn.style.background = "#fff";
        cancelBtn.style.cursor = "pointer";

        var okBtn = document.createElement("button");
        okBtn.textContent = "确认";
        okBtn.style.padding = "6px 12px";
        okBtn.style.borderRadius = "4px";
        okBtn.style.border = "1px solid #1677ff";
        okBtn.style.background = "#1677ff";
        okBtn.style.color = "#fff";
        okBtn.style.cursor = "pointer";

        cancelBtn.onclick = function () {
          try {
            document.body.removeChild(overlay);
          } catch (_e) {}
        };
        okBtn.onclick = function () {
          try {
            document.body.removeChild(overlay);
          } catch (_e) {}
          try {
            onConfirm();
          } catch (_e) {}
        };

        dialog.appendChild(title);
        dialog.appendChild(urlEl);
        buttons.appendChild(cancelBtn);
        buttons.appendChild(okBtn);
        dialog.appendChild(buttons);
        overlay.appendChild(dialog);
        document.body.appendChild(overlay);
      }

      document.addEventListener(
        "click",
        function (ev) {
          try {
            var t = ev.target;
            var a = null;
            if (t && typeof t.closest === "function") {
              a = t.closest("a");
            }
            if (!a) return;
            var href = a.getAttribute("href") || a.href || "";
            href = href ? String(href).trim() : "";
            if (!href) return;
            if (href === "#") return;
            var lower = href.toLowerCase();
            if (lower.indexOf("javascript:") === 0) return;
            if (ev.defaultPrevented) return;
            if (ev.button !== 0) return;
            if (ev.metaKey || ev.ctrlKey || ev.shiftKey || ev.altKey) return;
            ev.preventDefault();
            var isBlank = a.target === "_blank";
            showConfirm(href, function () {
              if (isBlank) {
                try {
                  window.open(href);
                } catch (_e) {}
              } else {
                try {
                  window.location.href = href;
                } catch (_e) {}
              }
            });
          } catch (_e) {}
        },
        true,
      );
    }
  } catch (e) {}
})();
