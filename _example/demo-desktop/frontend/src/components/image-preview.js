const PREVIEW_STORAGE_PREFIX = "demo-desktop:image-preview:";
const PREVIEW_STORAGE_INDEX = "demo-desktop:image-preview:index";
const MAX_STORED_PREVIEWS = 16;

const ICONS = {
  actual:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="4" y="4" width="16" height="16" rx="2"></rect><path d="M8 8h8v8H8z"></path></svg>',
  brush:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M9 18c-2 0-4 1-5 3 4 0 7-1 7-4"></path><path d="M14 5l5 5"></path><path d="M10 15 20 5a2.1 2.1 0 0 0-3-3L7 12z"></path></svg>',
  close:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 6 6 18"></path><path d="m6 6 12 12"></path></svg>',
  copy:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><rect x="9" y="9" width="11" height="11" rx="2"></rect><rect x="4" y="4" width="11" height="11" rx="2"></rect></svg>',
  download:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M12 3v12"></path><path d="m7 10 5 5 5-5"></path><path d="M5 21h14"></path></svg>',
  fit:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M8 3H5a2 2 0 0 0-2 2v3"></path><path d="M16 3h3a2 2 0 0 1 2 2v3"></path><path d="M8 21H5a2 2 0 0 1-2-2v-3"></path><path d="M16 21h3a2 2 0 0 0 2-2v-3"></path></svg>',
  hand:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M18 11V7a2 2 0 0 0-4 0v4"></path><path d="M14 10V5a2 2 0 0 0-4 0v6"></path><path d="M10 10V6a2 2 0 0 0-4 0v8"></path><path d="M6 14v-2a2 2 0 0 0-4 0v3a7 7 0 0 0 7 7h3a6 6 0 0 0 6-6v-5a2 2 0 0 0-4 0v1"></path></svg>',
  pin:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m15 4 5 5-4 4v5l-2 2-5-5-5-5 2-2h5z"></path><path d="m9 15-5 5"></path></svg>',
  reset:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 12a9 9 0 1 0 3-6.7"></path><path d="M3 4v6h6"></path></svg>',
  rotateLeft:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 12a9 9 0 1 0 3-6.7"></path><path d="M3 4v6h6"></path></svg>',
  rotateRight:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M21 12a9 9 0 1 1-3-6.7"></path><path d="M21 4v6h-6"></path></svg>',
  trash:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 6h18"></path><path d="M8 6V4h8v2"></path><path d="M19 6l-1 14H6L5 6"></path></svg>',
  zoomIn:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7"></circle><path d="M11 8v6"></path><path d="M8 11h6"></path><path d="m21 21-4.3-4.3"></path></svg>',
  zoomOut:
    '<svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="7"></circle><path d="M8 11h6"></path><path d="m21 21-4.3-4.3"></path></svg>',
};

