package master

import (
	"context"

	"github.com/ltaoo/velo/updater/types"
)

// VersionChecker defines the interface for checking version updates from various sources
type VersionChecker interface {
	// CheckLatest checks for the latest version available
	CheckLatest(ctx context.Context, currentVersion string) (*types.ReleaseInfo, error)

	// GetSourceName returns the name of this update source
	GetSourceName() string
}

// UpdateApplier defines the interface for platform-specific update operations
type UpdateApplier interface {
	// Backup creates a backup of the current executable
	Backup(execPath, backupPath string) error

	// Apply applies the update by extracting and replacing the executable
	Apply(updatePath, execPath string) error

	// Restore restores the executable from backup
	Restore(backupPath, execPath string) error

	// Cleanup removes backup and temporary files
	Cleanup(paths ...string) error

	// Restart is retained for source compatibility. Implementations must not
	// terminate the host application directly; callers should configure and use
	// updater/restart.Manager instead.
	Restart(exec_path string, arguments []string) error
}
