package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/autostart"
	veloerr "github.com/ltaoo/velo/error"
	"github.com/ltaoo/velo/file"
	"github.com/ltaoo/velo/shortcut"
	"github.com/ltaoo/velo/store"
	"github.com/ltaoo/velo/tray"
	updater "github.com/ltaoo/velo/updater/api"
	utypes "github.com/ltaoo/velo/updater/types"
	uversion "github.com/ltaoo/velo/updater/version"

	"github.com/rs/zerolog"
)

//go:embed frontend
var frontend_folder embed.FS

//go:embed app-config.json
var appConfigData []byte

//go:embed assets/appicon.png
var appIcon []byte

var Version = "1.0.0"
var Mode = "dev"

const cloudStorageSettingsKey = "demo-desktop:settings:cloud-storage:v1"
const globalVeloDirName = ".velo"
const globalVaultDataFileName = "data.json"
const vaultConfigDirName = ".velo"
const vaultMemoDirName = "memo"
const vaultSchemaVersion = 1

var memoTagPattern = regexp.MustCompile(`(?:^|\s)#([\p{L}\p{N}_-]+)`)
var memoReferencePattern = regexp.MustCompile(`!?\[\[([^\]|#]+)(?:[|#][^\]]*)?\]\]`)
var memoMarkdownURLPattern = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]*)\)`)
var memoAssetTokenPattern = regexp.MustCompile("@assets/[A-Za-z0-9_-]+/[^\\s\\]\\)<>'\"`]+")

var memoWindowCache = struct {
	sync.RWMutex
	items map[string]MemoWindowPayload
}{
	items: make(map[string]MemoWindowPayload),
}

var vaultRuntime = struct {
	sync.RWMutex
	active *VaultContext
}{
	active: nil,
}

type MemoWindowPayload struct {
	Fixed bool            `json:"fixed"`
	Memo  json.RawMessage `json:"memo"`
	Memos json.RawMessage `json:"memos"`
}

type OSSConfig struct {
	AccessKeyID     string `json:"accessKeyId"`
	Bucket          string `json:"bucket"`
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint"`
	ForcePathStyle  bool   `json:"forcePathStyle"`
	ID              string `json:"id"`
	Name            string `json:"name"`
	PathPrefix      string `json:"pathPrefix"`
	Provider        string `json:"provider"`
	PublicBaseURL   string `json:"publicBaseUrl"`
	Region          string `json:"region"`
	SecretAccessKey string `json:"secretAccessKey"`
	SessionToken    string `json:"sessionToken"`
	UseSSL          bool   `json:"useSSL"`
}

type CloudStorageSettings struct {
	ActiveStorageID     string      `json:"activeStorageId"`
	DefaultsInitialized bool        `json:"defaultsInitialized,omitempty"`
	Storages            []OSSConfig `json:"storages"`
}

type VaultRegistry struct {
	SchemaVersion int          `json:"schemaVersion"`
	ActiveVaultID string       `json:"activeVaultId"`
	Vaults        []VaultEntry `json:"vaults"`
}

type VaultEntry struct {
	ID           string `json:"id"`
	LastOpenedAt string `json:"lastOpenedAt"`
	Name         string `json:"name"`
	Path         string `json:"path"`
}

type VaultFile struct {
	CreatedAt     string `json:"createdAt"`
	ID            string `json:"id"`
	Name          string `json:"name"`
	SchemaVersion int    `json:"schemaVersion"`
	UpdatedAt     string `json:"updatedAt"`
}

type VaultContext struct {
	Entry   VaultEntry `json:"entry"`
	RootDir string     `json:"rootDir"`
	VeloDir string     `json:"veloDir"`
	MemoDir string     `json:"memoDir"`
}

type VaultOpenRequest struct {
	Path string `json:"path"`
}

type MemoRecord struct {
	Archived   bool     `json:"archived"`
	Content    string   `json:"content"`
	CreatedAt  string   `json:"createdAt"`
	ID         string   `json:"id"`
	Path       string   `json:"path"`
	Pinned     bool     `json:"pinned"`
	References []string `json:"references"`
	Tags       []string `json:"tags"`
	UpdatedAt  string   `json:"updatedAt"`
	Visibility string   `json:"visibility"`
}

type MemoCreateRequest struct {
	Content    string `json:"content"`
	Visibility string `json:"visibility"`
}

type MemoUpdateRequest struct {
	Archived   *bool   `json:"archived"`
	Content    *string `json:"content"`
	ID         string  `json:"id"`
	Pinned     *bool   `json:"pinned"`
	Visibility *string `json:"visibility"`
}

type MemoDeleteRequest struct {
	CleanupAssets *bool  `json:"cleanupAssets"`
	ID            string `json:"id"`
}

type MemoDeleteResult struct {
	AssetErrors   []string `json:"assetErrors,omitempty"`
	AssetsDeleted int      `json:"assetsDeleted"`
	AssetsSkipped int      `json:"assetsSkipped"`
}

type MemoDeleteOptions struct {
	CleanupAssets   bool
	Parent          context.Context
	StorageSettings json.RawMessage
	StorePath       string
}

type memoAssetReference struct {
	Key       string
	StorageID string
}

type OSSUploadRequest struct {
	Config        OSSConfig `json:"config"`
	ContentBase64 string    `json:"content_base64"`
	Name          string    `json:"name"`
	StorageID     string    `json:"storageId"`
	Type          string    `json:"type"`
}

