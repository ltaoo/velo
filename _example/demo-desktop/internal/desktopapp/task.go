package desktopapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const vaultTaskDirName = "tasks"
const vaultTaskIndexFileName = "task-index.json"
const vaultTaskEventsDirName = "task-events"

const (
	taskStatusOpen      = "open"
	taskStatusCompleted = "completed"
	taskStatusCancelled = "cancelled"
	taskStatusArchived  = "archived"
)

const (
	taskPriorityNone   = "none"
	taskPriorityLow    = "low"
	taskPriorityMedium = "medium"
	taskPriorityHigh   = "high"
)

type TaskRecord struct {
	CancelledAt   string         `json:"cancelledAt,omitempty"`
	CompletedAt   string         `json:"completedAt,omitempty"`
	Contexts      []string       `json:"contexts"`
	CreatedAt     string         `json:"createdAt"`
	DueAt         string         `json:"dueAt,omitempty"`
	ID            string         `json:"id"`
	Links         []TaskLink     `json:"links"`
	ListID        string         `json:"listId"`
	Notes         string         `json:"notes"`
	NoteRefs      []TaskNoteRef  `json:"noteRefs"`
	ParentID      string         `json:"parentId,omitempty"`
	Path          string         `json:"path"`
	Priority      string         `json:"priority"`
	ProjectID     string         `json:"projectId,omitempty"`
	Reminders     []TaskReminder `json:"reminders"`
	Repeat        TaskRepeat     `json:"repeat"`
	Source        TaskSource     `json:"source"`
	SchemaVersion int            `json:"schemaVersion"`
	StartAt       string         `json:"startAt,omitempty"`
	Status        string         `json:"status"`
	SubtaskIDs    []string       `json:"subtaskIds"`
	Tags          []string       `json:"tags"`
	Timezone      string         `json:"timezone"`
	Title         string         `json:"title"`
	UpdatedAt     string         `json:"updatedAt"`
}

type TaskReminder struct {
	At            string `json:"at,omitempty"`
	Base          string `json:"base,omitempty"`
	OffsetMinutes int    `json:"offsetMinutes,omitempty"`
	Type          string `json:"type"`
}

type TaskRepeat struct {
	End       TaskRepeatEnd `json:"end,omitempty"`
	Frequency string        `json:"frequency"`
	Interval  int           `json:"interval,omitempty"`
	Weekdays  []string      `json:"weekdays,omitempty"`
}

type TaskRepeatEnd struct {
	At    string `json:"at,omitempty"`
	Count int    `json:"count,omitempty"`
	Type  string `json:"type,omitempty"`
}

type TaskSource struct {
	Line     int    `json:"line,omitempty"`
	MemoID   string `json:"memoId,omitempty"`
	MemoPath string `json:"memoPath,omitempty"`
	Text     string `json:"text,omitempty"`
	Type     string `json:"type,omitempty"`
}

type TaskLink struct {
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
	Type  string `json:"type"`
	URL   string `json:"url,omitempty"`
}

type TaskNoteRef struct {
	CreatedAt string `json:"createdAt"`
	MemoID    string `json:"memoId"`
	Role      string `json:"role"`
	SortOrder int    `json:"sortOrder"`
}

type TaskCreateRequest struct {
	Contexts  []string       `json:"contexts"`
	DueAt     string         `json:"dueAt"`
	Links     []TaskLink     `json:"links"`
	ListID    string         `json:"listId"`
	Notes     string         `json:"notes"`
	NoteRefs  []TaskNoteRef  `json:"noteRefs"`
	ParentID  string         `json:"parentId"`
	Priority  string         `json:"priority"`
	ProjectID string         `json:"projectId"`
	Reminders []TaskReminder `json:"reminders"`
	Repeat    TaskRepeat     `json:"repeat"`
	Source    TaskSource     `json:"source"`
	StartAt   string         `json:"startAt"`
	Tags      []string       `json:"tags"`
	Timezone  string         `json:"timezone"`
	Title     string         `json:"title"`
}

type TaskUpdateRequest struct {
	CancelledAt *string         `json:"cancelledAt"`
	CompletedAt *string         `json:"completedAt"`
	Contexts    *[]string       `json:"contexts"`
	DueAt       *string         `json:"dueAt"`
	ID          string          `json:"id"`
	Links       *[]TaskLink     `json:"links"`
	ListID      *string         `json:"listId"`
	Notes       *string         `json:"notes"`
	NoteRefs    *[]TaskNoteRef  `json:"noteRefs"`
	ParentID    *string         `json:"parentId"`
	Priority    *string         `json:"priority"`
	ProjectID   *string         `json:"projectId"`
	Reminders   *[]TaskReminder `json:"reminders"`
	Repeat      *TaskRepeat     `json:"repeat"`
	Source      *TaskSource     `json:"source"`
	StartAt     *string         `json:"startAt"`
	Status      *string         `json:"status"`
	SubtaskIDs  *[]string       `json:"subtaskIds"`
	Tags        *[]string       `json:"tags"`
	Timezone    *string         `json:"timezone"`
	Title       *string         `json:"title"`
}

