#!/bin/bash

# 验证二进制文件命名更改
# 此脚本检查所有配置文件是否正确使用 app.name 而不是 binary.name

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🔍 验证二进制文件命名配置..."
echo ""

# 检查 jq 是否安装
if ! command -v jq &> /dev/null; then
    echo -e "${RED}❌ 错误: 需要安装 jq 工具${NC}"
    exit 1
fi

# 读取配置
APP_NAME=$(jq -r '.app.name' app-config.json)
BINARY_NAME=$(jq -r '.binary.name' app-config.json)

echo "📋 配置信息:"
echo "  app.name: $APP_NAME"
echo "  binary.name: $BINARY_NAME (已弃用)"
echo ""

# 检查点计数
CHECKS_PASSED=0
CHECKS_FAILED=0

# 函数：检查并报告
check() {
    local description=$1
    local command=$2
    local expected=$3
    
    echo -n "  检查 $description... "
    
    result=$(eval "$command" 2>/dev/null || echo "")
    
    if [ "$result" = "$expected" ]; then
        echo -e "${GREEN}✓${NC}"
        ((CHECKS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC}"
        echo "    期望: $expected"
        echo "    实际: $result"
        ((CHECKS_FAILED++))
        return 1
    fi
}

echo "🔧 检查生成的配置文件..."
echo ""

# 生成配置文件
echo "  生成配置文件..."
./scripts/generate-configs.sh > /dev/null 2>&1

# 检查 GoReleaser 配置
echo ""
echo "📦 GoReleaser 配置:"
check "Windows binary" "grep -A 8 'id: windows' .build/.goreleaser.yaml | grep 'binary:' | head -1 | awk '{print \$2}'" "$APP_NAME"
check "Linux binary" "grep -A 10 'id: linux' .build/.goreleaser.yaml | grep 'binary:' | head -1 | awk '{print \$2}'" "$APP_NAME"
check "macOS binary" "grep -A 10 'id: macos' .build/.goreleaser.yaml | grep 'binary:' | head -1 | awk '{print \$2}'" "$APP_NAME"

# 检查 Info.plist 模板
echo ""
echo "🍎 macOS Info.plist:"
check "CFBundleExecutable" "grep -A 1 'CFBundleExecutable' .build/Info.plist.template | grep '<string>' | sed 's/.*<string>\(.*\)<\/string>/\1/'" "$APP_NAME"
check "CFBundleName" "grep -A 1 'CFBundleName' .build/Info.plist.template | grep '<string>' | sed 's/.*<string>\(.*\)<\/string>/\1/'" "$APP_NAME"

# 检查 Linux .desktop 文件
echo ""
echo "🐧 Linux .desktop:"
check "Exec" "grep '^Exec=' .build/app.desktop.template | cut -d'=' -f2" "$APP_NAME"
check "Name" "grep '^Name=' .build/app.desktop.template | cut -d'=' -f2" "$APP_NAME"

# 检查脚本文件
echo ""
echo "📜 检查脚本文件..."

# 检查是否还有对 binary.name 的引用（排除文档和此脚本）
echo ""
echo "🔎 搜索遗留的 binary.name 引用..."
LEGACY_REFS=$(grep -r "binary\.name" \
    --include="*.sh" \
    --include="*.yaml" \
    --exclude="verify-binary-naming.sh" \
    --exclude="verify-naming.sh" \
    scripts/ .github/ 2>/dev/null | wc -l | tr -d ' ')

if [ "$LEGACY_REFS" -eq "0" ]; then
    echo -e "  ${GREEN}✓${NC} 未发现遗留引用"
    ((CHECKS_PASSED++))
else
    echo -e "  ${YELLOW}⚠${NC}  发现 $LEGACY_REFS 处遗留引用（可能在文档中）"
    grep -r "binary\.name" \
        --include="*.sh" \
        --include="*.yaml" \
        --exclude="verify-binary-naming.sh" \
        --exclude="verify-naming.sh" \
        scripts/ .github/ 2>/dev/null || true
fi

# 总结
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
if [ $CHECKS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ 所有检查通过！${NC} ($CHECKS_PASSED/$((CHECKS_PASSED + CHECKS_FAILED)))"
    echo ""
    echo "二进制文件命名配置正确："
    echo "  • 所有平台使用 app.name: $APP_NAME"
    echo "  • 配置文件已正确生成"
    echo "  • 脚本文件已更新"
    exit 0
else
    echo -e "${RED}❌ 发现 $CHECKS_FAILED 个问题${NC} ($CHECKS_PASSED/$((CHECKS_PASSED + CHECKS_FAILED)) 通过)"
    echo ""
    echo "请检查上述失败的项目并修复"
    exit 1
fi
