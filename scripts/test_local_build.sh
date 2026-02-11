#!/bin/bash
# Quick local build and test script

set -e

echo "=== Local Build and Test ==="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Get app name from config
APP_NAME=$(jq -r '.app.name' app-config.json)

echo "App: $APP_NAME"
echo ""

# Step 1: Build
echo -e "${GREEN}[1/4]${NC} Building binary..."
make build-macos || {
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
}
echo -e "${GREEN}✓${NC} Build successful"
echo ""

# Step 2: Create .app bundle
echo -e "${GREEN}[2/4]${NC} Creating .app bundle..."
BINARY_PATH="bin/$APP_NAME" \
OUTPUT_DIR="dist" \
./scripts/create_app_bundle.sh || {
    echo -e "${RED}❌ App bundle creation failed${NC}"
    exit 1
}
echo -e "${GREEN}✓${NC} App bundle created"
echo ""

# Find the created .app
APP_PATH=$(find dist -name "*.app" -type d | head -1)
if [ -z "$APP_PATH" ]; then
    echo -e "${RED}❌ App bundle not found${NC}"
    exit 1
fi

echo "App bundle: $APP_PATH"
echo ""

# Step 3: Test startup
echo -e "${GREEN}[3/4]${NC} Testing application startup..."
chmod +x ./scripts/test_app_startup.sh
./scripts/test_app_startup.sh "$APP_PATH" 10 || {
    echo -e "${RED}❌ Startup test failed${NC}"
    echo ""
    echo "=== Troubleshooting ==="
    echo "1. Check application logs:"
    echo "   tail -f ~/Library/Logs/wx_video_download/app_*.log"
    echo ""
    echo "2. Run from terminal to see errors:"
    echo "   $APP_PATH/Contents/MacOS/$APP_NAME"
    echo ""
    echo "3. Check crash reports:"
    echo "   ls -lt ~/Library/Logs/DiagnosticReports/${APP_NAME}*"
    exit 1
}
echo -e "${GREEN}✓${NC} Startup test passed"
echo ""

# Step 4: Manual test
echo -e "${GREEN}[4/4]${NC} Opening app for manual testing..."
echo "Press Ctrl+C to skip manual test, or wait 5 seconds..."
sleep 5 || {
    echo "Skipped manual test"
    exit 0
}

open "$APP_PATH"
echo ""
echo -e "${GREEN}✓${NC} App opened"
echo ""
echo "=== Manual Test Checklist ==="
echo "1. Does the app window appear?"
echo "2. Can you interact with the UI?"
echo "3. Are there any error messages?"
echo "4. Check the logs:"
echo "   tail -f ~/Library/Logs/wx_video_download/app_*.log"
echo ""
echo -e "${YELLOW}Press Enter when done testing...${NC}"
read

echo ""
echo -e "${GREEN}=== All tests completed ===${NC}"
echo ""
echo "Next steps:"
echo "1. If all tests passed, you can commit and push"
echo "2. Create a tag to trigger release: git tag v0.x.x && git push origin v0.x.x"
echo "3. Monitor GitHub Actions for the release build"
