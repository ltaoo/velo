package desktopapp

import (
	"os"
	"strings"
	"time"
)

func deleteVaultTasksForMemo(ctx *VaultContext, memoID string) (int, error) {
	memoID = strings.TrimSpace(memoID)
	if memoID == "" {
		return 0, nil
	}
	tasks, err := listVaultTasks(ctx)
	if err != nil {
		return 0, err
	}

	deleteIDs := map[string]bool{}
	for _, task := range tasks {
		if strings.TrimSpace(task.Source.MemoID) == memoID {
			deleteIDs[task.ID] = true
		}
	}
	if len(deleteIDs) == 0 {
		return 0, nil
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, task := range tasks {
		if deleteIDs[task.ID] || len(task.SubtaskIDs) == 0 {
			continue
		}
		next := make([]string, 0, len(task.SubtaskIDs))
		changed := false
		for _, subtaskID := range task.SubtaskIDs {
			if deleteIDs[subtaskID] {
				changed = true
				continue
			}
			next = append(next, subtaskID)
		}
		if !changed {
			continue
		}
		task.SubtaskIDs = next
		task.UpdatedAt = now
		if err := writeTaskRecord(ctx, task); err != nil {
			return 0, err
		}
	}

	deleted := 0
	for _, task := range tasks {
		if !deleteIDs[task.ID] {
			continue
		}
		path, err := findTaskFilePath(ctx, task.ID)
		if err != nil {
			return deleted, err
		}
		if err := os.Remove(path); err != nil {
			return deleted, err
		}
		deleted++
		_ = appendTaskEvent(ctx, task.ID, "deleted_with_memo", map[string]interface{}{
			"memoId": memoID,
		})
	}
	_, _ = rebuildTaskIndex(ctx)
	return deleted, nil
}
