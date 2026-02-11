# velo

[![Go Reference](https://pkg.go.dev/badge/github.com/ltaoo/velo.svg)](https://pkg.go.dev/github.com/ltaoo/velo)

A lightweight Go framework for building desktop applications with web frontends.

Velo provides native webview, system tray, file dialogs, and error dialogs across macOS, Windows, and Linux.

## Features

- **Webview** — Native webview window with JavaScript injection and message passing
- **System Tray** — System tray icon with menus, shortcuts, and click events
- **File Dialog** — Native file selection dialog
- **Error Dialog** — Native error dialog

## Installation

```bash
go get github.com/ltaoo/velo
```

## Quick Start

```go
package main

import "github.com/ltaoo/velo"

func main() {
	box := velo.NewBox()
	box.SetSize(1024, 640)

	box.Get("/api/hello", func(c *velo.BoxContext) interface{} {
		return c.Ok(velo.H{"message": "hello"})
	})

	box.Post("/api/echo", func(c *velo.BoxContext) interface{} {
		return c.Ok(c.Args())
	})

	velo.RunApp("./frontend/dist", box)
}
```

## Subpackages

| Package | Description |
|---------|-------------|
| `webview` | Native webview window management |
| `tray` | System tray icon and menu |
| `file` | Native file selection dialog |
| `error` | Native error dialog |
| `asset` | Embedded JS runtime assets |

## Supported Platforms

- macOS
- Windows
- Linux

## License

[MIT](LICENSE)
