#!/bin/bash

APP_PATH="$1"

if [ -z "$APP_PATH" ]; then
    echo "用法: $0 /path/to/App.app"
    exit 1
fi

echo "🔍 开始诊断签名问题..."
echo "======================================"

# 1. 检查应用结构
echo "1. 检查应用结构..."
if [ ! -d "$APP_PATH" ]; then
    echo "❌ 应用不存在: $APP_PATH"
    exit 1
fi

if [ ! -f "$APP_PATH/Contents/Info.plist" ]; then
    echo "❌ 缺少 Info.plist"
fi

if [ ! -d "$APP_PATH/Contents/MacOS" ]; then
    echo "❌ 缺少 MacOS 目录"
fi

EXECUTABLE=$(defaults read "$APP_PATH/Contents/Info.plist" CFBundleExecutable 2>/dev/null)
if [ -z "$EXECUTABLE" ]; then
    echo "❌ 无法获取可执行文件名"
else
    echo "可执行文件: $EXECUTABLE"
    if [ ! -f "$APP_PATH/Contents/MacOS/$EXECUTABLE" ]; then
        echo "❌ 可执行文件不存在: $EXECUTABLE"
    fi
fi

# 2. 检查文件权限
echo -e "\n2. 检查文件权限..."
ls -la "$APP_PATH/Contents/MacOS/"
chmod +x "$APP_PATH/Contents/MacOS/"* 2>/dev/null

# 3. 检查签名
echo -e "\n3. 检查签名..."
codesign -dv --verbose=4 "$APP_PATH" 2>&1 | head -50

# 4. 检查公证和订证
echo -e "\n4. 检查订证状态..."
xcrun stapler validate "$APP_PATH" 2>&1

# 5. 检查 Gatekeeper
echo -e "\n5. 检查 Gatekeeper..."
spctl -a -t exec -vv "$APP_PATH" 2>&1

# 6. 检查崩溃报告
echo -e "\n6. 检查控制台日志（最后5条相关日志）..."
log show --predicate 'eventMessage contains "$(basename "$APP_PATH")"' --last 1h | tail -20

# 7. 尝试运行并捕获输出
echo -e "\n7. 尝试运行应用..."
"$APP_PATH/Contents/MacOS/$EXECUTABLE" 2>&1 | head -20

echo -e "\n======================================"
echo "诊断完成！"
