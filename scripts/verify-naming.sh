#!/bin/bash

# 验证命名配置的脚本

set -e

echo "==================================="
echo "命名配置验证"
echo "==================================="
echo ""

# 检查 velo.json
if [ ! -f "velo.json" ]; then
    echo "❌ 找不到 velo.json"
    exit 1
fi

echo "📋 从 velo.json 读取配置..."
APP_NAME=$(jq -r '.app.name' velo.json)
BINARY_NAME=$(jq -r '.binary.name' velo.json)
PROJECT_NAME=$(jq -r '.binary.project_name' velo.json)

echo "  ✓ 项目名称 (project_name): ${PROJECT_NAME}"
echo "  ✓ 应用名称 (app.name): ${APP_NAME}"
echo "  ✓ 二进制名称 (binary.name): ${BINARY_NAME}"
echo ""

# 检查 GoReleaser 配置
echo "📋 检查 GoReleaser 配置..."
if [ ! -f "build/.goreleaser.yaml" ]; then
    echo "❌ 找不到 build/.goreleaser.yaml"
    echo "   请运行: ./scripts/generate-configs.sh"
    exit 1
fi

GORELEASER_PROJECT=$(grep "^project_name:" build/.goreleaser.yaml | awk '{print $2}')
GORELEASER_BINARY=$(grep "binary:" build/.goreleaser.yaml | head -1 | awk '{print $2}')

if [ "$GORELEASER_PROJECT" != "$PROJECT_NAME" ]; then
    echo "❌ GoReleaser project_name 不匹配"
    echo "   期望: ${PROJECT_NAME}"
    echo "   实际: ${GORELEASER_PROJECT}"
    exit 1
fi

if [ "$GORELEASER_BINARY" != "$BINARY_NAME" ]; then
    echo "❌ GoReleaser binary 不匹配"
    echo "   期望: ${BINARY_NAME}"
    echo "   实际: ${GORELEASER_BINARY}"
    exit 1
fi

echo "  ✓ project_name: ${GORELEASER_PROJECT}"
echo "  ✓ binary: ${GORELEASER_BINARY}"
echo ""

# 检查 Info.plist 模板
echo "📋 检查 Info.plist 模板..."
if [ ! -f ".build/Info.plist.template" ]; then
    echo "❌ 找不到 .build/Info.plist.template"
    echo "   请运行: ./scripts/generate-configs.sh"
    exit 1
fi

PLIST_EXECUTABLE=$(grep -A 1 "CFBundleExecutable" .build/Info.plist.template | tail -1 | sed 's/.*<string>\(.*\)<\/string>.*/\1/')
PLIST_NAME=$(grep -A 1 "CFBundleName" .build/Info.plist.template | tail -1 | sed 's/.*<string>\(.*\)<\/string>.*/\1/')

if [ "$PLIST_EXECUTABLE" != "$BINARY_NAME" ]; then
    echo "❌ Info.plist CFBundleExecutable 不匹配"
    echo "   期望: ${BINARY_NAME}"
    echo "   实际: ${PLIST_EXECUTABLE}"
    exit 1
fi

if [ "$PLIST_NAME" != "$APP_NAME" ]; then
    echo "❌ Info.plist CFBundleName 不匹配"
    echo "   期望: ${APP_NAME}"
    echo "   实际: ${PLIST_NAME}"
    exit 1
fi

echo "  ✓ CFBundleExecutable: ${PLIST_EXECUTABLE}"
echo "  ✓ CFBundleName: ${PLIST_NAME}"
echo ""

# 显示预期的文件名
echo "==================================="
echo "预期的文件命名"
echo "==================================="
echo ""

echo "📦 GoReleaser 归档文件:"
echo "  - ${PROJECT_NAME}_darwin_amd64.zip"
echo "  - ${PROJECT_NAME}_darwin_arm64.zip"
echo "  - ${PROJECT_NAME}_linux_amd64.tar.gz"
echo "  - ${PROJECT_NAME}_windows_amd64.zip"
echo ""

echo "📦 归档文件内容:"
echo "  - ${BINARY_NAME} (可执行文件)"
echo "  - config.yaml"
echo "  - update_config.yaml"
echo ""

echo "🍎 macOS .app Bundle:"
echo "  - ${APP_NAME}_v*_darwin_amd64.app"
echo "  - ${APP_NAME}_v*_darwin_arm64.app"
echo ""

echo "🔐 签名后的 .app.zip:"
echo "  - ${APP_NAME}_v*_darwin_amd64.app.zip"
echo "  - ${APP_NAME}_v*_darwin_arm64.app.zip"
echo ""

echo "==================================="
echo "✅ 所有配置验证通过！"
echo "==================================="
