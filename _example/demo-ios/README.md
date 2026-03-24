# Velo iOS Demo

This is a demonstration of running Velo on iOS using `purego` bindings.

## Prerequisites

- macOS with Xcode installed
- **Go 1.20 or later**
- `gomobile` tool installed and initialized

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
```

## Running

### Using gomobile (Recommended)

To build an `.app` bundle for iOS Simulator (Apple Silicon):

```bash
gomobile build -target=ios/arm64 -bundleid=com.velo.iosdemo -o ios-demo.app .
```

To build for physical device (requires signing):

```bash
gomobile build -target=ios -bundleid=com.velo.iosdemo -o ios-demo.app .
```

Then install the `.app` to your simulator or device using Xcode or `xcrun simctl`.

### Using standard Go build (Advanced)

You can compile the binary, but you'll need to manually package it into an app bundle structure.

```bash
CGO_ENABLED=1 GOOS=ios GOARCH=arm64 SDK=iphoneos go build .
```

## Notes

- This demo uses `purego` to call UIKit frameworks directly.
- The `webview` implementation is in `../../webview/webview_ios.go`.
- Only single-window mode is supported on iOS.
- **Important**: The custom scheme `velo://` is currently not implemented on iOS. Please use `ModeBridgeHttp` (which uses a local HTTP server) or load external URLs.

## Troubleshooting

### Go Version Mismatch
If you see errors like `compile: version "go1.24.0" does not match go tool version "go1.20"`, it means your `go` command and the Go toolchain used by `gomobile` or your build environment are out of sync. Ensure you are using a consistent Go version (1.20+ recommended). You may need to update your Go installation or check your `PATH`.

### SDK not found
If you see `xcrun: error: SDK "iphoneos" cannot be located`, ensure Xcode is installed and the command line tools are selected:
```bash
sudo xcode-select -s /Applications/Xcode.app/Contents/Developer
```