type TaskIDRequest struct {
	ID string `json:"id"`
}

func taskRootDir(ctx *VaultContext) string {
	return filepath.Join(ctx.RootDir, vaultTaskDirName)
}

func taskIndexPath(ctx *VaultContext) string {
	return filepath.Join(ctx.VeloDir, vaultTaskIndexFileName)
}

func taskEventsDir(ctx *VaultContext) string {
	return filepath.Join(ctx.VeloDir, vaultTaskEventsDirName)
}

func listVaultTasks(ctx *VaultContext) ([]TaskRecord, error) {
	tasks := []TaskRecord{}
	root := taskRootDir(ctx)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return tasks, nil
	} else if err != nil {
		return nil, err
	}
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
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
		tasks = append(tasks, task)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.SliceStable(tasks, func(i, j int) bool {
		return taskSortKey(tasks[i]).After(taskSortKey(tasks[j]))
	})
	return tasks, nil
}

func createVaultTask(ctx *VaultContext, req TaskCreateRequest) (TaskRecord, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return TaskRecord{}, fmt.Errorf("task title is required")
	}
	projectID, err := validateMemoProjectID(ctx, req.ProjectID)
	if err != nil {
		return TaskRecord{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	task := TaskRecord{
		Contexts:      normalizeTaskLabels(req.Contexts),
		CreatedAt:     now,
		DueAt:         normalizeTaskTime(req.DueAt),
		ID:            newTaskID(),
		Links:         normalizeTaskLinks(req.Links),
		ListID:        normalizeTaskListID(req.ListID),
		Notes:         normalizeMemoContent(req.Notes),
		NoteRefs:      normalizeTaskNoteRefs(req.NoteRefs),
		ParentID:      sanitizeTaskID(req.ParentID),
		Priority:      normalizeTaskPriority(req.Priority),
		ProjectID:     projectID,
		Reminders:     normalizeTaskReminders(req.Reminders),
		Repeat:        normalizeTaskRepeat(req.Repeat),
		Source:        normalizeTaskSource(req.Source),
		SchemaVersion: vaultSchemaVersion,
		StartAt:       normalizeTaskTime(req.StartAt),
		Status:        taskStatusOpen,
		SubtaskIDs:    []string{},
		Tags:          normalizeTaskLabels(req.Tags),
		Timezone:      normalizeTaskTimezone(req.Timezone),
		Title:         title,
		UpdatedAt:     now,
	}
	task.Path = taskRelativePath(task)
	if err := writeTaskRecord(ctx, task); err != nil {
		return TaskRecord{}, err
	}
	_ = appendTaskEvent(ctx, task.ID, "created", nil)
	_, _ = rebuildTaskIndex(ctx)
	return task, nil
}

func getVaultTask(ctx *VaultContext, id string) (TaskRecord, error) {
	path, err := findTaskFilePath(ctx, id)
	if err != nil {
		return TaskRecord{}, err
	}
	return readTaskFile(ctx, path)
}

func updateVaultTask(ctx *VaultContext, req TaskUpdateRequest) (TaskRecord, error) {
	id := sanitizeTaskID(req.ID)
	if id == "" {
		return TaskRecord{}, fmt.Errorf("task id is required")
	}
	path, err := findTaskFilePath(ctx, id)
	if err != nil {
		return TaskRecord{}, err
	}
	task, err := readTaskFile(ctx, path)
	if err != nil {
		return TaskRecord{}, err
	}
	oldPath := path
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return TaskRecord{}, fmt.Errorf("task title is required")
		}
		task.Title = title
	}
	if req.Status != nil {
		nextStatus := normalizeTaskStatus(*req.Status)
		if task.Status != nextStatus {
			task.Status = nextStatus
			now := time.Now().UTC().Format(time.RFC3339Nano)
			if nextStatus == taskStatusCompleted && task.CompletedAt == "" {
				task.CompletedAt = now
			}
			if nextStatus == taskStatusCancelled && task.CancelledAt == "" {
				task.CancelledAt = now
			}
		}
	}
	if req.ListID != nil {
		task.ListID = normalizeTaskListID(*req.ListID)
	}
	if req.ProjectID != nil {
		projectID, err := validateMemoProjectID(ctx, *req.ProjectID)
		if err != nil {
			return TaskRecord{}, err
		}
		task.ProjectID = projectID
	}
	if req.Priority != nil {
		task.Priority = normalizeTaskPriority(*req.Priority)
	}
	if req.Tags != nil {
		task.Tags = normalizeTaskLabels(*req.Tags)
	}
	if req.Contexts != nil {
		task.Contexts = normalizeTaskLabels(*req.Contexts)
	}
	if req.StartAt != nil {
		task.StartAt = normalizeTaskTime(*req.StartAt)
	}
	if req.DueAt != nil {
		task.DueAt = normalizeTaskTime(*req.DueAt)
	}
	if req.Timezone != nil {
		task.Timezone = normalizeTaskTimezone(*req.Timezone)
	}
	if req.Reminders != nil {
		task.Reminders = normalizeTaskReminders(*req.Reminders)
	}
	if req.Repeat != nil {
		task.Repeat = normalizeTaskRepeat(*req.Repeat)
	}
	if req.ParentID != nil {
		task.ParentID = sanitizeTaskID(*req.ParentID)
	}
	if req.SubtaskIDs != nil {
		task.SubtaskIDs = normalizeTaskIDs(*req.SubtaskIDs)
	}
	if req.Source != nil {
		task.Source = normalizeTaskSource(*req.Source)
	}
	if req.Links != nil {
		task.Links = normalizeTaskLinks(*req.Links)
	}
	if req.Notes != nil {
		task.Notes = normalizeMemoContent(*req.Notes)
	}
	if req.NoteRefs != nil {
		task.NoteRefs = normalizeTaskNoteRefs(*req.NoteRefs)
	}
	if req.CompletedAt != nil {
		task.CompletedAt = normalizeTaskTime(*req.CompletedAt)
	}
	if req.CancelledAt != nil {
		task.CancelledAt = normalizeTaskTime(*req.CancelledAt)
	}
	task.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	task.Path = taskRelativePath(task)
	if err := writeTaskRecord(ctx, task); err != nil {
		return TaskRecord{}, err
	}
	if oldPath != filepath.Join(ctx.RootDir, filepath.FromSlash(task.Path)) {
		_ = os.Remove(oldPath)
	}
	_ = appendTaskEvent(ctx, task.ID, "updated", nil)
	_, _ = rebuildTaskIndex(ctx)
	return task, nil
}