const PREVIEW_CSS = `
  * {
    box-sizing: border-box;
  }

  html,
  body {
    height: 100%;
    margin: 0;
    overscroll-behavior: none;
  }

  body.image-preview-page {
    background: #1f1f1f;
    color: rgba(255, 255, 255, 0.88);
    font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    overflow: hidden;
  }

  body.image-preview-page #root {
    height: 100%;
    min-height: 0;
    min-width: 0;
    width: 100%;
  }

  .image-preview-shell {
    background: #202020;
    display: grid;
    grid-template-rows: 48px minmax(0, 1fr);
    height: 100%;
    min-width: 0;
  }

  .image-preview-toolbar {
    align-items: center;
    background: #39393b;
    border-bottom: 1px solid rgba(255, 255, 255, 0.08);
    display: grid;
    gap: 12px;
    grid-template-columns: 78px minmax(120px, 1fr) auto;
    height: 48px;
    min-width: 0;
    padding: 0 14px;
    user-select: none;
  }

  .image-preview-native-controls {
    height: 100%;
    pointer-events: none;
  }

  .image-preview-title {
    color: rgba(255, 255, 255, 0.74);
    font-size: 13px;
    min-width: 0;
    overflow: hidden;
    text-align: center;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .image-preview-tools {
    align-items: center;
    display: flex;
    gap: 7px;
    justify-content: flex-end;
    min-width: 0;
    overflow-x: auto;
    scrollbar-width: none;
  }

  .image-preview-tools::-webkit-scrollbar {
    display: none;
  }

  .image-preview-divider {
    background: rgba(255, 255, 255, 0.14);
    flex: 0 0 auto;
    height: 24px;
    width: 1px;
  }

  .image-preview-button,
  .image-preview-swatch {
    align-items: center;
    background: transparent;
    border: 1px solid transparent;
    border-radius: 6px;
    color: rgba(255, 255, 255, 0.68);
    cursor: pointer;
    display: inline-flex;
    flex: 0 0 auto;
    height: 32px;
    justify-content: center;
    padding: 0;
    width: 32px;
  }

  .image-preview-button:hover,
  .image-preview-button.is-active,
  .image-preview-swatch:hover,
  .image-preview-swatch.is-active {
    background: rgba(255, 255, 255, 0.11);
    border-color: rgba(255, 255, 255, 0.1);
    color: #fff;
  }

  .image-preview-button svg {
    fill: none;
    height: 18px;
    stroke: currentColor;
    stroke-linecap: round;
    stroke-linejoin: round;
    stroke-width: 2;
    width: 18px;
  }

  .image-preview-zoom-label {
    color: rgba(255, 255, 255, 0.72);
    flex: 0 0 auto;
    font-size: 12px;
    font-variant-numeric: tabular-nums;
    min-width: 48px;
    text-align: center;
  }

  .image-preview-swatch::before {
    background: var(--preview-color);
    border: 1px solid rgba(255, 255, 255, 0.38);
    border-radius: 999px;
    content: "";
    display: block;
    height: 16px;
    width: 16px;
  }

  .image-preview-size {
    accent-color: #66d9a8;
    flex: 0 0 auto;
    width: 74px;
  }

  .image-preview-stage {
    background:
      linear-gradient(45deg, rgba(255, 255, 255, 0.035) 25%, transparent 25%),
      linear-gradient(-45deg, rgba(255, 255, 255, 0.035) 25%, transparent 25%),
      linear-gradient(45deg, transparent 75%, rgba(255, 255, 255, 0.035) 75%),
      linear-gradient(-45deg, transparent 75%, rgba(255, 255, 255, 0.035) 75%),
      #222;
    background-position: 0 0, 0 10px, 10px -10px, -10px 0;
    background-size: 20px 20px;
    min-height: 0;
    overflow: hidden;
    position: relative;
    touch-action: none;
  }

  .image-preview-stage canvas {
    display: block;
    height: 100%;
    width: 100%;
  }

  .image-preview-stage.is-move canvas {
    cursor: grab;
  }

  .image-preview-stage.is-moving canvas {
    cursor: grabbing;
  }

  .image-preview-stage.is-draw canvas {
    cursor: crosshair;
  }

  .image-preview-state,
  .image-preview-toast {
    left: 50%;
    position: absolute;
    transform: translateX(-50%);
  }

  .image-preview-state {
    background: rgba(0, 0, 0, 0.48);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    color: rgba(255, 255, 255, 0.78);
    font-size: 14px;
    max-width: min(520px, calc(100vw - 40px));
    padding: 14px 16px;
    text-align: center;
    top: 50%;
    transform: translate(-50%, -50%);
  }

  .image-preview-state[hidden] {
    display: none;
  }

  .image-preview-toast {
    background: rgba(0, 0, 0, 0.76);
    border: 1px solid rgba(255, 255, 255, 0.12);
    border-radius: 999px;
    bottom: 22px;
    color: rgba(255, 255, 255, 0.9);
    font-size: 13px;
    opacity: 0;
    padding: 8px 13px;
    pointer-events: none;
    transition: opacity 160ms ease, transform 160ms ease;
    white-space: nowrap;
  }

  .image-preview-toast.is-visible {
    opacity: 1;
    transform: translate(-50%, -4px);
  }

  body.is-fixed-window .image-preview-toolbar {
    box-shadow: inset 0 -2px 0 rgba(102, 217, 168, 0.66);
  }

  @media (max-width: 720px) {
    .image-preview-toolbar {
      grid-template-columns: 76px minmax(80px, 1fr) auto;
      padding: 0 8px;
    }

    .image-preview-title {
      text-align: left;
    }

    .image-preview-size {
      width: 58px;
    }
  }
`;

function storeImagePreviewPayload(payload) {
  const id = createPreviewID();
  const normalized = normalizePreviewPayload(payload);
  const item = {
    ...normalized,
    createdAt: new Date().toISOString(),
    id,
  };

  try {
    localStorage.setItem(PREVIEW_STORAGE_PREFIX + id, JSON.stringify(item));
    rememberPreviewID(id);
  } catch (_) {}

  return id;
}

function createPreviewID() {
  const random = Math.random().toString(36).slice(2, 9);
  return Date.now().toString(36) + "-" + random;
}

