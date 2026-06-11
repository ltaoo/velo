import {
  isMemoFenceLine,
  memoLines,
  memoSelectorLabel,
  memoTitle,
  parseMemoHeadingLine,
  parseMemoReferenceInner,
  parseStandaloneMemoEmbed,
  parseTaskLine,
  resolveMemoReferenceTarget,
} from "../../domain/memos.js";
import { codeBlockFence, fileDisplayName, isFileAttachment, isImageAttachment } from "../../domain/memo-resources.js";
import { parseAssetReference } from "../../domain/storage.js";
import { formatRelativeDate } from "./memo-date.js";
import { cloudStorageById, loadEditorSettings, normalizeFileEditor, normalizeFileEditorRules, resolveAssetUrl } from "./memo-editor.js";
import { SVG } from "./memo-icons.js";
import { escapeAttr, escapeHTML } from "./memo-utils.js";

function renderMemoMarkdown(content, context = {}, lineNumberOffset = 0) {
  const lines = memoLines(content);
  const lineNumberClass = context.showLineNumbers === false ? " is-line-numbers-hidden" : "";
  return `<div class="memo-line-list${lineNumberClass}">${renderMemoMarkdownLines(lines, context, lineNumberOffset)}</div>`;
}

function renderMemoMarkdownLines(lines, context, lineNumberOffset) {
  let html = "";

  for (let index = 0; index < lines.length; index++) {
    const line = lines[index];
    const lineNumber = index + 1;

    if (isQuoteLine(line)) {
      const quoteStart = index;
      const quoteLines = [];
      while (index < lines.length && isQuoteLine(lines[index])) {
        quoteLines.push(stripQuoteMarker(lines[index]));
        index++;
      }
      index--;
      html += memoLineTemplate(
        quoteStart + 1 + lineNumberOffset,
        `<blockquote>${renderMemoMarkdown(quoteLines.join("\n"), context, quoteStart + lineNumberOffset)}</blockquote>`,
        "is-quote",
      );
      continue;
    }

    if (isMemoFenceLine(line)) {
      const startIndex = index;
      const codeLines = [];
      let endIndex = startIndex;
      index++;
      while (index < lines.length) {
        if (isMemoFenceLine(lines[index])) {
          endIndex = index;
          break;
        }
        codeLines.push(lines[index]);
        endIndex = index;
        index++;
      }
      if (index >= lines.length) index = lines.length - 1;
      html += memoLineTemplate(
        memoLineNumberRange(startIndex, endIndex, lineNumberOffset),
        renderMemoCodeBlock(line, codeLines, context, startIndex + lineNumberOffset, endIndex + lineNumberOffset),
        "is-code is-code-block",
      );
      continue;
    }

    if (!line.trim()) {
      html += memoLineTemplate(lineNumber + lineNumberOffset, '<div class="memo-markdown-gap"></div>', "is-empty");
      continue;
    }

    const memoEmbed = parseStandaloneMemoEmbed(line);
    if (memoEmbed) {
      html += memoLineTemplate(lineNumber + lineNumberOffset, renderMemoEmbedCard(memoEmbed, context), "is-embed");
      continue;
    }

    const standaloneResource = parseStandaloneMarkdownResource(line);
    if (standaloneResource) {
      html += memoLineTemplate(
        lineNumber + lineNumberOffset,
        standaloneResource.type === "image"
          ? renderMemoImageBlock(standaloneResource)
          : renderMemoFileBlock(standaloneResource, context),
        "is-resource",
      );
      continue;
    }

    const task = parseTaskLine(line);
    if (task) {
      const sourceLineIndex = index + lineNumberOffset;
      html += memoLineTemplate(
        lineNumber + lineNumberOffset,
        `
        <div class="memo-task-line">
          <input type="checkbox" ${context.readonly ? "disabled" : `data-task-line="${sourceLineIndex}"`} ${task.checked ? "checked" : ""} />
          <span>${inlineMarkdown(task.text, context)}</span>
        </div>
      `,
        "is-task",
      );
      continue;
    }

    const unorderedMatch = line.match(/^(\s*)[-*+]\s+(.*)$/);
    if (unorderedMatch) {
      html += memoLineTemplate(
        lineNumber + lineNumberOffset,
        `<div class="memo-line-list-item is-ul" style="--memo-list-indent: ${listIndentWidth(unorderedMatch[1])}px"><span class="memo-line-list-content">${inlineMarkdown(unorderedMatch[2], context)}</span></div>`,
        "is-list",
      );
      continue;
    }

    const orderedMatch = line.match(/^(\s*)(\d+\.)\s+(.*)$/);
    if (orderedMatch) {
      html += memoLineTemplate(
        lineNumber + lineNumberOffset,
        `<div class="memo-line-list-item is-ol" style="--memo-list-indent: ${listIndentWidth(orderedMatch[1])}px"><span class="memo-line-list-marker">${escapeHTML(orderedMatch[2])}</span><span class="memo-line-list-content">${inlineMarkdown(orderedMatch[3], context)}</span></div>`,
        "is-list",
      );
      continue;
    }

    const heading = parseMemoHeadingLine(line);
    if (heading) {
      html += memoLineTemplate(
        lineNumber + lineNumberOffset,
        `<h${heading.level} class="memo-heading memo-heading-${heading.level}">${inlineMarkdown(heading.text, context)}</h${heading.level}>`,
        `is-heading is-heading-${heading.level}`,
      );
      continue;
    }

    html += memoLineTemplate(lineNumber + lineNumberOffset, `<p>${inlineMarkdown(line, context)}</p>`);
  }

  return html;
}

