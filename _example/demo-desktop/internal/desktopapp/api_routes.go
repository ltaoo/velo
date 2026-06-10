package desktopapp

import (
	"github.com/ltaoo/velo"
	updater "github.com/ltaoo/velo/updater/api"
	"github.com/rs/zerolog"
)

func registerRoutes(b *velo.Box, logger *zerolog.Logger, appUpdater *updater.AppUpdater, inputSourceLock *InputSourceLockService) {
	registerVaultProjectMemoRoutes(b)
	registerTaskRoutes(b)
	registerGTDRoutes(b)
	registerSnippetRoutes(b)
	registerDesktopRoutes(b, logger)
	registerStorageRoutes(b)
	registerInputSourceLockRoutes(b, inputSourceLock)
	registerClipboardRoutes(b, logger)
	registerUpdateAndWindowRoutes(b, appUpdater)
}
