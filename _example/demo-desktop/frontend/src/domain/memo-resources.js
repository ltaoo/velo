import {
  cleanMemoLine,
  compactText,
  isMemoFenceClosingLine,
  maskMemoInlineCode,
  memoLines,
  parseMemoFenceLine,
} from "./memos.js";
import { parseAssetReference } from "./storage.js";

export function collectLinks(memos) {
  return collectMemoReferences(memos).filter((reference) => reference.type === "link");
}

export function collectResources(memos) {
  return collectMemoReferences(memos).filter((reference) => reference.type === "file" || reference.type === "image");
}

export function collectCodeBlocks(memos) {
  const blocks = [];
  memos.forEach((memo) => {
    const lines = memoLines(memo.content);
    let activeBlock = null;

    lines.forEach((line, lineIndex) => {
      const openingFence = parseMemoFenceLine(line);
      if (!openingFence) {
        if (activeBlock) activeBlock.lines.push(line);
        return;
      }

      if (activeBlock) {
        if (isMemoFenceClosingLine(line, activeBlock.openingFence)) {
          blocks.push(codeBlockView(memo, lines, activeBlock, lineIndex));
          activeBlock = null;
        } else {
          activeBlock.lines.push(line);
        }
        return;
      }

      const blockFence = codeBlockFence(line);
      activeBlock = {
        fenceLine: line,
        language: blockFence.language,
        lines: [],
        marker: blockFence.marker || codeBlockMarkerFromPreviousLines(lines, lineIndex),
        openingFence,
        startLineIndex: lineIndex,
      };
    });

    if (activeBlock) {
      blocks.push(codeBlockView(memo, lines, activeBlock, lines.length - 1));
    }
  });
  return blocks;
}

export function getResourceStats(memos) {
  const resources = collectResources(memos);
  return {
    files: resources.filter((resource) => resource.type === "file").length,
    images: resources.filter((resource) => resource.type === "image").length,
    total: resources.length,
  };
}

export function collectMemoReferences(memos) {
  const references = [];
  memos.forEach((memo) => {
    const lines = memoLines(memo.content);
    let activeFence = null;
    lines.forEach((line, lineIndex) => {
      const fence = parseMemoFenceLine(line);
      if (activeFence) {
        if (fence && isMemoFenceClosingLine(line, activeFence)) activeFence = null;
        return;
      }
      if (fence) {
        activeFence = fence;
        return;
      }
      references.push(...collectLineReferences(memo, lines, line, lineIndex));
    });
  });
  return references;
}

export function collectLineReferences(memo, lines, line, lineIndex) {
  const searchableLine = maskMemoInlineCode(line);
  const references = [];
  const markdownRanges = [];
  const markdownLinkRegex = /(!?)\[([^\]]*)\]\(([^)]+)\)/g;
  let match;

  while ((match = markdownLinkRegex.exec(searchableLine))) {
    markdownRanges.push([match.index, match.index + match[0].length]);

    const url = markdownReferenceURL(match[3]);
    if (!url) continue;

    const label = (match[2] || "").trim();
    const type = markdownReferenceType(match[1], label, url);
    if (!type) continue;

    references.push(
      referenceView(memo, lines, lineIndex, references.length, {
        label: referenceLabel(type, label, url),
        syntax: match[1] === "!" ? "image" : "markdown",
        type,
        url,
      }),
    );
  }

  const rawURLRegex = /\bhttps?:\/\/[^\s<>"`]+/gi;
  while ((match = rawURLRegex.exec(searchableLine))) {
    if (rangeIncludes(markdownRanges, match.index)) continue;

    const url = cleanRawURL(match[0]);
    if (!url) continue;

    const type = rawURLReferenceType(url);
    references.push(
      referenceView(memo, lines, lineIndex, references.length, {
        label: referenceLabel(type, "", url),
        syntax: "raw",
        type,
        url,
      }),
    );
  }

  return references;
}

export function codeBlockView(memo, lines, block, endLineIndex) {
  const contentMarker = codeBlockMarkerFromFirstLine(block.lines);
  const marker = block.marker || contentMarker;
  const codeLines = contentMarker && !block.marker ? block.lines.slice(1) : block.lines;
  const code = codeLines.join("\n");
  const language = block.language || "";
  const title = marker && marker.title ? marker.title : "";
  const aliases = marker ? marker.aliases : [];
  const label = title || (language ? `${language} 代码片段` : "代码片段");
  return {
    aliases,
    code,
    endLineIndex,
    id: `${sourceId(memo)}:${block.startLineIndex}:${endLineIndex}:code`,
    label,
    language,
    lineIndex: block.startLineIndex,
    marked: Boolean(marker),
    memo,
    memoId: sourceMemoId(memo),
    sourceCommentId: sourceCommentId(memo),
    sourceId: sourceId(memo),
    sourceMemoId: sourceMemoId(memo),
    sourceType: sourceType(memo),
    sourceText: sourceTextFromLines(lines, block.startLineIndex, "仅包含代码块的 memo"),
    syntax: "fenced",
    title,
    type: "code",
  };
}

