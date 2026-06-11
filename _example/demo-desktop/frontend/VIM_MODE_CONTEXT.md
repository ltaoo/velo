# Vim Mode Context

## 2026-06-09: `yy` then `p` inserts at current line start

- Symptom: in the ProseMirror memo editor, normal-mode `yy` followed by `p` inserted the yanked line text at the current cursor position, which often looked like insertion at the current line start. Vim linewise paste should insert the copied line as a new line after the current line.
- Root cause: `vim.js` stored only a plain string in `pluginState.register`. `pasteRegister` always called `insertText` at the selection position, so linewise yanks from `yy` were handled like characterwise yanks.
- Fix: added `registerType` with `char` and `line`. `yy` and linewise deletes store `line`; characterwise operations store `char`. `p`/`P` now insert paragraph nodes after/before the current textblock when the register is linewise.
- Verification: loaded `prosemirror.umd.min.js` and `vim.js` in a browser-like Node VM and simulated `yy` then `p`; a two-paragraph doc `one / two` became `one / one / two`. Also checked `2p` and empty-line `yy p`.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260609-vim-linewise-paste`.
- Follow-up watch point: visual-line paste is now tagged linewise, but visual-mode replacement still uses the older characterwise replace path.

## 2026-06-09: `v`, `e`, `y` leaves cursor after selected word

- Symptom: entering visual mode with `v`, extending to the word end with `e`, then yanking with `y` left the normal-mode cursor at the selected word's right boundary plus one character. Expected behavior for this editor: return to the position where visual mode started.
- Root cause: `visualCommand` copied the range and then called `leaveVisual(view)`. The default `leaveVisual` implementation collapsed the selection to `state.selection.to`, which is the expanded head after `e`.
- Fix: `leaveVisual` now accepts an optional cursor position, and visual yank passes `pluginState.visualAnchor`.
- Verification: loaded `prosemirror.umd.min.js` and `vim.js` in a browser-like Node VM and simulated `v`, `e`, `y` on `hello world`; the register became `hello` and the final selection returned to the initial position.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260609-vim-visual-yank-cursor`.

## 2026-06-09: visual `p` inserts before the current cursor

- Symptom: after selecting content with `v`, pressing `p` pasted before the current visual cursor. Expected behavior for this editor: paste after the current visual cursor.
- Root cause: visual-mode `p` deleted the selected range and then inserted the register at `state.selection.from`, which is the start of the selected range.
- Fix: visual-mode `p` now computes the displayed visual cursor position and inserts at the next document position without deleting the selection.
- Verification: loaded `prosemirror.umd.min.js` and `vim.js` in a browser-like Node VM, set the register to `X`, and simulated `v`, `e`, `p` on `foo bar`; the document became `fooX bar`.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260609-vim-visual-paste-after-cursor`.

## 2026-06-09: normal `p` inserts before the current character

- Symptom: with text `123456`, normal cursor on `3`, and register `abc`, pressing `p` produced `12abc3456`. Expected behavior: `123abc456`.
- Root cause: charwise `pasteRegister` used `state.selection.to` as the `p` insertion point. In this editor's normal mode, the collapsed selection marks the current character position, so `selection.to` is still before that character.
- Fix: added `pasteInsertPos`; charwise normal `p` now inserts after `normalCursorPos(state)`, while `P` still inserts at the current position.
- Verification: loaded `prosemirror.umd.min.js` and `vim.js` in a browser-like Node VM, set the cursor on `3` in `123456`, set register `abc`, and pressed `p`; the document became `123abc456`.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260609-vim-char-paste-after-cursor`.

## 2026-06-09: normal `p` leaves cursor at the original character

- Symptom: after fixing insertion position, `123456` with cursor on `3` and register `abc` became `123abc456`, but the normal cursor stayed on `3`. Expected behavior: move to the last inserted character, `c`.
- Root cause: charwise `pasteRegister` inserted text without setting the transaction selection, so the editor's default mapping kept the cursor near its original position.
- Fix: charwise paste now sets selection to `insertAt + pastedText.length - 1`.
- Verification: loaded `prosemirror.umd.min.js` and `vim.js` in a browser-like Node VM, set the cursor on `3` in `123456`, set register `abc`, and pressed `p`; the document became `123abc456`, selection moved to position `6`, and that position reads as `c`.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260609-vim-char-paste-cursor`.

## 2026-06-09: IME composition inserts text in normal mode

- Symptom: when using a Chinese IME in normal mode, intermediate composition text was inserted at the cursor position. Expected behavior: composition input is invalid outside insert mode and should not change the document.
- Root cause: vim normal mode only handled keydown. IME composition can enter ProseMirror through native `beforeinput`, `composition*`, or text input paths even when no mapped vim key is produced.
- Fix: added native text input blocking for non-insert modes through `handleTextInput` and `handleDOMEvents` for `beforeinput`, `compositionstart`, `compositionupdate`, `compositionend`, and `textInput`.
- Verification: loaded `prosemirror.umd.min.js` and `vim.js` in a browser-like Node VM; in normal mode `handleTextInput` and `beforeinput` returned `true` and called `preventDefault`, while insert mode returned `false`.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260609-vim-normal-ime-block`.

