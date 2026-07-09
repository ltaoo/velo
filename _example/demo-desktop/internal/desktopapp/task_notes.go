package desktopapp

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var taskLinePattern = regexp.MustCompile(`^(\s*[-*]\s+\[)([ xX])(\]\s+)(.*)$`)
var taskProjectDirectivePattern = regexp.MustCompile(`(?:^|\s)/project:([^\s]+)`)

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

type ParsedTaskText struct {
	DueAt       string
	Notes       string
	ProjectName string
	Tags        []string
	Title       string
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
	originalTags := extractMemoTags(memo.Content)
	if err := syncMemoTaskLines(ctx, &memo); err != nil {
		return TaskRecord{}, MemoRecord{}, err
	}
	memo.Tags = uniqueStrings(append(extractMemoTags(memo.Content), originalTags...))
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
	if memoLineIndexInCodeBlock(lines, req.LineIndex) {
		return TaskRecord{}, TaskRecord{}, MemoRecord{}, fmt.Errorf("line is inside a code block")
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
	metadata := parseTaskMetadataFromTodoText(parsed.Text)
	child, err := createVaultTask(ctx, TaskCreateRequest{
		Contexts:  parent.Contexts,
		DueAt:     metadata.DueAt,
		ListID:    parent.ListID,
		Notes:     metadata.Notes,
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
		Tags:  uniqueStrings(append(parent.Tags, metadata.Tags...)),
		Title: metadata.Title,
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
		lines[req.LineIndex] = parsed.Prefix + parsed.statusMarker() + parsed.Suffix + taskReferenceMarkdown(child)
		memo.Content = strings.Join(lines, "\n")
		memo.Tags = uniqueStrings(append(extractMemoTags(memo.Content), metadata.Tags...))
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
	title := parseTaskMetadataFromTodoText(text).Title
	if title == "" {
		title = strings.TrimSpace(text)
	}
	if title == "" {
		return "Untitled task"
	}
	return title
}

func parseTaskMetadataFromTodoText(text string) ParsedTaskText {
	tags := extractMemoTags(text)
	dueAt := ""
	explicitTitle := ""
	projectName := ""
	withoutTitleSyntax := rewriteTaskTextOutsideInlineCode(text, func(segment string) string {
		if explicitTitle == "" {
			if next, title, ok := removeTaskTitlePropertySyntax(segment); ok {
				explicitTitle = title
				segment = next
			}
		}
		if explicitTitle == "" {
			if next, title, ok := removeTaskTitlePrefixSyntax(segment); ok {
				explicitTitle = title
				segment = next
			}
		}
		return segment
	})
	titleSource := rewriteTaskTextOutsideInlineCode(withoutTitleSyntax, func(segment string) string {
		withoutDates, foundDueAt := removeTaskDateSyntax(segment)
		if dueAt == "" {
			dueAt = foundDueAt
		}
		withoutProject, foundProject := removeTaskProjectDirective(withoutDates)
		if projectName == "" {
			projectName = foundProject
		}
		return removeTaskTagSyntax(withoutProject)
	})
	notes := ""
	title := cleanTaskTitleText(explicitTitle)
	if title != "" {
		notes = cleanTaskTitleText(titleSource)
		if notes == title {
			notes = ""
		}
	} else {
		title = cleanTaskTitleText(titleSource)
	}
	if title == "" {
		title = cleanTaskTitleText(text)
	}
	if title == "" {
		title = "Untitled task"
	}
	return ParsedTaskText{
		DueAt:       dueAt,
		Notes:       notes,
		ProjectName: projectName,
		Tags:        tags,
		Title:       title,
	}
}

var taskTitlePropertyPattern = regexp.MustCompile(`(?i)(^|[\s([{（【「『])\{(title|标题)\s*[:=：]\s*([^{}]{1,120})\}`)

func removeTaskTitlePropertySyntax(text string) (string, string, bool) {
	match := taskTitlePropertyPattern.FindStringSubmatchIndex(text)
	if len(match) < 8 {
		return text, "", false
	}
	title := trimTaskTitleValue(text[match[6]:match[7]])
	if title == "" {
		return text, "", false
	}
	leading := text[match[2]:match[3]]
	next := text[:match[0]] + leading + text[match[1]:]
	return next, title, true
}

func removeTaskTitlePrefixSyntax(text string) (string, string, bool) {
	leading := text[:len(text)-len(strings.TrimLeft(text, " \t"))]
	rest := strings.TrimLeft(text, " \t")
	pairs := []struct {
		open       string
		close      string
		requireGap bool
	}{
		{open: "[", close: "]", requireGap: true},
		{open: "【", close: "】"},
		{open: "「", close: "」"},
		{open: "『", close: "』"},
	}
	for _, pair := range pairs {
		if !strings.HasPrefix(rest, pair.open) {
			continue
		}
		closeIndex := strings.Index(rest[len(pair.open):], pair.close)
		if closeIndex < 0 {
			continue
		}
		titleEnd := len(pair.open) + closeIndex
		title := trimTaskTitleValue(rest[len(pair.open):titleEnd])
		if title == "" {
			continue
		}
		after := rest[titleEnd+len(pair.close):]
		if pair.requireGap && !taskTitlePrefixHasGap(after) {
			continue
		}
		return leading + strings.TrimLeft(after, " \t"), title, true
	}
	return text, "", false
}

func trimTaskTitleValue(value string) string {
	title := strings.TrimSpace(value)
	title = strings.Trim(title, `"'“”‘’`)
	return strings.TrimSpace(title)
}

func taskTitlePrefixHasGap(after string) bool {
	if after == "" {
		return true
	}
	if strings.HasPrefix(after, "(") || strings.HasPrefix(after, "[") {
		return false
	}
	return len(after) != len(strings.TrimLeft(after, " \t:-|,.;，。：；、"))
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

func taskReferenceMarkdown(task TaskRecord) string {
	return "[[task:" + task.ID + "|" + taskReferenceAlias(task.Title) + "]]"
}

func taskReferenceAlias(value string) string {
	return strings.TrimSpace(strings.NewReplacer(
		"\r", " ",
		"\n", " ",
		"|", " ",
		"]", " ",
		"[", " ",
	).Replace(value))
}

func rewriteTaskTextOutsideInlineCode(text string, rewrite func(string) string) string {
	value := strings.TrimSpace(text)
	if value == "" {
		return ""
	}
	var output strings.Builder
	for index := 0; index < len(value); {
		if value[index] != '`' {
			next := strings.IndexByte(value[index:], '`')
			if next < 0 {
				output.WriteString(rewrite(value[index:]))
				break
			}
			output.WriteString(rewrite(value[index : index+next]))
			index += next
			continue
		}
		runStart := index
		for index < len(value) && value[index] == '`' {
			index++
		}
		delimiter := value[runStart:index]
		closeIndex := strings.Index(value[index:], delimiter)
		if closeIndex < 0 {
			output.WriteString(value[index:])
			break
		}
		codeStart := index
		codeEnd := index + closeIndex
		output.WriteString(value[codeStart:codeEnd])
		index = codeEnd + len(delimiter)
	}
	return output.String()
}

func removeTaskTagSyntax(text string) string {
	return memoTagPattern.ReplaceAllString(text, " ")
}

func removeTaskProjectDirective(text string) (string, string) {
	match := taskProjectDirectivePattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return text, ""
	}
	cleaned := taskProjectDirectivePattern.ReplaceAllString(text, " ")
	return cleaned, strings.TrimSpace(match[1])
}

func removeTaskDateSyntax(text string) (string, string) {
	dueAt := ""
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(^|[\s([{（【「『])(::)((?:\d{4}(?:[-/]\d{1,2}(?:[-/]\d{1,2}(?:[ T]\d{1,2}:\d{2}(?::\d{2})?)?)?)?)|(?:\d{1,2}:\d{2}(?::\d{2})?)|(?:[^\s<>()\[\]{}，。！？、；;,.]{1,32}))`),
		regexp.MustCompile(`(^|[\s([{（【「『])(\d{4}[-/]\d{1,2}[-/]\d{1,2}(?:[ T]\d{1,2}:\d{2}(?::\d{2})?)?)`),
	}
	value := text
	for _, pattern := range patterns {
		value = pattern.ReplaceAllStringFunc(value, func(match string) string {
			parts := pattern.FindStringSubmatch(match)
			if len(parts) == 0 {
				return match
			}
			raw := parts[len(parts)-1]
			if dueAt == "" {
				dueAt = normalizeTaskInlineDate(raw)
			}
			return parts[1]
		})
	}
	return value, dueAt
}

func normalizeTaskInlineDate(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}
	if t := parseMemoTime(raw); !t.IsZero() {
		return t.Format(time.RFC3339Nano)
	}
	normalized := strings.ReplaceAll(raw, "/", "-")
	for _, layout := range []string{
		"2006-1-2 15:04:05",
		"2006-1-2 15:04",
		"2006-1-2T15:04:05",
		"2006-1-2T15:04",
	} {
		if t, err := time.ParseInLocation(layout, normalized, time.Local); err == nil {
			return t.Format(time.RFC3339Nano)
		}
	}
	if dateOnlyPattern := regexp.MustCompile(`^(\d{4})-(\d{1,2})-(\d{1,2})$`); dateOnlyPattern.MatchString(normalized) {
		parts := dateOnlyPattern.FindStringSubmatch(normalized)
		return parts[1] + "-" + leftPadTaskDatePart(parts[2]) + "-" + leftPadTaskDatePart(parts[3])
	}
	return raw
}

func leftPadTaskDatePart(value string) string {
	if len(value) >= 2 {
		return value
	}
	return "0" + value
}
