import { normalizeProjectID } from "./projects.js";

export const DEFAULT_VISIBILITY = "PRIVATE";

export const VISIBILITY = {
  PRIVATE: { label: "仅自己", icon: "lock" },
  PROTECTED: { label: "工作区", icon: "shield" },
  PUBLIC: { label: "公开", icon: "globe" },
};

const TASK_LINE_REGEX = /^(\s*[-*]\s+\[)([ xX])(\]\s+)(.*)$/;

export function normalizeMemoPayload(memo) {
  if (!memo || typeof memo !== "object") return null;
  const id = String(memo.id || "").trim();
  if (!id) return null;
  return {
    archived: Boolean(memo.archived),
    content: String(memo.content || ""),
    createdAt: memo.createdAt || new Date().toISOString(),
    id,
    kind: String(memo.kind || "").trim(),
    pinned: Boolean(memo.pinned),
    projectId: normalizeProjectID(memo.projectId),
    taskId: String(memo.taskId || "").trim(),
    updatedAt: memo.updatedAt || "",
    visibility: memo.visibility || DEFAULT_VISIBILITY,
  };
}

export function collectTags(memos) {
  const counts = new Map();
  memos.forEach((memo) => {
    extractTags(memo.content).forEach((tag) => {
      counts.set(tag, (counts.get(tag) || 0) + 1);
    });
  });
  return Array.from(counts.entries()).sort((a, b) => b[1] - a[1] || a[0].localeCompare(b[0]));
}

export function collectTodos(memos) {
  const todos = [];
  memos.forEach((memo) => {
    const lines = memoLines(memo.content);
    let inCode = false;
    lines.forEach((line, lineIndex) => {
      if (isMemoFenceLine(line)) {
        inCode = !inCode;
        return;
      }
      if (inCode) return;

      const task = parseTaskLine(line);
      if (!task) return;
      todos.push({
        checked: task.checked,
        id: `${memo.id}:${lineIndex}`,
        lineIndex,
        memo,
        memoId: memo.id,
        projectId: memo.projectId || "",
        sourceText: memoSourceText(lines, lineIndex),
        text: task.text,
      });
    });
  });
  return todos;
}

export function getTodoStats(memos) {
  const todos = collectTodos(memos);
  const done = todos.filter((todo) => todo.checked).length;
  return {
    done,
    open: todos.length - done,
    total: todos.length,
  };
}

export function parseTaskLine(line) {
  const match = String(line || "").match(TASK_LINE_REGEX);
  if (!match) return null;
  return {
    checked: match[2].toLowerCase() === "x",
    text: match[4].trim(),
  };
}

export function updateTaskLine(line, checked) {
  return String(line || "").replace(
    TASK_LINE_REGEX,
    function (_match, prefix, _marker, suffix, text) {
      return `${prefix}${checked ? "x" : " "}${suffix}${text}`;
    },
  );
}

export function memoSourceText(lines, lineIndex, fallbackText = "仅包含任务的 memo") {
  const before = lines
    .slice(0, lineIndex)
    .reverse()
    .find((line) => line.trim() && !parseTaskLine(line));
  const fallback = lines.find((line) => line.trim() && !parseTaskLine(line));
  return compactText(cleanMemoLine(before || fallback || "") || fallbackText, 84);
}

