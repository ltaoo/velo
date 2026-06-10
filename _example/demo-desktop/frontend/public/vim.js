(function () {
  "use strict";

  const PM = window.ProsemirrorMod;
  if (!PM) {
    throw new Error("ProsemirrorMod was not loaded before vim.js.");
  }

  const MODES = {
    NORMAL: "normal",
    INSERT: "insert",
    VISUAL: "visual",
  };

  const vimPluginKey = new PM.PluginKey("vanillaVim");
  const LINE_JUMP_COUNT = 10;
  const VIM_PLUGIN_VERSION = "20260610-vim-empty-line-paste-boundary";
  const REGISTER_TYPES = {
    CHAR: "char",
    LINE: "line",
  };

  function clamp(value, min, max) {
    return Math.max(min, Math.min(max, value));
  }

  function getMeta(tr) {
    return tr.getMeta(vimPluginKey) || null;
  }

  function setVimMeta(tr, meta) {
    return tr.setMeta(vimPluginKey, meta);
  }

  function insertedTextFromTransaction(tr) {
    if (!tr.steps) return "";
    return tr.steps
      .map((step) => {
        if (!step.slice || !step.slice.content) return "";
        return step.slice.content.textBetween(0, step.slice.content.size, "\n", "\ufffc");
      })
      .join("");
  }

  function getPluginState(state) {
    return vimPluginKey.getState(state);
  }

  function isWordChar(ch) {
    if (!ch) return false;
    return /[\p{L}\p{N}_]/u.test(ch);
  }

  function isBlank(ch) {
    return /\s/.test(ch || "");
  }

  function charKind(ch) {
    if (!ch || isBlank(ch)) return "blank";
    return isWordChar(ch) ? "word" : "symbol";
  }

  function keyName(event) {
    if (event.key === "Escape") return "Esc";
    if (event.key === "Enter") return "Enter";
    if (event.key === "Backspace") return "Backspace";
    if (event.key === "Delete") return "Delete";
    if (event.key === "ArrowLeft") return "ArrowLeft";
    if (event.key === "ArrowRight") return "ArrowRight";
    if (event.key === "ArrowUp") return "ArrowUp";
    if (event.key === "ArrowDown") return "ArrowDown";
    if (event.ctrlKey && event.key === "[") return "Ctrl-[";
    if (event.ctrlKey && event.key.toLowerCase() === "a") return "Ctrl-a";
    if (event.ctrlKey && event.key.toLowerCase() === "r") return "Ctrl-r";
    if (event.metaKey || event.altKey || event.ctrlKey) return null;
    if (event.shiftKey && (event.code === "KeyD" || event.key.toLowerCase() === "d")) return "D";
    if (event.shiftKey && (event.code === "KeyU" || event.key.toLowerCase() === "u")) return "U";
    if (event.key.length === 1) return event.key;
    return null;
  }

  function textSelection(doc, anchor, head) {
    const size = doc.content.size;
    const safeAnchor = clamp(anchor, 0, size);
    const safeHead = clamp(head == null ? anchor : head, 0, size);

    try {
      return PM.TextSelection.create(doc, safeAnchor, safeHead);
    } catch (error) {
      return PM.Selection.near(doc.resolve(safeHead), safeHead >= safeAnchor ? 1 : -1);
    }
  }

  function setCursor(state, dispatch, pos, meta, bias) {
    if (!dispatch) return true;
    const safePos = clamp(pos, 0, state.doc.content.size);
    let selection;
    try {
      selection = PM.TextSelection.create(state.doc, safePos, safePos);
    } catch (error) {
      selection = PM.Selection.near(state.doc.resolve(safePos), bias || 1);
    }
    let tr = state.tr.setSelection(selection).scrollIntoView();
    if (meta) tr = setVimMeta(tr, meta);
    dispatch(tr);
    return true;
  }

  function setRange(state, dispatch, anchor, head, meta) {
    if (!dispatch) return true;
    let tr = state.tr
      .setSelection(textSelection(state.doc, anchor, head))
      .scrollIntoView();
    if (meta) tr = setVimMeta(tr, meta);
    dispatch(tr);
    return true;
  }

  function textblockInfo(state, pos) {
    const $pos = state.doc.resolve(clamp(pos, 0, state.doc.content.size));
    for (let depth = $pos.depth; depth > 0; depth -= 1) {
      const node = $pos.node(depth);
      if (node.isTextblock) {
        return {
          depth,
          node,
          start: $pos.start(depth),
          end: $pos.end(depth),
          before: $pos.before(depth),
          after: $pos.after(depth),
        };
      }
    }
    return {
      depth: 0,
      node: state.doc,
      start: 0,
      end: state.doc.content.size,
      before: 0,
      after: state.doc.content.size,
    };
  }

  function textblockInsertEnd(state, pos) {
    const block = textblockInfo(state, pos);
    const hardBreak = state.schema && state.schema.nodes && state.schema.nodes.hard_break;
    if (!hardBreak || !block.node.isTextblock) return block.end;

    let end = block.end;
    for (let index = block.node.childCount - 1; index >= 0; index -= 1) {
      const child = block.node.child(index);
      if (child.type !== hardBreak) break;
      end -= child.nodeSize;
    }
    return clamp(end, block.start, block.end);
  }

  function textblockListForDoc(doc) {
    const blocks = [];
    doc.descendants((node, pos) => {
      if (!node.isTextblock) return true;
      blocks.push({
        node,
        start: pos + 1,
        end: pos + 1 + node.content.size,
        before: pos,
        after: pos + node.nodeSize,
      });
      return false;
    });
    return blocks;
  }

  function textblockList(state) {
    return textblockListForDoc(state.doc);
  }

  function textblockIndexAt(blocks, pos) {
    for (let i = 0; i < blocks.length; i += 1) {
      if (blocks[i].start <= pos && pos <= blocks[i].end) return i;
    }
    for (let i = 0; i < blocks.length; i += 1) {
      if (blocks[i].start > pos) return Math.max(0, i - 1);
    }
    return Math.max(0, blocks.length - 1);
  }

  function emptyTextblockAtCursor(state, pos) {
    const cursorPos = pos == null ? state.selection.from : pos;
    const blocks = textblockList(state);
    for (let i = 0; i < blocks.length; i += 1) {
      const block = blocks[i];
      if (block.node.content.size !== 0) continue;
      if (block.before <= cursorPos && cursorPos <= block.after) return block;
    }
    return null;
  }

  function firstNonBlankInBlock(state, pos) {
    const block = textblockInfo(state, pos);
    const text = block.node.textContent || "";
    for (let i = 0; i < text.length; i += 1) {
      if (!isBlank(text[i])) return block.start + i;
    }
    return block.start;
  }

  function currentLineRange(state, count) {
    const blocks = textblockList(state);
    if (blocks.length) {
      const index = textblockIndexAt(blocks, state.selection.from);
      const lastIndex = clamp(index + (count || 1) - 1, index, blocks.length - 1);
      return {
        from: blocks[index].start,
        to: blocks[lastIndex].end,
        block: blocks[index],
      };
    }

    const block = textblockInfo(state, state.selection.from);
    return {
      from: block.start,
      to: block.end,
      block,
    };
  }

  function currentLinewiseRange(state, count) {
    const blocks = textblockList(state);
    if (!blocks.length) return currentLineRange(state, count);

    const index = textblockIndexAt(blocks, state.selection.from);
    const lastIndex = clamp(index + count - 1, index, blocks.length - 1);
    const selectedBlocks = blocks.slice(index, lastIndex + 1);

    return {
      from: selectedBlocks[0].before,
      to: selectedBlocks[selectedBlocks.length - 1].after,
      linewise: true,
      blocks: selectedBlocks,
    };
  }

  function flatTextPositions(doc) {
    const chars = [];
    doc.descendants((node, pos) => {
      if (!node.isText) return true;
      for (let i = 0; i < node.text.length; i += 1) {
        chars.push({ ch: node.text[i], pos: pos + i });
      }
      return true;
    });
    return chars;
  }

  function textblockTextPositions(block, blockIndex) {
    const chars = [];
    block.node.descendants((node, relativePos) => {
      if (!node.isText) return true;
      for (let i = 0; i < node.text.length; i += 1) {
        chars.push({
          ch: node.text[i],
          pos: block.start + relativePos + i,
          blockIndex,
        });
      }
      return true;
    });
    return chars;
  }

  function wordMotionPositions(state) {
    const blocks = textblockList(state);
    if (!blocks.length) return flatTextPositions(state.doc);

    const chars = [];
    blocks.forEach((block, index) => {
      chars.push(...textblockTextPositions(block, index));
      if (index < blocks.length - 1) {
        chars.push({
          ch: "\n",
          pos: block.end,
          blockIndex: index,
          lineBreak: true,
        });
      }
    });
    return chars;
  }

  function charBeforePositionInBlock(state, pos) {
    const block = textblockInfo(state, pos);
    if (!block.node.isTextblock || block.node.content.size === 0) return block.start;
    if (pos <= block.start) return block.start;

    const chars = [];
    block.node.descendants((node, relativePos) => {
      if (!node.isText) return true;
      for (let i = 0; i < node.text.length; i += 1) {
        chars.push({ ch: node.text[i], pos: block.start + relativePos + i });
      }
      return true;
    });

    for (let i = chars.length - 1; i >= 0; i -= 1) {
      if (chars[i].pos < pos) return chars[i].pos;
    }
    return block.start;
  }

  function normalCursorRange(state, pos) {
    const cursorPos = pos == null ? state.selection.from : pos;
    const block = textblockInfo(state, cursorPos);
    if (!block.node.isTextblock) return null;

    if (block.node.content.size === 0) {
      return { from: block.start, to: block.start, empty: true };
    }

    const from = clamp(cursorPos, block.start, block.end - 1);
    return { from, to: from + 1, empty: false };
  }

  function normalCursorPos(state) {
    if (!state.selection.empty) return state.selection.from;
    const cursor = normalCursorRange(state);
    return cursor ? cursor.from : state.selection.from;
  }

  function indexAtOrAfter(chars, pos) {
    for (let i = 0; i < chars.length; i += 1) {
      if (chars[i].pos >= pos) return i;
    }
    return chars.length;
  }

  function indexBefore(chars, pos) {
    for (let i = chars.length - 1; i >= 0; i -= 1) {
      if (chars[i].pos < pos) return i;
    }
    return -1;
  }

  function previousLineEndAtLineStart(state, pos) {
    const blocks = textblockList(state);
    if (!blocks.length) return null;

    const index = textblockIndexAt(blocks, pos);
    const block = blocks[index];
    if (!block || pos !== block.start || index <= 0) return null;
    return blocks[index - 1].end;
  }

  function nextLineStartAtLineEnd(state, pos) {
    const blocks = textblockList(state);
    if (!blocks.length) return null;

    const index = textblockIndexAt(blocks, pos);
    const block = blocks[index];
    if (!block || index >= blocks.length - 1) return null;

    const cursorLineEnd = block.end > block.start ? block.end - 1 : block.end;
    if (pos !== block.end && pos !== cursorLineEnd) return null;
    return blocks[index + 1].start;
  }

  function sameWordRun(left, right) {
    const leftKind = charKind(left && left.ch);
    return leftKind !== "blank" && leftKind === charKind(right && right.ch);
  }

  function runEndIndex(chars, index) {
    if (index < 0 || index >= chars.length) return index;
    while (index + 1 < chars.length && sameWordRun(chars[index], chars[index + 1])) {
      index += 1;
    }
    return index;
  }

  function wordForwardPos(state, pos, count) {
    const chars = wordMotionPositions(state);
    if (!chars.length) return pos;
    let index = indexAtOrAfter(chars, pos);

    for (let step = 0; step < count; step += 1) {
      const kind = charKind(chars[index] && chars[index].ch);
      if (kind !== "blank") {
        while (index < chars.length && charKind(chars[index].ch) === kind) index += 1;
      }
      while (index < chars.length && charKind(chars[index].ch) === "blank") index += 1;
    }

    if (index >= chars.length) return docEnd(state);
    return chars[index].pos;
  }

  function wordBackwardPos(state, pos, count) {
    const chars = wordMotionPositions(state);
    if (!chars.length) return pos;
    let currentPos = pos;

    for (let step = 0; step < count; step += 1) {
      const previousLineEnd = previousLineEndAtLineStart(state, currentPos);
      if (previousLineEnd != null) {
        currentPos = previousLineEnd;
        continue;
      }

      let index = indexBefore(chars, currentPos);
      while (index >= 0 && charKind(chars[index].ch) === "blank") index -= 1;
      const kind = charKind(chars[index] && chars[index].ch);
      while (index > 0 && charKind(chars[index - 1].ch) === kind) index -= 1;
      currentPos = index < 0 ? docStart(state) : chars[index].pos;
    }

    return currentPos;
  }

  function wordEndPos(state, pos, count) {
    const chars = wordMotionPositions(state);
    if (!chars.length) return pos;
    let currentPos = pos;

    for (let step = 0; step < count; step += 1) {
      const nextLineStart = nextLineStartAtLineEnd(state, currentPos);
      if (nextLineStart != null) {
        currentPos = nextLineStart;
        continue;
      }

      let index = indexAtOrAfter(chars, currentPos);
      if (index < chars.length && charKind(chars[index].ch) !== "blank" && runEndIndex(chars, index) === index) {
        index += 1;
      }
      while (index < chars.length && charKind(chars[index].ch) === "blank") index += 1;
      index = runEndIndex(chars, index);
      currentPos = index >= chars.length ? docEnd(state) : chars[index].pos;
    }

    return currentPos;
  }

  function changeWordEndPos(state, pos, count) {
    const chars = wordMotionPositions(state);
    if (!chars.length) return pos;

    let index = indexAtOrAfter(chars, pos);
    if (index >= chars.length || charKind(chars[index].ch) === "blank") return null;

    const steps = Math.max(1, count || 1);
    for (let step = 0; step < steps; step += 1) {
      while (index < chars.length && charKind(chars[index].ch) === "blank") index += 1;
      if (index >= chars.length) return docEnd(state);

      const endIndex = runEndIndex(chars, index);
      if (step === steps - 1) return chars[endIndex].pos;
      index = endIndex + 1;
    }

    return pos;
  }

  function findCharForwardPos(state, pos, query, count) {
    if (!query || query.length !== 1) return null;

    const block = textblockInfo(state, pos);
    if (!block.node.isTextblock) return null;

    const chars = textblockTextPositions(block, 0);
    let remaining = Math.max(1, count || 1);

    for (let i = 0; i < chars.length; i += 1) {
      if (chars[i].pos <= pos) continue;
      if (chars[i].ch !== query) continue;

      remaining -= 1;
      if (remaining === 0) return chars[i].pos;
    }

    return null;
  }

  function moveHorizontalPos(state, pos, delta) {
    return clamp(pos + delta, 0, state.doc.content.size);
  }

  function moveVerticalPos(view, direction, count, posOverride) {
    const { state } = view;
    const pos = posOverride == null ? state.selection.head : posOverride;
    const blocks = textblockList(state);
    if (!blocks.length) return pos;

    const index = textblockIndexAt(blocks, pos);
    const targetIndex = clamp(index + direction * count, 0, blocks.length - 1);
    const current = blocks[index];
    const target = blocks[targetIndex];
    const column = clamp(pos - current.start, 0, current.end - current.start);

    return target.start + Math.min(column, target.end - target.start);
  }

  function docStart(state) {
    return PM.Selection.atStart(state.doc).from;
  }

  function docEnd(state) {
    return PM.Selection.atEnd(state.doc).to;
  }

  function lastLineStart(state) {
    const blocks = textblockList(state);
    return blocks.length ? blocks[blocks.length - 1].start : docStart(state);
  }

  function motionTarget(view, key, count, posOverride) {
    const { state } = view;
    const pos = posOverride == null ? state.selection.head : posOverride;
    const block = textblockInfo(state, pos);

    switch (key) {
      case "h":
      case "ArrowLeft":
        return moveHorizontalPos(state, pos, -count);
      case "l":
      case "ArrowRight":
        return moveHorizontalPos(state, pos, count);
      case "j":
      case "J":
      case "ArrowDown":
        return moveVerticalPos(view, 1, count, pos);
      case "k":
      case "K":
      case "ArrowUp":
        return moveVerticalPos(view, -1, count, pos);
      case "D":
        return moveVerticalPos(view, 1, count * LINE_JUMP_COUNT, pos);
      case "U":
        return moveVerticalPos(view, -1, count * LINE_JUMP_COUNT, pos);
      case "w":
        return wordForwardPos(state, pos, count);
      case "b":
        return wordBackwardPos(state, pos, count);
      case "e":
        return wordEndPos(state, pos, count);
      case "0":
        return block.start;
      case "^":
        return firstNonBlankInBlock(state, pos);
      case "$":
        return block.end;
      case "G":
        return lastLineStart(state);
      default:
        return null;
    }
  }

  function positionAfterCursorTarget(state, pos) {
    const block = textblockInfo(state, pos);
    if (!block.node.isTextblock) return pos;
    return clamp(pos + 1, block.start, block.end);
  }

  function rangeForMotion(view, key, count, operator) {
    const { state } = view;
    const from = normalCursorPos(state);

    if (key === operator) {
      if (operator === "d" || operator === "y") {
        return currentLinewiseRange(state, count);
      }
      if (operator === "c") {
        const line = currentLineRange(state, count);
        return { from: line.from, to: line.to, linewise: true };
      }
    }

    if (operator === "c" && key === "w") {
      const chars = wordMotionPositions(state);
      const current = chars[indexAtOrAfter(chars, from)];
      if (current && charKind(current.ch) !== "blank") {
        const target = changeWordEndPos(state, from, count);
        if (target == null) return null;
        const rangeTarget = positionAfterCursorTarget(state, target);
        return {
          from: Math.min(from, rangeTarget),
          to: Math.max(from, rangeTarget),
          linewise: false,
        };
      }
    }

    const target = motionTarget(view, key, count);
    if (target == null) return null;
    const nextLineStart = key === "e" ? nextLineStartAtLineEnd(state, from) : null;
    const rangeTarget = operator && key === "e" && target !== nextLineStart
      ? positionAfterCursorTarget(state, target)
      : target;
    return {
      from: Math.min(from, rangeTarget),
      to: Math.max(from, rangeTarget),
      linewise: false,
    };
  }

  function rangeForFindChar(view, query, count) {
    const { state } = view;
    const from = normalCursorPos(state);
    const target = findCharForwardPos(state, from, query, count);
    if (target == null) return null;

    const rangeTarget = positionAfterCursorTarget(state, target);
    return {
      from: Math.min(from, rangeTarget),
      to: Math.max(from, rangeTarget),
      linewise: false,
    };
  }

  function countFrom(pluginState) {
    const parsed = Number(pluginState.count || "1");
    return Number.isFinite(parsed) && parsed > 0 ? parsed : 1;
  }

  function clearTransient(meta) {
    return {
      pending: null,
      count: "",
      message: "",
      imeFeedback: null,
      ...(meta || {}),
    };
  }

  function withRepeat(meta, repeatAction) {
    if (!repeatAction) return meta;
    return {
      ...meta,
      lastRepeat: repeatAction,
    };
  }

  function insertSessionFor(action) {
    return action ? { ...action, text: action.text || "" } : null;
  }

  function insertRepeatFromSession(session) {
    if (!session) return null;
    return { ...session, text: session.text || "" };
  }

  function setMode(state, dispatch, mode, extra) {
    if (!dispatch) return true;
    const meta = clearTransient({
      mode,
      visualAnchor: mode === MODES.VISUAL ? state.selection.from : null,
      ...(extra || {}),
    });
    dispatch(setVimMeta(state.tr, meta));
    return true;
  }

  function enterInsertAt(view, pos) {
    const { state } = view;
    const meta = clearTransient({
      mode: MODES.INSERT,
      visualAnchor: null,
      insertSession: insertSessionFor({ type: "insertText", text: "" }),
    });
    setCursor(state, view.dispatch, pos, meta);
    view.focus();
    return true;
  }

  function leaveInsert(view) {
    const { state } = view;
    const pluginState = getPluginState(state);
    const repeatAction = insertRepeatFromSession(pluginState.insertSession);
    const pos = charBeforePositionInBlock(state, state.selection.from);
    const meta = clearTransient({
      mode: MODES.NORMAL,
      visualAnchor: null,
      insertSession: null,
      ...(repeatAction && repeatAction.text ? { lastRepeat: repeatAction } : {}),
    });
    return setCursor(state, view.dispatch, pos, meta, -1);
  }

  function enterVisual(view, linewise) {
    const { state } = view;
    const line = currentLineRange(state);
    const anchor = linewise ? line.from : normalCursorPos(state);
    const range = linewise
      ? { anchor, head: line.to }
      : visualSelectionRangeForCursor(state, anchor, anchor);
    return setRange(state, view.dispatch, range.anchor, range.head, {
      mode: MODES.VISUAL,
      visualAnchor: anchor,
      visualLine: !!linewise,
      pending: null,
      count: "",
      message: "",
    });
  }

  function leaveVisual(view, cursorPos) {
    const { state } = view;
    const pluginState = getPluginState(state);
    let pos = cursorPos;
    if (pos == null) {
      pos =
        pluginState && pluginState.mode === MODES.VISUAL
          ? visualCursorDisplayPos(state, pluginState)
          : state.selection.to;
    }
    return setCursor(state, view.dispatch, pos, {
      mode: MODES.NORMAL,
      visualAnchor: null,
      visualLine: false,
      pending: null,
      count: "",
      message: "",
    });
  }

  function yankRange(state, dispatch, from, to, message, registerType) {
    if (!dispatch) return true;
    const text = state.doc.textBetween(from, to, "\n");
    dispatch(
      setVimMeta(state.tr, {
        register: text,
        registerType: registerType || REGISTER_TYPES.CHAR,
        pending: null,
        count: "",
        message: message || "yanked",
      }),
    );
    return true;
  }

  function deleteRange(state, dispatch, from, to, change, repeatAction) {
    if (!dispatch) return true;
    const text = state.doc.textBetween(from, to, "\n");
    let tr = state.tr.delete(from, to);
    const nextPos = clamp(from, 0, tr.doc.content.size);
    tr = tr.setSelection(PM.Selection.near(tr.doc.resolve(nextPos), 1));
    const meta = withRepeat({
      register: text,
      registerType: REGISTER_TYPES.CHAR,
      mode: change ? MODES.INSERT : MODES.NORMAL,
      visualAnchor: null,
      visualLine: false,
      pending: null,
      count: "",
      insertSession: change ? insertSessionFor(repeatAction) : null,
      message: change ? "change" : "deleted",
    }, repeatAction);
    tr = setVimMeta(tr, meta);
    dispatch(tr.scrollIntoView());
    return true;
  }

  function deleteLinewiseRange(state, dispatch, range, repeatAction) {
    if (!dispatch) return true;
    const blocks = range.blocks && range.blocks.length ? range.blocks : null;
    const text = blocks
      ? blocks.map((block) => state.doc.textBetween(block.start, block.end, "\n")).join("\n")
      : state.doc.textBetween(range.from, range.to, "\n");

    let tr = state.tr;
    if (blocks) {
      for (let i = blocks.length - 1; i >= 0; i -= 1) {
        tr = tr.deleteRange(blocks[i].before, blocks[i].after);
      }
    } else {
      tr = tr.deleteRange(range.from, range.to);
    }

    const nextPos = clamp(tr.mapping.map(range.from, -1), 0, tr.doc.content.size);
    tr = tr.setSelection(PM.Selection.near(tr.doc.resolve(nextPos), 1));
    tr = setVimMeta(tr, withRepeat({
      register: text,
      registerType: REGISTER_TYPES.LINE,
      mode: MODES.NORMAL,
      visualAnchor: null,
      visualLine: false,
      pending: null,
      count: "",
      insertSession: null,
      message: "deleted",
    }, repeatAction));
    dispatch(tr.scrollIntoView());
    return true;
  }

  function applyOperator(view, operator, range, repeatAction) {
    if (!range || range.from === range.to) {
      view.dispatch(
        setVimMeta(view.state.tr, {
          pending: null,
          count: "",
          message: "empty range",
        }),
      );
      return true;
    }

    if (operator === "y") {
      return yankRange(
        view.state,
        view.dispatch,
        range.from,
        range.to,
        "yanked",
        range.linewise ? REGISTER_TYPES.LINE : REGISTER_TYPES.CHAR,
      );
    }
    if (operator === "d") {
      if (range.linewise) {
        return deleteLinewiseRange(view.state, view.dispatch, range, repeatAction);
      }
      return deleteRange(view.state, view.dispatch, range.from, range.to, false, repeatAction);
    }
    if (operator === "c") {
      return deleteRange(view.state, view.dispatch, range.from, range.to, true, repeatAction);
    }
    return false;
  }

  function deleteCharacter(view, count, repeatAction) {
    const { state } = view;
    const from = state.selection.from;
    const to = clamp(from + count, 0, state.doc.content.size);
    if (from === to) return true;
    return deleteRange(state, view.dispatch, from, to, false, repeatAction);
  }

  function substituteCharacter(view, count) {
    const { state } = view;
    const cursor = normalCursorRange(state);
    if (cursor && cursor.empty) return enterInsertAt(view, cursor.from);
    const from = cursor ? cursor.from : state.selection.from;
    const to = clamp(from + count, 0, state.doc.content.size);
    const repeatAction = { type: "change", operator: "c", motion: "l", count, text: "" };
    if (from === to) return enterInsertAt(view, from);
    return deleteRange(state, view.dispatch, from, to, true, repeatAction);
  }

  function toggleCaseText(text) {
    if (!text) return text;
    const upper = text.toLocaleUpperCase();
    const lower = text.toLocaleLowerCase();
    if (text === upper && text !== lower) return lower;
    if (text === lower && text !== upper) return upper;
    return upper;
  }

  function toggleCaseAtCursor(view, count, repeatAction) {
    const { state } = view;
    const block = textblockInfo(state, state.selection.from);
    if (!block.node.isTextblock || block.node.content.size === 0) {
      view.dispatch(setVimMeta(state.tr, clearTransient({ message: "empty line" })));
      return true;
    }

    const repeatTimes = Math.max(1, count || 1);
    let tr = state.tr;
    let pos = normalCursorPos(state);
    let changed = 0;

    for (let i = 0; i < repeatTimes && pos < block.end; i += 1) {
      const current = tr.doc.textBetween(pos, pos + 1, "");
      const toggled = toggleCaseText(current);
      if (current && toggled !== current) tr = tr.insertText(toggled, pos, pos + 1);
      pos += Math.max(1, toggled ? toggled.length : 1);
      changed += 1;
    }

    if (!changed) {
      view.dispatch(setVimMeta(state.tr, clearTransient({ message: "no character" })));
      return true;
    }

    const cursorPos = clamp(pos, block.start, Math.max(block.start, block.end - 1));
    tr = tr.setSelection(textSelection(tr.doc, cursorPos, cursorPos));
    tr = setVimMeta(
      tr,
      withRepeat(clearTransient({ message: "case toggled" }), repeatAction),
    );
    view.dispatch(tr.scrollIntoView());
    return true;
  }

  function leadingBlankEndInBlock(block) {
    let to = block.start;
    let scanning = true;

    block.node.descendants((node, relativePos) => {
      if (!scanning) return false;
      if (node.isText) {
        for (let i = 0; i < node.text.length; i += 1) {
          if (!isBlank(node.text[i])) {
            scanning = false;
            return false;
          }
          to = block.start + relativePos + i + 1;
        }
        return true;
      }
      if (node.isInline || node.isLeaf || node.isAtom) {
        scanning = false;
        return false;
      }
      return true;
    });

    return to;
  }

  function needsJoinSpace(leftBlock, rightBlock) {
    const leftText = leftBlock.node.textContent || "";
    const rightText = (rightBlock.node.textContent || "").replace(/^\s+/, "");
    if (!leftText || !rightText) return false;
    return !isBlank(leftText[leftText.length - 1]);
  }

  function joinLines(view, count, repeatAction) {
    const { state } = view;
    if (!view.dispatch) return true;

    let tr = state.tr;
    let joined = 0;
    let cursorPos = null;
    const joinCount = Math.max(1, (count || 1) - 1);

    for (let i = 0; i < joinCount; i += 1) {
      const blocks = textblockListForDoc(tr.doc);
      if (!blocks.length) break;

      const currentPos = cursorPos == null ? tr.mapping.map(state.selection.from) : cursorPos;
      const index = textblockIndexAt(blocks, currentPos);
      const current = blocks[index];
      const next = blocks[index + 1];
      if (!current || !next || !PM.canJoin(tr.doc, current.after)) break;

      const trimTo = leadingBlankEndInBlock(next);
      const insertSpace = needsJoinSpace(current, next);
      if (trimTo > next.start) tr = tr.delete(next.start, trimTo);

      tr = tr.join(current.after);
      if (insertSpace) tr = tr.insertText(" ", current.end, current.end);

      if (cursorPos == null) cursorPos = current.end;
      joined += 1;
    }

    if (!joined) {
      view.dispatch(
        setVimMeta(state.tr, {
          pending: null,
          count: "",
          message: "no next line",
        }),
      );
      return true;
    }

    const selectionPos = clamp(cursorPos, 0, tr.doc.content.size);
    tr = tr.setSelection(textSelection(tr.doc, selectionPos, selectionPos));
    tr = setVimMeta(tr, withRepeat({
      pending: null,
      count: "",
      insertSession: null,
      message: "joined",
    }, repeatAction));
    view.dispatch(tr.scrollIntoView());
    return true;
  }

  function linewisePasteNodes(state, text, count) {
    const paragraph = state.schema.nodes.paragraph;
    if (!paragraph) return [];

    const lines = String(text == null ? "" : text).replace(/\r\n?/g, "\n").split("\n");
    const repeatTimes = Math.max(1, count || 1);
    const nodes = [];
    for (let i = 0; i < repeatTimes; i += 1) {
      for (let j = 0; j < lines.length; j += 1) {
        const line = lines[j];
        nodes.push(paragraph.create(null, line ? state.schema.text(line) : null));
      }
    }
    return nodes;
  }

  function pasteLinewiseRegister(view, before, repeatAction, count) {
    const { state } = view;
    const pluginState = getPluginState(state);
    const block = textblockInfo(state, state.selection.from);
    const nodes = linewisePasteNodes(state, pluginState.register, count);
    if (!nodes.length) {
      view.dispatch(setVimMeta(state.tr, { message: "register empty" }));
      return true;
    }

    const insertAt = before ? block.before : block.after;
    let tr = state.tr.insert(insertAt, nodes.length === 1 ? nodes[0] : nodes);
    const selectionPos = clamp(insertAt + 1, 0, tr.doc.content.size);
    tr = tr.setSelection(textSelection(tr.doc, selectionPos, selectionPos));
    tr = setVimMeta(
      tr,
      withRepeat(
        {
          message: "pasted",
          pending: null,
          count: "",
          insertSession: null,
          registerType: REGISTER_TYPES.LINE,
        },
        repeatAction,
      ),
    );
    view.dispatch(tr.scrollIntoView());
    return true;
  }

  function pasteRegister(view, before, repeatAction, count) {
    const { state } = view;
    const pluginState = getPluginState(state);
    if (pluginState.registerType === REGISTER_TYPES.LINE) {
      return pasteLinewiseRegister(view, before, repeatAction, count);
    }
    if (!pluginState.register) {
      view.dispatch(setVimMeta(state.tr, { message: "register empty" }));
      return true;
    }

    const pos = pasteInsertPos(state, before);
    const text = pluginState.register.repeat(Math.max(1, count || 1));
    let tr = state.tr.insertText(text, pos, pos);
    const cursorPos = lastInsertedTextCursorPos(tr.doc, pos, text);
    tr = tr.setSelection(textSelection(tr.doc, cursorPos, cursorPos));
    tr = setVimMeta(
      tr,
      withRepeat({ message: "pasted", pending: null, count: "", insertSession: null }, repeatAction),
    );
    view.dispatch(tr.scrollIntoView());
    return true;
  }

  function insertNewParagraph(view, after, repeatAction) {
    const { state } = view;
    const block = textblockInfo(state, state.selection.from);
    const paragraph = state.schema.nodes.paragraph.createAndFill();
    if (!paragraph) return false;

    const insertAt = after ? block.after : block.before;
    const cursorPos = insertAt + 1;
    let tr = state.tr.insert(insertAt, paragraph);
    tr = tr.setSelection(textSelection(tr.doc, cursorPos, cursorPos));
    tr = setVimMeta(
      tr,
      withRepeat(
        clearTransient({
          mode: MODES.INSERT,
          visualAnchor: null,
          insertSession: insertSessionFor(repeatAction),
        }),
        repeatAction,
      ),
    );
    view.dispatch(tr.scrollIntoView());
    return true;
  }

  function repeatCount(action, countOverride) {
    return countOverride || action.count || 1;
  }

  function applyRepeatedChange(view, action, count) {
    const { state } = view;
    const range = action.motion === "f"
      ? rangeForFindChar(view, action.query, count)
      : rangeForMotion(view, action.motion, count, action.operator);
    if (!range || range.from === range.to || !view.dispatch) return true;

    const deletedText = state.doc.textBetween(range.from, range.to, "\n");
    let tr = state.tr.delete(range.from, range.to);
    const insertAt = clamp(range.from, 0, tr.doc.content.size);
    if (action.text) tr = tr.insertText(action.text, insertAt, insertAt);
    const selectionPos = clamp(insertAt + (action.text ? action.text.length : 0), 0, tr.doc.content.size);
    tr = tr.setSelection(PM.Selection.near(tr.doc.resolve(selectionPos), -1));
    tr = setVimMeta(
      tr,
      withRepeat({
        register: deletedText,
        registerType: REGISTER_TYPES.CHAR,
        mode: MODES.NORMAL,
        visualAnchor: null,
        visualLine: false,
        pending: null,
        count: "",
        insertSession: null,
        message: "change",
      }, { ...action, count }),
    );
    view.dispatch(tr.scrollIntoView());
    return true;
  }

  function repeatInsertText(view, action, count) {
    if (!view.dispatch || !action.text) return true;
    const { state } = view;
    const pos = state.selection.from;
    const text = action.text.repeat(Math.max(1, count || 1));
    let tr = state.tr.insertText(text, pos, pos);
    const selectionPos = clamp(pos + text.length, 0, tr.doc.content.size);
    tr = tr.setSelection(PM.Selection.near(tr.doc.resolve(selectionPos), -1));
    tr = setVimMeta(
      tr,
      withRepeat(clearTransient({
        mode: MODES.NORMAL,
        visualAnchor: null,
        insertSession: null,
        message: "repeated",
      }), { ...action, count }),
    );
    view.dispatch(tr.scrollIntoView());
    return true;
  }

  function repeatNewParagraph(view, action, count) {
    if (!view.dispatch) return true;
    const { state } = view;
    let tr = state.tr;
    let selectionPos = state.selection.from;
    const repeatTimes = Math.max(1, count || 1);

    for (let i = 0; i < repeatTimes; i += 1) {
      const currentPos = clamp(selectionPos, 0, tr.doc.content.size);
      const block = textblockInfo({ doc: tr.doc }, currentPos);
      const paragraph = state.schema.nodes.paragraph.createAndFill();
      if (!paragraph) break;
      const insertAt = action.after ? block.after : block.before;
      const textPos = insertAt + 1;
      tr = tr.insert(insertAt, paragraph);
      if (action.text) tr = tr.insertText(action.text, textPos, textPos);
      selectionPos = textPos + (action.text ? action.text.length : 0);
    }

    tr = tr.setSelection(PM.Selection.near(tr.doc.resolve(clamp(selectionPos, 0, tr.doc.content.size)), -1));
    tr = setVimMeta(
      tr,
      withRepeat(clearTransient({
        mode: MODES.NORMAL,
        visualAnchor: null,
        insertSession: null,
        message: "repeated",
      }), { ...action, count }),
    );
    view.dispatch(tr.scrollIntoView());
    return true;
  }

  function repeatLastChange(view, countOverride) {
    const pluginState = getPluginState(view.state);
    const action = pluginState.lastRepeat;
    if (!action) {
      view.dispatch(setVimMeta(view.state.tr, { message: "no repeat" }));
      return true;
    }

    const count = repeatCount(action, countOverride);
    if (action.type === "operator") {
      const range = action.motion === "f"
        ? rangeForFindChar(view, action.query, count)
        : rangeForMotion(view, action.motion, count, action.operator);
      return applyOperator(view, action.operator, range, { ...action, count });
    }
    if (action.type === "change") {
      return applyRepeatedChange(view, action, count);
    }
    if (action.type === "insertText") {
      return repeatInsertText(view, action, count);
    }
    if (action.type === "newParagraph") {
      return repeatNewParagraph(view, action, count);
    }
    if (action.type === "command") {
      if (action.key === "x") return deleteCharacter(view, count, { ...action, count });
      if (action.key === "p") return pasteRegister(view, false, { ...action, count }, count);
      if (action.key === "P") return pasteRegister(view, true, { ...action, count }, count);
      if (action.key === "J") return joinLines(view, count, { ...action, count });
      if (action.key === "~") return toggleCaseAtCursor(view, count, { ...action, count });
    }

    view.dispatch(setVimMeta(view.state.tr, { message: "repeat unsupported" }));
    return true;
  }

  function searchText(state, query, direction) {
    const chars = flatTextPositions(state.doc);
    if (!query || !chars.length) return null;

    const text = chars.map((item) => item.ch).join("");
    const currentIndex = Math.max(0, indexAtOrAfter(chars, state.selection.to));
    let matchIndex = -1;

    if (direction > 0) {
      matchIndex = text.indexOf(query, currentIndex + 1);
      if (matchIndex < 0) matchIndex = text.indexOf(query, 0);
    } else {
      matchIndex = text.lastIndexOf(query, Math.max(0, currentIndex - 1));
      if (matchIndex < 0) matchIndex = text.lastIndexOf(query);
    }

    if (matchIndex < 0 || !chars[matchIndex]) return null;
    const from = chars[matchIndex].pos;
    const last = chars[Math.min(chars.length - 1, matchIndex + query.length - 1)];
    return { from, to: last.pos + 1 };
  }

  function runSearch(view, query, direction) {
    const result = searchText(view.state, query, direction);
    if (!result) {
      view.dispatch(
        setVimMeta(view.state.tr, {
          searchQuery: query,
          searchDirection: direction,
          searchRange: null,
          message: "not found: " + query,
        }),
      );
      return true;
    }

    let tr = view.state.tr.setSelection(textSelection(view.state.doc, result.from, result.to));
    tr = setVimMeta(tr, {
      mode: MODES.NORMAL,
      searchQuery: query,
      searchDirection: direction,
      searchRange: result,
      pending: null,
      count: "",
      message: "/" + query,
    });
    view.dispatch(tr.scrollIntoView());
    return true;
  }

  function promptSearch(view) {
    const pluginState = getPluginState(view.state);
    const query = window.prompt("/", pluginState.searchQuery || "");
    if (!query) return true;
    return runSearch(view, query, 1);
  }

  function repeatSearch(view, reverse) {
    const pluginState = getPluginState(view.state);
    if (!pluginState.searchQuery) return true;
    const direction = reverse ? -pluginState.searchDirection : pluginState.searchDirection;
    return runSearch(view, pluginState.searchQuery, direction || 1);
  }

  function exResultMessage(result, fallback) {
    if (typeof result === "string") return result;
    if (result && typeof result === "object" && result.message) return result.message;
    return fallback;
  }

  function exErrorMessage(error, fallback) {
    if (error && error.message) return fallback + ": " + error.message;
    if (error) return fallback + ": " + String(error);
    return fallback;
  }

  function dispatchVimMessage(view, message, extra) {
    if (!view || view.isDestroyed || !view.state || !view.dispatch) return;
    try {
      view.dispatch(setVimMeta(view.state.tr, { ...(extra || {}), message }));
    } catch (error) {
      if (window.console && window.console.debug) {
        window.console.debug("Skipped vim message for inactive editor.", error);
      }
    }
  }

  function runAsyncExCommand(view, pendingMessage, fallbackMessage, failureMessage, runner) {
    dispatchVimMessage(view, pendingMessage);
    Promise.resolve()
      .then(runner)
      .then(
        function (result) {
          dispatchVimMessage(view, exResultMessage(result, fallbackMessage));
        },
        function (error) {
          dispatchVimMessage(view, exErrorMessage(error, failureMessage));
        },
      );
    return true;
  }

  function parseSimpleExCommand(raw) {
    const value = String(raw || "").trim().replace(/^:/, "");
    const bang = value.endsWith("!");
    const command = bang ? value.slice(0, -1).trim() : value;
    return { raw: value, command, bang };
  }

  function jumpToExLine(view, command) {
    let index = null;
    if (command === "$") {
      index = -1;
    } else if (/^[1-9]\d*$/.test(command)) {
      index = Number(command) - 1;
    }
    if (index == null) return false;
    const blocks = textblockList(view.state);
    if (!blocks.length) {
      dispatchVimMessage(view, "line number out of range");
      return true;
    }
    if (index === -1) index = blocks.length - 1;
    if (index < 0 || index >= blocks.length) {
      dispatchVimMessage(view, "line number out of range");
      return true;
    }
    setCursor(view.state, view.dispatch, blocks[index].start, clearTransient({
      message: ":" + command,
    }));
    return true;
  }

  function runExCommand(view, options) {
    const raw = window.prompt(":", "");
    if (raw == null) return true;
    const parsed = parseSimpleExCommand(raw);
    const command = parsed.command;

    if (!command) {
      dispatchVimMessage(view, "");
      return true;
    }

    if (jumpToExLine(view, command)) return true;

    if (command === "w" || command === "write" || command === "up" || command === "update") {
      const writeDraft = options.onWriteDraft || options.onSave;
      if (!writeDraft) {
        dispatchVimMessage(view, "write draft is not available");
        return true;
      }
      if ((command === "up" || command === "update") && options.isDirty && !options.isDirty()) {
        dispatchVimMessage(view, "draft unchanged");
        return true;
      }
      return runAsyncExCommand(
        view,
        "writing draft...",
        "draft written",
        "write draft failed",
        function () {
          return writeDraft({ bang: parsed.bang, command, source: "vim-write" });
        },
      );
    }

    if (command === "wq" || command === "x") {
      const commit = options.onCommit || options.onSave;
      if (!commit) {
        dispatchVimMessage(view, "commit is not available");
        return true;
      }
      return runAsyncExCommand(
        view,
        "committing...",
        "committed",
        "commit failed",
        function () {
          return commit({ bang: parsed.bang, command, source: command === "x" ? "vim-x" : "vim-wq" });
        },
      );
    }

    if (command === "q" || command === "quit") {
      const quit = parsed.bang ? (options.onDiscard || options.onQuit) : options.onQuit;
      if (!quit) {
        dispatchVimMessage(view, "quit is not available");
        return true;
      }
      return runAsyncExCommand(
        view,
        parsed.bang ? "discarding..." : "quitting...",
        parsed.bang ? "discarded" : "quit",
        parsed.bang ? "discard failed" : "quit failed",
        function () {
          return quit({ bang: parsed.bang, command, force: parsed.bang, source: parsed.bang ? "vim-q-bang" : "vim-q" });
        },
      );
    }

    if (command === "nohl" || command === "noh") {
      view.dispatch(
        setVimMeta(view.state.tr, {
          searchRange: null,
          message: "search highlight cleared",
        }),
      );
      return true;
    }

    if (command === "help") {
      dispatchVimMessage(view, "commands: w, wq, q, q!, x, nohl, {line}");
      return true;
    }

    dispatchVimMessage(view, "Not an editor command: " + command);
    return true;
  }

  function dispatchCount(state, dispatch, key) {
    if (!dispatch) return true;
    const pluginState = getPluginState(state);
    const count = (pluginState.count || "") + key;
    dispatch(setVimMeta(state.tr, { count, message: count }));
    return true;
  }

  function dispatchPending(state, dispatch, pending, message, keepCount) {
    if (!dispatch) return true;
    const pluginState = getPluginState(state);
    dispatch(
      setVimMeta(state.tr, {
        pending,
        count: keepCount ? pluginState.count : "",
        message: message || pending,
      }),
    );
    return true;
  }

  function findPendingValue(mode, operator) {
    return "find:" + mode + ":" + (operator || "");
  }

  function parseFindPending(value) {
    const match = String(value || "").match(/^find:(normal|visual|operator):([dyc]?)$/);
    if (!match) return null;
    return {
      mode: match[1],
      operator: match[2] || null,
    };
  }

  function isFindQueryKey(key) {
    return key && key.length === 1;
  }

  function dispatchFindPending(state, dispatch, mode, operator) {
    return dispatchPending(state, dispatch, findPendingValue(mode, operator), "f", true);
  }

  function runFindChar(view, query, count) {
    const { state } = view;
    const target = findCharForwardPos(state, state.selection.head, query, count);
    if (target == null) {
      view.dispatch(
        setVimMeta(state.tr, {
          pending: null,
          count: "",
          message: "not found: " + query,
        }),
      );
      return true;
    }

    return setCursor(state, view.dispatch, target, clearTransient({
      lastFind: { query },
      message: "f" + query,
    }));
  }

  function visualFindChar(view, query, count) {
    const { state } = view;
    const pluginState = getPluginState(state);
    const cursorPos = visualCursorDisplayPos(state, pluginState);
    const target = findCharForwardPos(state, cursorPos, query, count);
    if (target == null) {
      view.dispatch(
        setVimMeta(state.tr, {
          pending: null,
          count: "",
          message: "not found: " + query,
        }),
      );
      return true;
    }

    const range = visualSelectionRangeForCursor(state, pluginState.visualAnchor, target);
    return setRange(state, view.dispatch, range.anchor, range.head, {
      mode: MODES.VISUAL,
      visualAnchor: pluginState.visualAnchor,
      pending: null,
      count: "",
      lastFind: { query },
      message: "f" + query,
    });
  }

  function repeatFindChar(view, count, visual) {
    const pluginState = getPluginState(view.state);
    if (!pluginState.lastFind || !pluginState.lastFind.query) {
      view.dispatch(setVimMeta(view.state.tr, { message: "no find" }));
      return true;
    }

    return visual
      ? visualFindChar(view, pluginState.lastFind.query, count)
      : runFindChar(view, pluginState.lastFind.query, count);
  }

  function normalCommand(view, key, options) {
    const { state } = view;
    const pluginState = getPluginState(state);
    const count = countFrom(pluginState);
    const countOverride = pluginState.count ? count : null;
    const findPending = parseFindPending(pluginState.pending);

    if (findPending) {
      if (key === "Esc" || key === "Ctrl-[") {
        return setMode(state, view.dispatch, MODES.NORMAL);
      }
      if (!isFindQueryKey(key)) {
        return dispatchPending(state, view.dispatch, null, "not mapped: f" + key);
      }
      if (findPending.mode === "operator" && findPending.operator) {
        const range = rangeForFindChar(view, key, count);
        if (!range) {
          view.dispatch(
            setVimMeta(state.tr, {
              pending: null,
              count: "",
              message: "not found: " + key,
            }),
          );
          return true;
        }
        const repeatAction =
          findPending.operator === "y"
            ? null
            : {
                type: findPending.operator === "c" ? "change" : "operator",
                operator: findPending.operator,
                motion: "f",
                query: key,
                count,
                text: "",
              };
        return applyOperator(view, findPending.operator, range, repeatAction);
      }
      return runFindChar(view, key, count);
    }

    if (/^[1-9]$/.test(key) || (key === "0" && pluginState.count)) {
      return dispatchCount(state, view.dispatch, key);
    }

    if (key === "." && !pluginState.pending) return repeatLastChange(view, countOverride);
    if (key === ";" && !pluginState.pending) return repeatFindChar(view, count, false);

    if (pluginState.pending === "g") {
      if (key === "g") {
        return setCursor(state, view.dispatch, docStart(state), clearTransient());
      }
      return dispatchPending(state, view.dispatch, null, "unknown g" + key);
    }

    if (pluginState.pending === "d" || pluginState.pending === "y" || pluginState.pending === "c") {
      if (key === "f") {
        return dispatchFindPending(state, view.dispatch, "operator", pluginState.pending);
      }
      const range = rangeForMotion(view, key, count, pluginState.pending);
      const repeatAction =
        pluginState.pending === "y"
          ? null
          : {
              type: pluginState.pending === "c" ? "change" : "operator",
              operator: pluginState.pending,
              motion: key,
              count,
              text: "",
            };
      return applyOperator(view, pluginState.pending, range, repeatAction);
    }

    if (key === "Esc" || key === "Ctrl-[") {
      return setMode(state, view.dispatch, MODES.NORMAL);
    }

    if (key === "g") return dispatchPending(state, view.dispatch, "g", "g");
    if (key === "G") return setCursor(state, view.dispatch, lastLineStart(state), clearTransient());
    if (key === "J") {
      return joinLines(view, count, { type: "command", key: "J", count });
    }

    if ("hjklJKDUweb0^$".includes(key) || key.startsWith("Arrow")) {
      const target = motionTarget(view, key, count);
      if (target != null) {
        const cursorTarget = key === "$" ? charBeforePositionInBlock(state, target) : target;
        return setCursor(state, view.dispatch, cursorTarget, clearTransient());
      }
    }
    if (key === "f") return dispatchFindPending(state, view.dispatch, "normal", null);

    if (key === "i") return enterInsertAt(view, state.selection.from);
    if (key === "a") return enterInsertAt(view, moveHorizontalPos(state, state.selection.from, 1));
    if (key === "I") return enterInsertAt(view, firstNonBlankInBlock(state, state.selection.from));
    if (key === "A" || key === "Ctrl-a") {
      return enterInsertAt(view, textblockInsertEnd(state, state.selection.from));
    }
    if (key === "o") {
      return insertNewParagraph(view, true, { type: "newParagraph", after: true, count, text: "" });
    }
    if (key === "O") {
      return insertNewParagraph(view, false, { type: "newParagraph", after: false, count, text: "" });
    }

    if (key === "v") return enterVisual(view, false);
    if (key === "V") return enterVisual(view, true);

    if (key === "d" || key === "y" || key === "c") {
      return dispatchPending(state, view.dispatch, key, key, true);
    }

    if (key === "x" || key === "Delete") {
      return deleteCharacter(view, count, { type: "command", key: "x", count });
    }
    if (key === "~") return toggleCaseAtCursor(view, count, { type: "command", key: "~", count });
    if (key === "s") return substituteCharacter(view, count);
    if (key === "p") return pasteRegister(view, false, { type: "command", key: "p", count }, count);
    if (key === "P") return pasteRegister(view, true, { type: "command", key: "P", count }, count);

    if (key === "u") return PM.undo(state, view.dispatch, view);
    if (key === "Ctrl-r") return PM.redo(state, view.dispatch, view);

    if (key === "/") return promptSearch(view);
    if (key === "n") return repeatSearch(view, false);
    if (key === "N") return repeatSearch(view, true);
    if (key === ":") return runExCommand(view, options);

    view.dispatch(
      setVimMeta(state.tr, {
        pending: null,
        count: "",
        message: "not mapped: " + key,
      }),
    );
    return true;
  }

  function visualCommand(view, key) {
    const { state } = view;
    const pluginState = getPluginState(state);
    const count = countFrom(pluginState);
    const findPending = parseFindPending(pluginState.pending);

    if (key === "Esc" || key === "Ctrl-[") return leaveVisual(view);

    if (findPending) {
      if (!isFindQueryKey(key)) {
        return dispatchPending(state, view.dispatch, null, "not mapped: f" + key);
      }
      return visualFindChar(view, key, count);
    }

    if (key === "o") {
      const anchor = visualCursorDisplayPos(state, pluginState);
      const range = visualSelectionRangeForCursor(state, anchor, pluginState.visualAnchor);
      return setRange(state, view.dispatch, range.anchor, range.head, {
        mode: MODES.VISUAL,
        visualAnchor: anchor,
        message: "swapped visual anchor",
      });
    }

    if ("hjklJKDUweb0^$G".includes(key) || key.startsWith("Arrow")) {
      const cursorPos = visualCursorDisplayPos(state, pluginState);
      const target = motionTarget(view, key, count, cursorPos);
      if (target == null) return true;
      const range = visualSelectionRangeForCursor(state, pluginState.visualAnchor, target);
      return setRange(state, view.dispatch, range.anchor, range.head, {
        mode: MODES.VISUAL,
        visualAnchor: pluginState.visualAnchor,
        count: "",
        message: "",
      });
    }

    if (key === "f") return dispatchFindPending(state, view.dispatch, "visual", null);
    if (key === ";") return repeatFindChar(view, count, true);

    if (key === "d" || key === "x") {
      const selectedSize = state.selection.to - state.selection.from;
      return deleteRange(
        state,
        view.dispatch,
        state.selection.from,
        state.selection.to,
        false,
        { type: "command", key: "x", count: selectedSize },
      );
    }
    if (key === "c") {
      const selectedSize = state.selection.to - state.selection.from;
      return deleteRange(
        state,
        view.dispatch,
        state.selection.from,
        state.selection.to,
        true,
        { type: "change", operator: "c", motion: "l", count: selectedSize, text: "" },
      );
    }
    if (key === "y") {
      const handled = yankRange(
        state,
        view.dispatch,
        state.selection.from,
        state.selection.to,
        "yanked",
        pluginState.visualLine ? REGISTER_TYPES.LINE : REGISTER_TYPES.CHAR,
      );
      leaveVisual(view, pluginState.visualAnchor);
      return handled;
    }
    if (key === "p") {
      const register = pluginState.register;
      if (!register) return true;
      const insertAt = visualPasteInsertPos(state, pluginState);
      let tr = state.tr.insertText(register, insertAt, insertAt);
      const selectionPos = lastInsertedTextCursorPos(tr.doc, insertAt, register);
      tr = tr.setSelection(textSelection(tr.doc, selectionPos, selectionPos));
      tr = setVimMeta(tr, {
        mode: MODES.NORMAL,
        visualAnchor: null,
        visualLine: false,
        pending: null,
        count: "",
        register,
        registerType: pluginState.registerType || REGISTER_TYPES.CHAR,
        lastRepeat: { type: "command", key: "p", count: 1 },
        message: "pasted",
      });
      view.dispatch(tr.scrollIntoView());
      return true;
    }

    return true;
  }

  function handleKeyDown(view, event, options) {
    const pluginState = getPluginState(view.state);
    if (pluginState.mode !== MODES.INSERT && isComposingKeyEvent(event)) {
      updateImeFeedback(view, event, "advance");
      event.preventDefault();
      if (typeof event.stopPropagation === "function") event.stopPropagation();
      return true;
    }

    const key = keyName(event);

    if (pluginState.mode === MODES.INSERT) {
      if (key === "Ctrl-a") {
        event.preventDefault();
        return enterInsertAt(view, textblockInsertEnd(view.state, view.state.selection.from));
      }
      if (key === "Esc" || key === "Ctrl-[") {
        event.preventDefault();
        return leaveInsert(view);
      }
      return false;
    }

    if (!key) return false;

    event.preventDefault();
    if (pluginState.mode === MODES.VISUAL) return visualCommand(view, key);
    return normalCommand(view, key, options);
  }

  function shouldBlockNativeTextInput(view) {
    const pluginState = getPluginState(view.state);
    return pluginState && pluginState.mode !== MODES.INSERT;
  }

  function isComposingKeyEvent(event) {
    return !!(
      event &&
      (event.isComposing ||
        event.key === "Process" ||
        event.keyCode === 229 ||
        event.which === 229)
    );
  }

  function blockNativeTextInput(view, event) {
    if (!shouldBlockNativeTextInput(view)) return false;
    const type = event && event.type;
    if (type === "compositionstart") {
      updateImeFeedback(view, event, "start");
    } else if (type === "compositionend") {
      updateImeFeedback(view, event, "end");
    } else if (type === "compositionupdate") {
      updateImeFeedback(view, event, "advance");
    }
    if (event && typeof event.preventDefault === "function") event.preventDefault();
    if (event && typeof event.stopPropagation === "function") event.stopPropagation();
    return true;
  }

  function imeEventTextLength(event) {
    if (!event || typeof event.data !== "string") return null;
    return Array.from(event.data).length;
  }

  function updateImeFeedback(view, event, action) {
    const pluginState = getPluginState(view.state);
    if (!pluginState || pluginState.mode === MODES.INSERT || !view.dispatch) return false;

    if (action === "end") {
      view.dispatch(setVimMeta(view.state.tr, { imeFeedback: null }));
      return true;
    }

    const current = pluginState.imeFeedback || {
      anchor: normalCursorPos(view.state),
      offset: 0,
    };
    const textLength = imeEventTextLength(event);
    const offset =
      action === "start"
        ? 0
        : Math.max(1, textLength == null ? current.offset + 1 : textLength);
    view.dispatch(setVimMeta(view.state.tr, {
      imeFeedback: {
        anchor: current.anchor,
        offset,
      },
    }));
    return true;
  }

  function pushCursorDecoration(decorations, state, pos) {
    const cursor = normalCursorRange(state, pos);
    if (!cursor) return;

    if (cursor.empty) {
      decorations.push(
        PM.Decoration.widget(
          cursor.from,
          () => {
            const marker = document.createElement("span");
            marker.className = "vim-empty-line-cursor";
            return marker;
          },
          { side: 1 },
        ),
      );
      return;
    }

    decorations.push(
      PM.Decoration.inline(cursor.from, cursor.to, {
        class: "vim-cursor-char",
      }),
    );
  }

  function pushImeFeedbackDecoration(decorations, state, feedback) {
    if (!feedback) return;
    const anchor = clamp(feedback.anchor, 0, state.doc.content.size);
    const block = textblockInfo(state, anchor);
    if (!block.node.isTextblock) return;

    if (block.node.content.size === 0) {
      decorations.push(
        PM.Decoration.widget(
          block.start,
          () => {
            const marker = document.createElement("span");
            marker.className = "vim-ime-feedback-empty";
            return marker;
          },
          { side: 1 },
        ),
      );
      return;
    }

    const from = clamp(anchor, block.start, block.end - 1);
    const offset = Math.max(0, feedback.offset || 0);
    const to = clamp(anchor + offset + 1, from + 1, block.end);
    decorations.push(
      PM.Decoration.inline(from, to, {
        class: "vim-ime-feedback",
      }),
    );
  }

  function visualCursorDisplayPos(state, pluginState) {
    const anchor = pluginState.visualAnchor;
    const head = state.selection.head;
    if (anchor != null && head > anchor) {
      return charBeforePositionInBlock(state, head);
    }
    return head;
  }

  function visualSelectionRangeForCursor(state, anchor, cursorPos) {
    if (cursorPos < anchor) {
      return {
        anchor: positionAfterCursorTarget(state, anchor),
        head: cursorPos,
      };
    }

    return {
      anchor,
      head: positionAfterCursorTarget(state, cursorPos),
    };
  }

  function pushVisualRangeDecorations(decorations, state, pluginState) {
    if (state.selection.empty) return;

    if (pluginState.visualLine) {
      decorations.push(
        PM.Decoration.inline(state.selection.from, state.selection.to, {
          class: "vim-visual-range",
        }),
      );
      return;
    }

    const from = state.selection.from;
    const to = state.selection.to;
    const blocks = textblockList(state);
    if (!blocks.length) {
      decorations.push(
        PM.Decoration.inline(from, to, {
          class: "vim-visual-range",
        }),
      );
      return;
    }

    blocks.forEach((block) => {
      const rangeFrom = Math.max(from, block.start);
      const rangeTo = Math.min(to, block.end);
      if (rangeFrom >= rangeTo) return;
      decorations.push(
        PM.Decoration.inline(rangeFrom, rangeTo, {
          class: "vim-visual-range",
        }),
      );
    });
  }

  function visualPasteInsertPos(state, pluginState) {
    return moveHorizontalPos(state, visualCursorDisplayPos(state, pluginState), 1);
  }

  function lastInsertedTextCursorPos(doc, insertAt, text) {
    const insertedEnd = clamp(
      insertAt + String(text == null ? "" : text).length,
      0,
      doc.content.size,
    );
    return charBeforePositionInBlock({ doc }, insertedEnd);
  }

  function pasteInsertPos(state, before) {
    const emptyBlock = emptyTextblockAtCursor(state);
    if (emptyBlock) return emptyBlock.start;
    if (before) return state.selection.from;
    if (!state.selection.empty) return state.selection.to;
    const cursor = normalCursorRange(state);
    return moveHorizontalPos(state, cursor ? cursor.from : normalCursorPos(state), 1);
  }

  function decorationsFor(state) {
    const pluginState = getPluginState(state);
    if (!pluginState) return PM.DecorationSet.empty;

    const decorations = [];
    if (pluginState.mode === MODES.NORMAL && state.selection.empty) {
      pushCursorDecoration(decorations, state);
    }

    if (pluginState.mode !== MODES.INSERT) {
      pushImeFeedbackDecoration(decorations, state, pluginState.imeFeedback);
    }

    if (pluginState.mode === MODES.VISUAL) {
      pushVisualRangeDecorations(decorations, state, pluginState);
    }

    if (pluginState.mode === MODES.VISUAL) {
      pushCursorDecoration(decorations, state, visualCursorDisplayPos(state, pluginState));
    }

    if (pluginState.searchRange) {
      decorations.push(
        PM.Decoration.inline(pluginState.searchRange.from, pluginState.searchRange.to, {
          class: "vim-search-range",
        }),
      );
    }

    return decorations.length
      ? PM.DecorationSet.create(state.doc, decorations)
      : PM.DecorationSet.empty;
  }

  function createVimPlugin(options) {
    const pluginOptions = options || {};

    return [
      new PM.Plugin({
        key: vimPluginKey,
        state: {
          init() {
            return {
              mode: MODES.NORMAL,
              pending: null,
              count: "",
              register: "",
              registerType: REGISTER_TYPES.CHAR,
              visualAnchor: null,
              visualLine: false,
              searchQuery: "",
              searchDirection: 1,
              searchRange: null,
              lastRepeat: null,
              lastFind: null,
              insertSession: null,
              imeFeedback: null,
              message: "",
            };
          },
          apply(tr, value) {
            const meta = getMeta(tr);
            const next = {
              ...value,
              visualAnchor:
                value.visualAnchor == null ? null : tr.mapping.map(value.visualAnchor),
              searchRange: value.searchRange
                ? {
                    from: tr.mapping.map(value.searchRange.from),
                    to: tr.mapping.map(value.searchRange.to),
                  }
                : null,
              imeFeedback: value.imeFeedback
                ? {
                    anchor: tr.mapping.map(value.imeFeedback.anchor),
                    offset: value.imeFeedback.offset || 0,
                  }
                : null,
            };

            if (tr.docChanged && value.mode === MODES.INSERT && value.insertSession) {
              const text = insertedTextFromTransaction(tr);
              if (text) {
                next.insertSession = {
                  ...value.insertSession,
                  text: (value.insertSession.text || "") + text,
                };
              }
            }

            if (!meta) return next;
            if (Object.prototype.hasOwnProperty.call(meta, "mode")) next.mode = meta.mode;
            if (Object.prototype.hasOwnProperty.call(meta, "pending")) next.pending = meta.pending;
            if (Object.prototype.hasOwnProperty.call(meta, "count")) next.count = meta.count;
            if (Object.prototype.hasOwnProperty.call(meta, "register")) next.register = meta.register;
            if (Object.prototype.hasOwnProperty.call(meta, "registerType")) {
              next.registerType = meta.registerType;
            }
            if (Object.prototype.hasOwnProperty.call(meta, "visualAnchor")) {
              next.visualAnchor = meta.visualAnchor;
            }
            if (Object.prototype.hasOwnProperty.call(meta, "visualLine")) {
              next.visualLine = meta.visualLine;
            }
            if (Object.prototype.hasOwnProperty.call(meta, "searchQuery")) {
              next.searchQuery = meta.searchQuery;
            }
            if (Object.prototype.hasOwnProperty.call(meta, "searchDirection")) {
              next.searchDirection = meta.searchDirection;
            }
            if (Object.prototype.hasOwnProperty.call(meta, "searchRange")) {
              next.searchRange = meta.searchRange;
            }
            if (Object.prototype.hasOwnProperty.call(meta, "lastRepeat")) {
              next.lastRepeat = meta.lastRepeat;
            }
            if (Object.prototype.hasOwnProperty.call(meta, "lastFind")) {
              next.lastFind = meta.lastFind;
            }
            if (Object.prototype.hasOwnProperty.call(meta, "insertSession")) {
              next.insertSession = meta.insertSession;
            }
            if (Object.prototype.hasOwnProperty.call(meta, "imeFeedback")) {
              next.imeFeedback = meta.imeFeedback;
            }
            if (Object.prototype.hasOwnProperty.call(meta, "message")) {
              next.message = meta.message;
            }
            return next;
          },
        },
        props: {
          attributes(state) {
            const pluginState = getPluginState(state);
            return {
              "data-vim-mode": pluginState.mode,
              "data-vim-visual-line": pluginState.visualLine ? "true" : "false",
              class: "vim-enabled vim-mode-" + pluginState.mode,
            };
          },
          decorations: decorationsFor,
          handleTextInput(view) {
            return shouldBlockNativeTextInput(view);
          },
          handleDOMEvents: {
            beforeinput(view, event) {
              return blockNativeTextInput(view, event);
            },
            compositionstart(view, event) {
              return blockNativeTextInput(view, event);
            },
            compositionupdate(view, event) {
              return blockNativeTextInput(view, event);
            },
            compositionend(view, event) {
              return blockNativeTextInput(view, event);
            },
            textInput(view, event) {
              return blockNativeTextInput(view, event);
            },
          },
          handleKeyDown(view, event) {
            return handleKeyDown(view, event, pluginOptions);
          },
        },
        view(editorView) {
          if (pluginOptions.onStateChange) {
            pluginOptions.onStateChange(getPluginState(editorView.state));
          }
          return {
            update(view, prevState) {
              const previous = getPluginState(prevState);
              const current = getPluginState(view.state);
              if (
                pluginOptions.onStateChange &&
                JSON.stringify(previous) !== JSON.stringify(current)
              ) {
                pluginOptions.onStateChange(current);
              }
            },
          };
        },
      }),
    ];
  }

  window.createVimPlugin = createVimPlugin;
  window.vimPluginKey = vimPluginKey;
  window.vimPluginVersion = VIM_PLUGIN_VERSION;
})();
