package main

import (
	"embed"
	"fmt"

	"github.com/ltaoo/velo"
)

//go:embed frontend
var frontend_folder embed.FS

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
		}
	}()
	fmt.Println("Starting iOS demo...")

	// Initialize Velo app options
	opt := velo.VeloAppOpt{
		Mode:    velo.ModeBridgeHttp, // Use HTTP bridge mode for iOS local file support
		AppName: "VeloiOSDemo",
		Title:   "Velo iOS Demo",
	}
	app := velo.NewApp(&opt)

	// Define a simple handler
	app.Get("/hello", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"message": "Hello from iOS!"})
	})

	app.Get("/open_window", func(c *velo.BoxContext) interface{} {
		target := c.Query("target")
		if target == "profile" {
			app.OpenWindow(&velo.VeloWebviewOpt{
				Title:      "Profile",
				Width:      390,
				Height:     844,
				EntryPage:  "profile.html",
				FrontendFS: frontend_folder,
			})
		}
		return c.Ok(velo.H{"success": true})
	})

	app.Get("/api/app", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{
			"version": "1.0.0",
			"name":    "VeloiOSDemo",
		})
	})

	// Register the main window configuration
	// The actual UI initialization happens in app.Run()
	app.NewWebview(&velo.VeloWebviewOpt{
		Title:      "Velo iOS Demo",
		Width:      390,
		Height:     844,
		EntryPage:  "index.html",
		FrontendFS: frontend_folder,
	})

	app.Run()
}
