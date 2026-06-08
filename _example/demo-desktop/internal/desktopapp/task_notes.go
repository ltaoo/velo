package desktopapp

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var taskLinePattern = regexp.MustCompile(`^(\s*[-*]\s+\[)([ xX])(\]\s+)(.*)$`)

type TaskNoteCreateRequest struct {
	Content    string `json:"content"`
	Role       string `json:"role"`
	TaskID     string `json:"taskId"`
	Visibility string `json:"visibility"`
}

type TaskExtractRequest struct {
	LineIndex      int    `json:"lineIndex"`
	MemoID         string `json:"memoId"`
	ParentTaskID   string `json:"parentTaskId"`
	ReplaceWithRef bool   `json:"replaceWithRef"`
}

type ParsedTaskLine struct {
	Checked bool
	Prefix  string
	Suffix  string
	Text    string
}

func createVaultTaskNote(ctx *VaultContext, req TaskNoteCreateRequest) (TaskRecord, MemoRecord, error) {
	taskID := sanitizeTaskID(req.TaskID)
	if taskID == "" {
		return TaskRecord{}, MemoRecord{}, fmt.Errorf("task id is required")
	}
	task, err := getVaultTask(ctx, taskID)
	if err != nil {
		return TaskRecord{}, MemoRecord{}, err
	}
	content := normalizeMemoContent(req.Content)
	if strings.TrimSpace(content) == "" {
		return TaskRecord{}, MemoRecord{}, fmt.Errorf("note content is required")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	memo := MemoRecord{
		Archived:   false,
		Content:    content,
		CreatedAt:  now,
		ID:         newMemoID(),
		Kind:       "task_note",
		Pinned:     false,
		ProjectID:  task.ProjectID,
		TaskID:     task.ID,
		UpdatedAt:  "",
		Visibility: normalizeMemoVisibility(req.Visibility),
	}
	memo.Path = memoRelativePath(memo)
	if err := syncMemoTaskLines(ctx, &memo); err != nil {
		return TaskRecord{}, MemoRecord{}, err
	}
	memo.Tags = extractMemoTags(memo.Content)
	memo.References = extractMemoReferences(memo.Content)
	if err := writeMemoRecord(ctx, memo); err != nil {
		return TaskRecord{}, MemoRecord{}, err
	}
	task, err = getVaultTask(ctx, task.ID)
	if err != nil {
		return TaskRecord{}, MemoRecord{}, err
	}
	ref := TaskNoteRef{
		CreatedAt: now,
		MemoID:    memo.ID,
		Role:      firstNonEmpty(strings.TrimSpace(req.Role), "note"),
		SortOrder: len(task.NoteRefs),
	}
	task = addTaskNoteRef(task, ref)
	task.UpdatedAt = now
	if err := writeTaskRecord(ctx, task); err != nil {
		return TaskRecord{}, MemoRecord{}, err
	}
	_ = appendTaskEvent(ctx, task.ID, "note_created", map[string]interface{}{"memoId": memo.ID})
	_, _ = rebuildTaskIndex(ctx)
	return task, memo, nil
}

func extractSubtaskFromMemoLine(ctx *VaultContext, req TaskExtractRequest) (TaskRecord, TaskRecord, MemoRecord, error) {
	parentID := sanitizeTaskID(req.ParentTaskID)
	if parentID == "" {
		return TaskRecord{}, TaskRecord{}, MemoRecord{}, fmt.Errorf("parent task id is required")
	}
	parent, err := getVaultTask(ctx, parentID)
	if err != nil {
		return TaskRecord{}, TaskRecord{}, MemoRecord{}, err
	}
	memoPath, err := findMemoFilePath(ctx, strings.TrimSpace(req.MemoID))
	if err != nil {
		return TaskRecord{}, TaskRecord{}, MemoRecord{}, err
	}
	memo, err := readMemoFile(ctx, memoPath)
	if err != nil {
		return TaskRecord{}, TaskRecord{}, MemoRecord{}, err
	}
	lines := strings.Split(normalizeMemoContent(memo.Content), "\n")
	if req.LineIndex < 0 || req.LineIndex >= len(lines) {
		return TaskRecord{}, TaskRecord{}, MemoRecord{}, fmt.Errorf("line index is out of range")
	}
	parsed, ok := parseTaskLineText(lines[req.LineIndex])
	if !ok || strings.TrimSpace(parsed.Text) == "" {
		return TaskRecord{}, TaskRecord{}, MemoRecord{}, fmt.Errorf("line is not a todo item")
	}
	if taskID := memoTaskRefID(parsed.Text); taskID != "" {
		child, err := getVaultTask(ctx, taskID)
		if err != nil {
			return TaskRecord{}, TaskRecord{}, MemoRecord{}, err
		}
		return parent, child, memo, nil
	}
	child, err := createVaultTask(ctx, TaskCreateRequest{
		Contexts:  parent.Contexts,
		ListID:    parent.ListID,
		ParentID:  parent.ID,
		Priority:  parent.Priority,
		ProjectID: parent.ProjectID,
		Source: TaskSource{
			Line:     req.LineIndex + 1,
			MemoID:   memo.ID,
			MemoPath: memo.Path,
			Text:     lines[req.LineIndex],
			Type:     "memo",
		},
		Tags:  uniqueStrings(append(parent.Tags, extractMemoTags(parsed.Text)...)),
		Title: taskTitleFromTodoText(parsed.Text),
	})
	if err != nil {
		return TaskRecord{}, TaskRecord{}, MemoRecord{}, err
	}
	if parsed.Checked {
		child, err = completeVaultTask(ctx, child.ID)
		if err != nil {
			return TaskRecord{}, TaskRecord{}, MemoRecord{}, err
		}
	}
	parent = addTaskSubtaskID(parent, child.ID)
	parent.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := writeTaskRecord(ctx, parent); err != nil {
		return TaskRecord{}, TaskRecord{}, MemoRecord{}, err
	}
	if req.ReplaceWithRef {
		lines[req.LineIndex] = parsed.Prefix + parsed.statusMarker() + parsed.Suffix + "[[task:" + child.ID + "|" + child.Title + "]]"
		memo.Content = strings.Join(lines, "\n")
		memo.Tags = extractMemoTags(memo.Content)
		memo.References = extractMemoReferences(memo.Content)
		memo.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		memo.Path = relativeVaultPath(ctx, memoPath)
		if err := writeMemoRecord(ctx, memo); err != nil {
			return TaskRecord{}, TaskRecord{}, MemoRecord{}, err
		}
	}
	_ = appendTaskEvent(ctx, parent.ID, "subtask_extracted", map[string]interface{}{
		"childTaskId": child.ID,
		"memoId":      memo.ID,
		"line":        req.LineIndex + 1,
	})
	_, _ = rebuildTaskIndex(ctx)
	return parent, child, memo, nil
}

func parseTaskLineText(line string) (ParsedTaskLine, bool) {
	match := taskLinePattern.FindStringSubmatch(line)
	if len(match) != 5 {
		return ParsedTaskLine{}, false
	}
	return ParsedTaskLine{
		Checked: strings.ToLower(match[2]) == "x",
		Prefix:  match[1],
		Suffix:  match[3],
		Text:    strings.TrimSpace(match[4]),
	}, true
}

func (line ParsedTaskLine) statusMarker() string {
	if line.Checked {
		return "x"
	}
	return " "
}

func taskTitleFromTodoText(text string) string {
	title := cleanTaskTitleText(text)
	if title == "" {
		title = strings.TrimSpace(text)
	}
	if title == "" {
		return "Untitled task"
	}
	return title
}

func cleanTaskTitleText(text string) string {
	value := strings.TrimSpace(text)
	replacements := []struct {
		pattern *regexp.Regexp
		replace string
	}{
		{regexp.MustCompile(`!\[\[([^\]]+)\]\]`), "$1"},
		{regexp.MustCompile(`\[\[([^\]]+)\]\]`), "$1"},
		{regexp.MustCompile(`!\[([^\]]*)\]\([^)]+\)`), "$1"},
		{regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`), "$1"},
		{regexp.MustCompile("[`*_~>]+"), " "},
		{regexp.MustCompile(`\s+`), " "},
	}
	for _, item := range replacements {
		value = item.pattern.ReplaceAllString(value, item.replace)
	}
	return strings.TrimSpace(value)
}
