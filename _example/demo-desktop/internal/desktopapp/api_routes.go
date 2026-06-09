package desktopapp

import (
	"github.com/ltaoo/velo"
	updater "github.com/ltaoo/velo/updater/api"
	"github.com/rs/zerolog"
)

func registerRoutes(b *velo.Box, logger *zerolog.Logger, appUpdater *updater.AppUpdater) {
	registerVaultProjectMemoRoutes(b)
	registerTaskRoutes(b)
	registerGTDRoutes(b)
	registerDesktopRoutes(b, logger)
	registerStorageRoutes(b)
	registerUpdateAndWindowRoutes(b, appUpdater)
}
