const render = ($root) => {
  const { innerWidth, innerHeight } = window;

  const isWebView2 = !!(window.chrome && window.chrome.webview);
  const titlebarHeight = 42;
  const captionButtonWidth = 44;

  const shell = document.createElement("div");
  shell.className = "reader-shell";

  let contentHost = shell;

  if (isWebView2) {
    const titlebar = document.createElement("div");
    titlebar.className = "reader-titlebar";

    const edgeTop = document.createElement("div");
    edgeTop.className = "reader-edge-indicator top";
    shell.appendChild(edgeTop);

    const edgeBottom = document.createElement("div");
    edgeBottom.className = "reader-edge-indicator bottom";
    shell.appendChild(edgeBottom);

    const edgeLeft = document.createElement("div");
    edgeLeft.className = "reader-edge-indicator left";
    shell.appendChild(edgeLeft);

    const edgeRight = document.createElement("div");
    edgeRight.className = "reader-edge-indicator right";
    shell.appendChild(edgeRight);

    const drag = document.createElement("div");
    drag.className = "reader-titlebar-drag";
    drag.addEventListener("mousedown", (e) => {
      if (e.button !== 0) return;
      invoke("__velo/window/start_drag", { method: "GET", args: {} }).catch(() => {});
    });

    const title = document.createElement("div");
    title.className = "reader-titlebar-title";
    title.textContent = "Reader";
    drag.appendChild(title);

    const controls = document.createElement("div");
    controls.className = "reader-titlebar-controls";

    const mkBtn = (key, svg, title) => {
      const el = document.createElement("button");
      el.className = "reader-control-btn";
      el.dataset.btn = key;
      el.title = title;
      el.innerHTML = svg;
      return el;
    };

    const pinBtn = mkBtn(
      "pin",
      '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 2v8m0 0l3-3m-3 3L9 7"/><circle cx="12" cy="14" r="6"/></svg>',
      "固定窗口"
    );
    pinBtn.classList.add("pinned");
    pinBtn.addEventListener("click", () => {
      const isPinned = pinBtn.classList.contains("pinned");
      invoke("/api/window/set_pinned", {
        method: "GET",
        args: { pinned: !isPinned },
      }).then(() => {
        if (isPinned) {
          pinBtn.classList.remove("pinned");
        } else {
          pinBtn.classList.add("pinned");
        }
      }).catch(() => {});
    });

    const hideBtn = mkBtn(
      "hide",
      '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>',
      "隐藏窗口"
    );
    hideBtn.addEventListener("click", () => {
      invoke("/api/window/hide", { method: "GET", args: {} }).catch(() => {});
    });

    const closeBtn = mkBtn(
      "close",
      '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M18 6L6 18M6 6l12 12"/></svg>',
      "关闭"
    );
    closeBtn.addEventListener("click", () => {
      invoke("__velo/window/close", { method: "GET", args: {} }).catch(() => {});
    });

    controls.appendChild(pinBtn);
    controls.appendChild(hideBtn);
    controls.appendChild(closeBtn);

    titlebar.appendChild(drag);
    titlebar.appendChild(controls);

    shell.appendChild(titlebar);

    const content = document.createElement("div");
    content.className = "reader-content";
    contentHost = content;

    shell.appendChild(content);

    const checkEdge = (e) => {
      const rect = shell.getBoundingClientRect();
      const edgeThreshold = 20;

      const edges = {
        top: rect.top < edgeThreshold,
        bottom: rect.bottom > window.innerHeight - edgeThreshold,
        left: rect.left < edgeThreshold,
        right: rect.right > window.innerWidth - edgeThreshold,
      };

      edgeTop.classList.toggle("active", edges.top);
      edgeBottom.classList.toggle("active", edges.bottom);
      edgeLeft.classList.toggle("active", edges.left);
      edgeRight.classList.toggle("active", edges.right);

      if (edges.top) {
        titlebar.classList.add("highlight-top");
        titlebar.classList.remove("highlight-bottom");
      } else if (edges.bottom) {
        titlebar.classList.add("highlight-bottom");
        titlebar.classList.remove("highlight-top");
      } else {
        titlebar.classList.remove("highlight-top", "highlight-bottom");
      }
    };

    shell.addEventListener("mouseenter", () => {
      const interval = setInterval(checkEdge, 100);
      shell._edgeInterval = interval;
    });

    shell.addEventListener("mouseleave", () => {
      if (shell._edgeInterval) {
        clearInterval(shell._edgeInterval);
        shell._edgeInterval = null;
      }
      edgeTop.classList.remove("active");
      edgeBottom.classList.remove("active");
      edgeLeft.classList.remove("active");
      edgeRight.classList.remove("active");
      titlebar.classList.remove("highlight-top", "highlight-bottom");
    });
  }

  const appData = {
    articles: [
      { id: 1, title: "深入理解 JavaScript 闭包", meta: "技术 · 2024-01-15" },
      { id: 2, title: "React 18 新特性解析", meta: "前端 · 2024-01-14" },
      { id: 3, title: "TypeScript 最佳实践", meta: "开发 · 2024-01-13" },
      { id: 4, title: "CSS Grid 布局详解", meta: "前端 · 2024-01-12" },
      { id: 5, title: "Node.js 性能优化", meta: "后端 · 2024-01-11" },
      { id: 6, title: "微服务架构设计", meta: "架构 · 2024-01-10" },
    ],
  };

  const renderList = () => {
    const list = document.createElement("div");
    list.className = "reader-list";

    appData.articles.forEach((article) => {
      const item = document.createElement("div");
      item.className = "reader-item";

      const title = document.createElement("div");
      title.className = "reader-item-title";
      title.textContent = article.title;

      const meta = document.createElement("div");
      meta.className = "reader-item-meta";
      meta.textContent = article.meta;

      item.appendChild(title);
      item.appendChild(meta);
      list.appendChild(item);
    });

    return list;
  };

  const renderContent = () => {
    return renderList();
  };

  contentHost.appendChild(renderContent());
  $root.appendChild(shell);
};

document.addEventListener("DOMContentLoaded", function () {
  const $root = document.querySelector("#root");
  if (!$root) {
    console.error("[Render] Root element not found");
    return;
  }
  render($root);
});