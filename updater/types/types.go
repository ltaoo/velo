package types

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
)

// UpdaterOptions contains options for creating an updater instance
type UpdaterOptions struct {
	// Config is the update configuration
	Config *UpdateConfig

	// CurrentVersion is the current application version
	CurrentVersion string

	// Logger is the zerolog logger instance (optional, will create default if nil)
	Logger *zerolog.Logger

	// StatePath is the path to the state file (optional, will use default if empty)
	StatePath string
}

// UpdateConfig defines the configuration for the update system
type UpdateConfig struct {
	// Enabled controls whether auto-update is enabled
	Enabled bool `yaml:"enabled"`

	// CheckFrequency defines when to check for updates: "startup", "daily", "weekly", "manual"
	CheckFrequency string `yaml:"check_frequency"`

	// Channel defines the update channel: "stable", "beta"
	Channel string `yaml:"channel"`

	// AutoDownload controls whether to automatically download updates (false = notify only)
	AutoDownload bool `yaml:"auto_download"`

	// Sources is the list of update sources (ordered by priority)
	Sources []UpdateSource `yaml:"sources"`

	// Timeout is the timeout in seconds for update operations
	Timeout int `yaml:"timeout"`

	// DevModeEnabled enables update checks in development mode
	DevModeEnabled bool `yaml:"dev_mode_enabled"`

	// DevVersion is the fake version to use in development mode
	DevVersion string `yaml:"dev_version"`

	// DevUpdateSource is the update source to use in development mode
	DevUpdateSource *UpdateSource `yaml:"dev_update_source"`
}

// UpdateSource represents a single update source
type UpdateSource struct {
	// Type is the source type: "github", "http"
	Type string `yaml:"type"`

	// Priority determines the order (lower number = higher priority)
	Priority int `yaml:"priority"`

	// GitHubRepo is the GitHub repository in "owner/repo" format
	GitHubRepo string `yaml:"github_repo,omitempty"`

	// GitHubToken is the optional GitHub API token
	GitHubToken string `yaml:"github_token,omitempty"`

	// ManifestURL is the URL for custom HTTP manifest
	ManifestURL string `yaml:"manifest_url,omitempty"`

	// SelfURL is the URL for self-hosted update source
	SelfURL string `yaml:"self_url,omitempty"`

	// Enabled controls whether this source is active
	Enabled bool `yaml:"enabled"`

	// NeedCheckChecksum controls whether to verify checksum after download
	NeedCheckChecksum bool `yaml:"need_check_checksum"`
}

// ReleaseManifest represents the standardized release manifest format
type ReleaseManifest struct {
	// Version is the semantic version number
	Version string `json:"version"`

	// PublishedAt is the release timestamp in RFC3339 format
	PublishedAt string `json:"published_at"`

	// ReleaseNotes contains the release notes (supports Markdown)
	ReleaseNotes string `json:"release_notes"`

	// Assets contains platform-specific download information
	Assets map[string]AssetInfo `json:"assets"`
}

// AssetInfo contains information about a downloadable asset
type AssetInfo struct {
	// URL is the download URL
	URL string `json:"url"`

	// Size is the file size in bytes
	Size int64 `json:"size"`

	// Checksum is the SHA256 checksum
	Checksum string `json:"checksum"`

	// Name is the filename
	Name string `json:"name"`
}

// ReleaseInfo contains information about a release
type ReleaseInfo struct {
	Version           string
	PublishedAt       time.Time
	ReleaseNotes      string
	AssetURL          string
	Headers           map[string]string
	AssetSize         int64
	Checksum          string
	AssetName         string
	IsNewer           bool
	NeedCheckChecksum bool
}

// UpdateState represents the persisted update state
type UpdateState struct {
	Filepath        string
	LastCheckTime   time.Time `json:"last_check_time"`
	LastUpdateTime  time.Time `json:"last_update_time"`
	SkippedVersions []string  `json:"skipped_versions"`
	CurrentVersion  string    `json:"current_version"`
}

// Save persists the UpdateState to a JSON file at the specified path.
// It creates the directory if it doesn't exist and writes the state atomically.
func (us *UpdateState) Save() error {
	path := us.Filepath
	if us == nil {
		return fmt.Errorf("cannot save nil UpdateState")
	}

	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Marshal the state to JSON with indentation for readability
	data, err := json.MarshalIndent(us, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal UpdateState: %w", err)
	}

	// Write to a temporary file first for atomic operation
	tempPath := path + ".tmp"
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Atomically rename the temporary file to the target path
	if err := os.Rename(tempPath, path); err != nil {
		// Clean up temporary file on failure
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temporary file: %w", err)
	}

	return nil
}

// LoadUpdateState loads the UpdateState from a JSON file at the specified path.
// If the file doesn't exist, it returns a new empty UpdateState.
// If the file exists but is invalid, it returns an error.
func LoadUpdateState(path string) (*UpdateState, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return a new empty state if file doesn't exist
		return &UpdateState{
			SkippedVersions: []string{},
		}, nil
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	// Unmarshal the JSON
	var state UpdateState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal UpdateState: %w", err)
	}
	state.Filepath = path
	// Ensure SkippedVersions is not nil
	if state.SkippedVersions == nil {
		state.SkippedVersions = []string{}
	}

	return &state, nil
}

// DownloadProgress represents download progress information
type DownloadProgress struct {
	BytesDownloaded int64
	TotalBytes      int64
	Percentage      float64
	Speed           int64 // bytes per second
}

// DownloadCallback is called to report download progress
type DownloadCallback func(progress DownloadProgress)

// ErrorCategory 定义错误类别
type ErrorCategory int