function memoLineNumberRange(startIndex, endIndex, offset) {
  const start = startIndex + 1 + offset;
  const end = endIndex + 1 + offset;
  return start === end ? start : `${start}-${end}`;
}

function renderMemoCodeBlock(fenceLine, codeLines, context, startLineIndex, endLineIndex) {
  const sourceId = String((context && context.sourceId) || "").trim();
  const blockId = sourceId ? `${sourceId}:${startLineIndex}:${endLineIndex}:code` : "";
  const fence = codeBlockFence(fenceLine);
  const language = fence.language || "";
  const label = language || "代码";
  const code = codeLines.join("\n");
  return `
    <div class="memo-fenced-code-block" ${blockId ? `data-code-block-id="${escapeAttr(blockId)}"` : ""}>
      <div class="memo-fenced-code-toolbar">
        <span class="memo-fenced-code-label">${escapeHTML(label)}</span>
        <button class="memo-action-button memo-code-copy-button" type="button" data-action="copyCodeBlock" title="复制代码" aria-label="复制代码">
          ${SVG.copy}
        </button>
      </div>
      <pre class="memo-fenced-code-body"><code data-code-block-code>${escapeHTML(code)}</code></pre>
    </div>
  `;
}

function isQuoteLine(line) {
  return /^>\s?/.test(String(line || ""));
}

function stripQuoteMarker(line) {
  const match = String(line || "").match(/^>\s?(.*)$/);
  return match ? match[1] : line;
}

function memoLineTemplate(lineNumber, body, className = "") {
  return `
    <div class="memo-source-line ${className}">
      <span class="memo-line-number" aria-hidden="true" data-line-number="${escapeAttr(lineNumber)}"></span>
      <div class="memo-line-body">${body}</div>
    </div>
  `;
}

function listIndentWidth(whitespace) {
  const level = Math.min(6, Math.floor(String(whitespace || "").replace(/\t/g, "  ").length / 2));
  return level * 18;
}

function inlineMarkdown(value, context = {}) {
  const text = String(value || "");
  const pattern = /(!?)\[\[([^\]\n]+)\]\]/g;
  let html = "";
  let lastIndex = 0;
  let match = pattern.exec(text);

  while (match) {
    html += inlineMarkdownBase(text.slice(lastIndex, match.index), context);
    const ref = parseMemoReferenceInner(match[2], match[1] === "!");
    html += ref ? renderMemoRefChip(ref, context) : inlineMarkdownBase(match[0], context);
    lastIndex = match.index + match[0].length;
    match = pattern.exec(text);
  }

  html += inlineMarkdownBase(text.slice(lastIndex), context);
  return html;
}

