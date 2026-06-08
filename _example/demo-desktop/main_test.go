package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeExternalBrowserURL(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "http", value: "http://example.com/a?b=c", want: "http://example.com/a?b=c"},
		{name: "https uppercase scheme", value: "HTTPS://example.com/a%20b", want: "https://example.com/a%20b"},
		{name: "empty", value: "", wantErr: true},
		{name: "relative", value: "/docs", wantErr: true},
		{name: "missing host", value: "https:///docs", wantErr: true},
		{name: "javascript", value: "javascript:alert(1)", wantErr: true},
		{name: "mailto", value: "mailto:user@example.com", wantErr: true},
		{name: "raw space", value: "https://example.com/a b", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeExternalBrowserURL(tt.value)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateVaultMemoWritesMarkdownFile(t *testing.T) {
	ctx, existing, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	if existing {
		t.Fatalf("new temp vault should not be existing")
	}

	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "#idea hello [[memo:abc|source]]",
		Visibility: "PUBLIC",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	if memo.ID == "" {
		t.Fatalf("memo id is empty")
	}
	if len(memo.Tags) != 1 || memo.Tags[0] != "idea" {
		t.Fatalf("tags = %#v, want idea", memo.Tags)
	}
	if len(memo.References) != 1 || memo.References[0] != "memo:abc" {
		t.Fatalf("references = %#v, want memo:abc", memo.References)
	}

	path := filepath.Join(ctx.RootDir, filepath.FromSlash(memo.Path))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memo file: %v", err)
	}
	text := string(raw)
	for _, want := range []string{
		"---\n",
		"id: \"" + memo.ID + "\"",
		"visibility: \"PUBLIC\"",
		"tags:\n  - \"idea\"",
		"references:\n  - \"memo:abc\"",
		"#idea hello [[memo:abc|source]]",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("memo file missing %q:\n%s", want, text)
		}
	}

	listed, err := listVaultMemos(ctx)
	if err != nil {
		t.Fatalf("list memos: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != memo.ID {
		t.Fatalf("listed memos = %#v, want created memo", listed)
	}
}

func TestCreateVaultMemoPreservesBlankLines(t *testing.T) {
	ctx, existing, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	if existing {
		t.Fatalf("new temp vault should not be existing")
	}

	content := "first\n\nsecond\n\n"
	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    content,
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	if memo.Content != content {
		t.Fatalf("created content = %q, want %q", memo.Content, content)
	}

	path := filepath.Join(ctx.RootDir, filepath.FromSlash(memo.Path))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memo file: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "contentWhitespace: \"preserve\"") {
		t.Fatalf("memo file missing whitespace marker:\n%s", text)
	}
	if !strings.HasSuffix(text, "---\n"+content) {
		t.Fatalf("memo file content suffix = %q, want %q", text, "---\n"+content)
	}

	listed, err := listVaultMemos(ctx)
	if err != nil {
		t.Fatalf("list memos: %v", err)
	}
	if len(listed) != 1 || listed[0].Content != content {
		t.Fatalf("listed memos = %#v, want content %q", listed, content)
	}
}

func TestCreateVaultMemoCanBelongToProject(t *testing.T) {
	ctx, existing, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	if existing {
		t.Fatalf("new temp vault should not be existing")
	}

	project, err := createVaultProject(ctx, ProjectCreateRequest{Name: "Work", Color: "#10b981"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "project memo",
		ProjectID:  project.ID,
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}
	if memo.ProjectID != project.ID {
		t.Fatalf("memo project id = %q, want %q", memo.ProjectID, project.ID)
	}

	path := filepath.Join(ctx.RootDir, filepath.FromSlash(memo.Path))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memo file: %v", err)
	}
	if !strings.Contains(string(raw), "projectId: \""+project.ID+"\"") {
		t.Fatalf("memo file missing projectId:\n%s", string(raw))
	}

	listed, err := listVaultMemos(ctx)
	if err != nil {
		t.Fatalf("list memos: %v", err)
	}
	if len(listed) != 1 || listed[0].ProjectID != project.ID {
		t.Fatalf("listed memos = %#v, want project %s", listed, project.ID)
	}
}

func TestCreateVaultMemoRejectsUnknownProject(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}
	if _, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:   "orphan",
		ProjectID: "project_missing",
	}); err == nil {
		t.Fatalf("expected unknown project error")
	}
}

