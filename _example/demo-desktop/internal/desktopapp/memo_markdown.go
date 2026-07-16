package desktopapp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func readMemoFile(ctx *VaultContext, path string) (MemoRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return MemoRecord{}, err
	}
	info, _ := os.Stat(path)
	meta, content := parseMemoMarkdown(string(raw))
	createdAt := firstNonEmpty(meta["createdAt"], meta["created_at"])
	if createdAt == "" && info != nil {
		createdAt = info.ModTime().UTC().Format(time.RFC3339Nano)
	}
	id := strings.TrimSpace(meta["id"])
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	memo := MemoRecord{
		Archived:   parseMemoBool(meta["archived"]),
		Content:    normalizeStoredMemoContent(content, meta),
		CreatedAt:  createdAt,
		ID:         id,
		Kind:       strings.TrimSpace(meta["kind"]),
		Path:       relativeVaultPath(ctx, path),
		Pinned:     parseMemoBool(meta["pinned"]),
		Private:    parseMemoBool(meta["private"]),
		ProjectID:  sanitizeProjectID(meta["projectId"]),
		References: parseMemoList(meta, "references"),
		Tags:       parseMemoList(meta, "tags"),
		TaskID:     sanitizeTaskID(meta["taskId"]),
		UpdatedAt:  firstNonEmpty(meta["updatedAt"], meta["updated_at"]),
		Visibility: normalizeMemoVisibility(meta["visibility"]),
	}
	if len(memo.Tags) == 0 {
		memo.Tags = extractMemoTags(memo.Content)
	}
	if len(memo.References) == 0 {
		memo.References = extractMemoReferences(memo.Content)
	}
	return memo, nil
}

func writeMemoRecord(ctx *VaultContext, memo MemoRecord) error {
	if memo.ID == "" {
		return fmt.Errorf("memo id is required")
	}
	if memo.Path == "" {
		memo.Path = memoRelativePath(memo)
	}
	target, err := safeVaultRelativePath(ctx.RootDir, memo.Path)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(target, ctx.MemoDir+string(filepath.Separator)) && target != ctx.MemoDir {
		return fmt.Errorf("memo path must be inside memo directory")
	}
	return writeTextFileAtomic(target, renderMemoMarkdownFile(memo))
}

func writeTextFileAtomic(path string, text string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(text), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func renderMemoMarkdownFile(memo MemoRecord) string {
	tags := uniqueStrings(memo.Tags)
	refs := uniqueStrings(memo.References)
	lines := []string{
		"---",
		"schemaVersion: " + fmt.Sprintf("%d", vaultSchemaVersion),
		"id: " + yamlQuote(memo.ID),
	}
	if memo.ProjectID != "" {
		lines = append(lines, "projectId: "+yamlQuote(sanitizeProjectID(memo.ProjectID)))
	}
	if strings.TrimSpace(memo.Kind) != "" {
		lines = append(lines, "kind: "+yamlQuote(strings.TrimSpace(memo.Kind)))
	}
	if strings.TrimSpace(memo.TaskID) != "" {
		lines = append(lines, "taskId: "+yamlQuote(sanitizeTaskID(memo.TaskID)))
	}
	lines = append(lines,
		"createdAt: "+yamlQuote(memo.CreatedAt),
		"updatedAt: "+yamlQuote(memo.UpdatedAt),
		"visibility: "+yamlQuote(normalizeMemoVisibility(memo.Visibility)),
		"private: "+fmt.Sprintf("%t", memo.Private),
		"pinned: "+fmt.Sprintf("%t", memo.Pinned),
		"archived: "+fmt.Sprintf("%t", memo.Archived),
		"contentWhitespace: \"preserve\"",
	)
	if len(tags) == 0 {
		lines = append(lines, "tags: []")
	} else {
		lines = append(lines, "tags:")
		for _, tag := range tags {
			lines = append(lines, "  - "+yamlQuote(tag))
		}
	}
	if len(refs) == 0 {
		lines = append(lines, "references: []")
	} else {
		lines = append(lines, "references:")
		for _, ref := range refs {
			lines = append(lines, "  - "+yamlQuote(ref))
		}
	}
	lines = append(lines, "---")
	return strings.Join(lines, "\n") + "\n" + normalizeMemoContent(memo.Content)
}

func normalizeMemoContent(content string) string {
	text := strings.ReplaceAll(content, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

func normalizeStoredMemoContent(content string, meta map[string]string) string {
	content = normalizeMemoContent(content)
	if meta["contentWhitespace"] == "preserve" {
		return content
	}
	return strings.TrimSpace(content)
}

func parseMemoMarkdown(raw string) (map[string]string, string) {
	meta := map[string]string{}
	text := strings.ReplaceAll(raw, "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return meta, text
	}
	end := strings.Index(text[4:], "\n---")
	if end < 0 {
		return meta, text
	}
	frontmatter := text[4 : 4+end]
	contentStart := 4 + end + len("\n---")
	if strings.HasPrefix(text[contentStart:], "\n") {
		contentStart++
	}
	currentListKey := ""
	for _, rawLine := range strings.Split(frontmatter, "\n") {
		line := strings.TrimRight(rawLine, " \t")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if currentListKey != "" && strings.HasPrefix(trimmed, "- ") {
			value := yamlUnquote(strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
			if meta[currentListKey] == "" {
				meta[currentListKey] = value
			} else {
				meta[currentListKey] += "\n" + value
			}
			continue
		}
		currentListKey = ""
		index := strings.Index(trimmed, ":")
		if index < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:index])
		value := strings.TrimSpace(trimmed[index+1:])
		if value == "" {
			currentListKey = key
			meta[key] = ""
			continue
		}
		if value == "[]" {
			meta[key] = ""
			continue
		}
		meta[key] = yamlUnquote(value)
	}
	return meta, text[contentStart:]
}

func parseMemoBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "y":
		return true
	default:
		return false
	}
}

