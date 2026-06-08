package desktopapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ltaoo/velo"
)

var memoTagPattern = regexp.MustCompile(`(?:^|\s)#([\p{L}\p{N}_-]+)`)
var memoReferencePattern = regexp.MustCompile(`!?\[\[([^\]|#]+)(?:[|#][^\]]*)?\]\]`)
var memoMarkdownURLPattern = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]*)\)`)
var memoAssetTokenPattern = regexp.MustCompile("@assets/[A-Za-z0-9_-]+/[^\\s\\]\\)<>'\"`]+")

type memoAssetReference struct {
	Key       string
	StorageID string
}

func extractMemoTags(content string) []string {
	matches := memoTagPattern.FindAllStringSubmatch(content, -1)
	tags := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			tags = append(tags, match[1])
		}
	}
	return uniqueStrings(tags)
}

func extractMemoReferences(content string) []string {
	matches := memoReferencePattern.FindAllStringSubmatch(content, -1)
	refs := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			ref := strings.TrimSpace(match[1])
			if ref != "" {
				refs = append(refs, ref)
			}
		}
	}
	return uniqueStrings(refs)
}

func extractMemoAssetReferences(content string) []memoAssetReference {
	seen := map[string]bool{}
	refs := []memoAssetReference{}
	markdownRanges := [][2]int{}
	add := func(value string) {
		ref, ok := parseMemoAssetReference(value)
		if !ok {
			return
		}
		id := memoAssetReferenceID(ref)
		if seen[id] {
			return
		}
		seen[id] = true
		refs = append(refs, ref)
	}

	for _, match := range memoMarkdownURLPattern.FindAllStringSubmatchIndex(content, -1) {
		if len(match) >= 4 {
			markdownRanges = append(markdownRanges, [2]int{match[0], match[1]})
			add(content[match[2]:match[3]])
		}
	}
	for _, match := range memoAssetTokenPattern.FindAllStringIndex(content, -1) {
		if len(match) != 2 || memoByteRangeContains(markdownRanges, match[0]) {
			continue
		}
		add(content[match[0]:match[1]])
	}
	return refs
}

func memoByteRangeContains(ranges [][2]int, index int) bool {
	for _, item := range ranges {
		if index >= item[0] && index < item[1] {
			return true
		}
	}
	return false
}

func parseMemoAssetReference(value string) (memoAssetReference, bool) {
	text := strings.TrimSpace(value)
	if strings.HasPrefix(text, "<") && strings.HasSuffix(text, ">") {
		text = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(text, "<"), ">"))
	}
	if !strings.HasPrefix(strings.ToLower(text), "@assets/") {
		return memoAssetReference{}, false
	}
	parts := strings.SplitN(text[len("@assets/"):], "/", 2)
	if len(parts) != 2 {
		return memoAssetReference{}, false
	}
	storageID := sanitizeStorageID(parts[0])
	key := cleanOSSObjectPath(decodeMemoAssetReferenceKey(parts[1]))
	if storageID == "" || key == "" {
		return memoAssetReference{}, false
	}
	return memoAssetReference{Key: key, StorageID: storageID}, true
}

func decodeMemoAssetReferenceKey(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "%28", "("), "%29", ")")
}

func memoAssetReferenceID(ref memoAssetReference) string {
	return sanitizeStorageID(ref.StorageID) + "/" + cleanOSSObjectPath(ref.Key)
}

func memoAssetReferencesInOtherMemos(ctx *VaultContext, targetID string) (map[string]bool, error) {
	refs := map[string]bool{}
	if err := filepath.WalkDir(ctx.MemoDir, func(path string, entry os.DirEntry, err error) error {
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
			return nil
		}
		for _, ref := range extractMemoAssetReferences(memo.Content) {
			refs[memoAssetReferenceID(ref)] = true
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return refs, nil
}

func deleteMemoAssetReferences(parent context.Context, rawSettings json.RawMessage, storePath string, refs []memoAssetReference) MemoDeleteResult {
	result := MemoDeleteResult{}
	if len(refs) == 0 {
		return result
	}
	if parent == nil {
		parent = context.Background()
	}

	settings, err := loadStoredCloudStorageSettings(rawSettings)
	if err != nil {
		result.AssetsSkipped = len(refs)
		result.AssetErrors = append(result.AssetErrors, fmt.Sprintf("read storage settings: %v", err))
		return result
	}
	settings, _, err = prepareCloudStorageSettings(settings, storePath, len(settings.Storages) == 0)
	if err != nil {
		result.AssetsSkipped = len(refs)
		result.AssetErrors = append(result.AssetErrors, fmt.Sprintf("prepare storage settings: %v", err))
		return result
	}

	configs := map[string]OSSConfig{}
	for _, ref := range refs {
		storageID := sanitizeStorageID(ref.StorageID)
		cfg, ok := configs[storageID]
		if !ok {
			var err error
			cfg, err = activeOSSConfig(settings, storageID)
			if err != nil {
				result.AssetsSkipped++
				result.AssetErrors = append(result.AssetErrors, fmt.Sprintf("%s: %v", assetRef(storageID, ref.Key), err))
				continue
			}
			configs[storageID] = cfg
		}

		resp, err := deleteOSSFile(parent, cfg, OSSFileDeleteRequest{Path: ref.Key, StorageID: storageID})
		if err != nil {
			result.AssetsSkipped++
			result.AssetErrors = append(result.AssetErrors, fmt.Sprintf("%s: %v", assetRef(storageID, ref.Key), err))
			continue
		}
		result.AssetsDeleted += ossDeletedCount(resp)
	}
	return result
}

func ossDeletedCount(resp velo.H) int {
	if resp == nil {
		return 1
	}
	switch value := resp["deleted"].(type) {
	case int:
		if value > 0 {
			return value
		}
	case int64:
		if value > 0 {
			return int(value)
		}
	case float64:
		if value > 0 {
			return int(value)
		}
	}
	return 1
}