function inlineMarkdownBase(value, context = {}) {
  let html = escapeHTML(value);
  html = html.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (_, alt, src) => {
    return renderMemoImageToken({ label: alt, url: src });
  });
  html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_, label, url) => {
    if (isFileAttachment(label, url)) {
      return renderMemoFileToken({ label, url }, context);
    }
    const href = safeUrl(url);
    return `<a href="${escapeAttr(href)}" target="_blank" rel="noreferrer">${escapeHTML(label)}</a>`;
  });
  html = html.replace(/`([^`]+)`/g, "<code>$1</code>");
  html = html.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  html = html.replace(/(^|[^*])\*([^*]+)\*/g, "$1<em>$2</em>");
  html = html.replace(/(^|\s)#([\w\u4e00-\u9fa5-]+)/g, '$1<span class="memo-hashtag">#$2</span>');
  html = replaceMemoTimeSyntax(html);
  return html;
}

function replaceMemoTimeSyntax(html) {
  return String(html || "")
    .split(/(<code\b[^>]*>.*?<\/code>|<[^>]+>)/gi)
    .map(function (part) {
      if (!part || part.charAt(0) === "<") return part;
      return part.replace(
        /(^|[\s([{（【「『])(::)((?:\d{4}(?:-\d{1,2}(?:-\d{1,2}(?:[ T]\d{1,2}:\d{2}(?::\d{2})?)?)?)?)|(?:\d{1,2}:\d{2}(?::\d{2})?)|(?:[^\s<>()\[\]{}，。！？、；;,.]{1,32}))/g,
        function (_, prefix, trigger, value) {
          return prefix + renderMemoTimeToken(trigger, value);
        },
      );
    })
    .join("");
}

function renderMemoTimeToken(trigger, value) {
  const label = String(value || "");
  return `
    <span class="memo-time-token" title="${escapeAttr(trigger + label)}" aria-label="时间 ${escapeAttr(label)}">
      ${SVG.clock}
      <span>${label}</span>
    </span>
  `;
}

function renderMemoRefChip(ref, context) {
  const taskRef = parseTaskReferenceTarget(ref);
  if (taskRef) {
    return renderTaskRefChip(taskRef, ref);
  }

  const target = resolveMemoReferenceTarget(ref, context);
  const label = target ? memoRefTitle(ref, target) : ref.alias || ref.target;
  const range = target && ref.selector ? memoSelectorLabel(ref.selector) : "";

  if (!target) {
    return `<span class="memo-ref-chip is-missing" title="找不到 ${escapeAttr(ref.target)}">[[${escapeHTML(label)}]]</span>`;
  }

  return `
    <button class="memo-ref-chip ${ref.embed ? "is-embed" : ""}" type="button" data-memo-ref-target="${escapeAttr(target.id)}" title="打开 memo">
      <span>${escapeHTML(label)}</span>
      ${range ? `<small>${escapeHTML(range)}</small>` : ""}
    </button>
  `;
}

function renderMemoEmbedCard(ref, context) {
  const taskRef = parseTaskReferenceTarget(ref);
  if (taskRef) {
    return renderMemoRefStateCard("is-task", taskRef.label, "任务引用");
  }

  const target = resolveMemoReferenceTarget(ref, context);
  if (!target) {
    return renderMemoRefStateCard("is-missing", "引用不可用", "找不到 " + ref.target);
  }

  const stack = Array.isArray(context.stack) ? context.stack : [];
  if (stack.includes(target.id)) {
    return renderMemoRefStateCard("is-cycle", memoRefTitle(ref, target), "循环引用已停止");
  }

  const depth = Number(context.depth || 0);
  const maxDepth = Number(context.maxDepth || 2);
  if (depth >= maxDepth) {
    return renderMemoRefStateCard("is-collapsed", memoRefTitle(ref, target), "嵌套引用已折叠");
  }

  const excerpt = memoReferenceExcerpt(target.content, ref.selector);
  if (excerpt.error) {
    return renderMemoRefStateCard("is-missing", memoRefTitle(ref, target), excerpt.error);
  }

  const childContext = {
    ...context,
    depth: depth + 1,
    readonly: true,
    sourceId: target.id,
    stack: stack.concat(target.id),
  };
  const updatedAt = target.updatedAt || target.createdAt;
  const meta = [
    excerpt.label ? `<span>${escapeHTML(excerpt.label)}</span>` : "",
    updatedAt ? `<time datetime="${escapeAttr(updatedAt)}">${formatRelativeDate(updatedAt)}</time>` : "",
  ].filter(Boolean).join("");

  return `
    <aside class="memo-ref-card" data-memo-ref-card="${escapeAttr(target.id)}">
      ${meta ? `<div class="memo-ref-meta-line">${meta}</div>` : ""}
      <div class="memo-ref-body memo-content">${renderMemoMarkdown(excerpt.content, childContext)}</div>
    </aside>
  `;
}

function parseTaskReferenceTarget(ref) {
  const raw = String((ref && ref.target) || "").trim();
  if (!raw.toLowerCase().startsWith("task:")) return null;
  const id = raw.slice(5).trim();
  if (!id) return null;
  return {
    id,
    label: ref.alias || id,
  };
}

function renderTaskRefChip(taskRef, ref) {
  return `
    <span class="memo-ref-chip memo-task-ref-chip ${ref.embed ? "is-embed" : ""}" data-task-ref-target="${escapeAttr(taskRef.id)}" title="任务 ${escapeAttr(taskRef.id)}">
      <span>${escapeHTML(taskRef.label)}</span>
    </span>
  `;
}

function renderMemoRefStateCard(className, title, message) {
  return `
    <aside class="memo-ref-card ${className}">
      <header class="memo-ref-head">
        <span class="memo-ref-title is-static">${escapeHTML(title || "引用")}</span>
      </header>
      <div class="memo-ref-state">${escapeHTML(message || "")}</div>
    </aside>
  `;
}

function memoReferenceExcerpt(content, selector) {
  const text = String(content || "").replace(/\r\n/g, "\n");
  if (!selector) {
    return {
      content: text,
      label: "",
    };
  }

  if (selector.type === "invalid") {
    return { error: "行范围无效" };
  }
  if (selector.type !== "line") {
    return { error: "暂不支持选择器 #" + selector.raw };
  }

  const lines = memoLines(text);
  if (selector.start > lines.length) {
    return { error: "目标 memo 没有第 " + selector.start + " 行" };
  }

  const actualEnd = Math.min(selector.end, lines.length);
  const selected = lines.slice(selector.start - 1, actualEnd);
  const wrapped = wrapPartialCodeFence(lines, selector.start - 1, actualEnd, selected);
  return {
    content: wrapped.join("\n"),
    label: selector.start === actualEnd ? "L" + selector.start : "L" + selector.start + "-L" + actualEnd,
  };
}

function wrapPartialCodeFence(lines, startIndex, endIndex, selected) {
  const output = selected.slice();
  let inCode = false;

  for (let i = 0; i < startIndex; i += 1) {
    if (isMemoFenceLine(lines[i])) inCode = !inCode;
  }

  const startedInsideCode = inCode;
  for (let i = startIndex; i < endIndex; i += 1) {
    if (isMemoFenceLine(lines[i])) inCode = !inCode;
  }

  if (startedInsideCode) output.unshift("```");
  if (inCode) output.push("```");
  return output;
}

