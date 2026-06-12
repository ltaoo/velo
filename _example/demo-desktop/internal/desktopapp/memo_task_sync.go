package desktopapp

import (
	"regexp"
	"strings"
	"time"
)

var memoTaskRefPattern = regexp.MustCompile(`\[\[task:([A-Za-z0-9_-]+)(?:\|[^\]]*)?\]\]`)

func syncMemoTaskLines(ctx *VaultContext, memo *MemoRecord) error {
	if memo == nil {
		return nil
	}
	lines := strings.Split(normalizeMemoContent(memo.Content), "\n")
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
				return err
			}
			continue
		}

		metadata := parseTaskMetadataFromTodoText(parsed.Text)
		req := TaskCreateRequest{
			DueAt:     metadata.DueAt,
			Notes:     metadata.Notes,
			ProjectID: memo.ProjectID,
			Source: TaskSource{
				Line:     index + 1,
				MemoID:   memo.ID,
				MemoPath: memo.Path,
				Text:     line,
				Type:     "memo",
			},
			Tags:  metadata.Tags,
			Title: metadata.Title,
		}
		if memo.Kind == "task_note" && memo.TaskID != "" {
			if !parentLoaded {
				var err error
				parent, err = getVaultTask(ctx, memo.TaskID)
				parentOK = err == nil
				parentLoaded = true
			}
			if parentOK {
				req.Contexts = parent.Contexts
				req.ListID = parent.ListID
				req.ParentID = parent.ID
				req.Priority = parent.Priority
				req.ProjectID = firstNonEmpty(parent.ProjectID, memo.ProjectID)
				req.Tags = uniqueStrings(append(parent.Tags, req.Tags...))
			}
		}

		task, err := createVaultTask(ctx, req)
		if err != nil {
			return err
		}
		if parsed.Checked {
			task, err = completeVaultTask(ctx, task.ID)
			if err != nil {
				return err
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
			return err
		}
		_, _ = rebuildTaskIndex(ctx)
	}
	if changed {
		memo.Content = strings.Join(lines, "\n")
	}
	return nil
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
