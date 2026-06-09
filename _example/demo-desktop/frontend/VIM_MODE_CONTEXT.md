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