## 2026-06-09: IME composition keydown triggers vim commands

- Symptom: during Chinese IME composition in normal mode, composing keydown events whose key matched vim commands such as `h` or `j` still moved the cursor or triggered vim behavior. Expected behavior: while composing, no vim key command should run outside insert mode.
- Root cause: the previous IME fix blocked native text input events, but `handleKeyDown` still parsed composing keydown events through `keyName` and dispatched normal/visual commands.
- Fix: `handleKeyDown` now checks `event.isComposing`, `key === "Process"`, `keyCode === 229`, and `which === 229` before key mapping; in non-insert modes those events are prevented and swallowed.
- Verification: loaded `prosemirror.umd.min.js` and `vim.js` in a browser-like Node VM; a normal-mode composing `h` keydown returned `true`, called `preventDefault`, and left the cursor unchanged. A non-composing `h` still moved left, and insert-mode composing keydown returned `false`.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260609-vim-composing-key-block`.

## 2026-06-09: normal-mode IME composition has no visible feedback

- Symptom: after blocking IME input and vim key dispatch in normal mode, composing text produced no visible feedback. Expected behavior: like VS Code vim mode, show a white underline that advances as composition input changes, while still keeping the document unchanged.
- Fix: added transient `imeFeedback` plugin state with an anchor and offset. Non-insert composition events and composing keydowns update the offset; `compositionend` clears it. Decorations render a white underline at the feedback position without inserting text.
- Styling: added `.vim-ime-feedback` and `.vim-ime-feedback-empty` in `index.css`.
- Verification: loaded `prosemirror.umd.min.js` and `vim.js` in a browser-like Node VM; normal-mode `compositionstart` set offset `0`, composing `h` keydown advanced to `1`, `compositionupdate` with `hj` advanced to `2`, and `compositionend` cleared feedback while the document stayed unchanged.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260609-vim-ime-feedback`.

## 2026-06-09: IME feedback underline should grow continuously

- Symptom: the normal-mode IME feedback underline moved one character at a time. Expected behavior: a continuous white underline that grows forward as composition input advances.
- Fix: changed the IME feedback decoration from a single-character inline range to an anchored range from `anchor` to `anchor + offset + 1`.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260609-vim-ime-feedback-range`.

## 2026-06-09: IME feedback underline remains after composition ends

- Symptom: the white underline was still visible after IME composition ended.
- Root cause: `compositionend` cleared `imeFeedback`, but a following native `beforeinput`/`textInput` event could be blocked and treated as another feedback advance, recreating the underline.
- Fix: `beforeinput` and `textInput` now only block native input in non-insert modes; they no longer advance IME feedback. Feedback advances only on `compositionupdate` and composing keydown, and `compositionend` clears it.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260609-vim-ime-feedback-clear`.

## 2026-06-09: empty-line IME feedback adds visual blank space

- Symptom: when the normal cursor was on an empty line, IME composition made a blank area appear below the line without actually inserting a new paragraph.
- Root cause: the empty-line IME feedback widget used `display: inline-block` with `height: 1em`, so it participated in layout inside an otherwise empty paragraph.
- Fix: changed `.vim-ime-feedback-empty` to a zero-size inline overlay and moved the visible white underline into an absolutely positioned `::after` pseudo-element.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260609-vim-ime-empty-line-overlay`.

## 2026-06-09: Ex `:w` writes vault drafts

- Change: Ex `:w` now calls the editor's draft-write callback, while `:wq`/`:x` call commit and `:q`/`:q!` call quit/discard callbacks. The editor callback layer now returns async results so Vim messages can reflect draft, commit, and quit outcomes.
- Cache note: updated `vim.js` script query keys to `20260609-vim-ex-drafts` and `prosemirror-editor.umd.js` query keys to `20260609-editor-draft-events`.

## 2026-06-10: normal `~` toggles character case

- Symptom: `shift+~`/`~` was not mapped in normal mode. Expected standard Vim behavior: toggle the case of the character under the cursor.
- Fix: added `toggleCaseAtCursor` for normal mode. It supports counts such as `3~`, advances the cursor after the toggled range, and is repeatable with `.`.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260610-vim-toggle-case`.

## 2026-06-10: first `l` in visual mode does not extend selection

