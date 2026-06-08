import { cleanMemoLine, compactText, memoLines } from "./memos.js";
import { parseAssetReference } from "./storage.js";

export function collectLinks(memos) {
  return collectMemoReferences(memos).filter((reference) => reference.type === "link");
}

export function collectResources(memos) {
  return collectMemoReferences(memos).filter((reference) => reference.type === "file" || reference.type === "image");
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
    lines.forEach((line, lineIndex) => {
      references.push(...collectLineReferences(memo, lines, line, lineIndex));
    });
  });
  return references;
}

export function collectLineReferences(memo, lines, line, lineIndex) {
  const references = [];
  const markdownRanges = [];
  const markdownLinkRegex = /(!?)\[([^\]]*)\]\(([^)]+)\)/g;
  let match;

  while ((match = markdownLinkRegex.exec(line))) {
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
  while ((match = rawURLRegex.exec(line))) {
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

export function referenceView(memo, lines, lineIndex, index, reference) {
  return {
    id: `${memo.id}:${lineIndex}:${index}:${reference.type}`,
    label: reference.label,
    lineIndex,
    memo,
    memoId: memo.id,
    sourceText: memoSourceTextForResource(lines, lineIndex),
    syntax: reference.syntax,
    type: reference.type,
    url: reference.url,
  };
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