func completeVaultTask(ctx *VaultContext, id string) (TaskRecord, error) {
	status := taskStatusCompleted
	return updateVaultTask(ctx, TaskUpdateRequest{ID: id, Status: &status})
}

func deleteVaultTask(ctx *VaultContext, id string) error {
	id = sanitizeTaskID(id)
	if id == "" {
		return fmt.Errorf("task id is required")
	}
	path, err := findTaskFilePath(ctx, id)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	_ = appendTaskEvent(ctx, id, "deleted", nil)
	_, _ = rebuildTaskIndex(ctx)
	return nil
}

func taskSortKey(task TaskRecord) time.Time {
	for _, value := range []string{task.UpdatedAt, task.CreatedAt} {
		if t := parseMemoTime(value); !t.IsZero() {
			return t
		}
	}
	return time.Time{}
}

func newTaskID() string {
	return "task_" + time.Now().UTC().Format("20060102T150405") + "_" + randomVaultSuffix()
}

func sanitizeTaskID(value string) string {
	return sanitizeProjectID(value)
}

func normalizeTaskIDs(values []string) []string {
	seen := map[string]bool{}
	next := []string{}
	for _, value := range values {
		id := sanitizeTaskID(value)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		next = append(next, id)
	}
	return next
}

func normalizeTaskStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case taskStatusCompleted:
		return taskStatusCompleted
	case taskStatusCancelled:
		return taskStatusCancelled
	case taskStatusArchived:
		return taskStatusArchived
	default:
		return taskStatusOpen
	}
}

func normalizeTaskPriority(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case taskPriorityLow:
		return taskPriorityLow
	case taskPriorityMedium:
		return taskPriorityMedium
	case taskPriorityHigh:
		return taskPriorityHigh
	default:
		return taskPriorityNone
	}
}

func normalizeTaskListID(value string) string {
	id := sanitizeProjectID(value)
	if id == "" {
		return "inbox"
	}
	return id
}

func normalizeTaskLabels(values []string) []string {
	return uniqueStrings(values)
}

func normalizeTaskTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if t := parseMemoTime(value); !t.IsZero() {
		return t.Format(time.RFC3339Nano)
	}
	return value
}

func normalizeTaskTimezone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "UTC"
	}
	return value
}

func normalizeTaskReminders(reminders []TaskReminder) []TaskReminder {
	next := []TaskReminder{}
	for _, reminder := range reminders {
		reminder.Type = strings.ToLower(strings.TrimSpace(reminder.Type))
		if reminder.Type == "" {
			continue
		}
		reminder.At = normalizeTaskTime(reminder.At)
		reminder.Base = strings.TrimSpace(reminder.Base)
		next = append(next, reminder)
	}
	return next
}