export function cleanMemoLine(line) {
  return String(line || "")
    .replace(/^#{1,6}\s+/, "")
    .replace(/^>\s?/, "")
    .replace(/^\s*[-*+]\s+/, "")
    .replace(/^\s*\d+\.\s+/, "")
    .replace(/!\[\[([^\]]+)\]\]/g, "$1")
    .replace(/\[\[([^\]]+)\]\]/g, "$1")
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/[`*_~]/g, "")
    .trim();
}

export function extractTags(text) {
  const tags = new Set();
  const matches = memoSearchableText(text).match(/(^|\s)#([\w\u4e00-\u9fa5-]+)/g) || [];
  matches.forEach((match) => tags.add(match.trim().slice(1)));
  return Array.from(tags);
}

export function buildMemoReferenceIndex(memos) {
  const list = Array.isArray(memos) ? memos : [];
  const incoming = new Map();
  const memoById = new Map();
  const memoByTitleKey = new Map();
  const outgoing = new Map();
  const unresolved = [];

  list.forEach(function (memo) {
    if (!memo || !memo.id) return;
    memoById.set(memo.id, memo);
  });

  list.forEach(function (memo) {
    if (!memo || !memo.id) return;
    const key = memoTitleKey(memoTitle(memo));
    if (key && !memoByTitleKey.has(key)) memoByTitleKey.set(key, memo.id);
  });

  list.forEach(function (memo) {
    if (!memo || !memo.id) return;
    const refs = parseMemoReferences(memo.content).map(function (ref) {
      const target = resolveMemoReferenceTarget(ref, { index: { memoById, memoByTitleKey } });
      const edge = {
        ...ref,
        sourceId: memo.id,
        targetId: target ? target.id : "",
      };
      if (edge.targetId) {
        if (!incoming.has(edge.targetId)) incoming.set(edge.targetId, []);
        incoming.get(edge.targetId).push(edge);
      } else {
        unresolved.push(edge);
      }
      return edge;
    });
    outgoing.set(memo.id, refs);
  });

  return {
    incoming,
    memoById,
    memoByTitleKey,
    outgoing,
    unresolved,
  };
}

export function parseMemoReferences(content) {
  const refs = [];
  const lines = memoLines(content);
  let inCode = false;

  lines.forEach(function (line, index) {
    if (isMemoFenceLine(line)) {
      inCode = !inCode;
      return;
    }
    if (inCode) return;

    const searchableLine = maskMemoInlineCode(line);
    const pattern = /(!?)\[\[([^\]\n]+)\]\]/g;
    let match = pattern.exec(searchableLine);
    while (match) {
      const ref = parseMemoReferenceInner(match[2], match[1] === "!");
      if (ref) {
        refs.push({
          ...ref,
          line: index + 1,
          raw: match[0],
        });
      }
      match = pattern.exec(searchableLine);
    }
  });

  return refs;
}

export function parseStandaloneMemoEmbed(line) {
  const match = String(line || "").match(/^\s*!\[\[([^\]\n]+)\]\]\s*$/);
  return match ? parseMemoReferenceInner(match[1], true) : null;
}

export function parseMemoReferenceInner(value, embed) {
  const raw = String(value || "").trim();
  if (!raw) return null;

  const aliasIndex = raw.indexOf("|");
  const targetExpr = (aliasIndex >= 0 ? raw.slice(0, aliasIndex) : raw).trim();
  const alias = aliasIndex >= 0 ? raw.slice(aliasIndex + 1).trim() : "";
  const selectorIndex = targetExpr.indexOf("#");
  const target = (selectorIndex >= 0 ? targetExpr.slice(0, selectorIndex) : targetExpr).trim();
  const selectorRaw = selectorIndex >= 0 ? targetExpr.slice(selectorIndex + 1).trim() : "";

  if (!target) return null;

  return {
    alias,
    embed: Boolean(embed),
    selector: parseMemoSelector(selectorRaw),
    target,
  };
}

export function parseMemoSelector(value) {
  const raw = String(value || "").trim();
  if (!raw) return null;

  const line = raw.match(/^L([1-9]\d*)(?:-(?:L)?([1-9]\d*))?$/i);
  if (line) {
    const start = Number(line[1]);
    const end = Number(line[2] || line[1]);
    return {
      end,
      raw,
      start,
      type: end >= start ? "line" : "invalid",
    };
  }

  return {
    raw,
    type: "unsupported",
  };
}

export function resolveMemoReferenceTarget(ref, context = {}) {
  const index = context.index || {};
  const memoById = index.memoById || new Map();
  const memoByTitleKey = index.memoByTitleKey || new Map();
  const raw = String((ref && ref.target) || "").trim();
  if (!raw) return null;

  const id = raw.toLowerCase().startsWith("memo:") ? raw.slice(5).trim() : raw;
  if (memoById.has(id)) return memoById.get(id);

  const titleId = memoByTitleKey.get(memoTitleKey(raw));
  return titleId ? memoById.get(titleId) || null : null;
}

export function memoLines(content) {
  return String(content || "").replace(/\r\n/g, "\n").split("\n");
}

export function isMemoFenceLine(line) {
  const trimmed = String(line || "").trim();
  return trimmed.startsWith("```") || trimmed.startsWith("~~~");
}

