package applier

import (
	"github.com/rs/zerolog"

	"github.com/ltaoo/velo/updater/master"
)

// NewPlatformUpdater creates a platform-specific updater
func NewPlatformUpdater(logger *zerolog.Logger) master.UpdateApplier {
	// The actual implementation is selected at compile time based on build tags
	return newPlatformUpdaterImpl(logger)
}
