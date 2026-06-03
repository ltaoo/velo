package main

import (
	"embed"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ltaoo/velo"
	"github.com/ltaoo/velo/notification"
	"github.com/ltaoo/velo/store"
)

//go:embed frontend
var frontendFS embed.FS

//go:embed assets/appicon.png
var appIcon []byte

type sendNotificationRequest struct {
	Type    string `json:"type"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	AppName string `json:"app_name"`
	Icon    string `json:"icon"`
	Sound   bool   `json:"sound"`
}

type remotePushState struct {
	RegisteredAt string `json:"registered_at,omitempty"`
	Token        string `json:"token,omitempty"`
	Error        string `json:"error,omitempty"`
	Payload      string `json:"payload,omitempty"`
}

var (
	pushStateMu sync.RWMutex
	pushState   remotePushState
)

func main() {
	opt := velo.VeloAppOpt{Mode: velo.ModeBridge, IconData: appIcon}
	app := velo.NewApp(&opt)
	app.Store = store.NewWithDir(filepath.Join(os.TempDir(), "velo-notification-demo"))

	app.Get("/api/app", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{
			"name":    "Velo Notification Demo",
			"version": "1.0.0",
			"os":      runtime.GOOS,
		})
	})

	app.Get("/api/notification/status", func(c *velo.BoxContext) interface{} {
		return c.Ok(notification.PermissionStatus())
	})

	app.Post("/api/notification/cleanup", func(c *velo.BoxContext) interface{} {
		appName := c.Query("app_name")
		if appName == "" {
			appName = "Velo Notification Demo"
		}
		if err := notification.Cleanup(notification.CleanupOptions{AppName: appName}); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{
			"cleaned": true,
			"note":    "Delivered and pending notifications were cleared. System permission entries are managed by the OS.",
		})
	})

	app.Get("/api/push/state", func(c *velo.BoxContext) interface{} {
		pushStateMu.RLock()
		state := pushState
		pushStateMu.RUnlock()
		return c.Ok(state)
	})

	app.Post("/api/push/register", func(c *velo.BoxContext) interface{} {
		pushStateMu.Lock()
		pushState = remotePushState{RegisteredAt: time.Now().Format(time.RFC3339)}
		pushStateMu.Unlock()

		err := notification.RegisterRemotePush(notification.RemotePushCallbacks{
			OnToken: func(token string) {
				pushStateMu.Lock()
				pushState.Token = token
				pushState.Error = ""
				state := pushState
				pushStateMu.Unlock()
				app.SendMessage(velo.H{"type": "remote_push_token", "data": state})
			},
			OnError: func(err error) {
				pushStateMu.Lock()
				pushState.Error = err.Error()
				state := pushState
				pushStateMu.Unlock()
				app.SendMessage(velo.H{"type": "remote_push_error", "data": state})
			},
			OnNotification: func(payload string) {
				pushStateMu.Lock()
				pushState.Payload = payload
				state := pushState
				pushStateMu.Unlock()
				app.SendMessage(velo.H{"type": "remote_push_payload", "data": state})
			},
		})
		if err != nil {
			return c.Error(err.Error())
		}

		pushStateMu.RLock()
		state := pushState
		pushStateMu.RUnlock()
		return c.Ok(state)
	})

	app.Post("/api/notification/send", func(c *velo.BoxContext) interface{} {
		var req sendNotificationRequest
		if err := c.BindJSON(&req); err != nil {
			return c.Error(err.Error())
		}

		opts := notification.Options{
			Type:    normalizeType(req.Type),
			Title:   strings.TrimSpace(req.Title),
			Body:    strings.TrimSpace(req.Body),
			AppName: strings.TrimSpace(req.AppName),
			Icon:    strings.TrimSpace(req.Icon),
			Sound:   req.Sound,
		}
		if opts.Title == "" {
			opts.Title = defaultTitle(opts.Type)
		}
		if opts.Body == "" {
			opts.Body = "This notification was sent by the Velo notification demo."
		}

		if err := notification.Push(opts); err != nil {
			return c.Error(err.Error())
		}
		return c.Ok(velo.H{
			"sent":     true,
			"type":     opts.Type,
			"title":    opts.Title,
			"body":     opts.Body,
			"app_name": opts.AppName,
			"sound":    opts.Sound,
		})
	})

	app.NewWebview(&velo.VeloWebviewOpt{
		Title:      "Notification Demo",
		FrontendFS: frontendFS,
		Pathname:   "/",
		Width:      760,
		Height:     680,
	})
	app.Run()
}

func normalizeType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case notification.TypeSuccess:
		return notification.TypeSuccess
	case notification.TypeWarning:
		return notification.TypeWarning
	case notification.TypeError:
		return notification.TypeError
	default:
		return notification.TypeInfo
	}
}

func defaultTitle(notificationType string) string {
	switch notificationType {
	case notification.TypeSuccess:
		return "Success notification"
	case notification.TypeWarning:
		return "Warning notification"
	case notification.TypeError:
		return "Error notification"
	default:
		return "Info notification"
	}
}
