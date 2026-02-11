#!/bin/bash

# 开发环境测试脚本
# 自动设置测试环境并启动应用

set -e

echo "🚀 Setting up development test environment..."

# 获取配置目录
CONFIG_DIR="$HOME/.config/velo"

# 创建配置目录
mkdir -p "$CONFIG_DIR"

# 复制开发配置
echo "📋 Copying development config..."
cp config/update_config.dev.yaml "$CONFIG_DIR/"

echo "✅ Configuration ready at: $CONFIG_DIR/update_config.dev.yaml"
echo ""
echo "📝 Next steps:"
echo "   1. In another terminal, run:"
echo "      cd scripts && go run test_update_server.go --version=2.0.0"
echo ""
echo "   2. Then start the app:"
echo "      go run main.go"
echo ""
echo "   3. Click 'Check Update' button in the app"
echo ""
echo "💡 Tip: Change server version to test different scenarios"
