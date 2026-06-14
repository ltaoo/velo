package desktopapp

import (
	"runtime"
	"strings"

	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/autostart"
)

const defaultAutoStartAppName = "DemoDesktop"

type autoStartSaveRequest struct {
	Enabled bool `json:"enabled"`
}

var newAutoStart = autostart.New

func registerAutoStartRoutes(b *velo.Box) {
	b.Get("/api/settings/autostart", func(c *velo.BoxContext) interface{} {
		return c.Ok(autoStartConfig())
	})

	b.Post("/api/settings/autostart/save", func(c *velo.BoxContext) interface{} {
		var req autoStartSaveRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}
		starter := newAutoStart(autoStartAppName())
		if req.Enabled {
			if !autoStartSupported() {
				return c.Error("autostart is not supported on this platform")
			}
			if err := starter.Enable(); err != nil {
				return c.Error(err.Error())
			}
		} else if autoStartSupported() {
			if err := starter.Disable(); err != nil {
				return c.Error(err.Error())
			}
		}
		return c.Ok(autoStartConfig())
	})
}

func autoStartConfig() velo.H {
	appName := autoStartAppName()
	starter := newAutoStart(appName)
	return velo.H{
		"config": velo.H{
			"appName":   appName,
			"enabled":   starter.IsEnabled(),
			"supported": autoStartSupported(),
		},
	}
}

func autoStartAppName() string {
	cfg := velo.LoadAppConfig(appAssets.AppConfigData)
	if name := strings.TrimSpace(cfg.App.Name); name != "" {
		return name
	}
	if name := strings.TrimSpace(cfg.App.DisplayName); name != "" {
		return name
	}
	return defaultAutoStartAppName
}

func autoStartSupported() bool {
	return runtime.GOOS == "darwin" || runtime.GOOS == "windows" || runtime.GOOS == "linux"
}