export function codeBlockFence(line) {
  const fence = parseMemoFenceLine(line);
  const info = fence ? fence.info : "";
  return {
    language: codeBlockLanguageFromInfo(info),
    marker: codeBlockMarkerFromText(info),
  };
}

export function codeBlockLanguage(line) {
  return codeBlockFence(line).language;
}

export function codeBlockLanguageFromInfo(info) {
  const tokens = String(info || "").replace(/[{}]/g, " ").trim().split(/\s+/).filter(Boolean);
  if (!tokens.length || isSnippetMarkerToken(tokens[0])) return "";
  if (/^(?:title|alias|aliases|aka|as|别名|缩写)=/i.test(tokens[0])) return "";
  return tokens[0].replace(/[,:：;，；|].*$/, "").trim();
}

export function codeBlockMarkerFromPreviousLines(lines, lineIndex) {
  for (let index = lineIndex - 1; index >= 0; index -= 1) {
    const line = String(lines[index] || "");
    if (!line.trim()) continue;
    return codeBlockMarkerFromText(line);
  }
  return null;
}

export function codeBlockMarkerFromFirstLine(lines) {
  if (!Array.isArray(lines) || !lines.length) return null;
  return codeBlockMarkerFromText(lines[0], { allowCommentPrefix: true });
}

