package desktopapp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type MemoCommentRecord struct {
	Content    string   `json:"content"`
	CreatedAt  string   `json:"createdAt"`
	ID         string   `json:"id"`
	MemoID     string   `json:"memoId"`
	Path       string   `json:"path"`
	References []string `json:"references"`
	Tags       []string `json:"tags"`
	UpdatedAt  string   `json:"updatedAt"`
}

type MemoCommentCreateRequest struct {
	Content string `json:"content"`
	MemoID  string `json:"memoId"`
}

type MemoCommentUpdateRequest struct {
	Content *string `json:"content"`
	ID      string  `json:"id"`
}

type MemoCommentDeleteRequest struct {
	CleanupAssets *bool  `json:"cleanupAssets"`
	ID            string `json:"id"`
}

func listVaultMemoComments(ctx *VaultContext, memoID string) ([]MemoCommentRecord, error) {
	targetMemoID := strings.TrimSpace(memoID)
	if targetMemoID != "" {
		if _, err := findMemoFilePath(ctx, targetMemoID); err != nil {
			return nil, err
		}
	}

	comments := []MemoCommentRecord{}
	root := memoCommentDir(ctx)
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			return nil
		}
		comment, err := readMemoCommentFile(ctx, path)
		if err != nil {
			if targetMemoID == "" {
				return nil
			}
			return err
		}
		if strings.TrimSpace(comment.MemoID) == "" {
			return nil
		}
		if targetMemoID != "" && comment.MemoID != targetMemoID {
			return nil
		}
		comments = append(comments, comment)
		return nil
	}); err != nil {
		if os.IsNotExist(err) {
			return []MemoCommentRecord{}, nil
		}
		return nil, err
	}
	sortMemoComments(comments)
	return comments, nil
}

func createVaultMemoComment(ctx *VaultContext, req MemoCommentCreateRequest) (MemoCommentRecord, error) {
	memoID := strings.TrimSpace(req.MemoID)
	if memoID == "" {
		return MemoCommentRecord{}, fmt.Errorf("memo id is required")
	}
	memoPath, err := findMemoFilePath(ctx, memoID)
	if err != nil {
		return MemoCommentRecord{}, err
	}
	memo, err := readMemoFile(ctx, memoPath)
	if err != nil {
		return MemoCommentRecord{}, err
	}
	content := normalizeMemoContent(req.Content)
	if strings.TrimSpace(content) == "" {
		return MemoCommentRecord{}, fmt.Errorf("comment content is required")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	comment := MemoCommentRecord{
		Content:   content,
		CreatedAt: now,
		ID:        newMemoCommentID(),
		MemoID:    memoID,
		UpdatedAt: "",
	}
	comment.Path = memoCommentRelativePath(comment)
	originalTags := extractMemoTags(comment.Content)
	if err := syncMemoCommentTaskLines(ctx, &comment, memo); err != nil {
		return MemoCommentRecord{}, err
	}
	comment.Tags = uniqueStrings(append(extractMemoTags(comment.Content), originalTags...))
	comment.References = extractMemoReferences(comment.Content)
	if err := writeMemoCommentRecord(ctx, comment); err != nil {
		return MemoCommentRecord{}, err
	}
	return comment, nil
}

func updateVaultMemoComment(ctx *VaultContext, req MemoCommentUpdateRequest) (MemoCommentRecord, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return MemoCommentRecord{}, fmt.Errorf("comment id is required")
	}
	path, err := findMemoCommentFilePath(ctx, id)
	if err != nil {
		return MemoCommentRecord{}, err
	}
	comment, err := readMemoCommentFile(ctx, path)
	if err != nil {
		return MemoCommentRecord{}, err
	}
	if req.Content != nil {
		content := normalizeMemoContent(*req.Content)
		if strings.TrimSpace(content) == "" {
			return MemoCommentRecord{}, fmt.Errorf("comment content is required")
		}
		comment.Content = content
	}
	memoPath, err := findMemoFilePath(ctx, comment.MemoID)
	if err != nil {
		return MemoCommentRecord{}, err
	}
	memo, err := readMemoFile(ctx, memoPath)
	if err != nil {
		return MemoCommentRecord{}, err
	}
	comment.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	comment.Path = relativeVaultPath(ctx, path)
	originalTags := extractMemoTags(comment.Content)
	if err := syncMemoCommentTaskLines(ctx, &comment, memo); err != nil {
		return MemoCommentRecord{}, err
	}
	comment.Tags = uniqueStrings(append(extractMemoTags(comment.Content), originalTags...))
	comment.References = extractMemoReferences(comment.Content)
	if err := writeMemoCommentRecord(ctx, comment); err != nil {
		return MemoCommentRecord{}, err
	}
	return comment, nil
}

func deleteVaultMemoCommentWithOptions(ctx *VaultContext, id string, options MemoDeleteOptions) (MemoDeleteResult, error) {
	result := MemoDeleteResult{}
	id = strings.TrimSpace(id)
	if id == "" {
		return result, fmt.Errorf("comment id is required")
	}
	path, err := findMemoCommentFilePath(ctx, id)
	if err != nil {
		return result, err
	}
	comment, err := readMemoCommentFile(ctx, path)
	if err != nil {
		return result, err
	}

	assetsToDelete := []memoAssetReference{}
	if options.CleanupAssets {
		assets := extractMemoAssetReferences(comment.Content)
		if len(assets) > 0 {
			shared, err := memoAssetReferencesOutside(ctx, nil, map[string]bool{comment.ID: true})
			if err != nil {
				return result, err
			}
			for _, asset := range assets {
				if shared[memoAssetReferenceID(asset)] {
					result.AssetsSkipped++
					continue
				}
				assetsToDelete = appendMemoAssetReference(assetsToDelete, asset)
			}
		}
	}

	if err := os.Remove(path); err != nil {
		return result, err
	}
	if len(assetsToDelete) > 0 {
		cleanup := deleteMemoAssetReferences(options.Parent, options.StorageSettings, options.StorePath, assetsToDelete)
		result.AssetsDeleted += cleanup.AssetsDeleted
		result.AssetsSkipped += cleanup.AssetsSkipped
		result.AssetErrors = append(result.AssetErrors, cleanup.AssetErrors...)
	}
	return result, nil
}

