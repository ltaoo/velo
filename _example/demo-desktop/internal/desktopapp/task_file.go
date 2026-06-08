package desktopapp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func readTaskFile(ctx *VaultContext, path string) (TaskRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return TaskRecord{}, err
	}
	info, _ := os.Stat(path)
	var task TaskRecord
	if err := json.Unmarshal(raw, &task); err != nil {
		return TaskRecord{}, fmt.Errorf("read task: %w", err)
	}
	task = normalizeTaskRecord(task)
	if task.ID == "" {
		task.ID = sanitizeTaskID(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	}
	if task.CreatedAt == "" && info != nil {
		task.CreatedAt = info.ModTime().UTC().Format(time.RFC3339Nano)
	}
	if task.UpdatedAt == "" {
		task.UpdatedAt = task.CreatedAt
	}
	task.Path = relativeVaultPath(ctx, path)
	return task, nil
}

func writeTaskRecord(ctx *VaultContext, task TaskRecord) error {
	task = normalizeTaskRecord(task)
	if task.ID == "" {
		return fmt.Errorf("task id is required")
	}
	task.Path = taskRelativePath(task)
	target, err := safeVaultRelativePath(ctx.RootDir, task.Path)
	if err != nil {
		return err
	}
	root := taskRootDir(ctx)
	if !strings.HasPrefix(target, root+string(filepath.Separator)) && target != root {
		return fmt.Errorf("task path must be inside task directory")
	}
	return writeTaskJSONFileAtomic(target, task)
}

func writeTaskJSONFileAtomic(path string, task TaskRecord) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(bytes.TrimRight(raw, "\n"), '\n'), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func taskRelativePath(task TaskRecord) string {
	status := normalizeTaskStatus(task.Status)
	t := taskPathTime(task)
	id := sanitizeTaskID(task.ID)
	if id == "" {
		id = newTaskID()
	}
	switch status {
	case taskStatusCompleted, taskStatusCancelled, taskStatusArchived:
		return filepath.ToSlash(filepath.Join(
			vaultTaskDirName,
			status,
			fmt.Sprintf("%04d", t.Year()),
			id+".json",
		))
	default:
		return filepath.ToSlash(filepath.Join(
			vaultTaskDirName,
			taskStatusOpen,
			fmt.Sprintf("%04d", t.Year()),
			fmt.Sprintf("%02d", int(t.Month())),
			id+".json",
		))
	}
}

func taskPathTime(task TaskRecord) time.Time {
	for _, value := range []string{task.CompletedAt, task.CancelledAt, task.CreatedAt} {
		if t := parseMemoTime(value); !t.IsZero() {
			return t
		}
	}
	return time.Now()
}

func findTaskFilePath(ctx *VaultContext, id string) (string, error) {
	targetID := sanitizeTaskID(id)
	if targetID == "" {
		return "", fmt.Errorf("task id is required")
	}
	root := taskRootDir(ctx)
	var found string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if os.IsNotExist(err) {
			return filepath.SkipAll
		}
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			return nil
		}
		task, err := readTaskFile(ctx, path)
		if err != nil {
			return err
		}
		if task.ID == targetID {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("task not found: %s", targetID)
	}
	return found, nil
}