func TestExtractMemoAssetReferences(t *testing.T) {
	content := strings.Join([]string{
		"![image](@assets/memo-local/images/a%29.png)",
		"[file](@assets/Other/docs/report.pdf)",
		"[space](@assets/memo-local/docs/my file.pdf)",
		"raw @assets/memo-local/raw.txt reference",
		"![inline](data:image/png;base64,abc)",
	}, "\n")

	got := extractMemoAssetReferences(content)
	want := map[string]bool{
		"memo-local/images/a).png":    true,
		"memo-local/docs/my file.pdf": true,
		"other/docs/report.pdf":       true,
		"memo-local/raw.txt":          true,
	}
	if len(got) != len(want) {
		t.Fatalf("asset refs length = %d, want %d: %#v", len(got), len(want), got)
	}
	for _, ref := range got {
		id := memoAssetReferenceID(ref)
		if !want[id] {
			t.Fatalf("unexpected asset ref %q from %#v", id, got)
		}
		delete(want, id)
	}
	if len(want) > 0 {
		t.Fatalf("missing asset refs: %#v", want)
	}
}

func TestDeleteVaultMemoRemovesExclusiveManagedAssets(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	storePath := filepath.Join(ctx.VeloDir, "storage.json")
	cfg := defaultLocalMemoOSSConfig(storePath)
	settings := CloudStorageSettings{
		ActiveStorageID:     cfg.ID,
		DefaultsInitialized: true,
		Storages:            []OSSConfig{cfg},
	}
	rawSettings, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}

	uniqueKey := "images/unique.png"
	sharedKey := "images/shared.png"
	if err := writeLocalOSSObject(context.Background(), cfg, uniqueKey, []byte("unique")); err != nil {
		t.Fatalf("write unique asset: %v", err)
	}
	if err := writeLocalOSSObject(context.Background(), cfg, sharedKey, []byte("shared")); err != nil {
		t.Fatalf("write shared asset: %v", err)
	}
	uniquePath, err := localOSSObjectDiskPath(cfg, uniqueKey)
	if err != nil {
		t.Fatalf("unique asset path: %v", err)
	}
	sharedPath, err := localOSSObjectDiskPath(cfg, sharedKey)
	if err != nil {
		t.Fatalf("shared asset path: %v", err)
	}
	localOriginal := filepath.Join(t.TempDir(), "original.txt")
	if err := os.WriteFile(localOriginal, []byte("original"), 0644); err != nil {
		t.Fatalf("write local original: %v", err)
	}

	target, err := createVaultMemo(ctx, MemoCreateRequest{
		Content: strings.Join([]string{
			"![unique](@assets/memo-local/" + uniqueKey + ")",
			"![shared](@assets/memo-local/" + sharedKey + ")",
			"[local](file://" + localOriginal + ")",
			"[remote](https://example.com/file.png)",
		}, "\n"),
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create target memo: %v", err)
	}
	other, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "![shared](@assets/memo-local/" + sharedKey + ")",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create other memo: %v", err)
	}

	result, err := deleteVaultMemoWithAssets(context.Background(), ctx, target.ID, json.RawMessage(rawSettings), storePath)
	if err != nil {
		t.Fatalf("delete memo: %v", err)
	}
	if result.AssetsDeleted != 1 || result.AssetsSkipped != 1 || len(result.AssetErrors) != 0 {
		t.Fatalf("delete result = %#v, want 1 deleted, 1 skipped, no errors", result)
	}
	targetPath := filepath.Join(ctx.RootDir, filepath.FromSlash(target.Path))
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("target memo still exists or stat failed: %v", err)
	}
	if _, err := os.Stat(uniquePath); !os.IsNotExist(err) {
		t.Fatalf("unique asset still exists or stat failed: %v", err)
	}
	if _, err := os.Stat(sharedPath); err != nil {
		t.Fatalf("shared asset was removed: %v", err)
	}
	if _, err := os.Stat(localOriginal); err != nil {
		t.Fatalf("local original file was removed: %v", err)
	}

	listed, err := listVaultMemos(ctx)
	if err != nil {
		t.Fatalf("list memos: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != other.ID {
		t.Fatalf("listed memos = %#v, want only other memo %s", listed, other.ID)
	}
}