- Symptom: after pressing `v`, the first `l` appeared to do nothing; only the second `l` moved the visual selection to the right.
- Root cause: char visual mode entered with `anchor === head`, so the initial selection was empty. The first `l` only selected the current character instead of extending past it.
- Fix: char visual mode now normalizes the current cursor position and initializes `head` to the position after the current cursor target. This makes the current character selected immediately on `v`, so the first `l` extends one character to the right.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260610-vim-visual-initial-head`.

## 2026-06-10: visual motions used selection boundaries as Vim cursor positions

- Symptom: after `v`, pressing `h` had the same first-key no-op behavior, and the original cursor character could fall out of the selected range. Other visual motions had the same risk because they used ProseMirror's half-open `selection.head` as if it were the Vim cursor character.
- Root cause: char visual selections need inclusive Vim semantics, but ProseMirror ranges are half-open. When selecting forward, `selection.head` is after the visible cursor character; when selecting backward, the range must end after the original anchor character.
- Fix: added a single visual cursor/range conversion path. Visual motions now compute from `visualCursorDisplayPos()` and convert the target character back through `visualSelectionRangeForCursor()`, so `h/l/j/k/w/b/e/0/^/$/G`, arrow keys, and visual `f`/`;` keep both the anchor character and target character selected. Visual `o` was updated to swap Vim endpoints rather than raw ProseMirror boundaries.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260610-vim-visual-motion-inclusive`.

## 2026-06-10: `Esc` from visual mode lands after the final character

- Symptom: with `abc`, starting visual mode on `a`, pressing `l`, then `Esc` left the normal cursor on `c`. Expected behavior: the cursor should stay on the final visual cursor character, `b`.
- Root cause: `leaveVisual(view)` defaulted to `state.selection.to`. In a forward char visual selection, ProseMirror's `to` is the position after the final selected character.
- Fix: when visual mode exits without an explicit cursor override, `leaveVisual` now uses `visualCursorDisplayPos()`. Explicit exits such as visual yank still pass their own cursor target and keep their existing behavior.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260610-vim-visual-esc-cursor`.

## 2026-06-10: cross-line char visual selection paints to the row edge

- Symptom: with two paragraphs `abc` and `123`, starting char visual on `a` and pressing `j` visually painted the whole first row to the container edge. Native Vim char visual semantics should select `abc`, the line break, and `1`, not a full linewise row.
- Root cause: the actual ProseMirror selection was already `abc\n1`, but the browser native cross-paragraph selection background and a single cross-block visual decoration made the row-edge gap look selected.
- Fix: char visual range rendering is now split per textblock, so only real text content receives `.vim-visual-range`. The editor also exposes `data-vim-visual-line`, and CSS hides the browser native `::selection` background only for char visual mode; linewise `V` remains unaffected.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html`, and `index.css` query keys in `index.html`, `memo-window.html`, and `memo-slim.html`, to `20260610-vim-visual-cross-line-render`.

## 2026-06-10: paste cursor can miss the last inserted character

- Symptom: after `p`, the normal cursor could land before the final inserted character instead of on it.
- Root cause: paste cursor placement was implemented differently in normal and visual paste paths. Normal paste used a hand-rolled `length - 1` offset, while visual paste used `insertAt + register.length`, which is the position after the inserted text rather than the final inserted character.
- Fix: added `lastInsertedTextCursorPos`, which computes the cursor from the post-insert document by stepping back from the inserted range end to the actual character position. Both normal charwise paste and visual paste now use this shared helper.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260610-vim-paste-last-cursor`.

## 2026-06-10: `p` on an empty line lands on the penultimate pasted character

- Symptom: when the normal cursor was on an empty line at line start, `p` pasted text but left the cursor on the penultimate character.
- Root cause: `pasteInsertPos` always moved one position to the right for `p`. On an empty paragraph there is no current character, so that moved the insertion point outside the empty textblock. The pasted text was inserted after the empty paragraph, and cursor recovery from the inserted end resolved against the wrong block boundary.
- Fix: if `normalCursorRange` is empty, charwise `p` now inserts at the empty textblock start, same as `P`, and still uses `lastInsertedTextCursorPos` to land on the final pasted character.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260610-vim-empty-line-paste-cursor`.

## 2026-06-10: empty-line paste still fails when selection is on the paragraph boundary