function memoRefTitle(ref, memo) {
  return String((ref && ref.alias) || "").trim() || memoTitle(memo);
}

function parseStandaloneMarkdownResource(line) {
  const match = String(line || "").match(/^\s*(!?)\[([^\]]*)\]\(([^)]+)\)\s*$/);
  if (!match) return null;

  const url = match[3].trim();
  if (!url) return null;

  const label = (match[2] || "").trim() || fileDisplayName("", url);
  const type = match[1] === "!" || isImageAttachment(label, url) ? "image" : "file";

  return { type, label, url };
}

function renderMemoImageBlock(resource) {
  const src = safeImageUrl(resource.url);
  if (!src) return `<p>${renderMemoImageToken(resource)}</p>`;

  const label = resource.label || fileDisplayName("", resource.url);
  const previewAttrs = imagePreviewAttrs(src, label, resource.url, label);
  return `
    <figure class="memo-image-block" ${previewAttrs}>
      <img src="${escapeAttr(src)}" alt="${escapeAttr(label)}" loading="lazy" ${previewAttrs} />
      ${label ? `<figcaption>${escapeHTML(label)}</figcaption>` : ""}
    </figure>
  `;
}

function renderMemoFileBlock(resource, context = {}) {
  const href = safeUrl(resource.url);
  const name = fileDisplayName(resource.label, resource.url);
  const displayUrl = href !== "#" ? href : resource.url;
  const openButton = renderVSCodeOpenButton(resource.url, context);
  const localClass = openButton ? " has-editor-open" : "";
  const body = `
    <span class="memo-file-block-icon">${SVG.paperclip}</span>
    <span class="memo-file-block-text">
      <span class="memo-file-block-name">${escapeHTML(name)}</span>
      <span class="memo-file-block-url">${escapeHTML(compactFileURL(displayUrl))}</span>
    </span>
  `;

  if (href === "#" || openButton) {
    return `<div class="memo-file-block${localClass}">${body}${openButton}</div>`;
  }
  return `<a class="memo-file-block" href="${escapeAttr(href)}" target="_blank" rel="noreferrer">${body}</a>`;
}