type OSSFileListRequest struct {
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFilePreviewRequest struct {
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFileMkdirRequest struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFileDeleteRequest struct {
	IsDir     bool   `json:"isDir"`
	Path      string `json:"path"`
	StorageID string `json:"storageId"`
}

type OSSFileUploadRequest struct {
	ContentBase64 string `json:"content_base64"`
	Name          string `json:"name"`
	Path          string `json:"path"`
	StorageID     string `json:"storageId"`
	Type          string `json:"type"`
}

type OSSFileView struct {
	ID      string `json:"id"`
	IsDir   bool   `json:"isDir"`
	ModTime string `json:"modTime"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Ref     string `json:"ref"`
	Size    int64  `json:"size"`
	Type    string `json:"type"`
	URL     string `json:"url"`
}

func setupLogger() *zerolog.Logger {
	homeDir, _ := os.UserHomeDir()
	logDir := filepath.Join(homeDir, ".myapp")
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

	var writer io.Writer
	if err != nil {
		writer = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	} else if Mode == "release" {
		writer = logFile
	} else {
		writer = io.MultiWriter(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}, logFile)
	}
	logger := zerolog.New(writer).With().Timestamp().Logger()
	return &logger
}

func fatal(logger *zerolog.Logger, msg string) {
	logger.Error().Msg(msg)
	veloerr.ShowErrorDialog(msg)
	os.Exit(1)
}

func projectDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Dir(filename)
}

func globalVeloDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, globalVeloDirName), nil
}

func globalVaultDataPath() (string, error) {
	dir, err := globalVeloDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, globalVaultDataFileName), nil
}

func loadVaultRegistry() (VaultRegistry, error) {
	path, err := globalVaultDataPath()
	if err != nil {
		return VaultRegistry{}, err
	}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return VaultRegistry{SchemaVersion: vaultSchemaVersion, Vaults: []VaultEntry{}}, nil
	}
	if err != nil {
		return VaultRegistry{}, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return VaultRegistry{SchemaVersion: vaultSchemaVersion, Vaults: []VaultEntry{}}, nil
	}

	var registry VaultRegistry
	if err := json.Unmarshal(raw, &registry); err != nil {
		return VaultRegistry{}, fmt.Errorf("read vault registry: %w", err)
	}
	registry = normalizeVaultRegistry(registry)
	return registry, nil
}

func normalizeVaultRegistry(registry VaultRegistry) VaultRegistry {
	if registry.SchemaVersion == 0 {
		registry.SchemaVersion = vaultSchemaVersion
	}
	next := make([]VaultEntry, 0, len(registry.Vaults))
	seen := make(map[string]bool)
	for _, entry := range registry.Vaults {
		entry.ID = strings.TrimSpace(entry.ID)
		entry.Path = strings.TrimSpace(entry.Path)
		if entry.ID == "" || entry.Path == "" {
			continue
		}
		cleanPath, err := cleanVaultPath(entry.Path)
		if err == nil {
			entry.Path = cleanPath
		}
		if entry.Name == "" {
			entry.Name = vaultDisplayName(entry.Path)
		}
		key := entry.ID
		if seen[key] {
			continue
		}
		seen[key] = true
		next = append(next, entry)
	}
	registry.Vaults = next
	if registry.ActiveVaultID != "" && !vaultRegistryHasID(registry, registry.ActiveVaultID) {
		registry.ActiveVaultID = ""
	}
	return registry
}

func saveVaultRegistry(registry VaultRegistry) error {
	path, err := globalVaultDataPath()
	if err != nil {
		return err
	}
	registry = normalizeVaultRegistry(registry)
	if registry.SchemaVersion == 0 {
		registry.SchemaVersion = vaultSchemaVersion
	}
	return writeJSONFileAtomic(path, registry)
}

func writeJSONFileAtomic(path string, value interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(raw, '\n'), 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func activeVaultFromRegistry(registry VaultRegistry) (VaultEntry, bool) {
	activeID := strings.TrimSpace(registry.ActiveVaultID)
	if activeID == "" {
		return VaultEntry{}, false
	}
	for _, entry := range registry.Vaults {
		if entry.ID == activeID {
			return entry, true
		}
	}
	return VaultEntry{}, false
}

func loadStartupVault() (*VaultContext, error) {
	registry, err := loadVaultRegistry()
	if err != nil {
		return nil, err
	}
	entry, ok := activeVaultFromRegistry(registry)
	if !ok {
		return nil, nil
	}
	ctx, _, err := openVaultDirectory(entry.Path, false)
	if err != nil {
		return nil, err
	}
	return ctx, nil
}

func openVaultDirectory(value string, createIfMissing bool) (*VaultContext, bool, error) {
	rootDir, err := cleanVaultPath(value)
	if err != nil {
		return nil, false, err
	}
	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, false, fmt.Errorf("vault directory is not accessible: %w", err)
	}
	if !info.IsDir() {
		return nil, false, fmt.Errorf("vault path is not a directory")
	}
	if err := ensureDirectoryWritable(rootDir); err != nil {
		return nil, false, err
	}

	veloDir := filepath.Join(rootDir, vaultConfigDirName)
	veloInfo, err := os.Stat(veloDir)
	existingVault := false
	if err == nil {
		if !veloInfo.IsDir() {
			return nil, false, fmt.Errorf(".velo exists but is not a directory")
		}
		existingVault = true
	} else if os.IsNotExist(err) {
		if !createIfMissing {
			return nil, false, fmt.Errorf("vault config directory does not exist")
		}
		if err := os.MkdirAll(veloDir, 0755); err != nil {
			return nil, false, fmt.Errorf("create .velo directory: %w", err)
		}
	} else {
		return nil, false, fmt.Errorf("stat .velo directory: %w", err)
	}

	memoDir := filepath.Join(rootDir, vaultMemoDirName)
	if err := os.MkdirAll(memoDir, 0755); err != nil {
		return nil, false, fmt.Errorf("create memo directory: %w", err)
	}

	vaultFile, err := loadOrCreateVaultFile(rootDir, veloDir)
	if err != nil {
		return nil, false, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	entry := VaultEntry{
		ID:           vaultFile.ID,
		LastOpenedAt: now,
		Name:         firstNonEmpty(vaultFile.Name, vaultDisplayName(rootDir)),
		Path:         rootDir,
	}
	return &VaultContext{
		Entry:   entry,
		RootDir: rootDir,
		VeloDir: veloDir,
		MemoDir: memoDir,
	}, existingVault, nil
}

func cleanVaultPath(value string) (string, error) {
	path := strings.TrimSpace(value)
	if path == "" {
		return "", fmt.Errorf("vault path is required")
	}
	if strings.HasPrefix(path, "~"+string(filepath.Separator)) || path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if path == "~" {
			path = homeDir
		} else {
			path = filepath.Join(homeDir, strings.TrimPrefix(path, "~"+string(filepath.Separator)))
		}
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func ensureDirectoryWritable(dir string) error {
	probe := filepath.Join(dir, ".velo-write-test-"+randomVaultSuffix())
	if err := os.WriteFile(probe, []byte("ok"), 0600); err != nil {
		return fmt.Errorf("vault directory is not writable: %w", err)
	}
	_ = os.Remove(probe)
	return nil
}

func loadOrCreateVaultFile(rootDir string, veloDir string) (VaultFile, error) {
	path := filepath.Join(veloDir, "vault.json")
	raw, err := os.ReadFile(path)
	if err == nil && len(bytes.TrimSpace(raw)) > 0 {
		var file VaultFile
		if err := json.Unmarshal(raw, &file); err != nil {
			return VaultFile{}, fmt.Errorf("read vault config: %w", err)
		}
		changed := false
		if strings.TrimSpace(file.ID) == "" {
			file.ID = newVaultID()
			changed = true
		}
		if strings.TrimSpace(file.Name) == "" {
			file.Name = vaultDisplayName(rootDir)
			changed = true
		}
		if file.SchemaVersion == 0 {
			file.SchemaVersion = vaultSchemaVersion
			changed = true
		}
		if file.CreatedAt == "" {
			file.CreatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			changed = true
		}
		if changed {
			file.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
			if err := writeJSONFileAtomic(path, file); err != nil {
				return VaultFile{}, err
			}
		}
		return file, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return VaultFile{}, fmt.Errorf("read vault config: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	file := VaultFile{
		CreatedAt:     now,
		ID:            newVaultID(),
		Name:          vaultDisplayName(rootDir),
		SchemaVersion: vaultSchemaVersion,
		UpdatedAt:     now,
	}
	if err := writeJSONFileAtomic(path, file); err != nil {
		return VaultFile{}, fmt.Errorf("write vault config: %w", err)
	}
	return file, nil
}

func registerActiveVault(ctx *VaultContext) (VaultRegistry, error) {
	registry, err := loadVaultRegistry()
	if err != nil {
		return VaultRegistry{}, err
	}
	registry.SchemaVersion = vaultSchemaVersion
	registry.ActiveVaultID = ctx.Entry.ID
	updated := false
	for i, entry := range registry.Vaults {
		if entry.ID == ctx.Entry.ID || samePath(entry.Path, ctx.Entry.Path) {
			registry.Vaults[i] = ctx.Entry
			updated = true
			break
		}
	}
	if !updated {
		registry.Vaults = append(registry.Vaults, ctx.Entry)
	}
	sort.SliceStable(registry.Vaults, func(i, j int) bool {
		return registry.Vaults[i].LastOpenedAt > registry.Vaults[j].LastOpenedAt
	})
	if err := saveVaultRegistry(registry); err != nil {
		return VaultRegistry{}, err
	}
	return registry, nil
}

func setActiveVault(ctx *VaultContext) {
	vaultRuntime.Lock()
	vaultRuntime.active = ctx
	vaultRuntime.Unlock()
}

func activeVaultSnapshot() *VaultContext {
	vaultRuntime.RLock()
	defer vaultRuntime.RUnlock()
	if vaultRuntime.active == nil {
		return nil
	}
	cp := *vaultRuntime.active
	return &cp
}

func vaultRegistryHasID(registry VaultRegistry, id string) bool {
	for _, entry := range registry.Vaults {
		if entry.ID == id {
			return true
		}
	}
	return false
}

func samePath(a string, b string) bool {
	cleanA, errA := cleanVaultPath(a)
	cleanB, errB := cleanVaultPath(b)
	if errA == nil {
		a = cleanA
	}
	if errB == nil {
		b = cleanB
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func vaultDisplayName(path string) string {
	name := strings.TrimSpace(filepath.Base(path))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "Vault"
	}
	return name
}

func newVaultID() string {
	return "vault_" + randomVaultSuffix()
}

func randomVaultSuffix() string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err == nil {
		return hex.EncodeToString(buf[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func requireActiveVault() (*VaultContext, error) {
	ctx := activeVaultSnapshot()
	if ctx == nil || strings.TrimSpace(ctx.RootDir) == "" {
		return nil, fmt.Errorf("vault is not selected")
	}
	return ctx, nil
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
	now := time.Now().UTC().Format(time.RFC3339Nano)
	memo := MemoRecord{
		Archived:   false,
		Content:    content,
		CreatedAt:  now,
		ID:         newMemoID(),
		Pinned:     false,
		UpdatedAt:  "",
		Visibility: normalizeMemoVisibility(req.Visibility),
	}
	memo.Tags = extractMemoTags(memo.Content)
	memo.References = extractMemoReferences(memo.Content)
	memo.Path = memoRelativePath(memo)
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
	memo.Tags = extractMemoTags(memo.Content)
	memo.References = extractMemoReferences(memo.Content)
	memo.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	memo.Path = relativeVaultPath(ctx, path)
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
		Path:       relativeVaultPath(ctx, path),
		Pinned:     parseMemoBool(meta["pinned"]),
		References: parseMemoList(meta, "references"),
		Tags:       parseMemoList(meta, "tags"),
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
		"createdAt: " + yamlQuote(memo.CreatedAt),
		"updatedAt: " + yamlQuote(memo.UpdatedAt),
		"visibility: " + yamlQuote(normalizeMemoVisibility(memo.Visibility)),
		"pinned: " + fmt.Sprintf("%t", memo.Pinned),
		"archived: " + fmt.Sprintf("%t", memo.Archived),
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

func initUpdater(logger *zerolog.Logger) (*updater.AppUpdater, error) {
	appCfg := velo.LoadAppConfig(appConfigData)
	updateConfig := appCfg.Update.ToUpdaterConfig()
	versionInfo := uversion.ParseVersionInfo(Version, updateConfig)
	if !versionInfo.UpdateMode.IsEnabled() {
		return nil, fmt.Errorf("auto-update is disabled (mode: %s)", versionInfo.UpdateMode)
	}
	effectiveVersion := Version
	if versionInfo.IsDevelopment() && updateConfig.DevVersion != "" {
		effectiveVersion = updateConfig.DevVersion
	}
	homeDir, _ := os.UserHomeDir()
	statePath := filepath.Join(homeDir, ".myapp", "update_state.json")
	opts := utypes.UpdaterOptions{
		Config:         updateConfig,
		CurrentVersion: effectiveVersion,
		Logger:         logger,
		StatePath:      statePath,
	}
	u, err := updater.NewUpdaterWithOptions(&opts, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create updater: %w", err)
	}
	return u, nil
}

func main() {
	logger := setupLogger()
	logger.Info().Msgf("Version: %s, Velo: %s, Mode: %s, OS: %s/%s", Version, velo.GetVersion(), Mode, runtime.GOOS, runtime.GOARCH)

	app_updater, err := initUpdater(logger)
	if err != nil {
		logger.Warn().Msgf("Updater init: %v", err)
	}

	quitOnLastWindowClosed := true
	opt := velo.VeloAppOpt{Mode: velo.ModeBridge, IconData: appIcon, QuitOnLastWindowClosed: &quitOnLastWindowClosed}
	b := velo.NewApp(&opt)
	initialPathname := "/vault-picker"
	if startupVault, err := loadStartupVault(); err != nil {
		logger.Warn().Msgf("Active vault unavailable: %v", err)
	} else if startupVault != nil {
		setActiveVault(startupVault)
		if _, err := registerActiveVault(startupVault); err != nil {
			logger.Warn().Msgf("Failed to update active vault registry: %v", err)
		}
		b.Store = store.NewWithDir(startupVault.VeloDir)
		initialPathname = "/desktop"
		logger.Info().Msgf("Active vault: %s", startupVault.RootDir)
	} else if dir, err := globalVeloDir(); err == nil {
		if err := os.MkdirAll(dir, 0755); err != nil {
			logger.Warn().Msgf("Failed to create global velo dir: %v", err)
		} else {
			b.Store = store.NewWithDir(dir)
		}
	}
	logger.Info().Msgf("Store path: %s", b.Store.Path())

	b.Get("/api/ping", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"message": "pong"})
	})

	b.Get("/api/app", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"version": Version, "velo": velo.GetVersion(), "mode": Mode})
	})

	b.Get("/api/vault/status", func(c *velo.BoxContext) interface{} {
		registry, err := loadVaultRegistry()
		if err != nil {
			return c.Error(err.Error())
		}
		dataPath, err := globalVaultDataPath()
		if err != nil {
			return c.Error(err.Error())
		}
		_, statErr := os.Stat(dataPath)
		return c.Ok(velo.H{
			"active":         activeVaultSnapshot(),
			"activeVaultId":  registry.ActiveVaultID,
			"dataFileExists": statErr == nil,
			"dataPath":       dataPath,
			"vaults":         registry.Vaults,
		})
	})

	b.Get("/api/vault/select-directory", func(c *velo.BoxContext) interface{} {
		path, err := selectVaultDirectory()
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"path": path})
	})

	b.Post("/api/vault/open", func(c *velo.BoxContext) interface{} {
		var req VaultOpenRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		ctx, existing, err := openVaultDirectory(req.Path, true)
		if err != nil {
			return c.Error(err.Error())
		}
		registry, err := registerActiveVault(ctx)
		if err != nil {
			return c.Error(err.Error())
		}
		setActiveVault(ctx)
		b.Store = store.NewWithDir(ctx.VeloDir)
		return c.Ok(velo.H{
			"active":   ctx,
			"created":  !existing,
			"existing": existing,
			"registry": registry,
		})
	})

	b.Get("/api/memos", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		memos, err := listVaultMemos(ctx)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"memos": memos})
	})

	b.Post("/api/memos/create", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req MemoCreateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		memo, err := createVaultMemo(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"memo": memo})
	})

	b.Post("/api/memos/update", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req MemoUpdateRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		memo, err := updateVaultMemo(ctx, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"memo": memo})
	})

	b.Post("/api/memos/delete", func(c *velo.BoxContext) interface{} {
		ctx, err := requireActiveVault()
		if err != nil {
			return c.Error(err.Error())
		}
		var req MemoDeleteRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cleanupAssets := true
		if req.CleanupAssets != nil {
			cleanupAssets = *req.CleanupAssets
		}
		result, err := deleteVaultMemoWithOptions(ctx, req.ID, MemoDeleteOptions{
			CleanupAssets:   cleanupAssets,
			Parent:          c.Context(),
			StorageSettings: b.Store.Get(cloudStorageSettingsKey),
			StorePath:       b.Store.Path(),
		})
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{
			"assetErrors":   result.AssetErrors,
			"assetsDeleted": result.AssetsDeleted,
			"assetsSkipped": result.AssetsSkipped,
			"success":       true,
		})
	})

	b.Get("/api/window/show", func(c *velo.BoxContext) interface{} {
		b.Webview.Show()
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/window/hide", func(c *velo.BoxContext) interface{} {
		b.Webview.Hide()
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/window/state/restore", func(c *velo.BoxContext) interface{} {
		name := c.Query("name")
		if name == "" {
			name = "default"
		}
		ws := b.Store.GetWindow(name)
		if ws == nil {
			return c.Ok(velo.H{"found": false})
		}
		if ws.Width > 0 && ws.Height > 0 {
			b.Webview.SetSize(ws.Width, ws.Height)
		}
		if ws.X != 0 || ws.Y != 0 {
			b.Webview.SetPosition(ws.X, ws.Y)
		}
		return c.Ok(velo.H{"found": true, "x": ws.X, "y": ws.Y, "width": ws.Width, "height": ws.Height})
	})

	b.Get("/api/file/select", func(c *velo.BoxContext) interface{} {
		path, err := file.ShowFileSelectDialog("default")
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"path": path})
	})

	b.Get("/api/file/select-data-url", func(c *velo.BoxContext) interface{} {
		var allowedTypes []string
		if c.Query("accept") == "image" {
			allowedTypes = imageFileExtensions()
		}

		var path string
		var err error
		if len(allowedTypes) > 0 {
			path, err = file.ShowFileSelectDialogWithTypes("default", allowedTypes)
		} else {
			path, err = file.ShowFileSelectDialog("default")
		}
		if err != nil {
			return c.Error(err.Error())
		}

		selectedFile, err := droppedFileForPath(path)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"file": selectedFile})
	})

	b.Get("/api/editor/open", func(c *velo.BoxContext) interface{} {
		fileParam := strings.TrimSpace(c.Query("file"))
		if fileParam == "" {
			return c.Error("file is required")
		}

		fileTarget, embeddedLine, embeddedCol := splitEditorLocation(fileParam)
		line := editorPositionValue(c.Query("line"), embeddedLine)
		col := editorPositionValue(c.Query("col"), embeddedCol)
		resolvedFile, err := resolveEditorFileTarget(fileTarget, b.Store.Get(cloudStorageSettingsKey), b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}

		if err := openFileInEditor(resolvedFile, line, col, c.Query("app")); err != nil {
			logger.Error().Err(err).Str("file", resolvedFile).Msg("failed to open file in editor")
			return c.Error(fmt.Sprintf("Failed to open editor: %v", err))
		}

		return c.Ok(velo.H{
			"success": true,
			"file":    resolvedFile,
			"line":    line,
			"col":     col,
		})
	})

	b.Get("/api/external/open", func(c *velo.BoxContext) interface{} {
		target, err := normalizeExternalBrowserURL(c.Query("url"))
		if err != nil {
			return c.Error(err.Error())
		}

		confirmed, err := confirmExternalBrowserOpen(target)
		if err != nil {
			logger.Error().Err(err).Str("url", target).Msg("failed to confirm external URL")
			return c.Error(fmt.Sprintf("Failed to show confirm dialog: %v", err))
		}
		if !confirmed {
			return c.Ok(velo.H{"success": false, "cancelled": true, "url": target})
		}

		if err := openExternalBrowser(target); err != nil {
			logger.Error().Err(err).Str("url", target).Msg("failed to open external URL")
			return c.Error(fmt.Sprintf("Failed to open default browser: %v", err))
		}

		return c.Ok(velo.H{"success": true, "url": target})
	})

	b.Post("/api/memo-window/open", func(c *velo.BoxContext) interface{} {
		var req MemoWindowPayload
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}

		memoID, err := memoWindowMemoID(req.Memo)
		if err != nil {
			return c.Error(err.Error())
		}
		req.Memos = memoWindowMemosPayload(req.Memo, req.Memos)

		memoWindowCache.Lock()
		memoWindowCache.items[memoID] = req
		memoWindowCache.Unlock()

		params := url.Values{}
		params.Set("id", memoID)
		if req.Fixed {
			params.Set("fixed", "1")
		}

		nameSuffix := sanitizeStorageID(memoID)
		if nameSuffix == "" {
			nameSuffix = "memo"
		}
		b.OpenWindow(&velo.VeloWebviewOpt{
			Name:       "memo-window-" + nameSuffix,
			Title:      "Memo",
			Pathname:   "/memo-window?" + params.Encode(),
			Width:      460,
			Height:     560,
			Frameless:  true,
			EntryPage:  "memo-window.html",
			FrontendFS: frontend_folder,
		})
		return c.Ok(velo.H{"success": true, "id": memoID})
	})

	b.Get("/api/memo-window/get", func(c *velo.BoxContext) interface{} {
		memoID := strings.TrimSpace(c.Query("id"))
		if memoID == "" {
			return c.Error("id is required")
		}

		memoWindowCache.RLock()
		payload, ok := memoWindowCache.items[memoID]
		memoWindowCache.RUnlock()
		if !ok {
			return c.Ok(velo.H{"found": false})
		}
		return c.Ok(velo.H{
			"found": true,
			"fixed": payload.Fixed,
			"memo":  payload.Memo,
			"memos": payload.Memos,
		})
	})

	b.Get("/api/settings/cloud-storage", func(c *velo.BoxContext) interface{} {
		raw := b.Store.Get(cloudStorageSettingsKey)
		settings, err := loadStoredCloudStorageSettings(raw)
		if err != nil {
			return c.Error(err.Error())
		}
		settings, changed, err := prepareCloudStorageSettings(settings, b.Store.Path(), raw == nil || !settings.DefaultsInitialized)
		if err != nil {
			return c.Error(err.Error())
		}
		if raw == nil || changed {
			stored, err := json.Marshal(settings)
			if err != nil {
				return c.Error(err.Error())
			}
			if err := b.Store.Set(cloudStorageSettingsKey, json.RawMessage(stored)); err != nil {
				return c.Error(err.Error())
			}
		}
		return c.Ok(velo.H{"found": true, "config": settings, "defaults": cloudStorageDefaults(b.Store.Path())})
	})

	b.Post("/api/settings/cloud-storage/save", func(c *velo.BoxContext) interface{} {
		var settings CloudStorageSettings
		if err := c.BindJSON(&settings); err != nil {
			return c.Error(err.Error())
		}

		settings = normalizeCloudStorageSettings(settings)
		settings, _, err := prepareCloudStorageSettings(settings, b.Store.Path(), len(settings.Storages) == 0)
		if err != nil {
			return c.Error(err.Error())
		}
		raw, err := json.Marshal(settings)
		if err != nil {
			return c.Error(err.Error())
		}
		if err := b.Store.Set(cloudStorageSettingsKey, json.RawMessage(raw)); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true, "config": settings})
	})

	b.Get("/api/settings/cloud-storage/delete", func(c *velo.BoxContext) interface{} {
		if err := b.Store.Delete(cloudStorageSettingsKey); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Post("/api/oss/upload", func(c *velo.BoxContext) interface{} {
		var req OSSUploadRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		if !hasOSSConfig(req.Config) {
			settings, err := loadStoredCloudStorageSettings(b.Store.Get(cloudStorageSettingsKey))
			if err != nil {
				return c.Error(err.Error())
			}
			settings, _, err = prepareCloudStorageSettings(settings, b.Store.Path(), len(settings.Storages) == 0)
			if err != nil {
				return c.Error(err.Error())
			}
			cfg, err := activeOSSConfig(settings, req.StorageID)
			if err != nil {
				return c.Error(err.Error())
			}
			req.Config = cfg
		} else if strings.TrimSpace(req.Config.ID) == "" && strings.TrimSpace(req.StorageID) != "" {
			req.Config.ID = req.StorageID
		}

		result, err := uploadOSSObject(c.Context(), req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Post("/api/oss/files/list", func(c *velo.BoxContext) interface{} {
		var req OSSFileListRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), req.StorageID, b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}
		result, err := listOSSFiles(c.Context(), cfg, req.Path)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Post("/api/oss/files/preview", func(c *velo.BoxContext) interface{} {
		var req OSSFilePreviewRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), req.StorageID, b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}
		result, err := previewOSSFile(c.Context(), cfg, req.Path)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Post("/api/oss/files/mkdir", func(c *velo.BoxContext) interface{} {
		var req OSSFileMkdirRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), req.StorageID, b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}
		result, err := makeOSSFolder(c.Context(), cfg, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Post("/api/oss/files/delete", func(c *velo.BoxContext) interface{} {
		var req OSSFileDeleteRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), req.StorageID, b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}
		result, err := deleteOSSFile(c.Context(), cfg, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Post("/api/oss/files/upload", func(c *velo.BoxContext) interface{} {
		var req OSSFileUploadRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), req.StorageID, b.Store.Path())
		if err != nil {
			return c.Error(err.Error())
		}
		result, err := uploadOSSManagedFile(c.Context(), cfg, req)
		if err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(result)
	})

	b.Get("/api/oss/assets", func(c *velo.BoxContext) interface{} {
		cfg, err := storedOSSConfig(b.Store.Get(cloudStorageSettingsKey), c.Query("storageId"), b.Store.Path())
		if err != nil {
			writePlainError(c.Writer, http.StatusBadRequest, err.Error())
			return nil
		}
		objectPath := cleanOSSObjectPath(firstNonEmpty(c.Query("path"), c.Query("key")))
		if objectPath == "" {
			writePlainError(c.Writer, http.StatusBadRequest, "file path is required")
			return nil
		}
		if !isLocalOSSConfig(cfg) {
			endpoint := normalizeOSSEndpoint(cfg.Endpoint, cfg.UseSSL)
			c.Writer.Header().Set("Location", publicOSSObjectURL(cfg, endpoint, objectPath))
			c.Writer.WriteHeader(http.StatusFound)
			return nil
		}
		if err := serveLocalOSSAsset(c.Writer, cfg, objectPath); err != nil {
			writePlainError(c.Writer, http.StatusNotFound, err.Error())
		}
		return nil
	})

	b.Get("/api/update/check", func(c *velo.BoxContext) interface{} {
		if app_updater == nil {
			return c.Error("Updater not initialized")
		}
		ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
		defer cancel()
		releaseInfo, err := app_updater.CheckForUpdatesForce(ctx)
		if err != nil {
			return c.Error(err.Error())
		}
		if releaseInfo != nil && releaseInfo.IsNewer {
			return c.Ok(velo.H{"hasUpdate": true, "version": releaseInfo.Version, "currentVersion": Version, "releaseNotes": releaseInfo.ReleaseNotes})
		}
		return c.Ok(velo.H{"hasUpdate": false, "currentVersion": Version})
	})
	b.Get("/api/update/download", func(c *velo.BoxContext) interface{} {
		if app_updater == nil {
			return c.Ok(velo.H{"success": false, "error": "Updater not initialized"})
		}
		ctx := c.Context()
		releaseInfo, err := app_updater.CheckForUpdatesForce(ctx)
		if err != nil {
			return c.Ok(velo.H{"success": false, "error": err.Error()})
		}
		if releaseInfo == nil || !releaseInfo.IsNewer {
			return c.Ok(velo.H{"success": false, "error": "No update available"})
		}
		updatePath, err := app_updater.DownloadUpdate(ctx, releaseInfo, func(progress utypes.DownloadProgress) {
			b.SendMessage(velo.H{
				"type":            "download_progress",
				"bytesDownloaded": progress.BytesDownloaded,
				"totalBytes":      progress.TotalBytes,
				"percentage":      progress.Percentage,
				"speed":           progress.Speed,
			})
		})
		if err != nil {
			return c.Ok(velo.H{"success": false, "error": err.Error()})
		}
		return c.Ok(velo.H{"success": true, "updatePath": updatePath})
	})
	b.Get("/api/update/restart", func(c *velo.BoxContext) interface{} {
		if app_updater == nil {
			return c.Ok(velo.H{"success": false, "error": "Updater not initialized"})
		}
		if err := app_updater.ApplyUpdateThenRestartApplication(c.Context()); err != nil {
			return c.Ok(velo.H{"success": false, "error": err.Error()})
		}
		return c.Ok(velo.H{"success": true})
	})
	b.Get("/api/update/skip", func(c *velo.BoxContext) interface{} {
		if app_updater == nil {
			return c.Ok(velo.H{"success": false, "error": "Updater not initialized"})
		}
		args, _ := c.Args().(map[string]interface{})
		v, _ := args["version"].(string)
		if v == "" {
			return c.Ok(velo.H{"success": false, "error": "version required"})
		}
		if err := app_updater.SkipVersion(v); err != nil {
			return c.Ok(velo.H{"success": false, "error": err.Error()})
		}
		return c.Ok(velo.H{"success": true})
	})

	b.Get("/api/open_window", func(c *velo.BoxContext) interface{} {
		pathname := c.Query("pathname")
		if pathname == "" {
			pathname = "/settings"
		}
		storageID := sanitizeStorageID(c.Query("storageId"))
		objectPath := cleanOSSObjectPath(c.Query("objectPath"))
		provider := strings.ToLower(strings.TrimSpace(c.Query("provider")))
		if pathname == "/oss-manager" && storageID != "" {
			pathname += "?storageId=" + url.QueryEscape(storageID)
		}
		if pathname == "/oss-storage-editor" {
			params := url.Values{}
			if storageID != "" {
				params.Set("storageId", storageID)
			}
			if provider != "" {
				params.Set("provider", provider)
			}
			if encoded := params.Encode(); encoded != "" {
				pathname += "?" + encoded
			}
		}
		if pathname == "/oss-preview" {
			params := url.Values{}
			if storageID != "" {
				params.Set("storageId", storageID)
			}
			if objectPath != "" {
				params.Set("objectPath", objectPath)
			}
			if encoded := params.Encode(); encoded != "" {
				pathname += "?" + encoded
			}
		}
		pathBase := pathname
		if index := strings.Index(pathBase, "?"); index >= 0 {
			pathBase = pathBase[:index]
		}
		entryPage := "index.html"
		name := "app-window"
		title := "App"
		width := 760
		height := 640
		if pathBase == "/settings" {
			entryPage = "settings.html"
			name = "settings"
			title = "App-Settings"
		}
		if pathBase == "/oss-manager" {
			entryPage = "oss-manager.html"
			name = "oss-manager"
			title = "OSS 文件管理"
			width = 1040
			height = 720
			if storageID != "" {
				name += "-" + storageID
			}
		}
		if pathBase == "/oss-storage-editor" {
			entryPage = "oss-storage-editor.html"
			name = "oss-storage-editor"
			title = "OSS 存储编辑"
			width = 760
			height = 720
		}
		if pathBase == "/oss-preview" {
			entryPage = "oss-preview.html"
			name = "oss-preview"
			title = "OSS 文件预览"
			width = 860
			height = 680
			if storageID != "" {
				name += "-" + storageID
			}
			if objectPath != "" {
				name += "-" + sanitizeStorageID(objectPath)
			}
		}
		b.OpenWindow(&velo.VeloWebviewOpt{
			Name:       name,
			Title:      title,
			Pathname:   pathname,
			Width:      width,
			Height:     height,
			EntryPage:  entryPage,
			FrontendFS: frontend_folder,
		})
		return c.Ok(velo.H{"success": true})
	})

	fmt.Println("starting server...")

	// 注册全局快捷键: Cmd+Shift+M (macOS) / Win+Shift+M (Windows) 显示/隐藏主窗口
	sm := shortcut.NewManager()
	sm.Register("MetaLeft+ShiftLeft+KeyM", func() {
		b.Webview.Show()
	})
	sm.Register("MetaLeft+ShiftLeft+KeyH", func() {
		b.Webview.Hide()
	})
	_ = sm

	as := autostart.New("MyApp")

	proxyEnabled := false
	proxyItem := &tray.MenuItem{Label: "设置系统代理", Click: func(m *tray.MenuItem) {
		proxyEnabled = !proxyEnabled
		if proxyEnabled {
			m.Check()
		} else {
			m.Uncheck()
		}
	}}

	autoStartItem := &tray.MenuItem{Label: "开机自启动", Checked: as.IsEnabled(), Click: func(m *tray.MenuItem) {
		if as.IsEnabled() {
			as.Disable()
			m.Uncheck()
		} else {
			as.Enable()
			m.Check()
		}
	}}

	tray.Setup(&tray.Tray{
		Icon:    appIcon,
		Tooltip: "MyApp",
		Menu: &tray.Menu{
			Items: []*tray.MenuItem{
				{Label: "显示主窗口", Click: func(m *tray.MenuItem) {
					b.Webview.Show()
				}},
				proxyItem,
				autoStartItem,
				{IsSeparator: true},
				{Label: "退出", Click: func(m *tray.MenuItem) {
					tray.Quit()
				}},
			},
		},
	})

	b.NewWebview(&velo.VeloWebviewOpt{
		Name:       "desktop",
		Title:      "App-Main",
		FrontendFS: frontend_folder,
		Pathname:   initialPathname,
		Width:      1024,
		Height:     768,
		OnDragDrop: func(event string, payload string) {
			if event != "drop" {
				return
			}
			files := droppedFilesFromPayload(payload, logger)
			if len(files) == 0 {
				return
			}
			b.SendMessage(velo.H{
				"type":  "memo_file_drop",
				"files": files,
			})
		},
	})
	b.Run()
}

func droppedFilesFromPayload(payload string, logger *zerolog.Logger) []velo.H {
	var paths []string
	if err := json.Unmarshal([]byte(payload), &paths); err != nil {
		logger.Error().Err(err).Msg("failed to parse dropped file payload")
		return nil
	}

	files := make([]velo.H, 0, len(paths))
	for _, path := range paths {
		file, err := droppedFileForPath(path)
		if err != nil {
			logger.Error().Err(err).Str("path", path).Msg("failed to read dropped file")
			continue
		}
		files = append(files, file)
	}
	return files
}

func droppedFileForPath(path string) (velo.H, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("dropped path is a directory")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return velo.H{
		"name":    filepath.Base(path),
		"path":    path,
		"size":    info.Size(),
		"type":    contentType,
		"dataURL": "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data),
	}, nil
}

func memoWindowMemoID(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || !json.Valid(raw) {
		return "", fmt.Errorf("memo is required")
	}
	var memo struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &memo); err != nil {
		return "", err
	}
	id := strings.TrimSpace(memo.ID)
	if id == "" {
		return "", fmt.Errorf("memo id is required")
	}
	return id, nil
}

func memoWindowMemosPayload(memo json.RawMessage, memos json.RawMessage) json.RawMessage {
	if len(memos) > 0 && json.Valid(memos) && strings.TrimSpace(string(memos)) != "null" {
		return memos
	}
	return json.RawMessage("[" + string(memo) + "]")
}

func normalizeExternalBrowserURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("url is required")
	}
	for _, r := range value {
		if r <= 0x20 || r == 0x7f {
			return "", fmt.Errorf("invalid url")
		}
	}

	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() {
		return "", fmt.Errorf("invalid url")
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("only http and https URLs can be opened externally")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return "", fmt.Errorf("url host is required")
	}
	parsed.Scheme = scheme
	return parsed.String(), nil
}

func openExternalBrowser(target string) error {
	cmd, err := externalBrowserCommand(target)
	if err != nil {
		return err
	}
	cmd.Env = os.Environ()
	return cmd.Start()
}

func externalBrowserCommand(target string) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target), nil
	default:
		return exec.Command("xdg-open", target), nil
	}
}

func externalBrowserConfirmMessage(target string) string {
	return "即将使用默认浏览器打开以下链接：\n\n" + target + "\n\n是否继续？"
}

type editorSpec struct {
	name        string
	priority    int
	appPath     string
	cmdBuilder  func(executable string, file string, line string, col string) *exec.Cmd
	fallbackURL func(file string, line string, col string) string
}

func splitEditorLocation(value string) (string, string, string) {
	file := strings.TrimSpace(value)
	line := "1"
	col := "1"
	if file == "" {
		return file, line, col
	}

	if parsed, err := url.Parse(file); err == nil && parsed.RawQuery != "" {
		query := parsed.Query()
		if value := query.Get("line"); value != "" {
			line = editorPositionValue(value, line)
			query.Del("line")
		}
		if value := firstNonEmpty(query.Get("col"), query.Get("column")); value != "" {
			col = editorPositionValue(value, col)
			query.Del("col")
			query.Del("column")
		}
		parsed.RawQuery = query.Encode()
		file = parsed.String()
	}

	if path, suffixLine, suffixCol, ok := splitEditorPositionSuffix(file); ok {
		file = path
		line = editorPositionValue(suffixLine, line)
		col = editorPositionValue(suffixCol, col)
	}

	return file, line, col
}

func editorPositionValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if isPositiveInteger(value) {
		return value
	}
	if isPositiveInteger(fallback) {
		return fallback
	}
	return "1"
}

func splitEditorPositionSuffix(value string) (string, string, string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || isEditorAssetReference(value) || isEditorOSSAssetURL(value) {
		return "", "", "", false
	}
	if hasNonLocalEditorScheme(value) {
		return "", "", "", false
	}

	lastColon := strings.LastIndex(value, ":")
	if lastColon <= 0 || lastColon == len(value)-1 {
		return "", "", "", false
	}
	lastPart := value[lastColon+1:]
	if !isPositiveInteger(lastPart) {
		return "", "", "", false
	}

	before := value[:lastColon]
	line := lastPart
	col := "1"
	path := before
	secondColon := strings.LastIndex(before, ":")
	if secondColon > 0 && secondColon < len(before)-1 && isPositiveInteger(before[secondColon+1:]) {
		path = before[:secondColon]
		line = before[secondColon+1:]
		col = lastPart
	}
	if strings.TrimSpace(path) == "" {
		return "", "", "", false
	}
	return path, line, col, true
}

func hasNonLocalEditorScheme(value string) bool {
	if isWindowsDrivePath(value) {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	return scheme != "file" && scheme != "local"
}

func isWindowsDrivePath(value string) bool {
	if len(value) < 3 || value[1] != ':' {
		return false
	}
	drive := value[0]
	return ((drive >= 'a' && drive <= 'z') || (drive >= 'A' && drive <= 'Z')) && (value[2] == '\\' || value[2] == '/')
}

func isPositiveInteger(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return strings.TrimLeft(value, "0") != ""
}

func resolveEditorFileTarget(value string, rawSettings json.RawMessage, storePath string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("file is required")
	}

	if storageID, key, ok, err := parseEditorAssetReference(value); ok || err != nil {
		if err != nil {
			return "", err
		}
		return resolveEditorAssetPath(storageID, key, rawSettings, storePath)
	}

	if storageID, key, ok, err := parseEditorOSSAssetURL(value); ok || err != nil {
		if err != nil {
			return "", err
		}
		return resolveEditorAssetPath(storageID, key, rawSettings, storePath)
	}

	if path, ok, err := editorLocalURLPath(value); ok || err != nil {
		if err != nil {
			return "", err
		}
		value = path
	}

	return expandLocalPath(value), nil
}

func resolveEditorAssetPath(storageID string, key string, rawSettings json.RawMessage, storePath string) (string, error) {
	cleanKey := cleanOSSObjectPath(key)
	if cleanKey == "" {
		return "", fmt.Errorf("file path is required")
	}
	cfg, err := storedOSSConfig(rawSettings, storageID, storePath)
	if err != nil {
		return "", err
	}
	if !isLocalOSSConfig(cfg) {
		return "", fmt.Errorf("asset is not in local storage")
	}
	return localOSSObjectDiskPath(cfg, cleanKey)
}

func parseEditorAssetReference(value string) (string, string, bool, error) {
	if !isEditorAssetReference(value) {
		return "", "", false, nil
	}
	rest := strings.TrimPrefix(strings.TrimSpace(value), "@assets/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", "", true, fmt.Errorf("invalid asset reference")
	}
	key, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", "", true, err
	}
	return parts[0], key, true, nil
}

func isEditorAssetReference(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "@assets/")
}

func parseEditorOSSAssetURL(value string) (string, string, bool, error) {
	if !isEditorOSSAssetURL(value) {
		return "", "", false, nil
	}
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", "", true, err
	}
	query := parsed.Query()
	storageID := firstNonEmpty(query.Get("storageId"), query.Get("storageID"), query.Get("id"))
	key := firstNonEmpty(query.Get("path"), query.Get("key"))
	if strings.TrimSpace(key) == "" {
		return "", "", true, fmt.Errorf("file path is required")
	}
	return storageID, key, true, nil
}

func isEditorOSSAssetURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Path == "/api/oss/assets"
}

func editorLocalURLPath(value string) (string, bool, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return "", false, err
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "file" && scheme != "local" {
		return "", false, nil
	}

	path := parsed.Path
	if parsed.Host != "" && !strings.EqualFold(parsed.Host, "localhost") {
		if runtime.GOOS == "windows" {
			path = parsed.Host + parsed.Path
		} else {
			path = string(os.PathSeparator) + parsed.Host + parsed.Path
		}
	}
	if path == "" {
		path = parsed.Opaque
	}
	if path == "" {
		return "", true, fmt.Errorf("file path is required")
	}
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return "", true, err
	}
	return decoded, true, nil
}

func openFileInEditor(file string, line string, col string, preferredEditor string) error {
	absoluteFile := expandLocalPath(file)
	if !filepath.IsAbs(absoluteFile) && !isWindowsDrivePath(absoluteFile) {
		cwd, err := os.Getwd()
		if err == nil {
			absoluteFile = filepath.Join(cwd, absoluteFile)
		}
	}
	info, err := os.Stat(absoluteFile)
	if err != nil {
		return fmt.Errorf("file not found: %s", absoluteFile)
	}
	if info.IsDir() {
		return fmt.Errorf("folder cannot be opened in editor: %s", absoluteFile)
	}

	spec, executable, err := chooseEditor(preferredEditor)
	if err != nil {
		return err
	}

	line = editorPositionValue(line, "1")
	col = editorPositionValue(col, "1")
	cmd := spec.cmdBuilder(executable, absoluteFile, line, col)
	cmd.Env = os.Environ()
	if err := cmd.Start(); err != nil {
		if spec.fallbackURL == nil || runtime.GOOS != "darwin" {
			return fmt.Errorf("failed to launch editor: %w", err)
		}
		fallback := exec.Command("open", spec.fallbackURL(absoluteFile, line, col))
		fallback.Env = os.Environ()
		if fallbackErr := fallback.Start(); fallbackErr != nil {
			return fmt.Errorf("failed to launch editor via URL scheme: %w", fallbackErr)
		}
	}
	return nil
}

func chooseEditor(preferredEditor string) (*editorSpec, string, error) {
	editors := editorSpecs()
	preferredEditor = normalizeEditorName(preferredEditor)
	if preferredEditor != "" {
		for i := range editors {
			if editors[i].name != preferredEditor {
				continue
			}
			if executable := editorExecutable(editors[i]); executable != "" {
				return &editors[i], executable, nil
			}
			break
		}
	}

	for _, envName := range []string{os.Getenv("EDITOR"), os.Getenv("GIT_EDITOR")} {
		fields := strings.Fields(envName)
		if len(fields) == 0 {
			continue
		}
		base := normalizeEditorName(filepath.Base(fields[0]))
		if base == "" {
			continue
		}
		for i := range editors {
			if editors[i].name != base {
				continue
			}
			if executable := editorExecutable(editors[i]); executable != "" {
				return &editors[i], executable, nil
			}
		}
	}

	var chosen *editorSpec
	chosenExecutable := ""
	for i := range editors {
		executable := editorExecutable(editors[i])
		if executable == "" {
			continue
		}
		if chosen == nil || editors[i].priority > chosen.priority {
			chosen = &editors[i]
			chosenExecutable = executable
		}
	}
	if chosen == nil {
		return nil, "", fmt.Errorf("no editor found")
	}
	return chosen, chosenExecutable, nil
}

func editorSpecs() []editorSpec {
	return []editorSpec{
		{
			name:     "trae",
			priority: 10,
			appPath:  "/Applications/Trae.app/Contents/MacOS/trae",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, file)
			},
			fallbackURL: func(file string, line string, col string) string {
				return fmt.Sprintf("trae://file/%s:%s:%s", file, line, col)
			},
		},
		{
			name:     "cursor",
			priority: 10,
			appPath:  "/Applications/Cursor.app/Contents/MacOS/cursor",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "--goto", fmt.Sprintf("%s:%s:%s", file, line, col))
			},
			fallbackURL: func(file string, line string, col string) string {
				return fmt.Sprintf("cursor://file/%s:%s:%s", file, line, col)
			},
		},
		{
			name:     "code",
			priority: 10,
			appPath:  "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "-g", fmt.Sprintf("%s:%s:%s", file, line, col))
			},
		},
		{
			name:     "code-insiders",
			priority: 10,
			appPath:  "/Applications/Visual Studio Code - Insiders.app/Contents/Resources/app/bin/code-insiders",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "-g", fmt.Sprintf("%s:%s:%s", file, line, col))
			},
		},
		{
			name:     "webstorm",
			priority: 5,
			appPath:  "/Applications/WebStorm.app/Contents/MacOS/webstorm",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "--line", line, file)
			},
		},
		{
			name:     "idea",
			priority: 5,
			appPath:  "/Applications/IntelliJ IDEA.app/Contents/MacOS/idea",
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "--line", line, file)
			},
		},
		{
			name:     "vim",
			priority: 3,
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "+"+line, file)
			},
		},
		{
			name:     "nvim",
			priority: 3,
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "+"+line, file)
			},
		},
		{
			name:     "emacs",
			priority: 2,
			cmdBuilder: func(executable string, file string, line string, col string) *exec.Cmd {
				return exec.Command(executable, "+"+line, file)
			},
		},
	}
}

func editorExecutable(spec editorSpec) string {
	if path, err := exec.LookPath(spec.name); err == nil {
		return path
	}
	if runtime.GOOS == "darwin" && spec.appPath != "" {
		if _, err := os.Stat(spec.appPath); err == nil {
			return spec.appPath
		}
	}
	return ""
}

func normalizeEditorName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "vscode", "vs-code", "visual-studio-code":
		return "code"
	default:
		return value
	}
}

func imageFileExtensions() []string {
	return []string{"avif", "bmp", "gif", "jpg", "jpeg", "png", "svg", "webp"}
}

func uploadOSSObject(parent context.Context, req OSSUploadRequest) (velo.H, error) {
	cfg := req.Config
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return uploadLocalOSSObject(parent, req)
	}

	data, err := decodeUploadContent(req.ContentBase64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()

	key := objectKey(cfg.PathPrefix, req.Name)
	contentType := strings.TrimSpace(req.Type)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, err
	}

	return velo.H{
		"bucket":    cfg.Bucket,
		"key":       key,
		"name":      req.Name,
		"ref":       assetRef(cfg.ID, key),
		"size":      len(data),
		"storageId": cfg.ID,
		"type":      contentType,
		"url":       publicOSSObjectURL(cfg, endpoint, key),
	}, nil
}

func storedOSSConfig(raw json.RawMessage, storageID string, storePath string) (OSSConfig, error) {
	settings, err := loadStoredCloudStorageSettings(raw)
	if err != nil {
		return OSSConfig{}, err
	}
	settings, _, err = prepareCloudStorageSettings(settings, storePath, len(settings.Storages) == 0)
	if err != nil {
		return OSSConfig{}, err
	}
	cfg, err := activeOSSConfig(settings, storageID)
	if err != nil {
		return OSSConfig{}, err
	}
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, storageID, "default"))
	return cfg, nil
}

func listOSSFiles(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return listLocalOSSFiles(parent, cfg, objectPath)
	}

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	cleanPath := cleanOSSObjectPath(objectPath)
	prefix := ossFolderPrefix(cleanPath)
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(cfg.Bucket),
		Delimiter: aws.String("/"),
		MaxKeys:   1000,
		Prefix:    aws.String(prefix),
	}
	seen := map[string]bool{}
	items := make([]OSSFileView, 0)
	for {
		out, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, err
		}

		for _, commonPrefix := range out.CommonPrefixes {
			key := stringValue(commonPrefix.Prefix)
			view := ossFileView(cfg, endpoint, cleanPath, key, true, 0, nil, "")
			if view.Path == "" || seen[view.Path] {
				continue
			}
			seen[view.Path] = true
			items = append(items, view)
		}

		for _, object := range out.Contents {
			key := stringValue(object.Key)
			if key == "" || key == prefix {
				continue
			}
			isDir := strings.HasSuffix(key, "/")
			view := ossFileView(cfg, endpoint, cleanPath, key, isDir, object.Size, object.LastModified, "")
			if view.Path == "" || seen[view.Path] {
				continue
			}
			seen[view.Path] = true
			items = append(items, view)
		}

		if !out.IsTruncated || out.NextContinuationToken == nil {
			break
		}
		input.ContinuationToken = out.NextContinuationToken
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	return velo.H{
		"bucket":    cfg.Bucket,
		"list":      items,
		"path":      cleanPath,
		"prefix":    prefix,
		"storageId": cfg.ID,
	}, nil
}

func previewOSSFile(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return previewLocalOSSFile(parent, cfg, objectPath)
	}
	key := cleanOSSObjectPath(objectPath)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}

	client, _, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	head, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	if head.ContentLength > 8*1024*1024 {
		return nil, fmt.Errorf("file is too large to preview, max size is 8 MB")
	}

	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(cfg.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer out.Body.Close()

	content, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, err
	}

	name := pathpkg.Base(key)
	ext := strings.ToLower(filepath.Ext(name))
	contentType := firstNonEmpty(stringValue(out.ContentType), stringValue(head.ContentType), mime.TypeByExtension(ext), "application/octet-stream")
	if isTextPreview(ext, contentType) {
		return velo.H{
			"content":  string(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "text",
		}, nil
	}
	if strings.HasPrefix(contentType, "image/") {
		return velo.H{
			"content":  base64.StdEncoding.EncodeToString(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "image",
		}, nil
	}
	if contentType == "application/pdf" {
		return velo.H{
			"content":  base64.StdEncoding.EncodeToString(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "pdf",
		}, nil
	}
	return velo.H{
		"mimeType": contentType,
		"name":     name,
		"path":     key,
		"size":     head.ContentLength,
		"type":     "unknown",
	}, nil
}

func makeOSSFolder(parent context.Context, cfg OSSConfig, req OSSFileMkdirRequest) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return makeLocalOSSFolder(parent, cfg, req)
	}

	folderPath := cleanOSSObjectPath(req.Path)
	if strings.TrimSpace(req.Name) != "" {
		folderPath = objectPathJoin(folderPath, req.Name)
	}
	if folderPath == "" {
		return nil, fmt.Errorf("folder path is required")
	}

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	key := ossFolderPrefix(folderPath)
	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(nil),
		ContentType: aws.String("application/x-directory"),
	})
	if err != nil {
		return nil, err
	}

	return velo.H{
		"file":      ossFileView(cfg, endpoint, pathpkg.Dir(folderPath), key, true, 0, nil, "application/x-directory"),
		"path":      folderPath,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func deleteOSSFile(parent context.Context, cfg OSSConfig, req OSSFileDeleteRequest) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return deleteLocalOSSFile(parent, cfg, req)
	}

	key := cleanOSSObjectPath(req.Path)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}

	client, _, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	deleted := 0
	if req.IsDir {
		prefix := ossFolderPrefix(key)
		input := &s3.ListObjectsV2Input{
			Bucket:  aws.String(cfg.Bucket),
			MaxKeys: 1000,
			Prefix:  aws.String(prefix),
		}
		for {
			out, err := client.ListObjectsV2(ctx, input)
			if err != nil {
				return nil, err
			}
			for _, object := range out.Contents {
				objectKey := stringValue(object.Key)
				if objectKey == "" {
					continue
				}
				if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
					Bucket: aws.String(cfg.Bucket),
					Key:    aws.String(objectKey),
				}); err != nil {
					return nil, err
				}
				deleted++
			}
			if !out.IsTruncated || out.NextContinuationToken == nil {
				break
			}
			input.ContinuationToken = out.NextContinuationToken
		}
	} else {
		if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(cfg.Bucket),
			Key:    aws.String(key),
		}); err != nil {
			return nil, err
		}
		deleted = 1
	}

	return velo.H{
		"deleted":   deleted,
		"path":      key,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func uploadOSSManagedFile(parent context.Context, cfg OSSConfig, req OSSFileUploadRequest) (velo.H, error) {
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, err
	}
	if isLocalOSSConfig(cfg) {
		return uploadLocalOSSManagedFile(parent, cfg, req)
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("file name is required")
	}

	data, err := decodeUploadContent(req.ContentBase64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}

	client, endpoint, err := newOSSClient(cfg)
	if err != nil {
		return nil, err
	}

	key := objectPathJoin(req.Path, req.Name)
	if key == "" {
		return nil, fmt.Errorf("object key is required")
	}
	contentType := strings.TrimSpace(req.Type)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx, cancel := context.WithTimeout(parent, 90*time.Second)
	defer cancel()
	_, err = client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(cfg.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, err
	}

	return velo.H{
		"bucket":    cfg.Bucket,
		"file":      ossFileView(cfg, endpoint, cleanOSSObjectPath(req.Path), key, false, int64(len(data)), nil, contentType),
		"key":       key,
		"name":      sanitizeObjectName(req.Name),
		"ref":       assetRef(cfg.ID, key),
		"size":      len(data),
		"storageId": cfg.ID,
		"success":   true,
		"type":      contentType,
		"url":       publicOSSObjectURL(cfg, endpoint, key),
	}, nil
}

func uploadLocalOSSObject(parent context.Context, req OSSUploadRequest) (velo.H, error) {
	cfg := req.Config
	cfg.ID = sanitizeStorageID(firstNonEmpty(cfg.ID, req.StorageID, "default"))
	if err := validateOSSConfig(cfg); err != nil {
		return nil, err
	}
	data, err := decodeUploadContent(req.ContentBase64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	key := objectKey(cfg.PathPrefix, req.Name)
	contentType := strings.TrimSpace(req.Type)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := writeLocalOSSObject(parent, cfg, key, data); err != nil {
		return nil, err
	}
	return velo.H{
		"bucket":    cfg.Bucket,
		"key":       key,
		"name":      req.Name,
		"ref":       assetRef(cfg.ID, key),
		"size":      len(data),
		"storageId": cfg.ID,
		"type":      contentType,
		"url":       publicOSSObjectURL(cfg, "", key),
	}, nil
}

func listLocalOSSFiles(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	cleanPath := cleanOSSObjectPath(objectPath)
	target, err := localOSSObjectDiskPath(cfg, cleanPath)
	if err != nil {
		return nil, err
	}
	if _, err := ensureLocalOSSBucket(cfg); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		if os.IsNotExist(err) {
			return velo.H{
				"bucket":    cfg.Bucket,
				"list":      []OSSFileView{},
				"path":      cleanPath,
				"prefix":    ossFolderPrefix(cleanPath),
				"storageId": cfg.ID,
			}, nil
		}
		return nil, err
	}
	items := make([]OSSFileView, 0, len(entries))
	for _, entry := range entries {
		select {
		case <-parent.Done():
			return nil, parent.Err()
		default:
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		key := objectPathJoin(cleanPath, entry.Name())
		modTime := info.ModTime()
		size := info.Size()
		if entry.IsDir() {
			size = 0
		}
		items = append(items, ossFileView(cfg, "", cleanPath, key, entry.IsDir(), size, &modTime, ""))
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})
	return velo.H{
		"bucket":    cfg.Bucket,
		"list":      items,
		"path":      cleanPath,
		"prefix":    ossFolderPrefix(cleanPath),
		"storageId": cfg.ID,
	}, nil
}

func previewLocalOSSFile(parent context.Context, cfg OSSConfig, objectPath string) (velo.H, error) {
	key := cleanOSSObjectPath(objectPath)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(target)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("folder cannot be previewed")
	}
	if info.Size() > 8*1024*1024 {
		return nil, fmt.Errorf("file is too large to preview, max size is 8 MB")
	}
	select {
	case <-parent.Done():
		return nil, parent.Err()
	default:
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	name := pathpkg.Base(key)
	ext := strings.ToLower(filepath.Ext(name))
	contentType := firstNonEmpty(mime.TypeByExtension(ext), http.DetectContentType(content), "application/octet-stream")
	if isTextPreview(ext, contentType) {
		return velo.H{
			"content":  string(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "text",
		}, nil
	}
	if strings.HasPrefix(contentType, "image/") {
		return velo.H{
			"content":  base64.StdEncoding.EncodeToString(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "image",
		}, nil
	}
	if contentType == "application/pdf" {
		return velo.H{
			"content":  base64.StdEncoding.EncodeToString(content),
			"mimeType": contentType,
			"name":     name,
			"path":     key,
			"size":     len(content),
			"type":     "pdf",
		}, nil
	}
	return velo.H{
		"mimeType": contentType,
		"name":     name,
		"path":     key,
		"size":     info.Size(),
		"type":     "unknown",
	}, nil
}

func makeLocalOSSFolder(parent context.Context, cfg OSSConfig, req OSSFileMkdirRequest) (velo.H, error) {
	folderPath := cleanOSSObjectPath(req.Path)
	if strings.TrimSpace(req.Name) != "" {
		folderPath = objectPathJoin(folderPath, req.Name)
	}
	if folderPath == "" {
		return nil, fmt.Errorf("folder path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, folderPath)
	if err != nil {
		return nil, err
	}
	select {
	case <-parent.Done():
		return nil, parent.Err()
	default:
	}
	if err := os.MkdirAll(target, 0755); err != nil {
		return nil, err
	}
	return velo.H{
		"file":      ossFileView(cfg, "", pathpkg.Dir(folderPath), folderPath, true, 0, nil, "application/x-directory"),
		"path":      folderPath,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func deleteLocalOSSFile(parent context.Context, cfg OSSConfig, req OSSFileDeleteRequest) (velo.H, error) {
	key := cleanOSSObjectPath(req.Path)
	if key == "" {
		return nil, fmt.Errorf("file path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return nil, err
	}
	select {
	case <-parent.Done():
		return nil, parent.Err()
	default:
	}
	if err := os.RemoveAll(target); err != nil {
		return nil, err
	}
	return velo.H{
		"deleted":   1,
		"path":      key,
		"storageId": cfg.ID,
		"success":   true,
	}, nil
}

func uploadLocalOSSManagedFile(parent context.Context, cfg OSSConfig, req OSSFileUploadRequest) (velo.H, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("file name is required")
	}
	data, err := decodeUploadContent(req.ContentBase64)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file content is empty")
	}
	key := objectPathJoin(req.Path, req.Name)
	if key == "" {
		return nil, fmt.Errorf("object key is required")
	}
	contentType := strings.TrimSpace(req.Type)
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := writeLocalOSSObject(parent, cfg, key, data); err != nil {
		return nil, err
	}
	return velo.H{
		"bucket":    cfg.Bucket,
		"file":      ossFileView(cfg, "", cleanOSSObjectPath(req.Path), key, false, int64(len(data)), nil, contentType),
		"key":       key,
		"name":      sanitizeObjectName(req.Name),
		"ref":       assetRef(cfg.ID, key),
		"size":      len(data),
		"storageId": cfg.ID,
		"success":   true,
		"type":      contentType,
		"url":       publicOSSObjectURL(cfg, "", key),
	}, nil
}

func writeLocalOSSObject(parent context.Context, cfg OSSConfig, key string, data []byte) error {
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return err
	}
	select {
	case <-parent.Done():
		return parent.Err()
	default:
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return os.WriteFile(target, data, 0644)
}

func serveLocalOSSAsset(w http.ResponseWriter, cfg OSSConfig, objectPath string) error {
	key := cleanOSSObjectPath(objectPath)
	if key == "" {
		return fmt.Errorf("file path is required")
	}
	target, err := localOSSObjectDiskPath(cfg, key)
	if err != nil {
		return err
	}
	info, err := os.Stat(target)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("folder cannot be served")
	}
	file, err := os.Open(target)
	if err != nil {
		return err
	}
	defer file.Close()
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(target)))
	if contentType == "" {
		buffer := make([]byte, 512)
		n, _ := file.Read(buffer)
		contentType = http.DetectContentType(buffer[:n])
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return err
		}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, err = io.Copy(w, file)
	return err
}

func writePlainError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(message))
}

func isLocalOSSConfig(cfg OSSConfig) bool {
	provider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	return provider == "local" || provider == "local-oss"
}

func ensureLocalOSSBucket(cfg OSSConfig) (string, error) {
	root, err := localOSSBucketRoot(cfg)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return "", err
	}
	return root, nil
}

func localOSSBucketRoot(cfg OSSConfig) (string, error) {
	if err := validateOSSAccessConfig(cfg); err != nil {
		return "", err
	}
	root := expandLocalPath(strings.TrimSpace(cfg.Endpoint))
	if root == "" {
		return "", fmt.Errorf("local root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	bucket := strings.TrimSpace(cfg.Bucket)
	if err := validateLocalOSSBucket(bucket); err != nil {
		return "", err
	}
	return filepath.Join(absRoot, bucket), nil
}

func localOSSObjectDiskPath(cfg OSSConfig, objectPath string) (string, error) {
	bucketRoot, err := ensureLocalOSSBucket(cfg)
	if err != nil {
		return "", err
	}
	cleanKey := cleanOSSObjectPath(objectPath)
	target := bucketRoot
	if cleanKey != "" {
		target = filepath.Join(bucketRoot, filepath.FromSlash(cleanKey))
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if absTarget != bucketRoot && !strings.HasPrefix(absTarget, bucketRoot+string(os.PathSeparator)) {
		return "", fmt.Errorf("object path escapes bucket root: %s", objectPath)
	}
	return absTarget, nil
}

func expandLocalPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(value, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(value, "~/"))
		}
	}
	return value
}

func validateLocalOSSBucket(bucket string) error {
	bucket = strings.TrimSpace(bucket)
	if bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if bucket == "." || bucket == ".." {
		return fmt.Errorf("invalid bucket: %s", bucket)
	}
	if strings.ContainsAny(bucket, `/\`) {
		return fmt.Errorf("bucket must not contain path separators: %s", bucket)
	}
	return nil
}

func loadStoredCloudStorageSettings(raw json.RawMessage) (CloudStorageSettings, error) {
	if raw == nil {
		return normalizeCloudStorageSettings(CloudStorageSettings{}), nil
	}

	var settings CloudStorageSettings
	if err := json.Unmarshal(raw, &settings); err == nil && (settings.Storages != nil || strings.TrimSpace(settings.ActiveStorageID) != "") {
		return normalizeCloudStorageSettings(settings), nil
	}

	var cfg OSSConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return CloudStorageSettings{}, fmt.Errorf("read cloud storage config: %w", err)
	}
	if !hasOSSConfig(cfg) {
		return CloudStorageSettings{}, fmt.Errorf("cloud storage config is empty")
	}
	cfg.ID = firstNonEmpty(cfg.ID, "default")
	cfg.Name = firstNonEmpty(cfg.Name, "默认存储")
	return normalizeCloudStorageSettings(CloudStorageSettings{
		ActiveStorageID: cfg.ID,
		Storages:        []OSSConfig{cfg},
	}), nil
}

func normalizeCloudStorageSettings(settings CloudStorageSettings) CloudStorageSettings {
	seen := map[string]int{}
	next := make([]OSSConfig, 0, len(settings.Storages))
	for i, cfg := range settings.Storages {
		cfg.Provider = strings.ToLower(strings.TrimSpace(cfg.Provider))
		if cfg.Provider == "" {
			cfg.Provider = "s3"
		}
		cfg.Endpoint = strings.TrimSpace(cfg.Endpoint)
		cfg.Bucket = strings.TrimSpace(cfg.Bucket)
		cfg.PathPrefix = strings.TrimSpace(cfg.PathPrefix)
		cfg.PublicBaseURL = strings.TrimSpace(cfg.PublicBaseURL)
		cfg.Region = strings.TrimSpace(cfg.Region)
		baseID := sanitizeStorageID(cfg.ID)
		if baseID == "" {
			baseID = sanitizeStorageID(firstNonEmpty(cfg.Name, cfg.Provider, cfg.Bucket))
		}
		if baseID == "" {
			baseID = fmt.Sprintf("storage-%d", i+1)
		}
		seen[baseID]++
		if seen[baseID] > 1 {
			baseID = fmt.Sprintf("%s-%d", baseID, seen[baseID])
		}
		cfg.ID = baseID
		if strings.TrimSpace(cfg.Name) == "" {
			cfg.Name = storageDisplayName(cfg, i)
		} else {
			cfg.Name = strings.TrimSpace(cfg.Name)
		}
		next = append(next, cfg)
	}

	activeID := sanitizeStorageID(settings.ActiveStorageID)
	if !storageIDExists(next, activeID) {
		activeID = ""
		for _, cfg := range next {
			if cfg.Enabled {
				activeID = cfg.ID
				break
			}
		}
		if activeID == "" && len(next) > 0 {
			activeID = next[0].ID
		}
	}

	return CloudStorageSettings{
		ActiveStorageID:     activeID,
		DefaultsInitialized: settings.DefaultsInitialized,
		Storages:            next,
	}
}

func prepareCloudStorageSettings(settings CloudStorageSettings, storePath string, initializeDefault bool) (CloudStorageSettings, bool, error) {
	settings = normalizeCloudStorageSettings(settings)
	changed := false
	if initializeDefault || len(settings.Storages) == 0 {
		defaultCfg := defaultLocalMemoOSSConfig(storePath)
		if !storageIDExists(settings.Storages, defaultCfg.ID) {
			settings.Storages = append(settings.Storages, defaultCfg)
			changed = true
		}
		if settings.ActiveStorageID == "" {
			settings.ActiveStorageID = defaultCfg.ID
			changed = true
		}
	}
	defaultRoot := defaultLocalStorageRoot(storePath)
	legacyRoot := legacyDefaultLocalStorageRoot(storePath)
	for i := range settings.Storages {
		cfg := settings.Storages[i]
		if cfg.ID == "memo-local" && isLocalOSSConfig(cfg) && samePath(cfg.Endpoint, legacyRoot) && !samePath(cfg.Endpoint, defaultRoot) {
			settings.Storages[i].Endpoint = defaultRoot
			changed = true
		}
	}
	if !settings.DefaultsInitialized {
		settings.DefaultsInitialized = true
		changed = true
	}
	settings = normalizeCloudStorageSettings(settings)
	for _, cfg := range settings.Storages {
		if isLocalOSSConfig(cfg) && strings.TrimSpace(cfg.Endpoint) != "" && strings.TrimSpace(cfg.Bucket) != "" {
			if _, err := ensureLocalOSSBucket(cfg); err != nil {
				return CloudStorageSettings{}, changed, err
			}
		}
	}
	return settings, changed, nil
}

func cloudStorageDefaults(storePath string) velo.H {
	return velo.H{
		"localRoot":  defaultLocalStorageRoot(storePath),
		"memoBucket": "memos",
	}
}

func defaultLocalMemoOSSConfig(storePath string) OSSConfig {
	return OSSConfig{
		Bucket:         "memos",
		Enabled:        true,
		Endpoint:       defaultLocalStorageRoot(storePath),
		ForcePathStyle: true,
		ID:             "memo-local",
		Name:           "本地 Memo 存储",
		Provider:       "local",
		UseSSL:         false,
	}
}

func defaultLocalStorageRoot(storePath string) string {
	if vault := activeVaultSnapshot(); vault != nil && strings.TrimSpace(vault.RootDir) != "" {
		return filepath.Join(vault.RootDir, "storage")
	}
	if root := vaultRootFromStorePath(storePath); root != "" {
		return filepath.Join(root, "storage")
	}
	base := filepath.Dir(strings.TrimSpace(storePath))
	if base == "" || base == "." {
		base = projectDir()
	}
	return filepath.Join(base, "storage")
}

func legacyDefaultLocalStorageRoot(storePath string) string {
	base := filepath.Dir(strings.TrimSpace(storePath))
	if base == "" || base == "." {
		base = projectDir()
	}
	return filepath.Join(base, "workdir", "storage")
}

func vaultRootFromStorePath(storePath string) string {
	path := strings.TrimSpace(storePath)
	if path == "" {
		return ""
	}
	configDir := filepath.Dir(path)
	if filepath.Base(configDir) != vaultConfigDirName {
		return ""
	}
	root := filepath.Dir(configDir)
	if root == "." || root == string(filepath.Separator) {
		return ""
	}
	return root
}

func activeOSSConfig(settings CloudStorageSettings, storageID string) (OSSConfig, error) {
	settings = normalizeCloudStorageSettings(settings)
	targetID := sanitizeStorageID(storageID)
	if targetID == "" {
		targetID = settings.ActiveStorageID
	}
	if targetID == "" {
		return OSSConfig{}, fmt.Errorf("cloud storage config is not saved")
	}
	for _, cfg := range settings.Storages {
		if cfg.ID == targetID {
			return cfg, nil
		}
	}
	return OSSConfig{}, fmt.Errorf("cloud storage profile not found: %s", targetID)
}

func storageIDExists(storages []OSSConfig, id string) bool {
	if id == "" {
		return false
	}
	for _, cfg := range storages {
		if cfg.ID == id {
			return true
		}
	}
	return false
}

func storageDisplayName(cfg OSSConfig, index int) string {
	for _, value := range []string{cfg.Bucket, cfg.Provider, cfg.ID} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return fmt.Sprintf("存储 %d", index+1)
}

func sanitizeStorageID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-_")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func hasOSSConfig(cfg OSSConfig) bool {
	return cfg.Enabled ||
		strings.TrimSpace(cfg.Endpoint) != "" ||
		strings.TrimSpace(cfg.Bucket) != "" ||
		strings.TrimSpace(cfg.AccessKeyID) != "" ||
		strings.TrimSpace(cfg.SecretAccessKey) != "" ||
		strings.TrimSpace(cfg.SessionToken) != "" ||
		strings.TrimSpace(cfg.PublicBaseURL) != "" ||
		strings.TrimSpace(cfg.PathPrefix) != "" ||
		strings.TrimSpace(cfg.Region) != ""
}

func validateOSSConfig(cfg OSSConfig) error {
	if !cfg.Enabled {
		return fmt.Errorf("cloud storage is not enabled")
	}
	return validateOSSAccessConfig(cfg)
}

func validateOSSAccessConfig(cfg OSSConfig) error {
	missing := make([]string, 0, 5)
	if strings.TrimSpace(cfg.Endpoint) == "" {
		if isLocalOSSConfig(cfg) {
			missing = append(missing, "local root")
		} else {
			missing = append(missing, "endpoint")
		}
	}
	if strings.TrimSpace(cfg.Bucket) == "" {
		missing = append(missing, "bucket")
	}
	if !isLocalOSSConfig(cfg) {
		if strings.TrimSpace(cfg.AccessKeyID) == "" {
			missing = append(missing, "access key id")
		}
		if strings.TrimSpace(cfg.SecretAccessKey) == "" {
			missing = append(missing, "secret access key")
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("cloud storage config missing: %s", strings.Join(missing, ", "))
	}
	if isLocalOSSConfig(cfg) {
		return validateLocalOSSBucket(cfg.Bucket)
	}
	return nil
}

func newOSSClient(cfg OSSConfig) (*s3.Client, string, error) {
	if err := validateOSSAccessConfig(cfg); err != nil {
		return nil, "", err
	}

	endpoint := normalizeOSSEndpoint(cfg.Endpoint, cfg.UseSSL)
	region := strings.TrimSpace(cfg.Region)
	if region == "" {
		region = "auto"
	}

	awsCfg := aws.Config{
		Region: region,
		Credentials: aws.NewCredentialsCache(aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     cfg.AccessKeyID,
				SecretAccessKey: cfg.SecretAccessKey,
				SessionToken:    cfg.SessionToken,
				Source:          "oss-config",
			}, nil
		})),
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.EndpointResolver = s3.EndpointResolverFromURL(endpoint)
		o.UsePathStyle = cfg.ForcePathStyle
	})
	return client, endpoint, nil
}

func decodeUploadContent(contentBase64 string) ([]byte, error) {
	value := strings.TrimSpace(contentBase64)
	if value == "" {
		return nil, fmt.Errorf("content_base64 is required")
	}
	if comma := strings.Index(value, ","); strings.HasPrefix(value, "data:") && comma >= 0 {
		value = value[comma+1:]
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode content_base64: %w", err)
	}
	return data, nil
}

func cleanOSSObjectPath(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	value = strings.Trim(value, "/")
	if value == "" || value == "." {
		return ""
	}

	parts := strings.Split(value, "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			if len(clean) > 0 {
				clean = clean[:len(clean)-1]
			}
			continue
		}
		clean = append(clean, part)
	}
	return strings.Join(clean, "/")
}

func ossFolderPrefix(objectPath string) string {
	cleanPath := cleanOSSObjectPath(objectPath)
	if cleanPath == "" {
		return ""
	}
	return cleanPath + "/"
}

func objectPathJoin(parent string, name string) string {
	cleanParent := cleanOSSObjectPath(parent)
	cleanName := sanitizeObjectName(name)
	if cleanName == "" {
		return cleanParent
	}
	if cleanParent == "" {
		return cleanName
	}
	return pathpkg.Join(cleanParent, cleanName)
}

func ossFileView(cfg OSSConfig, endpoint string, parent string, key string, isDir bool, size int64, modTime *time.Time, contentType string) OSSFileView {
	cleanKey := cleanOSSObjectPath(key)
	name := ossFileName(parent, cleanKey)
	if name == "" {
		name = pathpkg.Base(cleanKey)
	}
	if contentType == "" && !isDir {
		contentType = mime.TypeByExtension(strings.ToLower(filepath.Ext(name)))
	}
	if contentType == "" {
		if isDir {
			contentType = "folder"
		} else {
			contentType = "application/octet-stream"
		}
	}

	ref := ""
	publicURL := ""
	if !isDir {
		ref = assetRef(cfg.ID, cleanKey)
		publicURL = publicOSSObjectURL(cfg, endpoint, cleanKey)
	}

	modTimeText := ""
	if modTime != nil && !modTime.IsZero() {
		modTimeText = modTime.Format(time.RFC3339)
	}
	return OSSFileView{
		ID:      cleanKey,
		IsDir:   isDir,
		ModTime: modTimeText,
		Name:    name,
		Path:    cleanKey,
		Ref:     ref,
		Size:    size,
		Type:    contentType,
		URL:     publicURL,
	}
}

func ossFileName(parent string, key string) string {
	cleanKey := cleanOSSObjectPath(key)
	cleanParent := cleanOSSObjectPath(parent)
	rel := cleanKey
	if cleanParent != "" && strings.HasPrefix(rel, cleanParent+"/") {
		rel = strings.TrimPrefix(rel, cleanParent+"/")
	}
	if index := strings.Index(rel, "/"); index >= 0 {
		rel = rel[:index]
	}
	return rel
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func isTextPreview(ext string, contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if strings.HasPrefix(contentType, "text/") ||
		strings.Contains(contentType, "json") ||
		strings.Contains(contentType, "javascript") ||
		strings.Contains(contentType, "xml") ||
		strings.Contains(contentType, "yaml") {
		return true
	}
	switch strings.ToLower(ext) {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".css", ".scss", ".html", ".htm", ".json", ".md", ".markdown", ".txt", ".csv", ".xml", ".yaml", ".yml", ".toml", ".ini", ".log", ".sql", ".sh", ".zsh", ".bash":
		return true
	default:
		return false
	}
}

func normalizeOSSEndpoint(endpoint string, useSSL bool) string {
	value := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	if useSSL {
		return "https://" + value
	}
	return "http://" + value
}

func assetRef(storageID string, key string) string {
	id := sanitizeStorageID(storageID)
	if id == "" {
		id = "default"
	}
	return "@assets/" + id + "/" + strings.TrimLeft(key, "/")
}

func objectKey(prefix string, name string) string {
	cleanPrefix := strings.Trim(pathpkg.Clean("/"+strings.TrimSpace(prefix)), "/")
	cleanName := sanitizeObjectName(name)
	if cleanName == "" {
		cleanName = "upload.bin"
	}
	fileName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), cleanName)
	if cleanPrefix == "" || cleanPrefix == "." {
		return fileName
	}
	return pathpkg.Join(cleanPrefix, fileName)
}

func sanitizeObjectName(name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	base = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`/\:?*<>|"`, r) {
			return '-'
		}
		return r
	}, base)
	return strings.Trim(base, ". ")
}