function rememberPreviewID(id) {
  const ids = readPreviewIndex().filter((item) => item !== id);
  ids.unshift(id);
  const stale = ids.slice(MAX_STORED_PREVIEWS);
  localStorage.setItem(PREVIEW_STORAGE_INDEX, JSON.stringify(ids.slice(0, MAX_STORED_PREVIEWS)));
  stale.forEach((item) => {
    try {
      localStorage.removeItem(PREVIEW_STORAGE_PREFIX + item);
    } catch (_) {}
  });
}

function readPreviewIndex() {
  try {
    const parsed = JSON.parse(localStorage.getItem(PREVIEW_STORAGE_INDEX) || "[]");
    return Array.isArray(parsed) ? parsed.map((item) => String(item || "")).filter(Boolean) : [];
  } catch (_) {
    return [];
  }
}

function readStoredPreviewPayload(id) {
  const key = PREVIEW_STORAGE_PREFIX + String(id || "").trim();
  if (!key || key === PREVIEW_STORAGE_PREFIX) return null;
  try {
    return normalizePreviewPayload(JSON.parse(localStorage.getItem(key) || "null"));
  } catch (_) {
    return null;
  }
}

function normalizePreviewPayload(payload) {
  const input = payload && typeof payload === "object" ? payload : {};
  return {
    caption: String(input.caption || ""),
    source: String(input.source || input.src || ""),
    src: String(input.src || input.source || "").trim(),
    title: String(input.title || input.caption || "图片预览").trim() || "图片预览",
  };
}

function previewPayloadFromElement(element) {
  const node = element || null;
  if (!node) return null;

  const image = node.tagName === "IMG" ? node : node.querySelector && node.querySelector("img");
  const dataset = node.dataset || {};
  const src = String(dataset.imagePreviewSrc || dataset.previewSrc || (image && (image.currentSrc || image.src)) || "").trim();
  if (!src) return null;

  return normalizePreviewPayload({
    caption: dataset.imagePreviewCaption || "",
    source: dataset.imagePreviewSource || src,
    src,
    title: dataset.imagePreviewTitle || (image && image.alt) || "",
  });
}

function openImagePreviewPayload(payload) {
  const normalized = normalizePreviewPayload(payload);
  if (!normalized.src) return Promise.reject(new Error("图片地址为空"));

  const id = storeImagePreviewPayload(normalized);
  const fallbackURL = imagePreviewWindowURL("image-preview.html", id, normalized);

  if (typeof invoke !== "function") {
    window.open(fallbackURL, "_blank", "noopener");
    return Promise.resolve();
  }

  const params = new URLSearchParams();
  params.set("pathname", "/image-preview");
  params.set("previewId", id);
  if (normalized.src.length <= 1800) params.set("previewSrc", normalized.src);
  if (normalized.title.length <= 240) params.set("previewTitle", normalized.title);
  const api = "/api/open_window?" + params.toString();
  return invoke(api, { method: "GET" }).then(function (resp) {
    if (!resp || resp.code !== 0) {
      throw new Error((resp && resp.msg) || "打开图片预览失败");
    }
    return resp;
  });
}

function imagePreviewWindowURL(base, id, payload) {
  const params = new URLSearchParams();
  params.set("id", id);
  if (payload.src && payload.src.length <= 1800) params.set("src", payload.src);
  if (payload.title && payload.title.length <= 240) params.set("title", payload.title);
  return base + "?" + params.toString();
}

function openImagePreviewFromElement(element) {
  const payload = previewPayloadFromElement(element);
  if (!payload) return Promise.reject(new Error("图片地址为空"));
  return openImagePreviewPayload(payload);
}

function mountImagePreview(root) {
  injectPreviewCSS();
  const payload = resolveWindowPreviewPayload();
  const state = {
    activePointer: 0,
    activeStroke: null,
    canvasHeight: 0,
    canvasWidth: 0,
    color: "#ff4d4f",
    fixed: false,
    image: null,
    imageHeight: 0,
    imageWidth: 0,
    loaded: false,
    mode: "move",
    moving: false,
    panStart: null,
    panX: 0,
    panY: 0,
    payload,
    resizeObserver: null,
    rotation: 0,
    scale: 1,
    strokeWidth: 5,
    strokes: [],
    toastTimer: null,
  };

  root.innerHTML = previewTemplate(payload);

  const els = {
    canvas: root.querySelector("[data-preview-canvas]"),
    state: root.querySelector("[data-preview-state]"),
    stage: root.querySelector("[data-preview-stage]"),
    title: root.querySelector("[data-preview-title]"),
    toast: root.querySelector("[data-preview-toast]"),
    zoom: root.querySelector("[data-preview-zoom]"),
  };

  const ctx = els.canvas.getContext("2d");
  bindPreviewEvents(root, els, state, ctx);
  updateMode(root, els, state);
  updateToolbarState(root, state);
  resizeCanvas(els, state, ctx);

  if (!payload.src) {
    showState(els, "缺少图片地址");
    return {
      destroy() {
        unmountImagePreview(state);
      },
    };
  }

  loadPreviewImage(els, state, ctx);

  return {
    destroy() {
      unmountImagePreview(state);
    },
  };
}