func TestDeleteVaultMemoCanKeepManagedAssets(t *testing.T) {
	ctx, _, err := openVaultDirectory(t.TempDir(), true)
	if err != nil {
		t.Fatalf("open vault: %v", err)
	}

	storePath := filepath.Join(ctx.VeloDir, "storage.json")
	cfg := defaultLocalMemoOSSConfig(storePath)
	settings := CloudStorageSettings{
		ActiveStorageID:     cfg.ID,
		DefaultsInitialized: true,
		Storages:            []OSSConfig{cfg},
	}
	rawSettings, err := json.Marshal(settings)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}

	key := "docs/keep.pdf"
	if err := writeLocalOSSObject(context.Background(), cfg, key, []byte("keep")); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	assetPath, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		t.Fatalf("asset path: %v", err)
	}
	memo, err := createVaultMemo(ctx, MemoCreateRequest{
		Content:    "[file](@assets/memo-local/" + key + ")",
		Visibility: "PRIVATE",
	})
	if err != nil {
		t.Fatalf("create memo: %v", err)
	}

	result, err := deleteVaultMemoWithOptions(ctx, memo.ID, MemoDeleteOptions{
		CleanupAssets:   false,
		Parent:          context.Background(),
		StorageSettings: json.RawMessage(rawSettings),
		StorePath:       storePath,
	})
	if err != nil {
		t.Fatalf("delete memo: %v", err)
	}
	if result.AssetsDeleted != 0 || result.AssetsSkipped != 0 || len(result.AssetErrors) != 0 {
		t.Fatalf("delete result = %#v, want no asset cleanup", result)
	}
	if _, err := os.Stat(assetPath); err != nil {
		t.Fatalf("asset should remain: %v", err)
	}
}

func TestDefaultLocalStorageRootUsesVaultStorage(t *testing.T) {
	vaultDir := t.TempDir()
	storePath := filepath.Join(vaultDir, ".velo", "storage.json")
	got := defaultLocalStorageRoot(storePath)
	want := filepath.Join(vaultDir, "storage")
	if got != want {
		t.Fatalf("defaultLocalStorageRoot() = %q, want %q", got, want)
	}
}

func TestPrepareCloudStorageSettingsRepointsLegacyLocalRoot(t *testing.T) {
	vaultDir := t.TempDir()
	storePath := filepath.Join(vaultDir, ".velo", "storage.json")
	settings := CloudStorageSettings{
		ActiveStorageID:     "memo-local",
		DefaultsInitialized: true,
		Storages: []OSSConfig{
			{
				Bucket:         "memos",
				Enabled:        true,
				Endpoint:       legacyDefaultLocalStorageRoot(storePath),
				ForcePathStyle: true,
				ID:             "memo-local",
				Name:           "本地 Memo 存储",
				Provider:       "local",
				UseSSL:         false,
			},
		},
	}

	got, changed, err := prepareCloudStorageSettings(settings, storePath, false)
	if err != nil {
		t.Fatalf("prepareCloudStorageSettings: %v", err)
	}
	if !changed {
		t.Fatalf("prepareCloudStorageSettings changed = false, want true")
	}
	if len(got.Storages) != 1 {
		t.Fatalf("storages length = %d, want 1", len(got.Storages))
	}
	wantEndpoint := filepath.Join(vaultDir, "storage")
	if got.Storages[0].Endpoint != wantEndpoint {
		t.Fatalf("endpoint = %q, want %q", got.Storages[0].Endpoint, wantEndpoint)
	}
	if got.Storages[0].Local == nil || got.Storages[0].Local.RootMode != localStorageRootModeVault || got.Storages[0].Local.Root != defaultLocalStorageRelativeRoot {
		t.Fatalf("local settings = %#v, want vault storage", got.Storages[0].Local)
	}
	if _, err := os.Stat(filepath.Join(wantEndpoint, "memos")); err != nil {
		t.Fatalf("local bucket was not created under vault storage: %v", err)
	}
}

