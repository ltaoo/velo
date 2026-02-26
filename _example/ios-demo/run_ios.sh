#!/bin/bash
set -e
cd "$(dirname "$0")"

# Check if mise is available and use go@1.25.0 if suggested
if command -v mise &> /dev/null; then
    echo "Using mise to set Go version..."
    eval "$(mise activate bash)"
    mise use go@1.25.0 || echo "Failed to switch Go version with mise, continuing with current version"
fi

APP_NAME=$(jq -r '.app.name // "Flix"' "app-config.json")
BUNDLE_ID=$(jq -r '.platforms.ios.bundle_id // "com.funzm.flix"' "app-config.json")
ICON_FILE=$(jq -r '.platforms.ios.icon_file // ""' "app-config.json")
MIN_OS=$(jq -r '.platforms.ios.minimum_os_version // "13.0"' "app-config.json")

echo "Cleaning..."
rm -rf "$APP_NAME.app" main

echo "Building binary (GOOS=ios GOARCH=arm64 CGO_ENABLED=1 SDK=iphonesimulator)..."
# Use a separate cache to avoid permissions issues with gomobile or other tools
export GOCACHE=/tmp/go-cache-ios-manual
mkdir -p $GOCACHE

# Get the path to the iOS Simulator SDK
SDK_PATH=$(xcrun --sdk iphonesimulator --show-sdk-path)
CC=$(xcrun --sdk iphonesimulator --find clang)

# Build with CGO enabled, targeting the simulator SDK
# Note: We must specify CGO_CFLAGS and CGO_LDFLAGS to point to the simulator SDK
# and explicitly set the target to ios-simulator to avoid linker errors
CGO_ENABLED=1 GOOS=ios GOARCH=arm64 \
CC=$CC \
CGO_CFLAGS="-target arm64-apple-ios$MIN_OS-simulator -isysroot $SDK_PATH -arch arm64" \
CGO_LDFLAGS="-target arm64-apple-ios$MIN_OS-simulator -isysroot $SDK_PATH -arch arm64" \
go build -tags ios -ldflags "-s -w" -o main .

echo "Creating App Bundle..."
mkdir -p "$APP_NAME.app"
mv main "$APP_NAME.app/$APP_NAME"

# Copy Icon if available
if [ -n "$ICON_FILE" ] && [ -f "$ICON_FILE" ]; then
    cp "$ICON_FILE" "$APP_NAME.app/AppIcon.png"
    ICON_ENTRY="<key>CFBundleIconFiles</key><array><string>AppIcon.png</string></array>"
else
    ICON_ENTRY=""
fi

# Create Info.plist
cat > "$APP_NAME.app/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>$APP_NAME</string>
    <key>CFBundleIdentifier</key>
    <string>$BUNDLE_ID</string>
    <key>CFBundleName</key>
    <string>$APP_NAME</string>
    <key>CFBundleDisplayName</key>
    <string>$(jq -r '.app.display_name // "Flix"' "app-config.json")</string>
    <key>CFBundleVersion</key>
    <string>$(jq -r '.app.version // "1.0.0"' "app-config.json")</string>
    <key>CFBundleShortVersionString</key>
    <string>$(jq -r '.app.version // "1.0.0"' "app-config.json")</string>
    <key>LSRequiresIPhoneOS</key>
    <true/>
    <key>UIDeviceFamily</key>
    <array>
        <integer>1</integer>
        <integer>2</integer>
    </array>
    <key>UIRequiredDeviceCapabilities</key>
    <array>
        <string>arm64</string>
    </array>
    <key>UISupportedInterfaceOrientations</key>
    <array>
        <string>UIInterfaceOrientationPortrait</string>
        <string>UIInterfaceOrientationLandscapeLeft</string>
        <string>UIInterfaceOrientationLandscapeRight</string>
    </array>
    <key>MinimumOSVersion</key>
    <string>$MIN_OS</string>
    $ICON_ENTRY
</dict>
</plist>
EOF

echo "Signing..."
codesign -s - --deep --force "$APP_NAME.app"

echo "Installing..."
xcrun simctl install booted "$APP_NAME.app"

echo "Launching..."
# Start logging in background
echo "---------------------------------------------------------------"
echo "Tailing logs for $APP_NAME..."
echo "---------------------------------------------------------------"
# Write logs to ios_debug.log for easier debugging
xcrun simctl spawn booted log stream --level debug --predicate "process == \"$APP_NAME\"" --style compact | grep --line-buffered "DEBUG:" | tee ios_debug.log &
LOG_PID=$!
trap "kill $LOG_PID 2>/dev/null" EXIT

xcrun simctl launch booted "$BUNDLE_ID"

sleep 5
echo "Done. Check logs above for output."