function resolveWindowPreviewPayload() {
  const params = new URLSearchParams(window.location.search || "");
  const id = params.get("id") || params.get("previewId") || "";
  const stored = readStoredPreviewPayload(id);
  if (stored && stored.src) return stored;
  return normalizePreviewPayload({
    caption: params.get("caption") || "",
    source: params.get("source") || params.get("src") || "",
    src: params.get("src") || "",
    title: params.get("title") || "",
  });
}

function previewTemplate(payload) {
  const title = escapeHTML((payload && payload.title) || "图片预览");
  return `
    <div class="image-preview-shell">
      <header class="image-preview-toolbar velo-drag" data-velo-drag>
        <div class="image-preview-native-controls" aria-hidden="true"></div>
        <div class="image-preview-title" data-preview-title>${title}</div>
        <div class="image-preview-tools velo-no-drag">
          ${toolbarButton("toggleFixed", ICONS.pin, "固定在最上方")}
          <span class="image-preview-divider" aria-hidden="true"></span>
          ${toolbarButton("fit", ICONS.fit, "适应窗口")}
          ${toolbarButton("actual", ICONS.actual, "实际大小")}
          ${toolbarButton("zoomOut", ICONS.zoomOut, "缩小")}
          <span class="image-preview-zoom-label" data-preview-zoom>100%</span>
          ${toolbarButton("zoomIn", ICONS.zoomIn, "放大")}
          <span class="image-preview-divider" aria-hidden="true"></span>
          ${toolbarButton("rotateLeft", ICONS.rotateLeft, "向左旋转")}
          ${toolbarButton("rotateRight", ICONS.rotateRight, "向右旋转")}
          <span class="image-preview-divider" aria-hidden="true"></span>
          ${toolbarButton("move", ICONS.hand, "移动")}
          ${toolbarButton("draw", ICONS.brush, "标注")}
          ${colorButton("#ff4d4f", "红色标注")}
          ${colorButton("#f7b731", "黄色标注")}
          ${colorButton("#40c057", "绿色标注")}
          ${colorButton("#4dabf7", "蓝色标注")}
          <input class="image-preview-size" type="range" min="2" max="18" value="5" data-preview-size title="画笔粗细" aria-label="画笔粗细" />
          ${toolbarButton("clearAnnotations", ICONS.trash, "清除标注")}
          <span class="image-preview-divider" aria-hidden="true"></span>
          ${toolbarButton("copy", ICONS.copy, "复制当前图片")}
          ${toolbarButton("download", ICONS.download, "下载当前图片")}
          ${toolbarButton("close", ICONS.close, "关闭窗口")}
        </div>
      </header>
      <main class="image-preview-stage" data-preview-stage>
        <canvas data-preview-canvas></canvas>
        <div class="image-preview-state" data-preview-state>正在载入图片...</div>
        <div class="image-preview-toast" data-preview-toast role="status"></div>
      </main>
    </div>
  `;
}

function toolbarButton(action, icon, label) {
  return `
    <button class="image-preview-button" type="button" data-preview-action="${escapeAttr(action)}" title="${escapeAttr(label)}" aria-label="${escapeAttr(label)}">
      ${icon}
    </button>
  `;
}

function colorButton(color, label) {
  return `
    <button class="image-preview-swatch" type="button" data-preview-color="${escapeAttr(color)}" style="--preview-color: ${escapeAttr(color)}" title="${escapeAttr(label)}" aria-label="${escapeAttr(label)}"></button>
  `;
}

