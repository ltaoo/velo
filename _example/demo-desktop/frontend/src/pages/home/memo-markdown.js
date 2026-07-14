import {
  isMemoFenceClosingLine,
  isMemoFenceLine,
  memoLines,
  memoSelectorLabel,
  memoTitle,
  parseMemoFenceLine,
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

function collectMemoHeadings(content) {
  const lines = memoLines(content);
  const headings = [];

  for (let index = 0; index < lines.length; index++) {
    const line = lines[index];

    if (isMemoFenceLine(line)) {
      const openingFence = parseMemoFenceLine(line);
      index++;
      while (index < lines.length && !isMemoFenceClosingLine(lines[index], openingFence)) {
        index++;
      }
      continue;
    }

    const imageLayout = parseMemoImageLayoutStartLine(line);
    if (imageLayout) {
      index++;
      while (index < lines.length && !isMemoImageLayoutEndLine(lines[index])) {
        index++;
      }
      continue;
    }

    const heading = parseMemoHeadingLine(line);
    if (!heading || !String(heading.text || "").trim()) continue;

    headings.push({
      level: heading.level,
      lineNumber: index + 1,
      text: heading.text,
    });
  }

  return headings;
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
      const openingFence = parseMemoFenceLine(line);
      const codeLines = [];
      let endIndex = startIndex;
      index++;
      while (index < lines.length) {
        if (isMemoFenceClosingLine(lines[index], openingFence)) {
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

    const imageLayout = parseMemoImageLayoutStartLine(line);
    if (imageLayout) {
      const startIndex = index;
      const layoutLines = [];
      let endIndex = startIndex;
      index++;
      while (index < lines.length) {
        if (isMemoImageLayoutEndLine(lines[index])) {
          endIndex = index;
          break;
        }
        layoutLines.push(lines[index]);
        endIndex = index;
        index++;
      }
      if (index >= lines.length) index = lines.length - 1;
      html += memoLineTemplate(
        memoLineNumberRange(startIndex, endIndex, lineNumberOffset),
        renderMemoImageLayoutBlock(imageLayout, parseMemoImageLayoutItems(layoutLines)),
        "is-image-layout",
      );
      continue;
    }

    if (!line.trim()) {
      html += memoLineTemplate(lineNumber + lineNumberOffset, '<div class="memo-markdown-gap"></div>', "is-empty");
      continue;
    }

    if (isMemoHorizontalRuleLine(line)) {
      html += memoLineTemplate(lineNumber + lineNumberOffset, '<hr class="memo-horizontal-rule" />', "is-horizontal-rule");
      continue;
    }

    const table = parseMemoTableBlock(lines, index);
    if (table) {
      html += memoLineTemplate(
        memoLineNumberRange(index, table.endIndex, lineNumberOffset),
        renderMemoTableBlock(table, context),
        "is-table",
      );
      index = table.endIndex;
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
          : standaloneResource.type === "link"
            ? renderMemoLinkBlock(standaloneResource)
            : renderMemoFileBlock(standaloneResource, context),
        "is-resource",
      );
      continue;
    }

    const task = parseTaskLine(line);
    if (task) {
      const sourceLineIndex = index + lineNumberOffset;
      const taskSourceAttrs = memoTaskSourceAttrs(context, sourceLineIndex);
      const taskDetailAttrs = memoTaskDetailAttrs(context, sourceLineIndex);
      html += memoLineTemplate(
        lineNumber + lineNumberOffset,
        `
        <div class="memo-task-line">
          <input type="checkbox" ${context.readonly ? "disabled" : taskSourceAttrs} ${task.checked ? "checked" : ""} />
          <span ${taskDetailAttrs}>${inlineMarkdown(task.text, context)}</span>
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
      const headingLineNumber = lineNumber + lineNumberOffset;
      html += memoLineTemplate(
        headingLineNumber,
        `<h${heading.level} class="memo-heading memo-heading-${heading.level}">${inlineMarkdown(heading.text, context)}</h${heading.level}>`,
        `is-heading is-heading-${heading.level}`,
        `data-heading-line="${escapeAttr(headingLineNumber)}" data-heading-level="${escapeAttr(heading.level)}"`,
      );
      continue;
    }

    html += memoLineTemplate(lineNumber + lineNumberOffset, `<p>${inlineMarkdown(line, context)}</p>`);
  }

  return html;
}

function memoTaskSourceAttrs(context, lineIndex) {
  const sourceId = String((context && context.sourceId) || "").trim();
  const sourceMemoId = String((context && context.sourceMemoId) || sourceId).trim();
  const sourceCommentId = String((context && context.sourceCommentId) || "").trim();
  const sourceType = String((context && context.sourceType) || (sourceCommentId ? "comment" : "memo")).trim().toLowerCase() || "memo";
  return [
    `data-task-line="${escapeAttr(lineIndex)}"`,
    `data-task-source-type="${escapeAttr(sourceType)}"`,
    sourceId ? `data-task-source-id="${escapeAttr(sourceId)}"` : "",
    sourceMemoId ? `data-task-source-memo-id="${escapeAttr(sourceMemoId)}"` : "",
    sourceCommentId ? `data-task-source-comment-id="${escapeAttr(sourceCommentId)}"` : "",
  ].filter(Boolean).join(" ");
}

function memoTaskDetailAttrs(context, lineIndex) {
  const sourceId = String((context && context.sourceId) || "").trim();
  const sourceMemoId = String((context && context.sourceMemoId) || sourceId).trim();
  const sourceCommentId = String((context && context.sourceCommentId) || "").trim();
  const sourceType = String((context && context.sourceType) || (sourceCommentId ? "comment" : "memo")).trim().toLowerCase() || "memo";
  return [
    `data-task-detail="${escapeAttr(lineIndex)}"`,
    `data-task-detail-source-type="${escapeAttr(sourceType)}"`,
    sourceMemoId ? `data-task-detail-memo-id="${escapeAttr(sourceMemoId)}"` : "",
    sourceCommentId ? `data-task-detail-comment-id="${escapeAttr(sourceCommentId)}"` : "",
  ].filter(Boolean).join(" ");
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
  const collapsible = codeLines.length > 10;
  const collapseButton = collapsible
    ? `<button class="memo-action-button memo-code-collapse-button" type="button" data-action="toggleCodeCollapse" title="收起代码" aria-label="收起代码">${SVG.chevronDown}</button>`
    : "";
  const collapsibleClass = collapsible ? " memo-fenced-code-collapsible" : "";
  return `
    <div class="memo-fenced-code-block${collapsibleClass}" ${blockId ? `data-code-block-id="${escapeAttr(blockId)}"` : ""}>
      <div class="memo-fenced-code-toolbar">
        <span class="memo-fenced-code-label">${escapeHTML(label)}</span>
        <div class="memo-fenced-code-actions">
          ${collapseButton}
          <button class="memo-action-button memo-code-copy-button" type="button" data-action="copyCodeBlock" title="复制代码" aria-label="复制代码">
            ${SVG.copy}
          </button>
        </div>
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

function isMemoHorizontalRuleLine(line) {
  return /^\s*-{3,}\s*$/.test(String(line || ""));
}

function parseMemoImageLayoutStartLine(line) {
  const match = String(line || "").match(/^\s*:::\s*([^\s]+)(?:\s+(.*?))?\s*$/);
  if (!match) return null;

  const marker = String(match[1] || "").trim().toLowerCase();
  const info = String(match[2] || "").trim();
  if (/^(?:images?|image-grid|imagegrid|gallery|photos?|pics?|九宫格|九图|图片九宫格)$/.test(marker)) {
    return { info, type: "grid" };
  }
  if (!/^(?:image-layout|imagelayout|图片布局)$/.test(marker)) return null;

  return {
    info,
    type: memoImageLayoutTypeFromInfo(info),
  };
}

function isMemoImageLayoutEndLine(line) {
  return /^\s*:::\s*$/.test(String(line || ""));
}

function memoLineTemplate(lineNumber, body, className = "", attrs = "") {
  return `
    <div class="memo-source-line ${className}"${attrs ? ` ${attrs}` : ""}>
      <span class="memo-line-number" aria-hidden="true" data-line-number="${escapeAttr(lineNumber)}"></span>
      <div class="memo-line-body">${body}</div>
    </div>
  `;
}

function listIndentWidth(whitespace) {
  const level = Math.min(6, Math.floor(String(whitespace || "").replace(/\t/g, "  ").length / 2));
  return level * 18;
}

function parseMemoTableBlock(lines, startIndex) {
  const header = parseMemoTableRow(lines[startIndex]);
  if (!header) return null;

  const separator = parseMemoTableSeparator(lines[startIndex + 1]);
  if (!separator) return null;

  const columnCount = Math.max(header.cells.length, separator.alignments.length);
  if (columnCount < 1) return null;

  const rows = [];
  let index = startIndex + 2;
  while (index < lines.length) {
    const row = parseMemoTableRow(lines[index]);
    if (!row) break;
    rows.push(normalizeMemoTableCells(row.cells, columnCount));
    index++;
  }

  return {
    alignments: normalizeMemoTableCells(separator.alignments, columnCount),
    endIndex: index - 1,
    headers: normalizeMemoTableCells(header.cells, columnCount),
    rows,
  };
}

function parseMemoTableRow(line) {
  const text = String(line || "").trim();
  if (!text || !hasUnescapedTablePipe(text)) return null;

  let body = text;
  if (body.charAt(0) === "|") body = body.slice(1);
  if (body.endsWith("|") && !body.endsWith("\\|")) body = body.slice(0, -1);

  const cells = splitMemoTableCells(body);
  if (!cells.length) return null;
  if (cells.length < 2 && text.charAt(0) !== "|" && !text.endsWith("|")) return null;
  return { cells };
}

function parseMemoTableSeparator(line) {
  const row = parseMemoTableRow(line);
  if (!row) return null;

  const alignments = [];
  for (const cell of row.cells) {
    const marker = String(cell || "").trim().replace(/\s+/g, "");
    if (!/^:?-{2,}:?$/.test(marker)) return null;
    const left = marker.startsWith(":");
    const right = marker.endsWith(":");
    alignments.push(left && right ? "center" : right ? "right" : left ? "left" : "");
  }

  return { alignments };
}

function splitMemoTableCells(value) {
  const text = String(value || "");
  const cells = [];
  let cell = "";
  let escaped = false;
  let inlineCode = false;

  for (const char of text) {
    if (escaped) {
      cell += char;
      escaped = false;
      continue;
    }
    if (char === "\\") {
      cell += char;
      escaped = true;
      continue;
    }
    if (char === "`") {
      inlineCode = !inlineCode;
      cell += char;
      continue;
    }
    if (char === "|" && !inlineCode) {
      cells.push(unescapeMemoTableCell(cell.trim()));
      cell = "";
      continue;
    }
    cell += char;
  }

  cells.push(unescapeMemoTableCell(cell.trim()));
  return cells;
}

function hasUnescapedTablePipe(value) {
  let escaped = false;
  let inlineCode = false;
  for (const char of String(value || "")) {
    if (escaped) {
      escaped = false;
      continue;
    }
    if (char === "\\") {
      escaped = true;
      continue;
    }
    if (char === "`") {
      inlineCode = !inlineCode;
      continue;
    }
    if (char === "|" && !inlineCode) return true;
  }
  return false;
}

function unescapeMemoTableCell(value) {
  return String(value || "").replace(/\\\|/g, "|");
}

function normalizeMemoTableCells(cells, count) {
  const output = Array.isArray(cells) ? cells.slice(0, count) : [];
  while (output.length < count) output.push("");
  return output;
}

function renderMemoTableBlock(table, context) {
  const alignments = table.alignments || [];
  const headers = table.headers || [];
  const rows = table.rows || [];
  return `
    <div class="memo-table-scroll">
      <table class="memo-markdown-table">
        <thead>
          <tr>${headers.map((cell, index) => renderMemoTableCell("th", cell, alignments[index], context)).join("")}</tr>
        </thead>
        <tbody>
          ${rows.map((row) => `<tr>${row.map((cell, index) => renderMemoTableCell("td", cell, alignments[index], context)).join("")}</tr>`).join("")}
        </tbody>
      </table>
    </div>
  `;
}

function renderMemoTableCell(tag, value, alignment, context) {
  const alignClass = alignment ? ` is-align-${alignment}` : "";
  const content = inlineMarkdown(value, context) || "&nbsp;";
  return `<${tag} class="${alignClass}">${content}</${tag}>`;
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
    return renderMemoLinkToken(label, url);
  });
  html = html.replace(/`([^`]+)`/g, "<code>$1</code>");
  html = html.replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>");
  html = html.replace(/(^|[^*])\*([^*]+)\*/g, "$1<em>$2</em>");
  html = html.replace(/(^|\s)#([\w\u4e00-\u9fa5-]+)/g, '$1<span class="memo-hashtag">#$2</span>');
  html = replaceMemoTimeSyntax(html);
  return html;
}

function renderMemoLinkToken(label, url) {
  const href = safeUrl(url);
  const copyUrl = href !== "#" ? href : url;
  const title = label || copyUrl;
  return `
    <span class="memo-inline-link" data-inline-link-url="${escapeAttr(copyUrl)}">
      <a class="memo-inline-link-anchor" href="${escapeAttr(href)}" target="_blank" rel="noreferrer" title="${escapeAttr(title)}">
        <span class="memo-inline-link-icon">${SVG.link}</span>
        <span class="memo-inline-link-text">${escapeHTML(label || copyUrl)}</span>
      </a>
      <button class="memo-inline-link-copy" type="button" data-action="copyInlineLink" title="复制链接" aria-label="复制链接">
        ${SVG.copy}
      </button>
    </span>
  `;
}

function replaceMemoTimeSyntax(html) {
  return String(html || "")
    .split(/(<code\b[^>]*>.*?<\/code>|<[^>]+>)/gi)
    .map(function (part) {
      if (!part || part.charAt(0) === "<") return part;
      return part.replace(
        /(^|[\s([{（【「『])(::)(?!:)((?:\d{4}(?:-\d{1,2}(?:-\d{1,2}(?:[ T]\d{1,2}:\d{2}(?::\d{2})?)?)?)?)|(?:\d{1,2}:\d{2}(?::\d{2})?)|(?:[^\s<>()\[\]{}，。！？、；;,.]{1,32}))/g,
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
  let activeFence = null;

  for (let i = 0; i < startIndex; i += 1) {
    const fence = parseMemoFenceLine(lines[i]);
    if (activeFence) {
      if (fence && isMemoFenceClosingLine(lines[i], activeFence)) activeFence = null;
    } else if (fence) {
      activeFence = fence;
    }
  }

  const startingFence = activeFence;
  const openingMarker = startingFence ? safeMemoFenceForLines(output, startingFence) : "";
  for (let i = startIndex; i < endIndex; i += 1) {
    const fence = parseMemoFenceLine(lines[i]);
    if (activeFence) {
      if (fence && isMemoFenceClosingLine(lines[i], activeFence)) activeFence = null;
    } else if (fence) {
      activeFence = fence;
    }
  }

  const closingMarker = activeFence ? safeMemoFenceForLines(output, activeFence) : "";
  if (startingFence) output.unshift(openingMarker);
  if (activeFence) output.push(closingMarker);
  return output;
}

function safeMemoFenceForLines(lines, openingFence) {
  const marker = openingFence && openingFence.marker === "~" ? "~" : "`";
  const pattern = marker === "`" ? /`+/g : /~+/g;
  let length = Math.max(3, (openingFence && openingFence.length) || 0);
  lines.forEach(function (line) {
    const matches = String(line || "").match(pattern) || [];
    matches.forEach(function (match) {
      if (match.length >= length) length = match.length + 1;
    });
  });
  return marker.repeat(length);
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
  const type = match[1] === "!" || isImageAttachment(label, url)
    ? "image"
    : isFileAttachment(label, url)
      ? "file"
      : isLinkResourceURL(url)
        ? "link"
        : "file";

  return { type, label, url };
}

function isLinkResourceURL(value) {
  return /^(https?:|mailto:)/i.test(String(value || "").trim());
}

function memoImageLayoutTypeFromInfo(info) {
  const text = String(info || "").trim();
  if (!text) return "grid";

  const keyValue = text.match(/(?:^|\s)(?:layout|type|view)=([^\s]+)/i);
  if (keyValue) return normalizeMemoImageLayoutType(keyValue[1]);

  const firstToken = text.split(/\s+/)[0];
  return normalizeMemoImageLayoutType(firstToken);
}

function normalizeMemoImageLayoutType(value) {
  const type = String(value || "").trim().toLowerCase();
  if (/^(?:grid|nine-grid|ninegrid|weibo|mosaic|九宫格|九图)$/.test(type)) return "grid";
  if (/^(?:carousel|slider|slideshow|轮播|轮播图)$/.test(type)) return "carousel";
  return type || "grid";
}

function parseMemoImageLayoutItems(lines) {
  return (Array.isArray(lines) ? lines : [])
    .map(parseMemoImageLayoutItem)
    .filter(Boolean);
}

function parseMemoImageLayoutItem(line) {
  const text = String(line || "").trim().replace(/^[-*+]\s+/, "");
  if (!text) return null;

  const resource = parseStandaloneMarkdownResource(text);
  if (resource && resource.type === "image") return resource;

  if (!safeImageUrl(text)) return null;
  return {
    type: "image",
    label: fileDisplayName("", text),
    url: text,
  };
}

function renderMemoImageLayoutBlock(layout, items) {
  const type = normalizeMemoImageLayoutType(layout && layout.type);
  switch (type) {
    case "grid":
      return renderMemoImageGridLayout(type, items);
    default:
      return renderMemoImageFallbackLayout(type, items);
  }
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

function renderMemoImageGridLayout(layoutType, items) {
  const images = (Array.isArray(items) ? items : [])
    .map(function (item) {
      const src = safeImageUrl(item.url);
      if (!src) return null;
      const label = item.label || fileDisplayName("", item.url);
      return {
        label,
        source: item.url,
        src,
      };
    })
    .filter(Boolean);

  if (!images.length) {
    return `<div class="memo-image-layout-empty">没有可显示的图片</div>`;
  }

  const visible = images.slice(0, 9);
  const hiddenCount = images.length - visible.length;
  const columnCount = memoImageGridColumnCount(visible.length);
  return `
    <div class="memo-image-layout memo-image-layout-grid is-count-${visible.length}" style="--memo-image-layout-columns: ${columnCount}" data-image-layout="${escapeAttr(layoutType || "grid")}" data-image-layout-count="${images.length}">
      ${visible.map(function (image, index) {
        const label = image.label || "图片 " + (index + 1);
        const previewAttrs = imagePreviewAttrs(image.src, label, image.source, label);
        const overflow = hiddenCount > 0 && index === visible.length - 1
          ? `<span class="memo-image-layout-more">+${hiddenCount}</span>`
          : "";
        return `
          <button class="memo-image-layout-item" type="button" ${previewAttrs} aria-label="预览 ${escapeAttr(label)}">
            <img src="${escapeAttr(image.src)}" alt="${escapeAttr(label)}" loading="lazy" />
            ${overflow}
          </button>
        `;
      }).join("")}
    </div>
  `;
}

function renderMemoImageFallbackLayout(layoutType, items) {
  return renderMemoImageGridLayout(layoutType || "grid", items);
}

function memoImageGridColumnCount(count) {
  if (count <= 1) return 1;
  if (count === 2 || count === 4) return 2;
  return 3;
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

function renderMemoLinkBlock(resource) {
  const href = safeUrl(resource.url);
  const copyUrl = href !== "#" ? href : resource.url;
  const label = String(resource.label || copyUrl || "").trim();
  return `
    <div class="memo-link-block" data-inline-link-url="${escapeAttr(copyUrl)}">
      <a class="memo-link-block-target" href="${escapeAttr(href)}" target="_blank" rel="noreferrer" title="${escapeAttr(label || copyUrl)}">
        <span class="memo-link-block-icon">${SVG.link}</span>
        <span class="memo-link-block-text">
          <span class="memo-link-block-name">${escapeHTML(label || copyUrl)}</span>
          <span class="memo-link-block-url">${escapeHTML(compactFileURL(copyUrl))}</span>
        </span>
      </a>
      <button class="memo-action-button memo-link-block-copy" type="button" data-action="copyInlineLink" title="复制链接" aria-label="复制链接">
        ${SVG.copy}
      </button>
    </div>
  `;
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
  collectMemoHeadings,
  compactFileURL,
  inlineMarkdown,
  renderMemoMarkdown,
  renderVSCodeOpenButton,
  safeImageUrl,
  safeUrl,
};