function renderMemoImageToken(resource) {
  const src = safeImageUrl(resource.url);
  const label = resource.label || fileDisplayName("", resource.url);
  const previewAttrs = src ? imagePreviewAttrs(src, label, resource.url, label) : "";
  return `
    <span class="memo-image-token" ${previewAttrs} ${src ? 'role="button" tabindex="0"' : ""}>
      ${SVG.image}
      <span>${escapeHTML(label || resource.url)}</span>
      ${src ? `<span class="memo-image-token-preview"><img src="${escapeAttr(src)}" alt="${escapeAttr(label)}" loading="lazy" /></span>` : ""}
    </span>
  `;
}

function imagePreviewAttrs(src, title, source, caption) {
  return [
    `data-image-preview-src="${escapeAttr(src)}"`,
    `data-image-preview-title="${escapeAttr(title || "图片预览")}"`,
    `data-image-preview-source="${escapeAttr(source || src)}"`,
    caption ? `data-image-preview-caption="${escapeAttr(caption)}"` : "",
  ]
    .filter(Boolean)
    .join(" ");
}

function renderMemoFileToken(resource, context = {}) {
  const href = safeUrl(resource.url);
  const name = fileDisplayName(resource.label, resource.url);
  const openButton = renderVSCodeOpenButton(resource.url, context);
  const localClass = openButton ? " has-editor-open" : "";
  const body = `${SVG.paperclip}<span>${escapeHTML(name)}</span>`;

  if (href === "#" || openButton) {
    return `<span class="memo-file-token${localClass}">${body}${openButton}</span>`;
  }
  return `<a class="memo-file-token" href="${escapeAttr(href)}" target="_blank" rel="noreferrer">${body}</a>`;
}

function renderVSCodeOpenButton(url, context = {}) {
  const target = localEditorTarget(url);
  if (!target) return "";
  const editor = selectedFileEditor(context, target.file);
  if (editor.id === "none") return "";
  const label = editor.name || "编辑器";
  const actionLabel = "在 " + label + " 中打开";
  return `
    <button
      class="memo-file-open-editor memo-file-open-vscode"
      type="button"
      data-editor-open="editor"
      data-editor-file="${escapeAttr(target.file)}"
      data-editor-line="${escapeAttr(target.line)}"
      data-editor-col="${escapeAttr(target.col)}"
      data-editor-app-id="${escapeAttr(editor.id)}"
      data-editor-app-name="${escapeAttr(label)}"
      data-editor-app-path="${escapeAttr(editor.path)}"
      data-editor-label="${escapeAttr(label)}"
      title="${escapeAttr(actionLabel)}"
      aria-label="${escapeAttr(actionLabel)}"
    >
      ${SVG.code}
      <span>${escapeHTML(actionLabel)}</span>
    </button>
  `;
}

function selectedFileEditor(context = {}, file = "") {
  const settings = context && context.editorSettings ? context.editorSettings : loadEditorSettings();
  const matched = fileEditorRuleForFile(file, settings);
  if (matched) return normalizeFileEditor(matched.editor);
  const fromContext = settings ? settings.fileEditor : null;
  if (fromContext) return normalizeFileEditor(fromContext);
  try {
    return normalizeFileEditor(loadEditorSettings().fileEditor);
  } catch (_) {
    return normalizeFileEditor(null);
  }
}