func publicOSSObjectURL(cfg OSSConfig, endpoint string, key string) string {
	escapedKey := escapedObjectKey(key)
	if isLocalOSSConfig(cfg) {
		return localOSSAssetURL(cfg.ID, key)
	}
	if base := strings.TrimRight(strings.TrimSpace(cfg.PublicBaseURL), "/"); base != "" {
		return base + "/" + escapedKey
	}
	if cfg.ForcePathStyle {
		return strings.TrimRight(endpoint, "/") + "/" + url.PathEscape(strings.Trim(cfg.Bucket, "/")) + "/" + escapedKey
	}
	parsed, err := url.Parse(endpoint)
	if err == nil && parsed.Host != "" {
		parsed.Host = strings.Trim(cfg.Bucket, ".") + "." + parsed.Host
		parsed.Path = "/" + escapedKey
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.String()
	}
	return strings.TrimRight(endpoint, "/") + "/" + escapedKey
}

func localOSSAssetURL(storageID string, key string) string {
	id := sanitizeStorageID(storageID)
	if id == "" {
		id = "default"
	}
	cleanKey := cleanOSSObjectPath(key)
	if cleanKey == "" {
		return ""
	}
	return "/api/oss/assets?storageId=" + url.QueryEscape(id) + "&path=" + url.QueryEscape(cleanKey)
}

func escapedObjectKey(key string) string {
	parts := strings.Split(strings.TrimLeft(key, "/"), "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
