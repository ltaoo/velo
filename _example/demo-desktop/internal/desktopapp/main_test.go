package desktopapp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"example/simple/internal/desktopapp/external"
)

func TestNormalizeExternalBrowserURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "http", value: "http://example.com/a?b=c", want: "http://example.com/a?b=c"},
		{name: "https uppercase scheme", value: "HTTPS://example.com/a%20b", want: "https://example.com/a%20b"},
		{name: "empty", value: "", wantErr: true},
		{name: "relative", value: "/docs", wantErr: true},
		{name: "missing host", value: "https:///docs", wantErr: true},
		{name: "javascript", value: "javascript:alert(1)", wantErr: true},
		{name: "mailto", value: "mailto:user@example.com", wantErr: true},
		{name: "raw space", value: "https://example.com/a b", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := external.NormalizeBrowserURL(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMemoWindowNameUsesMemoID(t *testing.T) {
	if got, want := memoWindowName("Memo:ABC/123"), "memo-window-memo-abc-123"; got != want {
		t.Fatalf("memoWindowName = %q, want %q", got, want)
	}
}

func stringSliceContains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestCreateVaultMemoWritesMarkdownFile(t *testing.T) {
	ctx, existing, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	if existing {
		t.Fatalf("new temp vault should not be existing")
	}

	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "#idea hello [[memo:abc|source]]",
		Visibility: "PUBLIC",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	if memo.ID == "" {
		t.Fatalf("memo id is empty")
	}
	if len(memo.Tags) != 1 || memo.Tags[0] != "idea" {
		t.Fatalf("tags = %#v, want idea", memo.Tags)
	}
	if len(memo.References) != 1 || memo.References[0] != "memo:abc" {
		t.Fatalf("references = %#v, want memo:abc", memo.References)
	}

	path := filepath.Join(ctx.RootDir, filepath.FromSlash(memo.Path))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memo file: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"---\n",
		"id: \"" + memo.ID + "\"",
		"visibility: \"PUBLIC\"",
		"tags:\n  - \"idea\"",
		"references:\n  - \"memo:abc\"",
		"#idea hello [[memo:abc|source]]",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("memo file missing %q:\n%s", want, text)
		}
	}

	listed, err := listVaultMemos(ctx)
	if err != nil {
		t.Fatalf("list memos: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != memo.ID {
		t.Fatalf("listed memos = %#v, want created memo", listed)
	}
}

func TestCreateVaultMemoIgnoresCodeWhenExtractingMetadata(t *testing.T) {
	ctx, existing, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	if existing {
		t.Fatalf("new temp vault should not be existing")
	}

	content := strings.Join([]string{
		"#real outside [[memo:real]]",
		"`#inline [[memo:inline]]`",
		"````markdown",
		"```",
		"#code [[memo:code]]",
		"- [ ] code task #todo",
		"```",
		"````",
		"- [ ] outside task #work",
	}, "\n")
	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    content,
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	if strings.Contains(strings.Join(memo.Tags, ","), "inline") || strings.Contains(strings.Join(memo.Tags, ","), "code") {
		t.Fatalf("tags = %#v, want inline/code ignored", memo.Tags)
	}
	if !stringSliceContains(memo.Tags, "real") || !stringSliceContains(memo.Tags, "work") {
		t.Fatalf("tags = %#v, want real and work", memo.Tags)
	}
	if !stringSliceContains(memo.References, "memo:real") {
		t.Fatalf("references = %#v, want memo:real", memo.References)
	}
	if stringSliceContains(memo.References, "memo:inline") || stringSliceContains(memo.References, "memo:code") {
		t.Fatalf("references = %#v, want inline/code references ignored", memo.References)
	}
	if strings.Contains(memo.Content, "code task [[task:") {
		t.Fatalf("memo content = %q, code block task should not be synced", memo.Content)
	}
	if !strings.Contains(memo.Content, "- [ ] [[task:") {
		t.Fatalf("memo content = %q, outside task should be synced", memo.Content)
	}
}

func TestCreateVaultMemoPreservesBlankLines(t *testing.T) {
	ctx, existing, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	if existing {
		t.Fatalf("new temp vault should not be existing")
	}

	content := "first\n\nsecond\n\n"
	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    content,
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	if memo.Content != content {
		t.Fatalf("created content = %q, want %q", memo.Content, content)
	}

	path := filepath.Join(ctx.RootDir, filepath.FromSlash(memo.Path))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memo file: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "contentWhitespace: \"preserve\"") {
		t.Fatalf("memo file missing whitespace marker:\n%s", text)
	}
	if !strings.HasSuffix(text, "---\n"+content) {
		t.Fatalf("memo file content suffix = %q, want %q", text, "---\n"+content)
	}

	listed, err := listVaultMemos(ctx)
	if err != nil {
		t.Fatalf("list memos: %v", err)
	}
	if len(listed) != 1 || listed[0].Content != content {
		t.Fatalf("listed memos = %#v, want content %q", listed, content)
	}
}