function fileEditorRuleForFile(file, settings) {
  const extension = fileExtension(file);
  if (!extension) return null;
  return normalizeFileEditorRules(settings && settings.fileEditorRules).find((rule) => rule.extension === extension) || null;
}

function fileExtension(file) {
  const path = String(file || "").split(/[?#]/)[0];
  const name = path.split(/[\\/]/).pop() || "";
  const index = name.lastIndexOf(".");
  if (index <= 0 || index === name.length - 1) return "";
  return "." + name.slice(index + 1).toLowerCase();
}

function localEditorTarget(value) {
  const target = parseEditorTarget(value);
  if (!target || !target.file) return null;
  if (!isLocalEditorResource(target.file)) return null;
  return target;
}

function parseEditorTarget(value) {
  let file = String(value || "").trim();
  if (!file) return null;

  let line = "1";
  let col = "1";
  try {
    const parsed = new URL(file, window.location.origin);
    line = editorPositionValue(parsed.searchParams.get("line"), line);
    col = editorPositionValue(parsed.searchParams.get("col") || parsed.searchParams.get("column"), col);
  } catch (_) {}

  if (!isLocalOSSAssetURL(file)) {
    const suffix = file.match(/^(.*):(\d+)(?::(\d+))?$/);
    if (suffix && !/^[a-zA-Z]:[\\/]/.test(file)) {
      file = suffix[1];
      line = editorPositionValue(suffix[2], line);
      col = editorPositionValue(suffix[3], col);
    }
  }

  return { file, line, col };
}

function editorPositionValue(value, fallback) {
  const text = String(value || "").trim();
  return /^[1-9]\d*$/.test(text) ? text : fallback || "1";
}

function isLocalEditorResource(value) {
  const url = String(value || "").trim();
  if (!url) return false;
  if (isLocalAssetReference(url)) return true;
  if (/^@assets\//i.test(url)) return false;
  if (isLocalOSSAssetURL(url)) return true;
  if (/^(local:\/\/|file:\/\/)/i.test(url)) return true;
  if (/^(https?:|mailto:|blob:|data:)/i.test(url)) return false;
  if (/^[a-z][a-z0-9+.-]*:/i.test(url) && !/^[a-zA-Z]:[\\/]/.test(url)) return false;
  return true;
}

function isLocalAssetReference(value) {
  const asset = typeof parseAssetReference === "function" ? parseAssetReference(value) : null;
  if (!asset) return false;
  const storage = typeof cloudStorageById === "function" ? cloudStorageById(asset.storageId) : null;
  return editorStorageIsLocal(storage);
}

function isLocalOSSAssetURL(value) {
  const raw = String(value || "").trim();
  try {
    const parsed = new URL(raw, window.location.origin);
    return parsed.pathname === "/api/oss/assets" && (parsed.origin === window.location.origin || raw.startsWith("/"));
  } catch (_) {
    return /^\/api\/oss\/assets(?:\?|$)/i.test(raw);
  }
}

function editorStorageIsLocal(storage) {
  const provider = String((storage && storage.provider) || "").trim().toLowerCase();
  return provider === "local" || provider === "local-oss";
}

function compactFileURL(url) {
  const value = String(url || "").trim();
  if (value.length <= 72) return value;
  return value.slice(0, 34) + "..." + value.slice(-28);
}

function safeImageUrl(value) {
  const url = resolveAssetUrl(value);
  if (/^\/(?!\/)/.test(url)) return url;
  if (/^(https?:|local:\/\/|blob:)/i.test(url)) return url;
  if (/^data:image\//i.test(url)) return url;
  return "";
}

function safeUrl(value) {
  const url = resolveAssetUrl(value);
  if (/^\/(?!\/)/.test(url)) return url;
  if (/^(https?:|mailto:|local:\/\/|blob:)/i.test(url)) return url;
  return "#";
}

export {
  compactFileURL,
  inlineMarkdown,
  renderMemoMarkdown,
  renderVSCodeOpenButton,
  safeImageUrl,
  safeUrl,
};
