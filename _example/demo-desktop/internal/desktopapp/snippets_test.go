package desktopapp

import (
	"strings"
	"testing"
)

func TestCollectMemoCodeSnippetsRecognizesMarkers(t *testing.T) {
	memo := MemoRecord{
		Content: strings.Join([]string{
			"# Snippet Memo",
			"",
			"snippet: Build app | build ship",
			"```sh",
			"npm run build",
			"```",
			"",
			"```go snippet Format code | fmt",
			"gofmt -w .",
			"```",
			"",
			"```js",
			"// snippet Copy value | copy-val",
			"navigator.clipboard.writeText(value)",
			"```",
			"",
			"```sql",
			"select * from memos;",
			"```",
		}, "\n"),
		CreatedAt:  "2026-06-10T01:00:00Z",
		ID:         "memo_snippets",
		Path:       "memos/2026/06/memo_snippets.md",
		Visibility: "PRIVATE",
	}

	items := collectMemoCodeSnippets(memo)
	if len(items) != 4 {
		t.Fatalf("snippet count = %d, want 4", len(items))
	}

	if !items[0].Marked || items[0].Title != "Build app" || items[0].Command != "build" || items[0].Language != "sh" {
		t.Fatalf("first snippet = %#v, want marked sh build snippet", items[0])
	}
	if items[0].Code != "npm run build" {
		t.Fatalf("first code = %q, want npm run build", items[0].Code)
	}

	if !items[1].Marked || items[1].Title != "Format code" || items[1].Command != "fmt" || items[1].Language != "go" {
		t.Fatalf("second snippet = %#v, want marked go fmt snippet", items[1])
	}

	if !items[2].Marked || items[2].Title != "Copy value" || items[2].Command != "copy-val" {
		t.Fatalf("third snippet = %#v, want comment-marked copy-val snippet", items[2])
	}
	if strings.Contains(items[2].Code, "snippet Copy value") {
		t.Fatalf("comment marker should be stripped from code: %q", items[2].Code)
	}

	if items[3].Marked || items[3].Language != "sql" {
		t.Fatalf("fourth snippet = %#v, want unmarked sql code block", items[3])
	}
}

func TestSearchVaultSnippetsSupportsSnippetDirective(t *testing.T) {
	ctx, existing, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	if existing {
		t.Fatalf("new temp vault should not be existing")
	}

	if _, err := createVaultMemo(ctx, MemoCreateRequest{
		Content: strings.Join([]string{
			"# Tools",
			"```sh snippet Deploy app | deploy",
			"rsync -av dist/ server:/app",
			"```",
			"",
			"```sql",
			"select * from memo_log;",
			"```",
		}, "\n"),
		Visibility: "PRIVATE",
	}); err != nil {
		t.Fatalf("create memo: %v", err)
	}

	items, err := searchVaultSnippets(ctx, "snippet deploy", 10)
	if err != nil {
		t.Fatalf("search snippets: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("snippet deploy result count = %d, want 1: %#v", len(items), items)
	}
	if !items[0].Marked || items[0].Command != "deploy" {
		t.Fatalf("snippet deploy result = %#v, want marked deploy snippet", items[0])
	}

	items, err = searchVaultSnippets(ctx, "snippet memo_log", 10)
	if err != nil {
		t.Fatalf("search unmarked code block: %v", err)
	}
	if len(items) != 1 || items[0].Marked {
		t.Fatalf("snippet memo_log results = %#v, want unmarked code block", items)
	}

	items, err = searchVaultSnippets(ctx, "memo_log", 10)
	if err != nil {
		t.Fatalf("plain search: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("plain search results = %#v, want no results until snippet command is explicit", items)
	}

	items, err = searchVaultSnippets(ctx, "", 10)
	if err != nil {
		t.Fatalf("empty search: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("empty search results = %#v, want no results", items)
	}

	items, err = searchVaultSnippets(ctx, "snippet", 10)
	if err != nil {
		t.Fatalf("snippet directive search: %v", err)
	}
	if len(items) != 2 || !items[0].Marked || items[1].Marked {
		t.Fatalf("snippet directive results = %#v, want marked snippet first then ordinary code blocks", items)
	}

	items, err = searchVaultSnippets(ctx, "snippet   ", 10)
	if err != nil {
		t.Fatalf("snippet directive search with spaces: %v", err)
	}
	if len(items) != 2 || !items[0].Marked || items[1].Marked {
		t.Fatalf("snippet directive with spaces results = %#v, want spaces ignored", items)
	}
}