func TestUpdateVaultMemoSourceMetadata(t *testing.T) {
	ctx, existing, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	if existing {
		t.Fatalf("new temp vault should not be existing")
	}

	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "补录的 memo",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}

	createdAt := "2024-01-02T03:04:05Z"
	updatedAt := ""
	kind := "note"
	taskID := "task_abc"
	pinned := true
	updated, err := updateVaultMemo(ctx, MemoUpdateRequest{
		CreatedAt: &createdAt,
		ID:        memo.ID,
		Kind:      &kind,
		Pinned:    &pinned,
		TaskID:    &taskID,
		UpdatedAt: &updatedAt,
	})
	if err != nil {
		t.Fatalf("update memo source metadata: %v", err)
	}
	if updated.Content != memo.Content {
		t.Fatalf("content = %q, want %q", updated.Content, memo.Content)
	}
	if updated.CreatedAt != createdAt {
		t.Fatalf("createdAt = %q, want %q", updated.CreatedAt, createdAt)
	}
	if updated.UpdatedAt != "" {
		t.Fatalf("updatedAt = %q, want empty", updated.UpdatedAt)
	}
	if updated.Kind != kind || updated.TaskID != taskID || !updated.Pinned {
		t.Fatalf("updated memo = %#v, want kind/task/pinned metadata", updated)
	}

	raw, err := os.ReadFile(filepath.Join(ctx.RootDir, filepath.FromSlash(updated.Path)))
	if err != nil {
		t.Fatalf("read updated memo: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"createdAt: \"" + createdAt + "\"",
		"updatedAt: \"\"",
		"kind: \"" + kind + "\"",
		"taskId: \"" + taskID + "\"",
		"补录的 memo",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("memo file missing %q:\n%s", want, text)
		}
	}
}

func TestCreateVaultMemoCanBelongToProject(t *testing.T) {
	ctx, existing, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	if existing {
		t.Fatalf("new temp vault should not be existing")
	}

	project, err := createVaultProject(ctx, ProjectCreateRequest{Name: "Work", Color: "#10b981"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "project memo",
		ProjectID:  project.ID,
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	if memo.ProjectID != project.ID {
		t.Fatalf("memo project id = %q, want %q", memo.ProjectID, project.ID)
	}

	path := filepath.Join(ctx.RootDir, filepath.FromSlash(memo.Path))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memo file: %v", err)
	}
	if !strings.Contains(string(raw), "projectId: \""+project.ID+"\"") {
		t.Fatalf("memo file missing projectId:\n%s", string(raw))
	}

	listed, err := listVaultMemos(ctx)
	if err != nil {
		t.Fatalf("list memos: %v", err)
	}
	if len(listed) != 1 || listed[0].ProjectID != project.ID {
		t.Fatalf("listed memos = %#v, want project %s", listed, project.ID)
	}
}

func TestCreateVaultMemoRejectsUnknownProject(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	if _, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:   "orphan",
		ProjectID: "project_missing",
	}); err == nil {
		t.Fatalf("expected unknown project error")
	}
}

