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

## Building the velo CLI

The `velo` CLI tool handles building, packaging, and signing your application.

```bash
# Install the velo CLI
go install github.com/ltaoo/velo/cmd/velo@latest

# Or build from source
go build -ldflags "-X main.version=1.0.0" ./cmd/velo
```

### Usage

```bash
# Build for current platform (in your project directory)
velo build

# Build a specific project
velo build /path/to/project

# Build for a specific platform
velo build -platform darwin
velo build -platform windows
velo build -platform linux
velo build -platform all

# Custom output directory
velo build -out build

# Override version
velo build -version 1.2.3
```

The `velo build` command reads `app-config.json` from the project directory, generates icons, platform configs, and compiles binaries for the target platform(s).

## Building the Example Project

```bash
cd _example/simple

# Run in development mode
go run .

# Build with the velo CLI (from the example directory)
velo build

# Or build with go directly
go build -o myapp .
```

The example project uses `replace` directive in `go.mod` to reference the local velo module, so no extra setup is needed.

### Project Structure

```
_example/simple/
├── app-config.json    # Application configuration (name, platforms, update, etc.)
├── main.go            # Application entry point
├── frontend/          # Web frontend assets
└── go.mod
```

### app-config.json

The `app-config.json` file configures your application. Key sections:

- `app` — Name, version, description, icon
- `binary` — Output binary name
- `platforms` — Platform-specific settings (macOS, Windows, Linux)
- `build` — Build options (config files, excludes)
- `release` — Release metadata
- `update` — Auto-update configuration

Example update configuration:

```json
{
  "update": {
    "enabled": true,
    "check_frequency": "startup",
    "channel": "stable",
    "auto_download": false,
    "timeout": 300,
    "sources": [
      {
        "type": "github",
        "priority": 1,
        "enabled": true,
        "need_check_checksum": true,
        "github_repo": "owner/repo"
      },
      {
        "type": "http",
        "priority": 2,
        "enabled": true,
        "manifest_url": "https://example.com/manifest.json"
      }
    ]
  }
}
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
| `updater` | Auto-update system |
| `buildcfg` | Build configuration and code generation |

## Supported Platforms

- macOS
- Windows
- Linux

## License

[MIT](LICENSE)
