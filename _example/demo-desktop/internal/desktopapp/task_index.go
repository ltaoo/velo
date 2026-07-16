package desktopapp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type TaskIndexFile struct {
	RebuiltAt     string                    `json:"rebuiltAt"`
	SchemaVersion int                       `json:"schemaVersion"`
	Tasks         map[string]TaskIndexEntry `json:"tasks"`
}

type TaskIndexEntry struct {
	CompletedAt  string     `json:"completedAt,omitempty"`
	CreatedAt    string     `json:"createdAt"`
	DueAt        string     `json:"dueAt,omitempty"`
	ID           string     `json:"id"`
	ListID       string     `json:"listId"`
	Contexts     []string   `json:"contexts"`
	NoteCount    int        `json:"noteCount"`
	ParentID     string     `json:"parentId,omitempty"`
	Path         string     `json:"path"`
	Priority     string     `json:"priority"`
	Private      bool       `json:"private"`
	ProjectID    string     `json:"projectId,omitempty"`
	Source       TaskSource `json:"source"`
	StartAt      string     `json:"startAt,omitempty"`
	Status       string     `json:"status"`
	SubtaskCount int        `json:"subtaskCount"`
	Tags         []string   `json:"tags"`
	Title        string     `json:"title"`
	UpdatedAt    string     `json:"updatedAt"`
	Visibility   string     `json:"visibility"`
}

type TaskEvent struct {
	At     string                 `json:"at"`
	Data   map[string]interface{} `json:"data,omitempty"`
	ID     string                 `json:"id"`
	TaskID string                 `json:"taskId"`
	Type   string                 `json:"type"`
}

func rebuildTaskIndex(ctx *VaultContext) (TaskIndexFile, error) {
	tasks, err := listVaultTasks(ctx)
	if err != nil {
		return TaskIndexFile{}, err
	}
	index := TaskIndexFile{
		RebuiltAt:     time.Now().UTC().Format(time.RFC3339Nano),
		SchemaVersion: vaultSchemaVersion,
		Tasks:         map[string]TaskIndexEntry{},
	}
	for _, task := range tasks {
		index.Tasks[task.ID] = taskIndexEntry(task)
	}
	if err := writeJSONFileAtomic(taskIndexPath(ctx), index); err != nil {
		return TaskIndexFile{}, err
	}
	return index, nil
}

func loadTaskIndex(ctx *VaultContext) (TaskIndexFile, error) {
	raw, err := os.ReadFile(taskIndexPath(ctx))
	if os.IsNotExist(err) {
		return rebuildTaskIndex(ctx)
	}
	if err != nil {
		return TaskIndexFile{}, err
	}
	var index TaskIndexFile
	if err := json.Unmarshal(raw, &index); err != nil {
		return rebuildTaskIndex(ctx)
	}
	if index.SchemaVersion == 0 || index.Tasks == nil {
		return rebuildTaskIndex(ctx)
	}
	if taskIndexMissingSourceField(raw) {
		return rebuildTaskIndex(ctx)
	}
	return index, nil
}

func taskIndexMissingSourceField(raw []byte) bool {
	var payload struct {
		Tasks map[string]map[string]json.RawMessage `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return false
	}
	for _, entry := range payload.Tasks {
		if _, ok := entry["source"]; !ok {
			return true
		}
	}
	return false
}

func taskIndexEntries(index TaskIndexFile) []TaskIndexEntry {
	entries := make([]TaskIndexEntry, 0, len(index.Tasks))
	for _, entry := range index.Tasks {
		entries = append(entries, entry)
	}
	sort.SliceStable(entries, func(i, j int) bool {
		left := parseMemoTime(firstNonEmpty(entries[i].UpdatedAt, entries[i].CreatedAt))
		right := parseMemoTime(firstNonEmpty(entries[j].UpdatedAt, entries[j].CreatedAt))
		if left.Equal(right) {
			return entries[i].ID > entries[j].ID
		}
		return left.After(right)
	})
	return entries
}

func taskIndexEntry(task TaskRecord) TaskIndexEntry {
	return TaskIndexEntry{
		CompletedAt:  task.CompletedAt,
		CreatedAt:    task.CreatedAt,
		DueAt:        task.DueAt,
		ID:           task.ID,
		ListID:       task.ListID,
		Contexts:     task.Contexts,
		NoteCount:    len(task.NoteRefs),
		ParentID:     task.ParentID,
		Path:         task.Path,
		Priority:     task.Priority,
		Private:      task.Private,
		ProjectID:    task.ProjectID,
		Source:       task.Source,
		StartAt:      task.StartAt,
		Status:       task.Status,
		SubtaskCount: len(task.SubtaskIDs),
		Tags:         task.Tags,
		Title:        task.Title,
		UpdatedAt:    task.UpdatedAt,
		Visibility:   task.Visibility,
	}
}

func appendTaskEvent(ctx *VaultContext, taskID string, eventType string, data map[string]interface{}) error {
	now := time.Now().UTC()
	event := TaskEvent{
		At:     now.Format(time.RFC3339Nano),
		Data:   data,
		ID:     "evt_" + now.Format("20060102T150405") + "_" + randomVaultSuffix(),
		TaskID: sanitizeTaskID(taskID),
		Type:   eventType,
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(taskEventsDir(ctx), 0755); err != nil {
		return err
	}
	path := filepath.Join(taskEventsDir(ctx), now.Format("2006-01")+".jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.Write(append(raw, '\n')); err != nil {
		return err
	}
	return nil
}
