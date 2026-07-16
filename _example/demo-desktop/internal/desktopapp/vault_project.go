package desktopapp

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

const globalVeloDirName = ".velo"
const globalVaultDataFileName = "data.json"
const vaultConfigDirName = ".velo"
const vaultMemoDirName = "memo"
const vaultMemoCommentDirName = "memo-comments"
const vaultProjectsFileName = "projects.json"
const vaultSchemaVersion = 1

var vaultRuntime = struct {
	sync.RWMutex
	active *VaultContext
}{
	active: nil,
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
	Entry           VaultEntry `json:"entry"`
	RootDir         string     `json:"rootDir"`
	VeloDir         string     `json:"veloDir"`
	MemoDir         string     `json:"memoDir"`
	MemoCommentDir  string     `json:"memoCommentDir"`
	PrivateUnlocked bool       `json:"-"`
}

type VaultOpenRequest struct {
	Path string `json:"path"`
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
	memoCommentDir := filepath.Join(rootDir, vaultMemoCommentDirName)
	if err := os.MkdirAll(memoCommentDir, 0755); err != nil {
		return nil, false, fmt.Errorf("create memo comment directory: %w", err)
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
		Entry:          entry,
		RootDir:        rootDir,
		VeloDir:        veloDir,
		MemoDir:        memoDir,
		MemoCommentDir: memoCommentDir,
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