function bindPreviewEvents(root, els, state, ctx) {
  root.addEventListener("click", function (event) {
    const color = closestElement(event.target, "[data-preview-color]");
    if (color && root.contains(color)) {
      state.color = color.dataset.previewColor || state.color;
      state.mode = "draw";
      updateMode(root, els, state);
      updateToolbarState(root, state);
      return;
    }

    const action = closestElement(event.target, "[data-preview-action]");
    if (!action || !root.contains(action)) return;
    runPreviewAction(action.dataset.previewAction, root, els, state, ctx);
  });

  root.addEventListener("input", function (event) {
    const size = closestElement(event.target, "[data-preview-size]");
    if (!size || !root.contains(size)) return;
    state.strokeWidth = clamp(Number(size.value || 5), 2, 18);
  });

  els.stage.addEventListener("pointerdown", function (event) {
    if (!state.loaded || event.button !== 0) return;
    els.stage.setPointerCapture(event.pointerId);
    state.activePointer = event.pointerId;

    if (state.mode === "draw") {
      beginStroke(event, els, state, ctx);
      return;
    }

    beginPan(event, els, state);
  });

  els.stage.addEventListener("pointermove", function (event) {
    if (!state.loaded || state.activePointer !== event.pointerId) return;
    if (state.activeStroke) {
      extendStroke(event, els, state, ctx);
      return;
    }
    if (state.moving) {
      movePan(event, els, state, ctx);
    }
  });

  els.stage.addEventListener("pointerup", function (event) {
    endPointer(event, els, state, ctx);
  });

  els.stage.addEventListener("pointercancel", function (event) {
    endPointer(event, els, state, ctx);
  });

  els.stage.addEventListener(
    "wheel",
    function (event) {
      if (!state.loaded) return;
      event.preventDefault();
      const rect = els.stage.getBoundingClientRect();
      const center = {
        x: event.clientX - rect.left,
        y: event.clientY - rect.top,
      };
      zoomAt(Math.pow(1.0015, -event.deltaY), center, els, state, ctx);
    },
    { passive: false },
  );

  els.stage.addEventListener("dblclick", function () {
    if (!state.loaded) return;
    fitImage(els, state, ctx);
  });

  window.addEventListener("keydown", function (event) {
    if (event.defaultPrevented) return;
    if (event.key === "Escape") {
      closeWindow();
    } else if (event.key === "+" || event.key === "=") {
      zoomAt(1.18, canvasCenter(state), els, state, ctx);
    } else if (event.key === "-") {
      zoomAt(1 / 1.18, canvasCenter(state), els, state, ctx);
    } else if (event.key === "0") {
      setActualSize(els, state, ctx);
    } else if (event.key.toLowerCase() === "f") {
      fitImage(els, state, ctx);
    } else if (event.key.toLowerCase() === "r") {
      rotateImage(90, els, state, ctx);
    }
  });

  if (typeof ResizeObserver === "function") {
    state.resizeObserver = new ResizeObserver(function () {
      resizeCanvas(els, state, ctx);
      if (state.loaded && !state.userZoomed) fitImage(els, state, ctx);
    });
    state.resizeObserver.observe(els.stage);
  } else {
    window.addEventListener("resize", function () {
      resizeCanvas(els, state, ctx);
    });
  }
}

function runPreviewAction(action, root, els, state, ctx) {
  switch (action) {
    case "actual":
      setActualSize(els, state, ctx);
      break;
    case "clearAnnotations":
      state.strokes = [];
      state.activeStroke = null;
      renderPreview(els, state, ctx);
      showToast(els, "已清除标注");
      break;
    case "close":
      closeWindow();
      break;
    case "copy":
      copyRenderedImage(els, state);
      break;
    case "download":
      downloadRenderedImage(els, state);
      break;
    case "draw":
      state.mode = "draw";
      updateMode(root, els, state);
      updateToolbarState(root, state);
      break;
    case "fit":
      fitImage(els, state, ctx);
      break;
    case "move":
      state.mode = "move";
      updateMode(root, els, state);
      updateToolbarState(root, state);
      break;
    case "rotateLeft":
      rotateImage(-90, els, state, ctx);
      break;
    case "rotateRight":
      rotateImage(90, els, state, ctx);
      break;
    case "toggleFixed":
      toggleFixed(root, state);
      updateToolbarState(root, state);
      break;
    case "zoomIn":
      zoomAt(1.18, canvasCenter(state), els, state, ctx);
      break;
    case "zoomOut":
      zoomAt(1 / 1.18, canvasCenter(state), els, state, ctx);
      break;
    default:
      break;
  }
}

function loadPreviewImage(els, state, ctx) {
  showState(els, "正在载入图片...");
  document.title = state.payload.title || "图片预览";
  els.title.textContent = state.payload.title || "图片预览";

  const image = new Image();
  state.image = image;
  image.onload = function () {
    state.loaded = true;
    state.imageWidth = image.naturalWidth || image.width || 1;
    state.imageHeight = image.naturalHeight || image.height || 1;
    hideState(els);
    els.title.textContent =
      (state.payload.title || "图片预览") + " · " + state.imageWidth + "×" + state.imageHeight;
    fitImage(els, state, ctx);
  };
  image.onerror = function () {
    showState(els, "图片载入失败");
  };
  image.decoding = "async";
  image.src = state.payload.src;
}

