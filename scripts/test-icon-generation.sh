#!/bin/bash
set -e

echo "🧪 测试图标生成功能..."
echo ""

# 检查源文件
if [ ! -f "build/icon.png" ]; then
    echo "❌ 错误: build/icon.png 不存在"
    exit 1
fi

echo "✓ 源文件存在: build/icon.png"
echo ""

# 运行图标生成脚本
echo "运行图标生成脚本..."
./scripts/generate-icons.sh

echo ""
echo "🔍 验证生成的文件..."
echo ""

# 检查生成的文件
declare -a required_files=(
    ".build/icons/AppIcon.icns"
    ".build/icons/icon.ico"
    ".build/icons/icon_256.png"
    ".build/icons/icon_16.png"
)

missing=0
for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        size=$(du -h "$file" | cut -f1)
        echo "✓ $file ($size)"
    else
        echo "❌ 缺失: $file"
        missing=1
    fi
done

echo ""

# 验证 Windows PNG 图标尺寸（go-winres 要求不超过 256x256）
echo "🔍 验证 Windows PNG 图标尺寸..."
if command -v identify &> /dev/null || command -v magick &> /dev/null; then
    if command -v magick &> /dev/null; then
        IDENTIFY_CMD="magick identify"
    else
        IDENTIFY_CMD="identify"
    fi
    
    for png_file in ".build/icons/icon_256.png" ".build/icons/icon_16.png"; do
        if [ -f "$png_file" ]; then
            dimensions=$($IDENTIFY_CMD -format "%wx%h" "$png_file")
            width=$(echo "$dimensions" | cut -d'x' -f1)
            height=$(echo "$dimensions" | cut -d'x' -f2)
            
            if [ "$width" -le 256 ] && [ "$height" -le 256 ]; then
                echo "✓ $png_file: ${dimensions} (符合 go-winres 要求)"
            else
                echo "❌ $png_file: ${dimensions} (超过 256x256 限制)"
                missing=1
            fi
        fi
    done
else
    echo "⚠️  ImageMagick 未安装，跳过尺寸验证"
fi

echo ""

if [ $missing -eq 0 ]; then
    echo "✅ 所有图标文件生成成功！"
    exit 0
else
    echo "❌ 部分图标文件生成失败"
    exit 1
fi