func parseMemoList(meta map[string]string, key string) []string {
	value := strings.TrimSpace(meta[key])
	if value == "" {
		return []string{}
	}
	return uniqueStrings(strings.Split(value, "\n"))
}

func memoRelativePath(memo MemoRecord) string {
	created := parseMemoTime(memo.CreatedAt)
	if created.IsZero() {
		created = time.Now()
	}
	return filepath.ToSlash(filepath.Join(
		vaultMemoDirName,
		fmt.Sprintf("%04d", created.Year()),
		fmt.Sprintf("%02d", int(created.Month())),
		sanitizeMemoID(memo.ID)+".md",
	))
}

func findMemoFilePath(ctx *VaultContext, id string) (string, error) {
	targetID := strings.TrimSpace(id)
	var found string
	err := filepath.WalkDir(ctx.MemoDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			return nil
		}
		memo, err := readMemoFile(ctx, path)
		if err != nil {
			return err
		}
		if memo.ID == targetID {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("memo not found: %s", targetID)
	}
	return found, nil
}

func safeVaultRelativePath(rootDir string, relativePath string) (string, error) {
	clean := filepath.Clean(strings.TrimSpace(relativePath))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("relative path is required")
	}
	if filepath.IsAbs(clean) {
		return "", fmt.Errorf("absolute path is not allowed")
	}
	target := filepath.Join(rootDir, clean)
	rel, err := filepath.Rel(rootDir, target)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes vault")
	}
	return target, nil
}

func relativeVaultPath(ctx *VaultContext, path string) string {
	rel, err := filepath.Rel(ctx.RootDir, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
func sanitizeMemoID(value string) string {
	id := strings.TrimSpace(value)
	if id == "" {
		return newMemoID()
	}
	var b strings.Builder
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	next := strings.Trim(b.String(), "-")
	if next == "" {
		return newMemoID()
	}
	return next
}

func parseMemoTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t
	}
	return time.Time{}
}

func memoSortTime(memo MemoRecord) time.Time {
	for _, value := range []string{memo.UpdatedAt, memo.CreatedAt} {
		if t := parseMemoTime(value); !t.IsZero() {
			return t
		}
	}
	return time.Time{}
}

func yamlQuote(value string) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}

func yamlUnquote(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var out string
	if err := json.Unmarshal([]byte(value), &out); err == nil {
		return out
	}
	return strings.Trim(value, `"'`)
}