function resizeCanvas(els, state, ctx) {
  const rect = els.stage.getBoundingClientRect();
  const width = Math.max(1, Math.round(rect.width || els.stage.clientWidth || 1));
  const height = Math.max(1, Math.round(rect.height || els.stage.clientHeight || 1));
  const dpr = Math.max(1, Math.min(3, window.devicePixelRatio || 1));
  state.canvasWidth = width;
  state.canvasHeight = height;
  els.canvas.width = Math.round(width * dpr);
  els.canvas.height = Math.round(height * dpr);
  els.canvas.style.width = width + "px";
  els.canvas.style.height = height + "px";
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  renderPreview(els, state, ctx);
}

function fitImage(els, state, ctx) {
  if (!state.loaded) return;
  const rotated = rotatedImageSize(state);
  const availableWidth = Math.max(1, state.canvasWidth - 56);
  const availableHeight = Math.max(1, state.canvasHeight - 56);
  const nextScale = Math.min(availableWidth / rotated.width, availableHeight / rotated.height, 1);
  state.scale = clamp(nextScale || 1, 0.02, 16);
  state.panX = 0;
  state.panY = 0;
  state.userZoomed = false;
  renderPreview(els, state, ctx);
}

function setActualSize(els, state, ctx) {
  if (!state.loaded) return;
  state.scale = 1;
  state.panX = 0;
  state.panY = 0;
  state.userZoomed = true;
  renderPreview(els, state, ctx);
}

function rotateImage(delta, els, state, ctx) {
  if (!state.loaded) return;
  state.rotation = normalizeRotation(state.rotation + delta);
  fitImage(els, state, ctx);
}

function zoomAt(factor, center, els, state, ctx) {
  if (!state.loaded) return;
  const before = viewportToImage(center.x, center.y, state);
  const nextScale = clamp(state.scale * factor, 0.02, 16);
  if (Math.abs(nextScale - state.scale) < 0.0001) return;
  state.scale = nextScale;
  const after = imageToViewport(before.x, before.y, state);
  state.panX += center.x - after.x;
  state.panY += center.y - after.y;
  state.userZoomed = true;
  renderPreview(els, state, ctx);
}

function beginPan(event, els, state) {
  state.moving = true;
  state.panStart = {
    clientX: event.clientX,
    clientY: event.clientY,
    panX: state.panX,
    panY: state.panY,
  };
  els.stage.classList.add("is-moving");
}

function movePan(event, els, state, ctx) {
  if (!state.panStart) return;
  state.panX = state.panStart.panX + event.clientX - state.panStart.clientX;
  state.panY = state.panStart.panY + event.clientY - state.panStart.clientY;
  state.userZoomed = true;
  renderPreview(els, state, ctx);
}

function beginStroke(event, els, state, ctx) {
  const point = pointerImagePoint(event, els, state);
  if (!point || !point.inside) return;
  state.activeStroke = {
    color: state.color,
    points: [point],
    width: state.strokeWidth,
  };
  state.strokes.push(state.activeStroke);
  renderPreview(els, state, ctx);
}

function extendStroke(event, els, state, ctx) {
  if (!state.activeStroke) return;
  const point = pointerImagePoint(event, els, state);
  if (!point) return;
  state.activeStroke.points.push(point);
  renderPreview(els, state, ctx);
}

function endPointer(event, els, state, ctx) {
  if (state.activePointer && state.activePointer !== event.pointerId) return;
  if (state.activeStroke && state.activeStroke.points.length < 2) {
    const point = state.activeStroke.points[0];
    state.activeStroke.points.push({ ...point, x: point.x + 0.01 });
  }
  state.activeStroke = null;
  state.moving = false;
  state.panStart = null;
  state.activePointer = 0;
  els.stage.classList.remove("is-moving");
  renderPreview(els, state, ctx);
}

function renderPreview(els, state, ctx) {
  if (!ctx) return;
  const dpr = Math.max(1, Math.min(3, window.devicePixelRatio || 1));
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, state.canvasWidth, state.canvasHeight);

  if (!state.loaded || !state.image) {
    updateZoomLabel(els, state);
    return;
  }

  ctx.save();
  applyImageTransform(ctx, state);
  ctx.imageSmoothingEnabled = true;
  ctx.imageSmoothingQuality = "high";
  ctx.drawImage(state.image, -state.imageWidth / 2, -state.imageHeight / 2, state.imageWidth, state.imageHeight);
  drawAnnotations(ctx, state);
  ctx.restore();
  updateZoomLabel(els, state);
}