func TestVaultMemoDraftLifecycle(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	project, err := createVaultProject(ctx, ProjectCreateRequest{Name: "Drafts", Color: "#10b981"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	draft, err := upsertVaultMemoDraft(ctx, MemoDraftUpsertRequest{
		Content:    "draft body",
		ID:         "draft_composer",
		Kind:       "composer",
		ProjectID:  project.ID,
		Visibility: "PUBLIC",
	})
	if err != nil {
		t.Fatalf("upsert draft: %v", err)
	}
	if draft.ProjectID != project.ID || draft.Visibility != "PUBLIC" || draft.UpdatedAt == "" {
		t.Fatalf("draft = %#v, want normalized project/visibility/time", draft)
	}

	path := filepath.Join(ctx.VeloDir, vaultMemoDraftsFileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read draft file: %v", err)
	}
	if !strings.Contains(string(raw), `"draft_composer"`) || !strings.Contains(string(raw), `"draft body"`) {
		t.Fatalf("draft file missing data:\n%s", string(raw))
	}

	updated, err := upsertVaultMemoDraft(ctx, MemoDraftUpsertRequest{
		Content:    "updated body",
		ID:         "draft_composer",
		Kind:       "composer",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("update draft: %v", err)
	}
	if updated.Content != "updated body" {
		t.Fatalf("updated content = %q", updated.Content)
	}

	drafts, err := listVaultMemoDrafts(ctx)
	if err != nil {
		t.Fatalf("list drafts: %v", err)
	}
	if len(drafts) != 1 || drafts[0].Content != "updated body" {
		t.Fatalf("drafts = %#v, want one updated draft", drafts)
	}

	if _, err := upsertVaultMemoDraft(ctx, MemoDraftUpsertRequest{
		Content: "edit draft",
		ID:      "draft_memo_missing",
		Kind:    "memo-edit",
	}); err == nil {
		t.Fatalf("expected memo-edit draft without memo id to fail")
	}

	if err := deleteVaultMemoDraft(ctx, "draft_composer"); err != nil {
		t.Fatalf("delete draft: %v", err)
	}
	drafts, err = listVaultMemoDrafts(ctx)
	if err != nil {
		t.Fatalf("list drafts after delete: %v", err)
	}
	if len(drafts) != 0 {
		t.Fatalf("drafts after delete = %#v, want empty", drafts)
	}
}

func TestCreateVaultTaskWritesJSONAndIndex(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	project, err := createVaultProject(ctx, ProjectCreateRequest{Name: "Release", Color: "#10b981"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	task, err := createVaultTask(ctx, TaskCreateRequest{
		Contexts:  []string{"office", "office"},
		DueAt:     "2026-06-09T18:00:00+08:00",
		ListID:    "",
		Notes:     "ship notes\n",
		Priority:  "high",
		ProjectID: project.ID,
		Reminders: []TaskReminder{{Type: "relative", Base: "dueAt", OffsetMinutes: -30}},
		Repeat:    TaskRepeat{Frequency: "weekly", Interval: 1, Weekdays: []string{"MO"}},
		Source:    TaskSource{Type: "memo", MemoID: "memo_source", Line: 12, Text: "- [ ] Ship"},
		Tags:      []string{"release", "release"},
		Title:     "Ship release",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.ID == "" {
		t.Fatalf("task id is empty")
	}
	if task.Status != taskStatusOpen {
		t.Fatalf("status = %q, want open", task.Status)
	}
	if task.ListID != "inbox" {
		t.Fatalf("list id = %q, want inbox", task.ListID)
	}
	if len(task.Tags) != 1 || task.Tags[0] != "release" {
		t.Fatalf("tags = %#v, want release", task.Tags)
	}
	if len(task.Contexts) != 1 || task.Contexts[0] != "office" {
		t.Fatalf("contexts = %#v, want office", task.Contexts)
	}

	raw, err := os.ReadFile(filepath.Join(ctx.RootDir, filepath.FromSlash(task.Path)))
	if err != nil {
		t.Fatalf("read task file: %v", err)
	}
	var stored TaskRecord
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("task file is not json: %v\n%s", err, string(raw))
	}
	if stored.ID != task.ID || stored.Title != "Ship release" || stored.Status != taskStatusOpen || stored.Priority != taskPriorityHigh {
		t.Fatalf("stored task = %#v, want json task", stored)
	}
	if len(stored.Tags) != 1 || stored.Tags[0] != "release" || len(stored.Reminders) != 1 || stored.Repeat.Frequency != "weekly" {
		t.Fatalf("stored task metadata = %#v, want task metadata", stored)
	}
	if stored.Notes != "ship notes\n" {
		t.Fatalf("stored notes = %q, want preserved notes", stored.Notes)
	}

	readBack, err := getVaultTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if readBack.Title != task.Title || readBack.ProjectID != project.ID || len(readBack.Reminders) != 1 {
		t.Fatalf("read task = %#v, want persisted task", readBack)
	}

	index, err := loadTaskIndex(ctx)
	if err != nil {
		t.Fatalf("load task index: %v", err)
	}
	entry, ok := index.Tasks[task.ID]
	if !ok {
		t.Fatalf("task missing from index: %#v", index.Tasks)
	}
	if entry.Path != task.Path || entry.Title != task.Title || entry.Status != taskStatusOpen {
		t.Fatalf("index entry = %#v, want task summary", entry)
	}
	if entry.Source.Type != "memo" || entry.Source.MemoID != "memo_source" || entry.Source.Line != 12 {
		t.Fatalf("index source = %#v, want memo source", entry.Source)
	}
}

func TestCompleteVaultTaskMovesFileAndRebuildsIndex(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	task, err := createVaultTask(ctx, TaskCreateRequest{Title: "Done task"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	oldPath := filepath.Join(ctx.RootDir, filepath.FromSlash(task.Path))

	completed, err := completeVaultTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("complete task: %v", err)
	}
	if completed.Status != taskStatusCompleted {
		t.Fatalf("status = %q, want completed", completed.Status)
	}
	if completed.CompletedAt == "" {
		t.Fatalf("completedAt is empty")
	}
	if !strings.Contains(completed.Path, "/completed/") {
		t.Fatalf("completed path = %q, want completed folder", completed.Path)
	}
	if filepath.Ext(completed.Path) != ".json" {
		t.Fatalf("completed path = %q, want json file", completed.Path)
	}
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf("old task file still exists or stat failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ctx.RootDir, filepath.FromSlash(completed.Path))); err != nil {
		t.Fatalf("completed task file missing: %v", err)
	}

	index, err := rebuildTaskIndex(ctx)
	if err != nil {
		t.Fatalf("rebuild task index: %v", err)
	}
	entry, ok := index.Tasks[task.ID]
	if !ok {
		t.Fatalf("completed task missing from index")
	}
	if entry.Status != taskStatusCompleted || entry.Path != completed.Path {
		t.Fatalf("index entry = %#v, want completed path %q", entry, completed.Path)
	}
}

func TestCreateVaultMemoAutoCreatesTaskFromTodoLine(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "计划\n- [ ] 跟进发布 Hello content `inline code` ::2026-06-09 #release #urgent\n",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	if !strings.Contains(memo.Content, "- [ ] [[task:") {
		t.Fatalf("memo content = %q, want task ref", memo.Content)
	}
	tasks, err := listVaultTasks(ctx)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks = %#v, want one task", tasks)
	}
	task := tasks[0]
	if task.Title != "跟进发布 Hello content inline code" || task.Source.MemoID != memo.ID || task.Source.Line != 2 {
		t.Fatalf("task = %#v, want task from memo line", task)
	}
	if task.DueAt != "2026-06-09" {
		t.Fatalf("task dueAt = %q, want date from todo text", task.DueAt)
	}
	if !stringSliceContains(task.Tags, "release") || !stringSliceContains(task.Tags, "urgent") {
		t.Fatalf("task tags = %#v, want tags from todo text", task.Tags)
	}
	if !strings.Contains(memo.Content, "[[task:"+task.ID+"|跟进发布 Hello content inline code]]") {
		t.Fatalf("memo content = %q, want full task title alias", memo.Content)
	}

	updatedContent := strings.Replace(memo.Content, "- [ ]", "- [x]", 1)
	updated, err := updateVaultMemo(ctx, MemoUpdateRequest{ID: memo.ID, Content: &updatedContent})
	if err != nil {
		t.Fatalf("update memo: %v", err)
	}
	if !strings.Contains(updated.Content, "[[task:"+task.ID) {
		t.Fatalf("updated memo content = %q, want same task ref", updated.Content)
	}
	completed, err := getVaultTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get completed task: %v", err)
	}
	if completed.Status != taskStatusCompleted {
		t.Fatalf("task status = %q, want completed", completed.Status)
	}
}

func TestCreateVaultMemoCommentAutoCreatesTaskFromTodoLine(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	project, err := createVaultProject(ctx, ProjectCreateRequest{Name: "Reply Project"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "Parent memo",
		ProjectID:  project.ID,
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}

	comment, err := createVaultMemoComment(ctx, MemoCommentCreateRequest{
		Content: "补充\n- [ ] 评论代办 ::2026-06-09 #reply\n",
		MemoID:  memo.ID,
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if !strings.Contains(comment.Content, "- [ ] [[task:") {
		t.Fatalf("comment content = %q, want task ref", comment.Content)
	}
	tasks, err := listVaultTasks(ctx)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks = %#v, want one task", tasks)
	}
	task := tasks[0]
	if task.Title != "评论代办" || task.ProjectID != project.ID {
		t.Fatalf("task = %#v, want comment task in project", task)
	}
	if task.Source.Type != "comment" || task.Source.MemoID != memo.ID || task.Source.CommentID != comment.ID || task.Source.Line != 2 {
		t.Fatalf("task source = %#v, want comment source", task.Source)
	}
	if !strings.Contains(comment.Content, "[[task:"+task.ID+"|评论代办]]") {
		t.Fatalf("comment content = %q, want task title alias", comment.Content)
	}

	updatedContent := strings.Replace(comment.Content, "- [ ]", "- [x]", 1)
	updated, err := updateVaultMemoComment(ctx, MemoCommentUpdateRequest{ID: comment.ID, Content: &updatedContent})
	if err != nil {
		t.Fatalf("update comment: %v", err)
	}
	if !strings.Contains(updated.Content, "[[task:"+task.ID) {
		t.Fatalf("updated comment content = %q, want same task ref", updated.Content)
	}
	completed, err := getVaultTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get completed task: %v", err)
	}
	if completed.Status != taskStatusCompleted {
		t.Fatalf("task status = %q, want completed", completed.Status)
	}
}

func TestSearchVaultSnippetsAndLinksIncludeMemoComments(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "Parent memo",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	comment, err := createVaultMemoComment(ctx, MemoCommentCreateRequest{
		Content: strings.Join([]string{
			"#snippet Comment Curl",
			"```sh",
			"curl https://example.com/api",
			"```",
			"https://example.com/docs",
		}, "\n"),
		MemoID: memo.ID,
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}

	snippets, err := searchVaultSnippets(ctx, "snippet curl", 10)
	if err != nil {
		t.Fatalf("search snippets: %v", err)
	}
	if len(snippets) != 1 {
		t.Fatalf("snippets = %#v, want one comment snippet", snippets)
	}
	if snippets[0].SourceType != "comment" || snippets[0].MemoID != memo.ID || snippets[0].CommentID != comment.ID {
		t.Fatalf("snippet source = %#v, want comment source", snippets[0])
	}

	links, err := searchVaultLinks(ctx, "link docs", 10)
	if err != nil {
		t.Fatalf("search links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("links = %#v, want one comment link", links)
	}
	if links[0].SourceType != "comment" || links[0].MemoID != memo.ID || links[0].CommentID != comment.ID {
		t.Fatalf("link source = %#v, want comment source", links[0])
	}
}

func TestCreateVaultMemoAutoCreatesTaskWithExplicitTitle(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "计划\n- [ ] [跟进发布] 这里写很长的上下文 `inline code` ::2026-06-09 #release\n",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	tasks, err := listVaultTasks(ctx)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("tasks = %#v, want one task", tasks)
	}
	task := tasks[0]
	if task.Title != "跟进发布" {
		t.Fatalf("task title = %q, want explicit title", task.Title)
	}
	if task.Notes != "这里写很长的上下文 inline code" {
		t.Fatalf("task notes = %q, want remaining context", task.Notes)
	}
	if task.DueAt != "2026-06-09" {
		t.Fatalf("task dueAt = %q, want date from todo text", task.DueAt)
	}
	if !strings.Contains(memo.Content, "[[task:"+task.ID+"|跟进发布]]") {
		t.Fatalf("memo content = %q, want short task title alias", memo.Content)
	}
}

func TestTaskTitleFromTodoTextUsesFullCleanedText(t *testing.T) {
	metadata := parseTaskMetadataFromTodoText("Hello content `inline code` ::2026-06-09 #work")
	if metadata.Title != "Hello content inline code" {
		t.Fatalf("title = %q, want full cleaned text", metadata.Title)
	}
	if metadata.DueAt != "2026-06-09" {
		t.Fatalf("dueAt = %q, want parsed date", metadata.DueAt)
	}
	if !stringSliceContains(metadata.Tags, "work") {
		t.Fatalf("tags = %#v, want work", metadata.Tags)
	}
}

func TestTaskTitleFromTodoTextSupportsExplicitTitlePrefix(t *testing.T) {
	metadata := parseTaskMetadataFromTodoText("[Ship release] Long context with `inline code` ::2026-06-09 #work")
	if metadata.Title != "Ship release" {
		t.Fatalf("title = %q, want explicit title", metadata.Title)
	}
	if metadata.Notes != "Long context with inline code" {
		t.Fatalf("notes = %q, want remaining context", metadata.Notes)
	}
	if metadata.DueAt != "2026-06-09" {
		t.Fatalf("dueAt = %q, want parsed date", metadata.DueAt)
	}
	if !stringSliceContains(metadata.Tags, "work") {
		t.Fatalf("tags = %#v, want work", metadata.Tags)
	}
}

func TestTaskTitleFromTodoTextSupportsExplicitTitleProperty(t *testing.T) {
	metadata := parseTaskMetadataFromTodoText("Long context {标题：跟进发布} ::2026-06-09 #release")
	if metadata.Title != "跟进发布" {
		t.Fatalf("title = %q, want explicit title", metadata.Title)
	}
	if metadata.Notes != "Long context" {
		t.Fatalf("notes = %q, want remaining context", metadata.Notes)
	}
	if metadata.DueAt != "2026-06-09" {
		t.Fatalf("dueAt = %q, want parsed date", metadata.DueAt)
	}
	if !stringSliceContains(metadata.Tags, "release") {
		t.Fatalf("tags = %#v, want release", metadata.Tags)
	}
}

func TestTaskTitleFromTodoTextIgnoresFullwidthTimeTrigger(t *testing.T) {
	metadata := parseTaskMetadataFromTodoText("Hello content ：：2026-06-09 #work")
	if metadata.Title != "Hello content ：：2026-06-09" {
		t.Fatalf("title = %q, want fullwidth trigger kept as text", metadata.Title)
	}
	if metadata.DueAt != "" {
		t.Fatalf("dueAt = %q, want no date from fullwidth trigger", metadata.DueAt)
	}
	if !stringSliceContains(metadata.Tags, "work") {
		t.Fatalf("tags = %#v, want work", metadata.Tags)
	}
}

func TestDeleteVaultMemoWithTasksDeletesSourceTasks(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "计划\n- [ ] 跟进发布\n- [ ] 写发布日志\n",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	keep, err := createVaultTask(ctx, TaskCreateRequest{Title: "Keep standalone task"})
	if err != nil {
		t.Fatalf("create standalone task: %v", err)
	}

	tasks, err := listVaultTasks(ctx)
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("tasks = %#v, want two memo tasks and one standalone", tasks)
	}

	result, err := deleteVaultMemoWithOptions(ctx, memo.ID, MemoDeleteOptions{DeleteTasks: true})
	if err != nil {
		t.Fatalf("delete memo: %v", err)
	}
	if result.TasksDeleted != 2 {
		t.Fatalf("tasks deleted = %d, want 2", result.TasksDeleted)
	}
	if _, err := findMemoFilePath(ctx, memo.ID); err == nil {
		t.Fatalf("memo file still exists")
	}

	tasks, err = listVaultTasks(ctx)
	if err != nil {
		t.Fatalf("list tasks after delete: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != keep.ID {
		t.Fatalf("remaining tasks = %#v, want standalone task", tasks)
	}
	index, err := loadTaskIndex(ctx)
	if err != nil {
		t.Fatalf("load index: %v", err)
	}
	if _, ok := index.Tasks[keep.ID]; !ok || len(index.Tasks) != 1 {
		t.Fatalf("index tasks = %#v, want standalone task only", index.Tasks)
	}
}

func TestTaskNoteAutoCreatesSubtaskFromTodoLine(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	parent, err := createVaultTask(ctx, TaskCreateRequest{Priority: "medium", Tags: []string{"release"}, Title: "Parent task"})
	if err != nil {
		t.Fatalf("create parent task: %v", err)
	}

	parent, note, err := createVaultTaskNote(ctx, TaskNoteCreateRequest{
		Content:    "执行记录\n- [ ] 检查更新包 #qa\n",
		TaskID:     parent.ID,
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create task note: %v", err)
	}
	if note.Kind != "task_note" || note.TaskID != parent.ID {
		t.Fatalf("note metadata = %#v, want task note for parent", note)
	}
	if len(parent.NoteRefs) != 1 || parent.NoteRefs[0].MemoID != note.ID {
		t.Fatalf("parent note refs = %#v, want created note", parent.NoteRefs)
	}
	notePath := filepath.Join(ctx.RootDir, filepath.FromSlash(note.Path))
	rawNote, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	if !strings.Contains(string(rawNote), "kind: \"task_note\"") || !strings.Contains(string(rawNote), "taskId: \""+parent.ID+"\"") {
		t.Fatalf("note file missing task metadata:\n%s", string(rawNote))
	}
	if !strings.Contains(note.Content, "- [ ] [[task:") {
		t.Fatalf("note content = %q, want task ref", note.Content)
	}
	parent, err = getVaultTask(ctx, parent.ID)
	if err != nil {
		t.Fatalf("get parent task: %v", err)
	}
	if len(parent.SubtaskIDs) != 1 {
		t.Fatalf("parent subtasks = %#v, want one child", parent.SubtaskIDs)
	}
	child, err := getVaultTask(ctx, parent.SubtaskIDs[0])
	if err != nil {
		t.Fatalf("get child task: %v", err)
	}
	if child.ParentID != parent.ID {
		t.Fatalf("child parent id = %q, want %q", child.ParentID, parent.ID)
	}
	if child.Title != "检查更新包" {
		t.Fatalf("child title = %q, want todo text", child.Title)
	}
	if !stringSliceContains(child.Tags, "qa") {
		t.Fatalf("child tags = %#v, want qa", child.Tags)
	}
	if !strings.Contains(note.Content, "[[task:"+child.ID+"|"+child.Title+"]]") {
		t.Fatalf("note content = %q, want child task ref", note.Content)
	}
}

func TestExtractMemoAssetReferences(t *testing.T) {
	content := strings.Join([]string{
		"![image](@assets/memo-local/images/a%29.png)",
		"[file](@assets/Other/docs/report.pdf)",
		"[space](@assets/memo-local/docs/my file.pdf)",
		"raw @assets/memo-local/raw.txt reference",
		"![inline](data:image/png;base64,abc)",
	}, "\n")

	got := extractMemoAssetReferences(content)
	want := map[string]bool{
		"memo-local/images/a).png":    true,
		"memo-local/docs/my file.pdf": true,
		"other/docs/report.pdf":       true,
		"memo-local/raw.txt":          true,
	}
	if len(got) != len(want) {
		t.Fatalf("asset refs length = %d, want %d: %#v", len(got), len(want), got)
	}
	for _, ref := range got {
		id := memoAssetReferenceID(ref)
		if !want[id] {
			t.Fatalf("unexpected asset ref %q from %#v", id, got)
		}
		delete(want, id)
	}
	if len(want) > 0 {
		t.Fatalf("missing asset refs: %#v", want)
	}
}

func TestExtractMemoAssetReferencesIgnoresCode(t *testing.T) {
	content := strings.Join([]string{
		"[real](@assets/memo-local/docs/real.pdf)",
		"`[inline](@assets/memo-local/docs/inline.pdf)`",
		"```",
		"[code](@assets/memo-local/docs/code.pdf)",
		"raw @assets/memo-local/raw-code.txt reference",
		"```",
		"raw @assets/memo-local/raw-real.txt reference",
	}, "\n")

	got := extractMemoAssetReferences(content)
	want := map[string]bool{
		"memo-local/docs/real.pdf": true,
		"memo-local/raw-real.txt":  true,
	}
	if len(got) != len(want) {
		t.Fatalf("asset refs length = %d, want %d: %#v", len(got), len(want), got)
	}
	for _, ref := range got {
		id := memoAssetReferenceID(ref)
		if !want[id] {
			t.Fatalf("unexpected asset ref %q from %#v", id, got)
		}
		delete(want, id)
	}
	if len(want) > 0 {
		t.Fatalf("missing asset refs: %#v", want)
	}
}

func TestDeleteVaultMemoRemovesExclusiveManagedAssets(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	storePath := filepath.Join(ctx.VeloDir, "storage.json")
	cfg := defaultLocalMemoOSSConfig(storePath)
	settings := CloudStorageSettings{
		ActiveStorageID:     cfg.ID,
		DefaultsInitialized: true,
		Storages:            []OSSConfig{cfg},
	}
	rawSettings, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}

	uniqueKey := "images/unique.png"
	sharedKey := "images/shared.png"
	if err := writeLocalOSSObject(context.Background(), cfg, uniqueKey, []byte("unique")); err != nil {
		t.Fatalf("write unique asset: %v", err)
	}
	if err := writeLocalOSSObject(context.Background(), cfg, sharedKey, []byte("shared")); err != nil {
		t.Fatalf("write shared asset: %v", err)
	}
	uniquePath, err := localOSSObjectDiskPath(cfg, uniqueKey)
	if err != nil {
		t.Fatalf("unique asset path: %v", err)
	}
	sharedPath, err := localOSSObjectDiskPath(cfg, sharedKey)
	if err != nil {
		t.Fatalf("shared asset path: %v", err)
	}
	localOriginal := filepath.Join(t.TempDir(), "original.txt")
	if err := os.WriteFile(localOriginal, []byte("original"), 0644); err != nil {
		t.Fatalf("write local original: %v", err)
	}

	target, err := createVaultMemo(ctx, MemoCreateRequest{
		Content: strings.Join([]string{
			"![unique](@assets/memo-local/" + uniqueKey + ")",
			"![shared](@assets/memo-local/" + sharedKey + ")",
			"[local](file://" + localOriginal + ")",
			"[remote](https://example.com/file.png)",
		}, "\n"),
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create target memo: %v", err)
	}
	other, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "![shared](@assets/memo-local/" + sharedKey + ")",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create other memo: %v", err)
	}

	result, err := deleteVaultMemoWithAssets(context.Background(), ctx, target.ID, json.RawMessage(rawSettings), storePath)
	if err != nil {
		t.Fatalf("delete memo: %v", err)
	}
	if result.AssetsDeleted != 1 || result.AssetsSkipped != 1 || len(result.AssetErrors) != 0 {
		t.Fatalf("delete result = %#v, want 1 deleted, 1 skipped, no errors", result)
	}
	targetPath := filepath.Join(ctx.RootDir, filepath.FromSlash(target.Path))
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("target memo still exists or stat failed: %v", err)
	}
	if _, err := os.Stat(uniquePath); !os.IsNotExist(err) {
		t.Fatalf("unique asset still exists or stat failed: %v", err)
	}
	if _, err := os.Stat(sharedPath); err != nil {
		t.Fatalf("shared asset was removed: %v", err)
	}
	if _, err := os.Stat(localOriginal); err != nil {
		t.Fatalf("local original file was removed: %v", err)
	}

	listed, err := listVaultMemos(ctx)
	if err != nil {
		t.Fatalf("list memos: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != other.ID {
		t.Fatalf("listed memos = %#v, want only other memo %s", listed, other.ID)
	}
}

func TestDeleteVaultMemoCanKeepManagedAssets(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	storePath := filepath.Join(ctx.VeloDir, "storage.json")
	cfg := defaultLocalMemoOSSConfig(storePath)
	settings := CloudStorageSettings{
		ActiveStorageID:     cfg.ID,
		DefaultsInitialized: true,
		Storages:            []OSSConfig{cfg},
	}
	rawSettings, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}

	key := "docs/keep.pdf"
	if err := writeLocalOSSObject(context.Background(), cfg, key, []byte("keep")); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	assetPath, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		t.Fatalf("asset path: %v", err)
	}
	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "[file](@assets/memo-local/" + key + ")",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}

	result, err := deleteVaultMemoWithOptions(ctx, memo.ID, MemoDeleteOptions{
		CleanupAssets:   false,
		Parent:          context.Background(),
		StorageSettings: json.RawMessage(rawSettings),
		StorePath:       storePath,
	})
	if err != nil {
		t.Fatalf("delete memo: %v", err)
	}
	if result.AssetsDeleted != 0 || result.AssetsSkipped != 0 || len(result.AssetErrors) != 0 {
		t.Fatalf("delete result = %#v, want no asset cleanup", result)
	}
	if _, err := os.Stat(assetPath); err != nil {
		t.Fatalf("asset should remain: %v", err)
	}
}

func TestMemoCommentsPersistAndDeleteWithParent(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "Parent memo",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	comment, err := createVaultMemoComment(ctx, MemoCommentCreateRequest{
		Content: "Comment body #reply\n\n[[memo:" + memo.ID + "|parent]]",
		MemoID:  memo.ID,
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if comment.MemoID != memo.ID {
		t.Fatalf("comment memo id = %q, want %q", comment.MemoID, memo.ID)
	}
	if len(comment.Tags) != 1 || comment.Tags[0] != "reply" {
		t.Fatalf("comment tags = %#v, want reply", comment.Tags)
	}
	if len(comment.References) != 1 || comment.References[0] != "memo:"+memo.ID {
		t.Fatalf("comment references = %#v, want parent memo ref", comment.References)
	}

	commentPath := filepath.Join(ctx.RootDir, filepath.FromSlash(comment.Path))
	raw, err := os.ReadFile(commentPath)
	if err != nil {
		t.Fatalf("read comment: %v", err)
	}
	if !strings.Contains(string(raw), "memoId: \""+memo.ID+"\"") || !strings.Contains(string(raw), "Comment body") {
		t.Fatalf("comment file missing metadata or content:\n%s", string(raw))
	}

	listed, err := listVaultMemoComments(ctx, memo.ID)
	if err != nil {
		t.Fatalf("list memo comments: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != comment.ID {
		t.Fatalf("listed comments = %#v, want %s", listed, comment.ID)
	}

	if _, err := deleteVaultMemoWithOptions(ctx, memo.ID, MemoDeleteOptions{}); err != nil {
		t.Fatalf("delete memo: %v", err)
	}
	if _, err := os.Stat(commentPath); !os.IsNotExist(err) {
		t.Fatalf("comment should be deleted with parent, stat err = %v", err)
	}
	listed, err = listVaultMemoComments(ctx, "")
	if err != nil {
		t.Fatalf("list comments after delete: %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("listed comments after delete = %#v, want empty", listed)
	}
}

func TestDeleteVaultMemoCleansManagedAssetsReferencedByComments(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	storePath := filepath.Join(ctx.VeloDir, "storage.json")
	cfg := defaultLocalMemoOSSConfig(storePath)
	settings := CloudStorageSettings{
		ActiveStorageID:     cfg.ID,
		DefaultsInitialized: true,
		Storages:            []OSSConfig{cfg},
	}
	rawSettings, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}

	uniqueKey := "images/comment-unique.png"
	sharedKey := "images/comment-shared.png"
	if err := writeLocalOSSObject(context.Background(), cfg, uniqueKey, []byte("unique")); err != nil {
		t.Fatalf("write unique asset: %v", err)
	}
	if err := writeLocalOSSObject(context.Background(), cfg, sharedKey, []byte("shared")); err != nil {
		t.Fatalf("write shared asset: %v", err)
	}
	uniquePath, err := localOSSObjectDiskPath(cfg, uniqueKey)
	if err != nil {
		t.Fatalf("unique asset path: %v", err)
	}
	sharedPath, err := localOSSObjectDiskPath(cfg, sharedKey)
	if err != nil {
		t.Fatalf("shared asset path: %v", err)
	}

	target, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "Target memo",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create target memo: %v", err)
	}
	comment, err := createVaultMemoComment(ctx, MemoCommentCreateRequest{
		Content: strings.Join([]string{
			"![unique](@assets/memo-local/" + uniqueKey + ")",
			"![shared](@assets/memo-local/" + sharedKey + ")",
		}, "\n"),
		MemoID: target.ID,
	})
	if err != nil {
		t.Fatalf("create comment: %v", err)
	}
	if _, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "![shared](@assets/memo-local/" + sharedKey + ")",
		Visibility: "PRIVATE",
	}); err != nil {
		t.Fatalf("create other memo: %v", err)
	}

	result, err := deleteVaultMemoWithOptions(ctx, target.ID, MemoDeleteOptions{
		CleanupAssets:   true,
		Parent:          context.Background(),
		StorageSettings: json.RawMessage(rawSettings),
		StorePath:       storePath,
	})
	if err != nil {
		t.Fatalf("delete memo: %v", err)
	}
	if result.AssetsDeleted != 1 || result.AssetsSkipped != 1 || len(result.AssetErrors) != 0 {
		t.Fatalf("delete result = %#v, want 1 deleted, 1 skipped, no errors", result)
	}
	if _, err := os.Stat(filepath.Join(ctx.RootDir, filepath.FromSlash(comment.Path))); !os.IsNotExist(err) {
		t.Fatalf("comment should be deleted with parent, stat err = %v", err)
	}
	if _, err := os.Stat(uniquePath); !os.IsNotExist(err) {
		t.Fatalf("unique comment asset still exists or stat failed: %v", err)
	}
	if _, err := os.Stat(sharedPath); err != nil {
		t.Fatalf("shared comment asset was removed: %v", err)
	}
}

func TestDefaultLocalStorageRootUsesVaultStorage(t *testing.T) {
	vaultDir := t.TempDir()
	storePath := filepath.Join(vaultDir, ".velo", "storage.json")
	got := defaultLocalStorageRoot(storePath)
	want := filepath.Join(vaultDir, "storage")
	if got != want {
		t.Fatalf("defaultLocalStorageRoot() = %q, want %q", got, want)
	}
}

func TestPrepareCloudStorageSettingsRepointsLegacyLocalRoot(t *testing.T) {
	vaultDir := t.TempDir()
	storePath := filepath.Join(vaultDir, ".velo", "storage.json")
	settings := CloudStorageSettings{
		ActiveStorageID:     "memo-local",
		DefaultsInitialized: true,
		Storages: []OSSConfig{
			{
				Bucket:         "memos",
				Enabled:        true,
				Endpoint:       legacyDefaultLocalStorageRoot(storePath),
				ForcePathStyle: true,
				ID:             "memo-local",
				Name:           "本地 Memo 存储",
				Provider:       "local",
				UseSSL:         false,
			},
		},
	}

	got, changed, err := prepareCloudStorageSettings(settings, storePath, false)
	if err != nil {
		t.Fatalf("prepareCloudStorageSettings: %v", err)
	}
	if !changed {
		t.Fatalf("prepareCloudStorageSettings changed = false, want true")
	}
	if len(got.Storages) != 1 {
		t.Fatalf("storages length = %d, want 1", len(got.Storages))
	}
	wantEndpoint := filepath.Join(vaultDir, "storage")
	if got.Storages[0].Endpoint != wantEndpoint {
		t.Fatalf("endpoint = %q, want %q", got.Storages[0].Endpoint, wantEndpoint)
	}
	if got.Storages[0].Local == nil || got.Storages[0].Local.RootMode != localStorageRootModeVault || got.Storages[0].Local.Root != defaultLocalStorageRelativeRoot {
		t.Fatalf("local settings = %#v, want vault storage", got.Storages[0].Local)
	}
	if _, err := os.Stat(filepath.Join(wantEndpoint, "memos")); err != nil {
		t.Fatalf("local bucket was not created under vault storage: %v", err)
	}
}

func TestPrepareCloudStorageSettingsResolvesSyncedLocalRootForCurrentVault(t *testing.T) {
	sourceVaultDir := t.TempDir()
	targetVaultDir := t.TempDir()
	sourceStorePath := filepath.Join(sourceVaultDir, ".velo", "storage.json")
	targetStorePath := filepath.Join(targetVaultDir, ".velo", "storage.json")
	settings := CloudStorageSettings{
		ActiveStorageID:     "attachments",
		DefaultsInitialized: true,
		Storages: []OSSConfig{
			{
				Bucket:         "files",
				Enabled:        true,
				Endpoint:       defaultLocalStorageRoot(sourceStorePath),
				ForcePathStyle: true,
				ID:             "attachments",
				Name:           "Attachments",
				Provider:       "local",
				UseSSL:         false,
			},
		},
	}

	got, changed, err := prepareCloudStorageSettings(settings, targetStorePath, false)
	if err != nil {
		t.Fatalf("prepareCloudStorageSettings: %v", err)
	}
	if !changed {
		t.Fatalf("prepareCloudStorageSettings changed = false, want true")
	}
	if len(got.Storages) != 1 {
		t.Fatalf("storages length = %d, want 1", len(got.Storages))
	}
	wantEndpoint := filepath.Join(targetVaultDir, "storage")
	if got.Storages[0].Endpoint != wantEndpoint {
		t.Fatalf("endpoint = %q, want %q", got.Storages[0].Endpoint, wantEndpoint)
	}
	if got.Storages[0].Local == nil || got.Storages[0].Local.RootMode != localStorageRootModeVault || got.Storages[0].Local.Root != defaultLocalStorageRelativeRoot {
		t.Fatalf("local settings = %#v, want vault storage", got.Storages[0].Local)
	}
	if _, err := os.Stat(filepath.Join(wantEndpoint, "files")); err != nil {
		t.Fatalf("local bucket was not created under target vault storage: %v", err)
	}

	raw, err := marshalCloudStorageSettingsForStore(got)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, sourceVaultDir) || strings.Contains(text, targetVaultDir) {
		t.Fatalf("stored settings should not include machine-specific vault paths: %s", text)
	}
	var stored CloudStorageSettings
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("unmarshal stored settings: %v", err)
	}
	if stored.Storages[0].Endpoint != "" {
		t.Fatalf("stored endpoint = %q, want empty portable endpoint", stored.Storages[0].Endpoint)
	}
	if stored.Storages[0].Local == nil || stored.Storages[0].Local.RootMode != localStorageRootModeVault || stored.Storages[0].Local.Root != defaultLocalStorageRelativeRoot {
		t.Fatalf("stored local settings = %#v, want vault storage", stored.Storages[0].Local)
	}
}

func TestPrepareCloudStorageSettingsKeepsCustomLocalAbsoluteRoot(t *testing.T) {
	vaultDir := t.TempDir()
	storePath := filepath.Join(vaultDir, ".velo", "storage.json")
	customRoot := filepath.Join(t.TempDir(), "asset-root")
	settings := CloudStorageSettings{
		ActiveStorageID:     "custom-local",
		DefaultsInitialized: true,
		Storages: []OSSConfig{
			{
				Bucket:         "files",
				Enabled:        true,
				Endpoint:       customRoot,
				ForcePathStyle: true,
				ID:             "custom-local",
				Name:           "Custom Local",
				Provider:       "local",
				UseSSL:         false,
			},
		},
	}

	got, _, err := prepareCloudStorageSettings(settings, storePath, false)
	if err != nil {
		t.Fatalf("prepareCloudStorageSettings: %v", err)
	}
	if got.Storages[0].Endpoint != customRoot {
		t.Fatalf("endpoint = %q, want custom root %q", got.Storages[0].Endpoint, customRoot)
	}
	if got.Storages[0].Local == nil || got.Storages[0].Local.RootMode != localStorageRootModeAbsolute || got.Storages[0].Local.Root != customRoot {
		t.Fatalf("local settings = %#v, want absolute custom root", got.Storages[0].Local)
	}

	raw, err := marshalCloudStorageSettingsForStore(got)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	var stored CloudStorageSettings
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("unmarshal stored settings: %v", err)
	}
	if stored.Storages[0].Endpoint != customRoot {
		t.Fatalf("stored endpoint = %q, want custom root %q", stored.Storages[0].Endpoint, customRoot)
	}
	if stored.Storages[0].Local == nil || stored.Storages[0].Local.RootMode != localStorageRootModeAbsolute || stored.Storages[0].Local.Root != customRoot {
		t.Fatalf("stored local settings = %#v, want absolute custom root", stored.Storages[0].Local)
	}
}
