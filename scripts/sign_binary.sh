#!/bin/bash

# 配置变量
APP_NAME="WXChannelsDownload"
VERSION="0.1.1"
BUNDLE_ID="com.funzm.box"
# APPLE_ID="litaowork@aliyun.com"
TEAM_ID="YG658VL4QS"  # 从开发者账户获取
APPLE_ID="tao li"
IDENTITY="Developer ID Application: $APPLE_ID ($TEAM_ID)"
P12_FILE="certs/YG658VL4QS_Mayfair.p12"
P12_PASSWORD="li1218040201."
API_KEY_ID="8XLC66Q3UY"  # 例如：ABC123DEF456
API_KEY_ISSUER_ID="fb055ea6-b243-4491-a498-d31164ec0ce1"  # 例如：a1b2c3d4-e5f6-7890-1234-567890abcdef
P8_FILE="certs/AuthKey_8XLC66Q3UY.p8"
INFO_FILE=".build/Info.plist"
APP_ICON_FILE=".build/icons/AppIcon.icns"
ENTITLEMENT_FILE="build/entitlements.plist"

APP_PASSWORD="@keychain:AC_PASSWORD"  # 或使用 app-specific 密码
NOTARIZE_PROFILE="My Notarization Profile"  # 钥匙串中的描述文件名称

# 清理
rm -rf dist
rm -rf "$APP_NAME.app"
rm -f "$APP_NAME.dmg"

# 创建目录结构
mkdir -p "dist/$APP_NAME.app/Contents/MacOS"
mkdir -p "dist/$APP_NAME.app/Contents/Resources"

# 编译 Go 程序
echo "编译 Go 程序..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o "dist/$APP_NAME.app/Contents/MacOS/$APP_NAME" -ldflags="-s -w" main.go
# CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o "dist/WXChannelsDownload" -ldflags="-s -w" main.go

# 复制资源文件
cp $INFO_FILE "dist/$APP_NAME.app/Contents/"
cp $APP_ICON_FILE "dist/$APP_NAME.app/Contents/Resources/"
# cp -r other_resources/ "dist/$APP_NAME.app/Contents/Resources/" 2>/dev/null || true

# 签名应用程序
echo "签名应用程序..."


# 1. 首先签名所有框架和库
find "dist/$APP_NAME.app" \
    -type f \
    \( -name "*.so" -o -name "*.dylib" -o -name "*.framework" \) \
    | while read file; do
        echo "签名: $file"
        codesign --force --sign "$IDENTITY" --timestamp "$file"
    done

# 2. 签名Plugins（如果有）
if [ -d "dist/$APP_NAME.app/Contents/PlugIns" ]; then
    find "dist/$APP_NAME.app/Contents/PlugIns" -name "*.appex" | while read plugin; do
        codesign --force --sign "$IDENTITY" --timestamp "$plugin"
    done
fi

# 3. 签名所有Mach-O二进制文件（除了主程序）
find "dist/$APP_NAME.app/Contents" \
    -type f \
    -perm +111 \
    ! -path "*/MacOS/$APP_NAME" \
    -exec file {} \; \
    | grep "Mach-O" | cut -d: -f1 | while read binary; do
        echo "签名二进制: $binary"
        codesign --force --sign "$IDENTITY" --timestamp "$binary"
    done

echo "签名主程序..."
# 签名主程序（使用修复后的 entitlements）
codesign --force \
    --sign "$IDENTITY" \
    --options runtime \
    --timestamp \
    --entitlements "$ENTITLEMENT_FILE" \
    "dist/$APP_NAME.app/Contents/MacOS/$APP_NAME"

# 4. 最后签名主应用（包含entitlements）
echo "签名主应用..."
codesign --force \
    --sign "$IDENTITY" \
    --options runtime \
    --timestamp \
    --entitlements "$ENTITLEMENT_FILE" \
    "dist/$APP_NAME.app"

# 5. 验证签名
echo "验证签名..."
codesign -vvv --deep --strict "dist/$APP_NAME.app"

# 6. 验证entitlements
echo "查看entitlements..."
codesign -d --entitlements :- "dist/$APP_NAME.app"