const (
	// ErrCategoryNetwork 网络错误
	ErrCategoryNetwork ErrorCategory = iota
	// ErrCategoryValidation 验证错误（校验和、格式等）
	ErrCategoryValidation
	// ErrCategoryFileSystem 文件系统错误
	ErrCategoryFileSystem
	// ErrCategoryPermission 权限错误
	ErrCategoryPermission
	// ErrCategorySecurity 安全错误
	ErrCategorySecurity
	// ErrCategoryConfiguration 配置错误
	ErrCategoryConfiguration
)

// String 返回错误类别的字符串表示
func (ec ErrorCategory) String() string {
	switch ec {
	case ErrCategoryNetwork:
		return "network"
	case ErrCategoryValidation:
		return "validation"
	case ErrCategoryFileSystem:
		return "filesystem"
	case ErrCategoryPermission:
		return "permission"
	case ErrCategorySecurity:
		return "security"
	case ErrCategoryConfiguration:
		return "configuration"
	default:
		return "unknown"
	}
}

// UpdateError 结构化的更新错误类型
type UpdateError struct {
	// Category 错误类别
	Category ErrorCategory
	// Message 错误消息
	Message string
	// Cause 原始错误
	Cause error
	// Context 错误上下文信息
	Context map[string]interface{}
}

// Error 实现 error 接口
func (e *UpdateError) Error() string {
	msg := fmt.Sprintf("[%s] %s", e.Category, e.Message)
	if e.Cause != nil {
		msg += fmt.Sprintf(": %v", e.Cause)
	}
	for k, v := range e.Context {
		msg += fmt.Sprintf(" [%s=%v]", k, v)
	}
	return msg
}

// Unwrap 实现 errors.Unwrap 接口
func (e *UpdateError) Unwrap() error {
	return e.Cause
}

// NewUpdateError 创建新的更新错误
func NewUpdateError(category ErrorCategory, message string, cause error) *UpdateError {
	return &UpdateError{
		Category: category,
		Message:  message,
		Cause:    cause,
		Context:  make(map[string]interface{}),
	}
}

// WithContext 添加上下文信息
func (e *UpdateError) WithContext(key string, value interface{}) *UpdateError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithContextMap 批量添加上下文信息
func (e *UpdateError) WithContextMap(context map[string]interface{}) *UpdateError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	for k, v := range context {
		e.Context[k] = v
	}
	return e
}

// IsCategory 检查错误是否属于指定类别
func IsCategory(err error, category ErrorCategory) bool {
	if updateErr, ok := err.(*UpdateError); ok {
		return updateErr.Category == category
	}
	return false
}

// GetCategory 获取错误类别
func GetCategory(err error) ErrorCategory {
	if updateErr, ok := err.(*UpdateError); ok {
		return updateErr.Category
	}
	return -1
}

// 预定义的错误构造函数

// NewNetworkError 创建网络错误
func NewNetworkError(message string, cause error) *UpdateError {
	return NewUpdateError(ErrCategoryNetwork, message, cause)
}

// NewValidationError 创建验证错误
func NewValidationError(message string, cause error) *UpdateError {
	return NewUpdateError(ErrCategoryValidation, message, cause)
}

// NewFileSystemError 创建文件系统错误
func NewFileSystemError(message string, cause error) *UpdateError {
	return NewUpdateError(ErrCategoryFileSystem, message, cause)
}

// NewPermissionError 创建权限错误
func NewPermissionError(message string, cause error) *UpdateError {
	return NewUpdateError(ErrCategoryPermission, message, cause)
}

// NewSecurityError 创建安全错误
func NewSecurityError(message string, cause error) *UpdateError {
	return NewUpdateError(ErrCategorySecurity, message, cause)
}

// NewConfigurationError 创建配置错误
func NewConfigurationError(message string, cause error) *UpdateError {
	return NewUpdateError(ErrCategoryConfiguration, message, cause)
}

// UpdateEvent represents an update event
type UpdateEvent struct {
	// Type is the event type
	Type UpdateEventType

	// Message is a human-readable message
	Message string

	// ReleaseInfo contains release information (for UpdateAvailable events)
	ReleaseInfo *ReleaseInfo

	// Progress contains download progress (for DownloadProgress events)
	Progress *DownloadProgress

	// Error contains error information (for Error events)
	Error error
}

// UpdateEventType represents the type of update event
type UpdateEventType string

const (
	// EventCheckStarted is emitted when update check starts
	EventCheckStarted UpdateEventType = "check_started"

	// EventCheckCompleted is emitted when update check completes
	EventCheckCompleted UpdateEventType = "check_completed"

	// EventUpdateAvailable is emitted when a new version is available
	EventUpdateAvailable UpdateEventType = "update_available"

	// EventNoUpdateAvailable is emitted when no update is available
	EventNoUpdateAvailable UpdateEventType = "no_update_available"

	// EventDownloadStarted is emitted when download starts
	EventDownloadStarted UpdateEventType = "download_started"

	// EventDownloadProgress is emitted during download
	EventDownloadProgress UpdateEventType = "download_progress"

	// EventDownloadCompleted is emitted when download completes
	EventDownloadCompleted UpdateEventType = "download_completed"

	// EventApplyStarted is emitted when update application starts
	EventApplyStarted UpdateEventType = "apply_started"

	// EventApplyCompleted is emitted when update application completes
	EventApplyCompleted UpdateEventType = "apply_completed"

	// EventError is emitted when an error occurs
	EventError UpdateEventType = "error"
)

type UpdateCallback func(event UpdateEvent)

type UpdateCheckResult struct {
	Version      string
	IsNewer      bool
	ReleaseNotes string
}