- Symptom: after yanking `abc` with `v e y`, moving to a new empty line and pressing `p` could still create a blank line below and leave the cursor on `b`.
- Root cause: the previous empty-line paste fix only handled the exact `emptyBlock.start` cursor position. In the real editor, an empty paragraph cursor can resolve to the paragraph boundary (`before`/`after`) after creating or moving to a new line, so `pasteInsertPos` still treated it as a normal non-empty cursor and inserted outside the empty paragraph.
- Fix: added `emptyTextblockAtCursor`, which recognizes an empty textblock across its full `before..after` boundary span. Charwise `p` and `P` now normalize any such cursor position to the empty textblock start before inserting.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260610-vim-empty-line-paste-boundary`.

## 2026-06-11: visual `gg` does not extend selection to the top

- Symptom: after entering char visual mode with `v`, pressing `gg` did not select from the original cursor position up to the document top. Visual motions should select the content traversed by the motion, same as the already fixed `h/j/k/l/e/G` paths.
- Root cause: normal mode had a `g` pending state for `gg`, but visual mode only handled single-key motions. The first `g` in visual mode was unmapped instead of waiting for the second `g`.
- Fix: visual mode now supports a `g` pending state. `v gg` moves the visual cursor to `docStart(state)` and reuses `visualSelectionRangeForCursor`, so the selection keeps Vim's inclusive visual semantics while ProseMirror still receives a half-open range.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260611-vim-visual-gg-motion`.

## 2026-06-11: visual selection is too faint in dark mode

- Symptom: char visual selection was hard to see in dark mode because `.vim-visual-range` used a low-alpha green background that blended into the editor surface.
- Fix: added dedicated `--vim-visual-*` theme variables. Dark mode now uses a brighter blue selection fill, a subtle inset ring, and high-contrast selected text; light mode keeps a softer blue fill.
- Cache note: updated `index.css` query keys in `index.html`, `memo-window.html`, and `memo-slim.html` to `20260611-vim-visual-dark-selection`.

## 2026-06-11: support `Ctrl-d` and `Ctrl-u` multi-line motions

- Symptom: `Ctrl-d` and `Ctrl-u` were ignored because `keyName` filtered unmapped Ctrl combinations before they could reach Vim command handling.
- Fix: mapped `Ctrl-d` and `Ctrl-u` in `keyName`, added them to `motionTarget`, and shared motion-key detection between normal and visual mode. They now move down/up by `LINE_JUMP_COUNT` lines and preserve the current column through the existing vertical motion helper.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260611-vim-ctrl-du-motion`.

## 2026-06-11: char visual selection is invisible on empty lines

- Symptom: with char visual mode, moving vertically onto an empty line selected the newline semantically, but there was no visible highlight because empty paragraphs have no inline content for `.vim-visual-range` to decorate.
- Fix: char visual rendering now adds a `vim-visual-empty-line` widget when a selected range covers an empty textblock. The widget reuses the visual selection theme variables and acts as a visible newline placeholder without changing document content.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html`, and `index.css` query keys in `index.html`, `memo-window.html`, and `memo-slim.html`, to `20260611-vim-visual-empty-line`.

## 2026-06-11: empty-line visual marker creates an extra blank row

- Symptom: after selecting an empty line in char visual mode, an extra blank row without a line number appeared between lines.
- Root cause: the empty-line visual marker was an `inline-block`, so inside ProseMirror's empty paragraph it could participate in line layout and combine with the editor's empty-block placeholder behavior.
- Fix: changed `.vim-visual-empty-line` into a zero-size inline marker and moved the visible highlight into an absolutely positioned `::after`, matching the existing empty-line cursor pattern. The marker no longer contributes layout height.
- Cache note: updated `index.css` query keys in `index.html`, `memo-window.html`, and `memo-slim.html` to `20260611-vim-visual-empty-line-layout`.

## 2026-06-11: `v gg Esc` leaves native browser selection visible

- Symptom: after entering visual mode, pressing `gg`, then `Esc`, the editor left a native browser selection highlight over the content below the first line.
- Root cause: ProseMirror state was correctly collapsed to the top cursor position, but the browser DOM selection could remain as the previous backward visual range after the keydown cycle. Once Vim mode returned to normal, the native `::selection` color became visible.
- Fix: visual exit now explicitly focuses/synchronizes the editor DOM selection to the collapsed ProseMirror cursor, then repeats the sync on a microtask to cover browser selection flush timing.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260611-vim-visual-exit-dom-selection`.

## 2026-06-11: `v gg Esc` native selection still visible after microtask sync

- Symptom: the browser-native blue selection could still remain after `v gg Esc`, even though ProseMirror state was collapsed and a microtask DOM selection sync had run.
- Root cause: in the desktop WebView, the native DOM selection may be restored later than a microtask after the keydown path. Also, once Vim returns to normal mode, any residual native selection becomes visible unless normal mode suppresses editor `::selection`.
- Fix: visual exit now also flushes DOM selection on the next animation frame or a 16ms timeout fallback. Normal mode now hides native `::selection` inside the editor, matching Vim behavior where selection feedback should come from Vim decorations rather than browser selection paint.
- Cache note: updated `vim.js` script query keys in `index.html` and `memo-window.html` to `20260611-vim-visual-exit-selection-flush`, and `index.css` query keys in `index.html`, `memo-window.html`, and `memo-slim.html` to `20260611-vim-normal-native-selection-hide`.