func TestPrepareCloudStorageSettingsResolvesSyncedLocalRootForCurrentVault(t *testing.T) {
	sourceVaultDir := t.TempDir()
	targetVaultDir := t.TempDir()
	sourceStorePath := filepath.Join(sourceVaultDir, ".velo", "storage.json")
	targetStorePath := filepath.Join(targetVaultDir, ".velo", "storage.json")
	settings := CloudStorageSettings{
		ActiveStorageID:     "attachments",
		DefaultsInitialized: true,
		Storages: []OSSConfig{
			{
				Bucket:         "files",
				Enabled:        true,
				Endpoint:       defaultLocalStorageRoot(sourceStorePath),
				ForcePathStyle: true,
				ID:             "attachments",
				Name:           "Attachments",
				Provider:       "local",
				UseSSL:         false,
			},
		},
	}

	got, changed, err := prepareCloudStorageSettings(settings, targetStorePath, false)
	if err != nil {
		t.Fatalf("prepareCloudStorageSettings: %v", err)
	}
	if !changed {
		t.Fatalf("prepareCloudStorageSettings changed = false, want true")
	}
	if len(got.Storages) != 1 {
		t.Fatalf("storages length = %d, want 1", len(got.Storages))
	}
	wantEndpoint := filepath.Join(targetVaultDir, "storage")
	if got.Storages[0].Endpoint != wantEndpoint {
		t.Fatalf("endpoint = %q, want %q", got.Storages[0].Endpoint, wantEndpoint)
	}
	if got.Storages[0].Local == nil || got.Storages[0].Local.RootMode != localStorageRootModeVault || got.Storages[0].Local.Root != defaultLocalStorageRelativeRoot {
		t.Fatalf("local settings = %#v, want vault storage", got.Storages[0].Local)
	}
	if _, err := os.Stat(filepath.Join(wantEndpoint, "files")); err != nil {
		t.Fatalf("local bucket was not created under target vault storage: %v", err)
	}

	raw, err := marshalCloudStorageSettingsForStore(got)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, sourceVaultDir) || strings.Contains(text, targetVaultDir) {
		t.Fatalf("stored settings should not include machine-specific vault paths: %s", text)
	}
	var stored CloudStorageSettings
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("unmarshal stored settings: %v", err)
	}
	if stored.Storages[0].Endpoint != "" {
		t.Fatalf("stored endpoint = %q, want empty portable endpoint", stored.Storages[0].Endpoint)
	}
	if stored.Storages[0].Local == nil || stored.Storages[0].Local.RootMode != localStorageRootModeVault || stored.Storages[0].Local.Root != defaultLocalStorageRelativeRoot {
		t.Fatalf("stored local settings = %#v, want vault storage", stored.Storages[0].Local)
	}
}

func TestPrepareCloudStorageSettingsKeepsCustomLocalAbsoluteRoot(t *testing.T) {
	vaultDir := t.TempDir()
	storePath := filepath.Join(vaultDir, ".velo", "storage.json")
	customRoot := filepath.Join(t.TempDir(), "asset-root")
	settings := CloudStorageSettings{
		ActiveStorageID:     "custom-local",
		DefaultsInitialized: true,
		Storages: []OSSConfig{
			{
				Bucket:         "files",
				Enabled:        true,
				Endpoint:       customRoot,
				ForcePathStyle: true,
				ID:             "custom-local",
				Name:           "Custom Local",
				Provider:       "local",
				UseSSL:         false,
			},
		},
	}

	got, _, err := prepareCloudStorageSettings(settings, storePath, false)
	if err != nil {
		t.Fatalf("prepareCloudStorageSettings: %v", err)
	}
	if got.Storages[0].Endpoint != customRoot {
		t.Fatalf("endpoint = %q, want custom root %q", got.Storages[0].Endpoint, customRoot)
	}
	if got.Storages[0].Local == nil || got.Storages[0].Local.RootMode != localStorageRootModeAbsolute || got.Storages[0].Local.Root != customRoot {
		t.Fatalf("local settings = %#v, want absolute custom root", got.Storages[0].Local)
	}

	raw, err := marshalCloudStorageSettingsForStore(got)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	var stored CloudStorageSettings
	if err := json.Unmarshal(raw, &stored); err != nil {
		t.Fatalf("unmarshal stored settings: %v", err)
	}
	if stored.Storages[0].Endpoint != customRoot {
		t.Fatalf("stored endpoint = %q, want custom root %q", stored.Storages[0].Endpoint, customRoot)
	}
	if stored.Storages[0].Local == nil || stored.Storages[0].Local.RootMode != localStorageRootModeAbsolute || stored.Storages[0].Local.Root != customRoot {
		t.Fatalf("stored local settings = %#v, want absolute custom root", stored.Storages[0].Local)
	}
}
