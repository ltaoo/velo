package desktopapp

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ltaoo/velo"
)

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

func marshalCloudStorageSettingsForStore(settings CloudStorageSettings) ([]byte, error) {
	return json.Marshal(cloudStorageSettingsForStore(settings))
}

func cloudStorageSettingsForStore(settings CloudStorageSettings) CloudStorageSettings {
	settings = normalizeCloudStorageSettings(settings)
	storages := make([]OSSConfig, len(settings.Storages))
	for i, cfg := range settings.Storages {
		if cfg.Local != nil {
			local := *cfg.Local
			cfg.Local = &local
		}
		if isLocalOSSConfig(cfg) {
			cfg.Local = normalizeLocalOSSSettings(cfg.Local)
			if cfg.Local != nil {
				if cfg.Local.RootMode == localStorageRootModeVault {
					cfg.Endpoint = ""
				} else if cfg.Local.RootMode == localStorageRootModeAbsolute {
					cfg.Endpoint = cfg.Local.Root
				}
			}
		} else {
			cfg.Local = nil
		}
		storages[i] = cfg
	}
	settings.Storages = storages
	return settings
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
		if isLocalOSSConfig(cfg) {
			cfg.Local = normalizeLocalOSSSettings(cfg.Local)
		} else {
			cfg.Local = nil
		}
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
		cfg, localChanged := prepareLocalOSSConfig(settings.Storages[i], storePath)
		if localChanged {
			settings.Storages[i] = cfg
			changed = true
		}
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
		Local: &LocalOSSSettings{
			Root:     defaultLocalStorageRelativeRoot,
			RootMode: localStorageRootModeVault,
		},
		Name:     "本地 Memo 存储",
		Provider: "local",
		UseSSL:   false,
	}
}

func prepareLocalOSSConfig(cfg OSSConfig, storePath string) (OSSConfig, bool) {
	if !isLocalOSSConfig(cfg) {
		if cfg.Local != nil {
			cfg.Local = nil
			return cfg, true
		}
		return cfg, false
	}

	changed := false
	local := normalizeLocalOSSSettings(cfg.Local)
	if !sameLocalOSSSettings(cfg.Local, local) {
		changed = true
	}
	cfg.Local = local
	if cfg.Local == nil {
		cfg.Local = inferLocalOSSSettings(cfg, storePath)
		if cfg.Local != nil {
			changed = true
		}
	}

	if cfg.Local != nil {
		root := resolveLocalOSSRoot(*cfg.Local, storePath)
		if root != "" && !samePath(cfg.Endpoint, root) {
			cfg.Endpoint = root
			changed = true
		}
	}
	return cfg, changed
}

func inferLocalOSSSettings(cfg OSSConfig, storePath string) *LocalOSSSettings {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" ||
		samePath(endpoint, defaultLocalStorageRoot(storePath)) ||
		samePath(endpoint, legacyDefaultLocalStorageRoot(storePath)) ||
		looksLikeVaultLocalStorageRoot(endpoint) {
		return &LocalOSSSettings{
			Root:     defaultLocalStorageRelativeRoot,
			RootMode: localStorageRootModeVault,
		}
	}
	return &LocalOSSSettings{
		Root:     endpoint,
		RootMode: localStorageRootModeAbsolute,
	}
}

func normalizeLocalOSSSettings(local *LocalOSSSettings) *LocalOSSSettings {
	if local == nil {
		return nil
	}
	root := strings.TrimSpace(local.Root)
	rootMode := strings.ToLower(strings.TrimSpace(local.RootMode))
	if rootMode == "" {
		if root == "" {
			return nil
		}
		if filepath.IsAbs(expandLocalPath(root)) || strings.HasPrefix(root, "~") {
			rootMode = localStorageRootModeAbsolute
		} else {
			rootMode = localStorageRootModeVault
		}
	}

	switch rootMode {
	case localStorageRootModeVault:
		root = cleanLocalRelativeRoot(root)
		if root == "" {
			root = defaultLocalStorageRelativeRoot
		}
	case localStorageRootModeAbsolute:
		root = strings.TrimSpace(root)
		if root == "" {
			return nil
		}
	default:
		return nil
	}

	return &LocalOSSSettings{
		Root:     root,
		RootMode: rootMode,
	}
}

func sameLocalOSSSettings(a *LocalOSSSettings, b *LocalOSSSettings) bool {
	a = normalizeLocalOSSSettings(a)
	b = normalizeLocalOSSSettings(b)
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Root == b.Root && a.RootMode == b.RootMode
}

func resolveLocalOSSRoot(local LocalOSSSettings, storePath string) string {
	normalized := normalizeLocalOSSSettings(&local)
	if normalized == nil {
		return ""
	}
	switch normalized.RootMode {
	case localStorageRootModeVault:
		base := vaultRootForLocalStorage(storePath)
		if base == "" {
			return ""
		}
		return filepath.Join(base, filepath.FromSlash(normalized.Root))
	case localStorageRootModeAbsolute:
		return expandLocalPath(normalized.Root)
	default:
		return ""
	}
}

func vaultRootForLocalStorage(storePath string) string {
	if vault := activeVaultSnapshot(); vault != nil && strings.TrimSpace(vault.RootDir) != "" {
		return vault.RootDir
	}
	if root := vaultRootFromStorePath(storePath); root != "" {
		return root
	}
	base := filepath.Dir(strings.TrimSpace(storePath))
	if base == "" || base == "." {
		base = projectDir()
	}
	return base
}

func cleanLocalRelativeRoot(value string) string {
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

func looksLikeVaultLocalStorageRoot(endpoint string) bool {
	root := expandLocalPath(strings.TrimSpace(endpoint))
	if root == "" {
		return false
	}
	clean := filepath.Clean(root)
	return filepath.Base(clean) == defaultLocalStorageRelativeRoot
}

func defaultLocalStorageRoot(storePath string) string {
	return filepath.Join(vaultRootForLocalStorage(storePath), defaultLocalStorageRelativeRoot)
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
		cfg.Local != nil ||
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
			if localOSSAbsoluteRoot(cfg) == "" {
				missing = append(missing, "local root")
			}
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