export function codeBlockMarkerFromText(value, options = {}) {
  const text = normalizeSnippetMarkerText(value, options);
  if (!text) return null;

  const match = text.match(/(?:^|\s)(#?snippet|snip|code[-\s]?snippet|代码片段|片段)(?:\s*[:：-]\s*|\s+|$)(.*)$/i);
  if (!match) return null;
  return parseSnippetMarkerMeta(match[2] || "");
}

export function normalizeSnippetMarkerText(value, options = {}) {
  let text = String(value || "").trim();
  if (!text) return "";
  text = text.replace(/^<!--\s*/, "").replace(/\s*-->$/, "");
  text = text.replace(/^\/\*\s*/, "").replace(/\s*\*\/$/, "");
  text = text.replace(/[{}]/g, " ");
  text = text.replace(/^>\s*/, "").replace(/^\s*[-*+]\s+/, "");
  text = text.replace(/^#{1,6}\s+/, "");
  if (options.allowCommentPrefix) {
    text = text.replace(/^(?:\/\/|#|--|;)\s*/, "");
  }
  return text.trim();
}

export function parseSnippetMarkerMeta(value) {
  const raw = String(value || "").trim();
  const parts = raw
    .split(/\s*(?:\||；|;|，|,)\s*/)
    .map((item) => item.replace(/^(?:alias|aliases|aka|as|别名|缩写)\s*[:：=]\s*/i, "").trim())
    .filter(Boolean);
  const title = compactText(parts[0] || "", 120);
  const aliases = uniqueStrings(parts.slice(1).flatMap(splitAliasText));
  return {
    aliases,
    title,
  };
}

export function splitAliasText(value) {
  const text = String(value || "").trim();
  if (!text) return [];
  if (/\s/.test(text) && !/^[\w.-]+$/i.test(text)) return [text];
  return text.split(/\s+/).filter(Boolean);
}

export function isSnippetMarkerToken(value) {
  return /^(?:#?snippet|snip|code[-_]?snippet|代码片段|片段)$/i.test(String(value || "").replace(/[{}:：,，;；|]/g, ""));
}

export function uniqueStrings(items) {
  const seen = new Set();
  return items.filter((item) => {
    const value = String(item || "").trim();
    const key = value.toLowerCase();
    if (!value || seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

export function referenceView(memo, lines, lineIndex, index, reference) {
  return {
    id: `${sourceId(memo)}:${lineIndex}:${index}:${reference.type}`,
    label: reference.label,
    lineIndex,
    memo,
    memoId: sourceMemoId(memo),
    sourceCommentId: sourceCommentId(memo),
    sourceId: sourceId(memo),
    sourceMemoId: sourceMemoId(memo),
    sourceType: sourceType(memo),
    sourceText: memoSourceTextForResource(lines, lineIndex),
    syntax: reference.syntax,
    type: reference.type,
    url: reference.url,
  };
}

function sourceId(memo) {
  return String((memo && memo.sourceId) || (memo && memo.id) || "").trim();
}

function sourceMemoId(memo) {
  return String((memo && memo.sourceMemoId) || (memo && memo.id) || "").trim();
}

function sourceCommentId(memo) {
  return String((memo && memo.sourceCommentId) || "").trim();
}

function sourceType(memo) {
  return String((memo && memo.sourceType) || "memo").trim().toLowerCase() || "memo";
}

export function memoSourceTextForResource(lines, lineIndex) {
  return sourceTextFromLines(lines, lineIndex, "仅包含资源的 memo");
}

export function sourceTextFromLines(lines, lineIndex, fallbackText) {
  const before = lines
    .slice(0, lineIndex)
    .reverse()
    .find((line) => cleanMemoLine(line));
  const fallback = lines.find((line) => cleanMemoLine(line));
  return compactText(cleanMemoLine(before || fallback || "") || fallbackText, 84);
}

export function markdownReferenceURL(value) {
  return String(value || "").trim();
}

export function markdownReferenceType(marker, label, url) {
  if (marker === "!" || isImageAttachment(label, url)) return "image";
  if (isFileAttachment(label, url)) return "file";
  if (isHyperlinkURL(url)) return "link";
  return "";
}

export function rawURLReferenceType(url) {
  if (isImageAttachment("", url)) return "image";
  if (isFileAttachment("", url)) return "file";
  return "link";
}

export function referenceLabel(type, label, url) {
  const text = cleanMemoLine(label || "");
  if (text) return compactText(text, 120);
  if (type === "file" || type === "image") return fileDisplayName("", url);
  return linkDisplayName(url);
}

export function linkDisplayName(url) {
  try {
    const parsed = new URL(url, globalThis.location && globalThis.location.origin);
    return parsed.pathname && parsed.pathname !== "/" ? `${parsed.host}${parsed.pathname}` : parsed.host || url;
  } catch (_) {
    return url;
  }
}

export function cleanRawURL(value) {
  let url = String(value || "").trim();
  while (/[),.;:!?，。；：！？]$/.test(url)) {
    url = url.slice(0, -1);
  }
  return url;
}

export function rangeIncludes(ranges, index) {
  return ranges.some(([start, end]) => index >= start && index < end);
}

export function isHyperlinkURL(url) {
  return /^(https?:|mailto:)/i.test(String(url || "")) || /^\/(?!\/)/.test(String(url || ""));
}

export function sortMemoReference(a, b, sortDesc = true) {
  const created = new Date(a.memo.createdAt).getTime() - new Date(b.memo.createdAt).getTime();
  if (created !== 0) return sortDesc ? -created : created;
  if (a.lineIndex !== b.lineIndex) return a.lineIndex - b.lineIndex;
  return a.id.localeCompare(b.id);
}

export function isFileAttachment(label, url) {
  const pattern = /\.(?:7z|aac|apk|avi|csv|dmg|docx?|flac|gz|heic|ics|json|key|log|m4a|mkv|mov|mp3|mp4|numbers|pages|pdf|pptx?|rar|rtf|tar|txt|wav|webm|xlsx?|xml|yaml|yml|zip)(?:[?#].*)?$/i;
  if (parseAssetReference(url)) return true;
  if (/^(local:\/\/|blob:|data:)/i.test(String(url || ""))) return true;
  return pattern.test(String(label || "")) || pattern.test(String(url || ""));
}

export function isImageAttachment(label, url) {
  const pattern = /\.(?:avif|bmp|gif|jpe?g|png|svg|webp)(?:[?#].*)?$/i;
  const asset = parseAssetReference(url);
  if (/^data:image\//i.test(String(url || ""))) return true;
  return pattern.test(String(label || "")) || pattern.test(String(url || "")) || pattern.test(asset ? asset.key : "");
}

export function fileDisplayName(label, url) {
  const asset = parseAssetReference(url);
  const raw = String(label || "").trim() || (asset ? asset.key : String(url || "").trim());
  const clean = raw.split(/[?#]/)[0].replace(/\/+$/, "");
  const last = clean.split("/").pop() || raw;
  try {
    return decodeURIComponent(last) || raw;
  } catch (_) {
    return last || raw;
  }
}
