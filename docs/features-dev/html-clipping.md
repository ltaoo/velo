# HTML Clipping Feature Design

## Background

The desktop demo already supports memo capture from clipboard text, links, and images. HTML clipboard content is currently treated as plain text, so rich snippets copied from a browser lose their original structure and presentation.

The memo system stores user notes as Markdown files with front matter. Memo creation and update paths parse content for tags, memo references, task lines, and asset references. Because of that, storing raw HTML directly inside memo content would interfere with existing Markdown parsing and would also introduce unsafe third-party markup into the main memo DOM.

The recommended design is to store clipped HTML as a separate vault resource and let memos reference it.

## Goals

- Capture a selected HTML fragment from the system clipboard.
- Persist the fragment in the active vault.
- Create a memo that contains source metadata, searchable summary text, and a reference to the HTML clip.
- Render the clipped HTML in the app with Obsidian-like embedded preview behavior.
- Keep third-party HTML isolated from the main memo DOM.

## Non-Goals

- Full web page archiving in the first phase.
- Browser extension implementation in the first phase.
- Perfect offline reproduction of all remote assets in the first phase.
- Executing scripts from clipped HTML.

## Existing Code Touchpoints

- Clipboard reading lives in `_example/demo-desktop/internal/desktopapp/clipboard.go`.
- Memo creation and persistence live in `_example/demo-desktop/internal/desktopapp/memo.go` and `_example/demo-desktop/internal/desktopapp/memo_markdown.go`.
- Vault directory setup lives in `_example/demo-desktop/internal/desktopapp/vault_project.go`.
- API route registration lives in `_example/demo-desktop/internal/desktopapp/api_routes.go`.
- Clipboard UI and accept flow live in `_example/demo-desktop/frontend/src/pages/home/memos.js`.
- Markdown rendering lives in `_example/demo-desktop/frontend/src/pages/home/memo-markdown.js`.
- Memo resource indexing lives in `_example/demo-desktop/frontend/src/domain/memo-resources.js`.

## Storage Model

Add a new vault directory:

```text
clips/
  2026/
    06/
      clip_20260612T153045_ab12cd34/
        index.html
        meta.json
        assets/
```

Extend `VaultContext` with:

```go
ClipDir string `json:"clipDir"`
```

Add constants:

```go
const vaultClipDirName = "clips"
```

Create the directory in `openVaultDirectory` alongside `memo` and `memo-comments`.

## Data Model

Create `_example/demo-desktop/internal/desktopapp/clip.go`.

```go
type ClipRecord struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	SourceURL  string `json:"sourceUrl,omitempty"`
	SourceHost string `json:"sourceHost,omitempty"`
	Excerpt    string `json:"excerpt"`
	Text       string `json:"text,omitempty"`
	HTMLPath   string `json:"htmlPath"`
	MemoID     string `json:"memoId,omitempty"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
	Size       int    `json:"size"`
}

type ClipCreateRequest struct {
	HTML       string `json:"html"`
	HTMLBase64 string `json:"htmlBase64,omitempty"`
	Title      string `json:"title,omitempty"`
	SourceURL  string `json:"sourceUrl,omitempty"`
	Text       string `json:"text,omitempty"`
	ProjectID  string `json:"projectId,omitempty"`
	Visibility string `json:"visibility,omitempty"`
}
```

`HTMLBase64` is optional. It is useful if clipboard HTML contains encoding-sensitive content or if request serialization starts producing escaping issues.

## Memo Reference Syntax

Use Obsidian-style embed syntax with a typed target:

```md
![[clip:clip_20260612T153045_ab12cd34|Article title]]
```

The memo created for a clip should contain human-readable text for search and context:

```md
#clipped Article title

