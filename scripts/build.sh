#!/bin/bash

# 配置变量
APP_NAME="WXChannelsDownload"
VERSION="1.1.0"
BUNDLE_ID="com.funzm.box"
TEAM_ID="YG658VL4QS"  # 从开发者账户获取
# APPLE_ID="litaowork@aliyun.com"
APPLE_ID="tao li"
P12_FILE="certs/DeveloperCertificates.p12"
P12_PASSWORD="li1218040201."
API_KEY_ID="8XLC66Q3UY"  # 例如：ABC123DEF456
API_KEY_ISSUER_ID="fb055ea6-b243-4491-a498-d31164ec0ce1"  # 例如：a1b2c3d4-e5f6-7890-1234-567890abcdef
IDENTITY="Developer ID Application: $APPLE_ID ($TEAM_ID)"
P8_FILE="certs/AuthKey_8XLC66Q3UY.p8"
INFO_FILE=".build/Info.plist"
APP_ICON_FILE=".build/icons/AppIcon.icns"
ENTITLEMENT_FILE="build/entitlements.plist"

APP_PASSWORD="@keychain:AC_PASSWORD"  # 或使用 app-specific 密码
NOTARIZE_PROFILE="My Notarization Profile"  # 钥匙串中的描述文件名称

# 临时钥匙串设置
KEYCHAIN_PATH="/tmp/build.keychain"
KEYCHAIN_PASSWORD="build_keychain_temp"

# 创建并配置临时钥匙串
echo "创建临时钥匙串..."
security create-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN_PATH" 2>/dev/null || true
security default-keychain -s "$KEYCHAIN_PATH"
security unlock-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN_PATH"

# 导入 p12 证书到临时钥匙串
echo "导入证书到临时钥匙串..."
security import "$P12_FILE" -P "$P12_PASSWORD" -k "$KEYCHAIN_PATH" -T /usr/bin/codesign 2>/dev/null || true

# 自动允许证书访问
security set-key-partition-list -S apple-tool:,apple: -s -k "$KEYCHAIN_PASSWORD" "$KEYCHAIN_PATH" 2>/dev/null || true

# 添加临时钥匙串到搜索列表
security list-keychains -s "$KEYCHAIN_PATH" "/Library/Keychains/System.keychain"

# 清理
rm -rf dist
rm -rf "$APP_NAME.app"

# 创建目录结构
mkdir -p "dist/$APP_NAME.app/Contents/MacOS"
mkdir -p "dist/$APP_NAME.app/Contents/Resources"

# 编译 Go 程序
echo "编译 Go 程序..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o "dist/$APP_NAME.app/Contents/MacOS/$APP_NAME" -ldflags="-s -w -X velo/pkg/version.Version=$VERSION" main.go
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
codesign -dv --verbose=4 "dist/$APP_NAME.app"
spctl -a -t exec -vv "dist/$APP_NAME.app"

# 创建 .dmg
echo "创建 DMG..."
mkdir -p dmg_root
cp -r "dist/$APP_NAME.app" dmg_root/

# 获取当前构建的平台和架构
BUILD_PLATFORM=$(go env GOOS)
BUILD_ARCH=$(go env GOARCH)
DMG_FILENAME="dist/${APP_NAME}_v${VERSION}_${BUILD_PLATFORM}_${BUILD_ARCH}.dmg"

# 使用 create-dmg 创建专业 DMG
create-dmg \
    --volname "$APP_NAME" \
    --window-pos 100 100 \
    --window-size 660 400 \
    --icon-size 128 \
    --icon "$APP_NAME.app" 230 136 \
    --app-drop-link 430 136 \
    --background "build/dmg-background.png" \
    --no-internet-enable \
    "$DMG_FILENAME" \
    dmg_root/ 2>/dev/null

# 签名 DMG
echo "签名 DMG..."
codesign --sign "Developer ID Application: $APPLE_ID ($TEAM_ID)" \
    --timestamp \
    "$DMG_FILENAME"

# 清理临时文件
rm -rf dmg_root

# 清理临时钥匙串（公证前清理，避免冲突）
echo "清理临时钥匙串..."
security list-keychains -s "/Library/Keychains/System.keychain"
security delete-keychain "$KEYCHAIN_PATH" 2>/dev/null || true

# 6. 提交公证
if [ -f "$P8_FILE" ]; then
    echo "提交公证..."
    NOTARIZE_RESULT=$(xcrun notarytool submit "dist/${APP_NAME}_v${VERSION}_${BUILD_PLATFORM}_${BUILD_ARCH}.dmg" \
        -k "$P8_FILE" \
        -d "$API_KEY_ID" \
        -i "$API_KEY_ISSUER_ID" \
        --wait 2>&1)

    echo "$NOTARIZE_RESULT"

    if echo "$NOTARIZE_RESULT" | grep -q "status: Accepted"; then
        echo "公证成功，添加凭证..."
        xcrun stapler apply "dist/${APP_NAME}_v${VERSION}_${BUILD_PLATFORM}_${BUILD_ARCH}.dmg"
        echo "公证完成"
    else
        echo "公证失败，请检查上面的输出"
    fi
else
    echo "警告: 未找到 p8 密钥文件 ($P8_FILE)，跳过公证"
    echo "请从 Apple Developer 门户下载 API Key 的私钥文件后重新公证"
fi

echo "构建完成: dist/${APP_NAME}_v${VERSION}_${BUILD_PLATFORM}_${BUILD_ARCH}.dmg"