func readMemoCommentFile(ctx *VaultContext, path string) (MemoCommentRecord, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return MemoCommentRecord{}, err
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
	comment := MemoCommentRecord{
		Content:    normalizeStoredMemoContent(content, meta),
		CreatedAt:  createdAt,
		ID:         id,
		MemoID:     strings.TrimSpace(meta["memoId"]),
		Path:       relativeVaultPath(ctx, path),
		References: parseMemoList(meta, "references"),
		Tags:       parseMemoList(meta, "tags"),
		UpdatedAt:  firstNonEmpty(meta["updatedAt"], meta["updated_at"]),
	}
	if comment.MemoID == "" {
		comment.MemoID = memoIDFromCommentPath(ctx, path)
	}
	if len(comment.Tags) == 0 {
		comment.Tags = extractMemoTags(comment.Content)
	}
	if len(comment.References) == 0 {
		comment.References = extractMemoReferences(comment.Content)
	}
	return comment, nil
}

func writeMemoCommentRecord(ctx *VaultContext, comment MemoCommentRecord) error {
	if strings.TrimSpace(comment.ID) == "" {
		return fmt.Errorf("comment id is required")
	}
	if strings.TrimSpace(comment.MemoID) == "" {
		return fmt.Errorf("comment memo id is required")
	}
	if comment.Path == "" {
		comment.Path = memoCommentRelativePath(comment)
	}
	target, err := safeVaultRelativePath(ctx.RootDir, comment.Path)
	if err != nil {
		return err
	}
	root := memoCommentDir(ctx)
	if !strings.HasPrefix(target, root+string(filepath.Separator)) && target != root {
		return fmt.Errorf("comment path must be inside memo comment directory")
	}
	return writeTextFileAtomic(target, renderMemoCommentMarkdownFile(comment))
}

func renderMemoCommentMarkdownFile(comment MemoCommentRecord) string {
	tags := uniqueStrings(comment.Tags)
	refs := uniqueStrings(comment.References)
	lines := []string{
		"---",
		"schemaVersion: " + fmt.Sprintf("%d", vaultSchemaVersion),
		"id: " + yamlQuote(comment.ID),
		"memoId: " + yamlQuote(strings.TrimSpace(comment.MemoID)),
		"createdAt: " + yamlQuote(comment.CreatedAt),
		"updatedAt: " + yamlQuote(comment.UpdatedAt),
		"contentWhitespace: \"preserve\"",
	}
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
	return strings.Join(lines, "\n") + "\n" + normalizeMemoContent(comment.Content)
}

func memoCommentRelativePath(comment MemoCommentRecord) string {
	created := parseMemoTime(comment.CreatedAt)
	if created.IsZero() {
		created = time.Now()
	}
	return filepath.ToSlash(filepath.Join(
		vaultMemoCommentDirName,
		sanitizeMemoID(comment.MemoID),
		fmt.Sprintf("%04d", created.Year()),
		fmt.Sprintf("%02d", int(created.Month())),
		sanitizeMemoCommentID(comment.ID)+".md",
	))
}

func findMemoCommentFilePath(ctx *VaultContext, id string) (string, error) {
	targetID := strings.TrimSpace(id)
	var found string
	err := filepath.WalkDir(memoCommentDir(ctx), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			return nil
		}
		comment, err := readMemoCommentFile(ctx, path)
		if err != nil {
			return err
		}
		if comment.ID == targetID {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("comment not found: %s", targetID)
	}
	return found, nil
}

func memoCommentDir(ctx *VaultContext) string {
	if ctx != nil && strings.TrimSpace(ctx.MemoCommentDir) != "" {
		return ctx.MemoCommentDir
	}
	if ctx == nil {
		return ""
	}
	return filepath.Join(ctx.RootDir, vaultMemoCommentDirName)
}

func memoIDFromCommentPath(ctx *VaultContext, path string) string {
	rel, err := filepath.Rel(memoCommentDir(ctx), path)
	if err != nil {
		return ""
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 0 {
		return ""
	}
	return strings.TrimSpace(parts[0])
}

func newMemoCommentID() string {
	return "comment_" + time.Now().UTC().Format("20060102T150405") + "_" + randomVaultSuffix()
}

func sanitizeMemoCommentID(value string) string {
	id := strings.TrimSpace(value)
	if id == "" {
		return newMemoCommentID()
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
		return newMemoCommentID()
	}
	return next
}

func sortMemoComments(comments []MemoCommentRecord) {
	sort.SliceStable(comments, func(i, j int) bool {
		left := parseMemoTime(comments[i].CreatedAt)
		right := parseMemoTime(comments[j].CreatedAt)
		if left.Equal(right) {
			return comments[i].ID > comments[j].ID
		}
		if left.IsZero() {
			return true
		}
		if right.IsZero() {
			return false
		}
		return left.After(right)
	})
}
