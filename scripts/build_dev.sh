#!/bin/bash

# 配置变量
APP_NAME="WXChannelsDownload"
VERSION="1.0.0"
BUNDLE_ID="com.funzm.box"
TEAM_ID="YG658VL4QS"
APPLE_ID="tao li"
P12_FILE="certs/DeveloperCertificates.p12"
P12_PASSWORD="li1218040201."
IDENTITY="Developer ID Application: $APPLE_ID ($TEAM_ID)"
INFO_FILE=".build/Info.plist"
APP_ICON_FILE=".build/icons/AppIcon.icns"

# 临时钥匙串设置
KEYCHAIN_PATH="/tmp/build.keychain"
KEYCHAIN_PASSWORD="build_keychain_temp"

# 创建并配置临时钥匙串
echo "创建临时钥匙串..."
security create-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN_PATH" 2>/dev/null || true
security default-keychain -s "$KEYCHAIN_PATH"
security unlock-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN_PATH"

# 导入 p12 证书
echo "导入证书..."
security import "$P12_FILE" -P "$P12_PASSWORD" -k "$KEYCHAIN_PATH" -T /usr/bin/codesign 2>/dev/null || true
security set-key-partition-list -S apple-tool:,apple: -s -k "$KEYCHAIN_PASSWORD" "$KEYCHAIN_PATH" 2>/dev/null || true
security list-keychains -s "$KEYCHAIN_PATH" "/Library/Keychains/System.keychain"

# 清理
rm -rf dist
rm -rf "$APP_NAME.app"
rm -f "$APP_NAME.dmg"

# 创建目录结构
mkdir -p "dist/$APP_NAME.app/Contents/MacOS"
mkdir -p "dist/$APP_NAME.app/Contents/Resources"

# 编译
echo "编译..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o "dist/$APP_NAME.app/Contents/MacOS/$APP_NAME" -ldflags="-s -w -X velo/pkg/version.Version=$VERSION" main.go

# 复制资源
cp "$INFO_FILE" "dist/$APP_NAME.app/Contents/"
cp "$APP_ICON_FILE" "dist/$APP_NAME.app/Contents/Resources/"

# 签名（不使用沙盒）
echo "签名..."

# 签名主程序（无沙盒）
codesign --force --sign "$IDENTITY" --timestamp "dist/$APP_NAME.app/Contents/MacOS/$APP_NAME"

# 签名主应用（无沙盒）
codesign --force --sign "$IDENTITY" --timestamp "dist/$APP_NAME.app"

# 验证
echo "验证签名..."
codesign -dv --verbose=4 "dist/$APP_NAME.app"

# 创建 DMG
echo "创建 DMG..."
mkdir -p dmg_root
cp -r "dist/$APP_NAME.app" dmg_root/

create-dmg \
    --volname "$APP_NAME" \
    --window-pos 100 100 \
    --window-size 660 400 \
    --icon-size 128 \
    --icon "$APP_NAME.app" 230 136 \
    --app-drop-link 430 136 \
    --background "build/dmg-background.png" \
    --no-internet-enable \
    "$APP_NAME.dmg" \
    dmg_root/ 2>/dev/null

# 签名 DMG
codesign --sign "$IDENTITY" --timestamp "$APP_NAME.dmg"

# 清理
rm -rf dmg_root
security list-keychains -s "/Library/Keychains/System.keychain"
security delete-keychain "$KEYCHAIN_PATH" 2>/dev/null || true

echo "构建完成: $APP_NAME.dmg"
echo ""
echo "提示: 此版本未使用沙盒，可直接运行用于测试"
echo "正式分发请使用 build.sh（包含沙盒和公证）"
