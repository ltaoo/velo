package desktopapp

import (
	"regexp"
	"strings"
	"time"
)

var memoTaskRefPattern = regexp.MustCompile(`\[\[task:([A-Za-z0-9_-]+)(?:\|[^\]]*)?\]\]`)

type taskLineSyncSource struct {
	CommentID    string
	CommentPath  string
	MemoID       string
	MemoPath     string
	ParentTaskID string
	ProjectID    string
	Type         string
}

func syncMemoTaskLines(ctx *VaultContext, memo *MemoRecord) error {
	if memo == nil {
		return nil
	}
	source := taskLineSyncSource{
		MemoID:    memo.ID,
		MemoPath:  memo.Path,
		ProjectID: memo.ProjectID,
		Type:      "memo",
	}
	if memo.Kind == "task_note" && memo.TaskID != "" {
		source.ParentTaskID = memo.TaskID
	}
	content, changed, err := syncTaskLinesForSource(ctx, memo.Content, source)
	if err != nil {
		return err
	}
	if changed {
		memo.Content = content
	}
	return nil
}

func syncMemoCommentTaskLines(ctx *VaultContext, comment *MemoCommentRecord, memo MemoRecord) error {
	if comment == nil {
		return nil
	}
	source := taskLineSyncSource{
		CommentID:   comment.ID,
		CommentPath: comment.Path,
		MemoID:      memo.ID,
		MemoPath:    memo.Path,
		ProjectID:   memo.ProjectID,
		Type:        "comment",
	}
	if memo.Kind == "task_note" && memo.TaskID != "" {
		source.ParentTaskID = memo.TaskID
	}
	content, changed, err := syncTaskLinesForSource(ctx, comment.Content, source)
	if err != nil {
		return err
	}
	if changed {
		comment.Content = content
	}
	return nil
}

func syncTaskLinesForSource(ctx *VaultContext, content string, source taskLineSyncSource) (string, bool, error) {
	lines := strings.Split(normalizeMemoContent(content), "\n")
	changed := false
	parentChanged := false
	var parent TaskRecord
	var parentLoaded bool
	var parentOK bool
	inCode := false
	var activeFence memoCodeFence

	for index, line := range lines {
		fence, hasFence := parseMemoCodeFenceLine(line)
		if inCode {
			if hasFence && memoCodeFenceCloses(fence, activeFence) {
				inCode = false
			}
			continue
		}
		if hasFence {
			activeFence = fence
			inCode = true
			continue
		}
		parsed, ok := parseTaskLineText(line)
		if !ok || strings.TrimSpace(parsed.Text) == "" {
			continue
		}
		if taskID := memoTaskRefID(parsed.Text); taskID != "" {
			if err := syncExistingTaskLineState(ctx, taskID, parsed.Checked); err != nil {
				return "", false, err
			}
			continue
		}

		metadata := parseTaskMetadataFromTodoText(parsed.Text)
		taskProjectID := source.ProjectID
		if metadata.ProjectName != "" {
			resolved, err := resolveOrCreateProjectByName(ctx, metadata.ProjectName)
			if err == nil && resolved != "" {
				taskProjectID = resolved
			}
		}
		req := TaskCreateRequest{
			DueAt:     metadata.DueAt,
			Notes:     metadata.Notes,
			ProjectID: taskProjectID,
			Source:    taskSourceForLine(source, index+1, line),
			Tags:      metadata.Tags,
			Title:     metadata.Title,
		}
		if source.ParentTaskID != "" {
			if !parentLoaded {
				var err error
				parent, err = getVaultTask(ctx, source.ParentTaskID)
				parentOK = err == nil
				parentLoaded = true
			}
			if parentOK {
				req.Contexts = parent.Contexts
				req.ListID = parent.ListID
				req.ParentID = parent.ID
				req.Priority = parent.Priority
				req.ProjectID = firstNonEmpty(parent.ProjectID, source.ProjectID)
				req.Tags = uniqueStrings(append(parent.Tags, req.Tags...))
			}
		}

		task, err := createVaultTask(ctx, req)
		if err != nil {
			return "", false, err
		}
		if parsed.Checked {
			task, err = completeVaultTask(ctx, task.ID)
			if err != nil {
				return "", false, err
			}
		}
		if parentOK {
			parent = addTaskSubtaskID(parent, task.ID)
			parent.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			parentChanged = true
		}
		lines[index] = parsed.Prefix + parsed.statusMarker() + parsed.Suffix + taskReferenceMarkdown(task)
		changed = true
	}

	if parentChanged {
		if err := writeTaskRecord(ctx, parent); err != nil {
			return "", false, err
		}
		_, _ = rebuildTaskIndex(ctx)
	}
	return strings.Join(lines, "\n"), changed, nil
}

func taskSourceForLine(source taskLineSyncSource, lineNumber int, text string) TaskSource {
	return TaskSource{
		CommentID:   source.CommentID,
		CommentPath: source.CommentPath,
		Line:        lineNumber,
		MemoID:      source.MemoID,
		MemoPath:    source.MemoPath,
		Text:        text,
		Type:        firstNonEmpty(source.Type, "memo"),
	}
}

func memoTaskRefID(text string) string {
	match := memoTaskRefPattern.FindStringSubmatch(maskMemoInlineCode(text))
	if len(match) < 2 {
		return ""
	}
	return sanitizeTaskID(match[1])
}

func syncExistingTaskLineState(ctx *VaultContext, taskID string, checked bool) error {
	task, err := getVaultTask(ctx, taskID)
	if err != nil {
		return nil
	}
	if checked && task.Status != taskStatusCompleted {
		_, err := completeVaultTask(ctx, taskID)
		return err
	}
	if !checked && task.Status == taskStatusCompleted {
		status := taskStatusOpen
		completedAt := ""
		_, err := updateVaultTask(ctx, TaskUpdateRequest{
			ID:          taskID,
			Status:      &status,
			CompletedAt: &completedAt,
		})
		return err
	}
	return nil
}