function applyImageTransform(ctx, state) {
  ctx.translate(state.canvasWidth / 2 + state.panX, state.canvasHeight / 2 + state.panY);
  ctx.rotate((state.rotation * Math.PI) / 180);
  ctx.scale(state.scale, state.scale);
}

function drawAnnotations(ctx, state) {
  if (!state.strokes.length) return;
  ctx.save();
  ctx.beginPath();
  ctx.rect(-state.imageWidth / 2, -state.imageHeight / 2, state.imageWidth, state.imageHeight);
  ctx.clip();
  state.strokes.forEach(function (stroke) {
    if (!stroke || !stroke.points || !stroke.points.length) return;
    ctx.beginPath();
    stroke.points.forEach(function (point, index) {
      const x = point.x - state.imageWidth / 2;
      const y = point.y - state.imageHeight / 2;
      if (index === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    ctx.lineWidth = Math.max(1, Number(stroke.width || 5)) / Math.max(0.01, state.scale);
    ctx.strokeStyle = stroke.color || "#ff4d4f";
    ctx.stroke();
  });
  ctx.restore();
}

function pointerImagePoint(event, els, state) {
  const rect = els.stage.getBoundingClientRect();
  const x = event.clientX - rect.left;
  const y = event.clientY - rect.top;
  const point = viewportToImage(x, y, state);
  return {
    ...point,
    inside: point.x >= 0 && point.x <= state.imageWidth && point.y >= 0 && point.y <= state.imageHeight,
  };
}

function viewportToImage(x, y, state) {
  try {
    const matrix = imageMatrix(state).inverse();
    const point = new DOMPoint(x, y).matrixTransform(matrix);
    return {
      x: point.x + state.imageWidth / 2,
      y: point.y + state.imageHeight / 2,
    };
  } catch (_) {
    return { x: 0, y: 0 };
  }
}

function imageToViewport(x, y, state) {
  try {
    const point = new DOMPoint(x - state.imageWidth / 2, y - state.imageHeight / 2).matrixTransform(imageMatrix(state));
    return { x: point.x, y: point.y };
  } catch (_) {
    return canvasCenter(state);
  }
}

function imageMatrix(state) {
  return new DOMMatrix()
    .translate(state.canvasWidth / 2 + state.panX, state.canvasHeight / 2 + state.panY)
    .rotate(state.rotation)
    .scale(state.scale);
}

function canvasCenter(state) {
  return {
    x: state.canvasWidth / 2,
    y: state.canvasHeight / 2,
  };
}

function rotatedImageSize(state) {
  const quarter = Math.abs(normalizeRotation(state.rotation)) % 180 === 90;
  return {
    height: quarter ? state.imageWidth : state.imageHeight,
    width: quarter ? state.imageHeight : state.imageWidth,
  };
}

function normalizeRotation(value) {
  return ((Math.round(Number(value || 0) / 90) * 90) % 360 + 360) % 360;
}

function updateMode(root, els, state) {
  els.stage.classList.toggle("is-draw", state.mode === "draw");
  els.stage.classList.toggle("is-move", state.mode !== "draw");
  root.querySelectorAll("[data-preview-action]").forEach(function (button) {
    const action = button.dataset.previewAction || "";
    button.classList.toggle("is-active", action === state.mode || (action === "toggleFixed" && state.fixed));
  });
}

function updateToolbarState(root, state) {
  root.querySelectorAll("[data-preview-color]").forEach(function (button) {
    button.classList.toggle("is-active", normalizeColor(button.dataset.previewColor) === normalizeColor(state.color));
  });
  root.querySelectorAll("[data-preview-action]").forEach(function (button) {
    const action = button.dataset.previewAction || "";
    button.classList.toggle("is-active", action === state.mode || (action === "toggleFixed" && state.fixed));
  });
}

function updateZoomLabel(els, state) {
  if (!els.zoom) return;
  const zoom = Math.round((state.scale || 1) * 100);
  els.zoom.textContent = zoom + "%";
}

function toggleFixed(root, state) {
  state.fixed = !state.fixed;
  document.body.classList.toggle("is-fixed-window", state.fixed);
  callNativeWindow("__velo/window/set_always_on_top", { onTop: state.fixed }).catch(function () {});
  updateToolbarState(root, state);
}

function copyRenderedImage(els, state) {
  if (!state.loaded) return;
  if (!navigator.clipboard || typeof ClipboardItem === "undefined") {
    showToast(els, "当前环境不支持复制图片");
    return;
  }
  renderedBlob(state).then(
    function (blob) {
      return navigator.clipboard.write([new ClipboardItem({ [blob.type || "image/png"]: blob })]);
    },
    function (err) {
      throw err;
    },
  ).then(
    function () {
      showToast(els, "已复制当前图片");
    },
    function () {
      showToast(els, "复制失败");
    },
  );
}

function downloadRenderedImage(els, state) {
  if (!state.loaded) return;
  renderedBlob(state).then(
    function (blob) {
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = safeFilename(state.payload.title || "image-preview") + ".png";
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.setTimeout(function () {
        URL.revokeObjectURL(url);
      }, 1000);
      showToast(els, "已下载当前图片");
    },
    function () {
      window.open(state.payload.src, "_blank", "noopener");
      showToast(els, "已打开原图");
    },
  );
}

function renderedBlob(state) {
  return new Promise(function (resolve, reject) {
    const canvas = document.createElement("canvas");
    const ctx = canvas.getContext("2d");
    if (!ctx || !state.image) {
      reject(new Error("canvas unavailable"));
      return;
    }
    canvas.width = Math.max(1, state.imageWidth);
    canvas.height = Math.max(1, state.imageHeight);
    ctx.drawImage(state.image, 0, 0, state.imageWidth, state.imageHeight);
    drawExportAnnotations(ctx, state);
    try {
      canvas.toBlob(function (blob) {
        if (blob) resolve(blob);
        else reject(new Error("blob unavailable"));
      }, "image/png");
    } catch (err) {
      reject(err);
    }
  });
}

function drawExportAnnotations(ctx, state) {
  state.strokes.forEach(function (stroke) {
    if (!stroke || !stroke.points || !stroke.points.length) return;
    ctx.beginPath();
    stroke.points.forEach(function (point, index) {
      if (index === 0) ctx.moveTo(point.x, point.y);
      else ctx.lineTo(point.x, point.y);
    });
    ctx.lineCap = "round";
    ctx.lineJoin = "round";
    ctx.lineWidth = Math.max(1, Number(stroke.width || 5));
    ctx.strokeStyle = stroke.color || "#ff4d4f";
    ctx.stroke();
  });
}

function closeWindow() {
  callNativeWindow("__velo/window/close").catch(function () {
    window.close();
  });
}

function callNativeWindow(method, args) {
  if (typeof invoke !== "function") {
    return Promise.reject(new Error("go bridge not available"));
  }
  return invoke(method, { args: args || {} });
}

function showState(els, message) {
  els.state.textContent = message || "";
  els.state.hidden = false;
}

function hideState(els) {
  els.state.hidden = true;
}

function showToast(els, message) {
  if (!els.toast) return;
  els.toast.textContent = message || "";
  els.toast.classList.add("is-visible");
  if (els.toastTimer) window.clearTimeout(els.toastTimer);
  els.toastTimer = window.setTimeout(function () {
    els.toast.classList.remove("is-visible");
  }, 1800);
}

function unmountImagePreview(state) {
  if (state.resizeObserver) {
    state.resizeObserver.disconnect();
    state.resizeObserver = null;
  }
}

function injectPreviewCSS() {
  if (document.getElementById("image-preview-component-style")) return;
  const style = document.createElement("style");
  style.id = "image-preview-component-style";
  style.textContent = PREVIEW_CSS;
  document.head.appendChild(style);
}

function safeFilename(value) {
  const text = String(value || "image-preview")
    .trim()
    .replace(/[\\/:*?"<>|]+/g, "-")
    .replace(/\s+/g, " ");
  return text.slice(0, 80) || "image-preview";
}

function normalizeColor(value) {
  return String(value || "").trim().toLowerCase();
}

function clamp(value, min, max) {
  const number = Number(value);
  if (!Number.isFinite(number)) return min;
  return Math.min(max, Math.max(min, number));
}

function closestElement(target, selector) {
  let node = target;
  if (node && node.nodeType === 3) node = node.parentElement || node.parentNode;
  while (node && node !== document) {
    if (node.nodeType === 1 && typeof node.matches === "function" && node.matches(selector)) return node;
    node = node.parentElement || node.parentNode;
  }
  return null;
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

export {
  mountImagePreview,
  openImagePreviewFromElement,
  openImagePreviewPayload,
  previewPayloadFromElement,
};