[来源](https://example.com/path)

![[clip:clip_20260612T153045_ab12cd34|Article title]]

> Short excerpt from the clipped content.
```

This keeps memo search useful while preserving the HTML separately.

## Clipboard Changes

Current behavior reads `public.html` and returns it as text. Change this to preserve HTML:

- `public.html` should return `type: "html"`.
- `content` should be a plain-text excerpt.
- Add `html` or `htmlBase64` to `ClipboardSnapshot`.
- Preserve `rawType` as `public.html`.
- If source URL metadata is available later, include it as `sourceUrl`.

Suggested additions:

```go
type ClipboardSnapshot struct {
	// existing fields...
	HTML       string `json:"html,omitempty"`
	HTMLBase64 string `json:"htmlBase64,omitempty"`
	SourceURL  string `json:"sourceUrl,omitempty"`
}
```

Read order should remain:

1. HTML
2. Plain text
3. Image

This preserves rich content when the clipboard has multiple formats for the same copy operation.

## Backend API

Add `_example/demo-desktop/internal/desktopapp/routes_clips.go`.

Routes:

```text
POST /api/clips/create
GET  /api/clips/read?id=clip_xxx
GET  /api/clips/render?id=clip_xxx
```

`POST /api/clips/create` should:

1. Require an active vault.
2. Decode and normalize HTML.
3. Enforce size limits.
4. Sanitize or wrap HTML.
5. Write `index.html`.
6. Write `meta.json`.
7. Create a memo that references the clip.
8. Return `{ clip, memo }`.

`GET /api/clips/read` should return metadata only.

`GET /api/clips/render` should serve the clipped HTML with defensive headers:

```text
Content-Type: text/html; charset=utf-8
Content-Security-Policy: default-src 'none'; img-src https: http: data:; style-src 'unsafe-inline'; font-src https: data:;
X-Content-Type-Options: nosniff
```

Do not serve arbitrary relative paths from the clip directory. Resolve by clip ID, validate the ID, locate the known `index.html`, and reject path traversal.

Register routes in `_example/demo-desktop/internal/desktopapp/api_routes.go`:

```go
registerClipRoutes(b)
```

## HTML Processing

The first implementation can use a conservative wrapper:

```html
<!doctype html>
<html>
  <head>
    <meta charset="utf-8">
    <base target="_blank">
    <style>
      body {
        margin: 0;
        font: 14px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      }
      img, video {
        max-width: 100%;
        height: auto;
      }
    </style>
  </head>
  <body>
    <!-- sanitized clip HTML -->
  </body>
</html>
```

Sanitization policy:

- Remove `script`, `iframe`, `object`, `embed`, `form`, `input`, `button`, `textarea`, `select`.
- Remove event attributes such as `onclick`.
- Remove `javascript:` and `data:` URLs except `data:image/*`.
- Keep basic layout tags, tables, images, links, and inline styles.
- Add `rel="noreferrer noopener"` to links.

Prefer a well-tested sanitizer dependency if dependency policy allows it. If adding a dependency is not desirable, implement a minimal sanitizer for MVP and keep iframe sandbox plus CSP as the primary safety boundary.

## Frontend Rendering

Update `_example/demo-desktop/frontend/src/pages/home/memo-markdown.js`:

- Detect standalone `![[clip:id|title]]`.
- Render a clip embed card before falling back to normal memo reference rendering.
- Use a sandboxed iframe:

```html
<iframe
  class="memo-clip-frame"
  sandbox=""
  src="/api/clips/render?id=clip_xxx"
></iframe>
```

Recommended card controls:

- Open original source URL if present.
- Copy clip reference.
- Expand or collapse preview.
- Open clip in detached window later.

Do not inject clipped HTML with `innerHTML` in the main document.

## Clipboard UI Flow

Update `_example/demo-desktop/frontend/src/pages/home/memos.js`:

- `normalizeClipboardItem` should preserve `html`, `htmlBase64`, and `sourceUrl`.
- `clipboardTypeLabel("html")` should return `HTML 剪藏`.
- `clipboardActionLabel("html")` should return `保存剪藏`.
- `acceptClipboardItem` should call a new `createClipFromClipboard` branch.

Suggested flow:

```js
if (item.type === "html") {
  task = createClipFromClipboard(item);
} else if (item.type === "image") {
  task = uploadClipboardImage(item);
} else if (item.type === "link") {
  task = createMemoFromContent(item.content, "链接已保存");
} else {
  task = createMemoFromContent(item.content, "已创建 memo");
}
```

Add a frontend domain helper, for example `_example/demo-desktop/frontend/src/domain/clips.js`:

```js
export function createClipInVault(payload) {
  return globalThis.invoke("/api/clips/create", {
    method: "POST",
    args: payload,
  }).then(function (resp) {
    if (!resp || resp.code !== 0 || !resp.data || !resp.data.memo) {
      throw new Error((resp && resp.msg) || "保存剪藏失败");
    }
    return resp.data;
  });
}
```

## Search and Resource Views

For MVP, the created memo contains title, source URL, and excerpt, so existing memo search can find clips without changing search indexing.

Later improvements:

- Add `collectClips(memos)` in `memo-resources.js`.
- Add a Clips navigation view.
- Read `meta.json` for richer list cards.
- Include extracted plain text in a search index.

## Cleanup Behavior

When deleting a memo that owns a clip, the clip should be deleted only if no other memo references it.

Add a clip reference extractor similar to asset reference extraction:

```go
var memoClipReferencePattern = regexp.MustCompile(`!?\[\[clip:([A-Za-z0-9_-]+)(?:[|#][^\]]*)?\]\]`)
```

In `deleteVaultMemoWithOptions`, collect clip references and remove unshared clip directories after memo deletion. This can be implemented after the MVP if deletion semantics need to stay low-risk initially.

## Tests

Backend tests:

- Creating a clip writes `index.html` and `meta.json`.
- Creating a clip creates a memo with `![[clip:...]]`.
- Invalid clip IDs are rejected.
- Render route cannot path-traverse.
- HTML sanitizer removes scripts and event handlers.
- Deleting a memo does not delete clips still referenced elsewhere.

Frontend tests or manual checks:

- Copy rich HTML from a browser and save it.
- The memo list shows an embedded preview.
- Links in the preview do not navigate the app frame.
- Script tags do not execute.
- Plain text, link, and image clipboard flows still work.

## Phased Implementation

Phase 1:

- Add clip vault directory.
- Preserve `public.html` as clipboard type `html`.
- Add `/api/clips/create` and `/api/clips/render`.
- Create memo with clip reference.
- Render clip reference as sandboxed iframe.

Phase 2:

- Add clip metadata cards and a Clips view.
- Add clip reference cleanup.
- Improve source URL/title extraction.

Phase 3:

- Download remote images and CSS into `clips/.../assets`.
- Rewrite asset URLs to local files.
- Add a detached clip preview window.

Phase 4:

- Add browser extension or share target.
- Capture selected DOM, page title, page URL, and simplified readable text directly.