export function memoSearchableText(content) {
  const lines = memoLines(content);
  let inCode = false;
  return lines
    .map(function (line) {
      if (isMemoFenceLine(line)) {
        inCode = !inCode;
        return " ".repeat(line.length);
      }
      if (inCode) return " ".repeat(line.length);
      return maskMemoInlineCode(line);
    })
    .join("\n");
}

export function maskMemoInlineCode(value) {
  const chars = String(value || "").split("");
  let index = 0;
  while (index < chars.length) {
    if (chars[index] !== "`") {
      index += 1;
      continue;
    }
    const runStart = index;
    while (index < chars.length && chars[index] === "`") index += 1;
    const delimiter = chars.slice(runStart, index).join("");
    const closeIndex = chars.slice(index).join("").indexOf(delimiter);
    const end = closeIndex >= 0 ? index + closeIndex + delimiter.length : chars.length;
    for (let i = runStart; i < end; i += 1) chars[i] = " ";
    index = end;
  }
  return chars.join("");
}

export function parseMemoHeadingLine(line) {
  const match = String(line || "").match(/^\s{0,3}(#{1,6})(?:[ \t]+|$)(.*)$/);
  if (!match) return null;

  return {
    level: match[1].length,
    text: match[2].replace(/[ \t]+#{1,}\s*$/, "").trim(),
  };
}

export function memoReferenceAlias(value) {
  return String(value || "")
    .replace(/\|+/g, " ")
    .replace(/\]\]+/g, "]")
    .replace(/\s+/g, " ")
    .trim();
}

export function memoTitle(memo) {
  const lines = memoLines((memo && memo.content) || "");
  const first = lines.find(function (line) {
    return line.trim();
  });
  if (!first) return memo && memo.id ? memo.id : "Untitled memo";

  const heading = parseMemoHeadingLine(first);
  return compactMemoTitle(stripMemoTitleMarkdown(heading ? heading.text : first));
}

export function stripMemoTitleMarkdown(value) {
  return String(value || "")
    .replace(/!\[\[([^\]]+)\]\]/g, "$1")
    .replace(/\[\[([^\]]+)\]\]/g, "$1")
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, "$1")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/[`*_>#-]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

export function compactMemoTitle(value) {
  const title = String(value || "").trim();
  if (!title) return "Untitled memo";
  return title.length > 48 ? title.slice(0, 47) + "..." : title;
}

export function memoTitleKey(value) {
  return String(value || "").trim().toLowerCase();
}

export function memoSelectorLabel(selector) {
  if (!selector) return "";
  if (selector.type === "line") {
    return selector.start === selector.end ? "L" + selector.start : "L" + selector.start + "-L" + selector.end;
  }
  return "#" + selector.raw;
}

export function memoBacklinkCount(context, memoId) {
  const incoming = context && context.index && context.index.incoming;
  return incoming && incoming.has(memoId) ? incoming.get(memoId).length : 0;
}

export function compactText(value, length) {
  const text = String(value || "").replace(/\s+/g, " ").trim();
  return text.length > length ? text.slice(0, length - 1) + "..." : text;
}
