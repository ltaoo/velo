package desktopapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type MemoRecord struct {
	Archived   bool     `json:"archived"`
	Content    string   `json:"content"`
	CreatedAt  string   `json:"createdAt"`
	ID         string   `json:"id"`
	Kind       string   `json:"kind,omitempty"`
	Path       string   `json:"path"`
	Pinned     bool     `json:"pinned"`
	ProjectID  string   `json:"projectId,omitempty"`
	References []string `json:"references"`
	Tags       []string `json:"tags"`
	TaskID     string   `json:"taskId,omitempty"`
	UpdatedAt  string   `json:"updatedAt"`
	Visibility string   `json:"visibility"`
}

type MemoCreateRequest struct {
	Content    string `json:"content"`
	ProjectID  string `json:"projectId,omitempty"`
	Visibility string `json:"visibility"`
}

type MemoUpdateRequest struct {
	Archived   *bool   `json:"archived"`
	Content    *string `json:"content"`
	ID         string  `json:"id"`
	Pinned     *bool   `json:"pinned"`
	ProjectID  *string `json:"projectId,omitempty"`
	Visibility *string `json:"visibility"`
}

type MemoDeleteRequest struct {
	CleanupAssets *bool  `json:"cleanupAssets"`
	DeleteTasks   *bool  `json:"deleteTasks"`
	ID            string `json:"id"`
}

type MemoDeleteResult struct {
	AssetErrors   []string `json:"assetErrors,omitempty"`
	AssetsDeleted int      `json:"assetsDeleted"`
	AssetsSkipped int      `json:"assetsSkipped"`
	TasksDeleted  int      `json:"tasksDeleted"`
}

type MemoDeleteOptions struct {
	CleanupAssets   bool
	DeleteTasks     bool
	Parent          context.Context
	StorageSettings json.RawMessage
	StorePath       string
}

func listVaultMemos(ctx *VaultContext) ([]MemoRecord, error) {
	memos := []MemoRecord{}
	if err := filepath.WalkDir(ctx.MemoDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			return nil
		}
		memo, err := readMemoFile(ctx, path)
		if err != nil {
			return err
		}
		memos = append(memos, memo)
		return nil
	}); err != nil {
		return nil, err
	}
	sort.SliceStable(memos, func(i, j int) bool {
		left := memoSortTime(memos[i])
		right := memoSortTime(memos[j])
		if left.Equal(right) {
			return memos[i].ID > memos[j].ID
		}
		return left.After(right)
	})
	return memos, nil
}

func createVaultMemo(ctx *VaultContext, req MemoCreateRequest) (MemoRecord, error) {
	content := normalizeMemoContent(req.Content)
	if strings.TrimSpace(content) == "" {
		return MemoRecord{}, fmt.Errorf("memo content is required")
	}
	projectID, err := validateMemoProjectID(ctx, req.ProjectID)
	if err != nil {
		return MemoRecord{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	memo := MemoRecord{
		Archived:   false,
		Content:    content,
		CreatedAt:  now,
		ID:         newMemoID(),
		Pinned:     false,
		ProjectID:  projectID,
		UpdatedAt:  "",
		Visibility: normalizeMemoVisibility(req.Visibility),
	}
	memo.Path = memoRelativePath(memo)
	originalTags := extractMemoTags(memo.Content)
	if err := syncMemoTaskLines(ctx, &memo); err != nil {
		return MemoRecord{}, err
	}
	memo.Tags = uniqueStrings(append(extractMemoTags(memo.Content), originalTags...))
	memo.References = extractMemoReferences(memo.Content)
	if err := writeMemoRecord(ctx, memo); err != nil {
		return MemoRecord{}, err
	}
	return memo, nil
}

func updateVaultMemo(ctx *VaultContext, req MemoUpdateRequest) (MemoRecord, error) {
	id := strings.TrimSpace(req.ID)
	if id == "" {
		return MemoRecord{}, fmt.Errorf("memo id is required")
	}
	path, err := findMemoFilePath(ctx, id)
	if err != nil {
		return MemoRecord{}, err
	}
	memo, err := readMemoFile(ctx, path)
	if err != nil {
		return MemoRecord{}, err
	}
	if req.Content != nil {
		content := normalizeMemoContent(*req.Content)
		if strings.TrimSpace(content) == "" {
			return MemoRecord{}, fmt.Errorf("memo content is required")
		}
		memo.Content = content
	}
	if req.Visibility != nil {
		memo.Visibility = normalizeMemoVisibility(*req.Visibility)
	}
	if req.Pinned != nil {
		memo.Pinned = *req.Pinned
	}
	if req.Archived != nil {
		memo.Archived = *req.Archived
	}
	if req.ProjectID != nil {
		projectID, err := validateMemoProjectID(ctx, *req.ProjectID)
		if err != nil {
			return MemoRecord{}, err
		}
		memo.ProjectID = projectID
	}
	memo.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	memo.Path = relativeVaultPath(ctx, path)
	originalTags := extractMemoTags(memo.Content)
	if err := syncMemoTaskLines(ctx, &memo); err != nil {
		return MemoRecord{}, err
	}
	memo.Tags = uniqueStrings(append(extractMemoTags(memo.Content), originalTags...))
	memo.References = extractMemoReferences(memo.Content)
	if err := writeMemoRecord(ctx, memo); err != nil {
		return MemoRecord{}, err
	}
	return memo, nil
}

func deleteVaultMemo(ctx *VaultContext, id string) error {
	_, err := deleteVaultMemoWithOptions(ctx, id, MemoDeleteOptions{})
	return err
}

func deleteVaultMemoWithAssets(parent context.Context, ctx *VaultContext, id string, storageSettings json.RawMessage, storePath string) (MemoDeleteResult, error) {
	return deleteVaultMemoWithOptions(ctx, id, MemoDeleteOptions{
		CleanupAssets:   true,
		Parent:          parent,
		StorageSettings: storageSettings,
		StorePath:       storePath,
	})
}

func deleteVaultMemoWithOptions(ctx *VaultContext, id string, options MemoDeleteOptions) (MemoDeleteResult, error) {
	result := MemoDeleteResult{}
	id = strings.TrimSpace(id)
	if id == "" {
		return result, fmt.Errorf("memo id is required")
	}
	path, err := findMemoFilePath(ctx, id)
	if err != nil {
		return result, err
	}
	memo, err := readMemoFile(ctx, path)
	if err != nil {
		return result, err
	}

	assetsToDelete := []memoAssetReference{}
	if options.CleanupAssets {
		assets := extractMemoAssetReferences(memo.Content)
		if len(assets) > 0 {
			shared, err := memoAssetReferencesInOtherMemos(ctx, memo.ID)
			if err != nil {
				return result, err
			}
			for _, asset := range assets {
				if shared[memoAssetReferenceID(asset)] {
					result.AssetsSkipped++
					continue
				}
				assetsToDelete = append(assetsToDelete, asset)
			}
		}
	}

	if options.DeleteTasks {
		deleted, err := deleteVaultTasksForMemo(ctx, memo.ID)
		if err != nil {
			return result, err
		}
		result.TasksDeleted = deleted
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

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool)
	next := []string{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		next = append(next, value)
	}
	return next
}

func normalizeMemoVisibility(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PUBLIC":
		return "PUBLIC"
	case "PROTECTED":
		return "PROTECTED"
	default:
		return "PRIVATE"
	}
}

func newMemoID() string {
	return "memo_" + time.Now().UTC().Format("20060102T150405") + "_" + randomVaultSuffix()
}