func normalizeTaskRepeat(repeat TaskRepeat) TaskRepeat {
	repeat.Frequency = strings.ToLower(strings.TrimSpace(repeat.Frequency))
	if repeat.Frequency == "" {
		repeat.Frequency = "none"
	}
	if repeat.Interval < 0 {
		repeat.Interval = 0
	}
	repeat.Weekdays = normalizeTaskLabels(repeat.Weekdays)
	repeat.End.Type = strings.ToLower(strings.TrimSpace(repeat.End.Type))
	repeat.End.At = normalizeTaskTime(repeat.End.At)
	return repeat
}

func normalizeTaskSource(source TaskSource) TaskSource {
	source.Type = strings.ToLower(strings.TrimSpace(source.Type))
	source.MemoID = strings.TrimSpace(source.MemoID)
	source.MemoPath = cleanOSSObjectPath(source.MemoPath)
	source.Text = strings.TrimSpace(source.Text)
	if source.Type == "" && (source.MemoID != "" || source.MemoPath != "" || source.Text != "") {
		source.Type = "memo"
	}
	if source.Line < 0 {
		source.Line = 0
	}
	return source
}

func normalizeTaskLinks(links []TaskLink) []TaskLink {
	next := []TaskLink{}
	for _, link := range links {
		link.Type = strings.ToLower(strings.TrimSpace(link.Type))
		link.ID = strings.TrimSpace(link.ID)
		link.URL = strings.TrimSpace(link.URL)
		link.Label = strings.TrimSpace(link.Label)
		if link.Type == "" || (link.ID == "" && link.URL == "") {
			continue
		}
		next = append(next, link)
	}
	return next
}

func normalizeTaskRecord(task TaskRecord) TaskRecord {
	task.ID = sanitizeTaskID(task.ID)
	task.Status = normalizeTaskStatus(task.Status)
	task.Priority = normalizeTaskPriority(task.Priority)
	task.ListID = normalizeTaskListID(task.ListID)
	task.ProjectID = sanitizeProjectID(task.ProjectID)
	task.ParentID = sanitizeTaskID(task.ParentID)
	task.Contexts = normalizeTaskLabels(task.Contexts)
	task.Tags = normalizeTaskLabels(task.Tags)
	task.SubtaskIDs = normalizeTaskIDs(task.SubtaskIDs)
	task.StartAt = normalizeTaskTime(task.StartAt)
	task.DueAt = normalizeTaskTime(task.DueAt)
	task.CompletedAt = normalizeTaskTime(task.CompletedAt)
	task.CancelledAt = normalizeTaskTime(task.CancelledAt)
	task.CreatedAt = normalizeTaskTime(task.CreatedAt)
	task.UpdatedAt = normalizeTaskTime(task.UpdatedAt)
	task.Timezone = normalizeTaskTimezone(task.Timezone)
	task.Reminders = normalizeTaskReminders(task.Reminders)
	task.Repeat = normalizeTaskRepeat(task.Repeat)
	task.Source = normalizeTaskSource(task.Source)
	if task.SchemaVersion == 0 {
		task.SchemaVersion = vaultSchemaVersion
	}
	task.Links = normalizeTaskLinks(task.Links)
	task.NoteRefs = normalizeTaskNoteRefs(task.NoteRefs)
	task.Notes = normalizeMemoContent(task.Notes)
	task.Title = strings.TrimSpace(task.Title)
	return task
}

func normalizeTaskNoteRefs(refs []TaskNoteRef) []TaskNoteRef {
	next := []TaskNoteRef{}
	seen := map[string]bool{}
	for _, ref := range refs {
		ref.MemoID = strings.TrimSpace(ref.MemoID)
		if ref.MemoID == "" || seen[ref.MemoID] {
			continue
		}
		seen[ref.MemoID] = true
		ref.Role = strings.ToLower(strings.TrimSpace(ref.Role))
		if ref.Role == "" {
			ref.Role = "note"
		}
		ref.CreatedAt = normalizeTaskTime(ref.CreatedAt)
		next = append(next, ref)
	}
	sort.SliceStable(next, func(i, j int) bool {
		if next[i].SortOrder == next[j].SortOrder {
			return next[i].CreatedAt < next[j].CreatedAt
		}
		return next[i].SortOrder < next[j].SortOrder
	})
	return next
}

func addTaskNoteRef(task TaskRecord, ref TaskNoteRef) TaskRecord {
	task.NoteRefs = append(task.NoteRefs, ref)
	task.NoteRefs = normalizeTaskNoteRefs(task.NoteRefs)
	return task
}

func addTaskSubtaskID(task TaskRecord, subtaskID string) TaskRecord {
	task.SubtaskIDs = append(task.SubtaskIDs, subtaskID)
	task.SubtaskIDs = normalizeTaskIDs(task.SubtaskIDs)
	return task
}
