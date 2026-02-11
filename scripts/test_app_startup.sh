#!/bin/bash
# Test application startup and collect diagnostics

set -e

APP_PATH="${1:-dist/*.app}"
TIMEOUT="${2:-5}"

echo "=== Application Startup Test ==="
echo "App path: $APP_PATH"
echo "Timeout: ${TIMEOUT}s"
echo "Current arch: $(uname -m)"
echo ""

# Find the app
if [ ! -d "$APP_PATH" ]; then
    echo "❌ App not found at: $APP_PATH"
    exit 1
fi

# Get binary path
BINARY_PATH=$(find "$APP_PATH/Contents/MacOS" -type f -perm +111 | head -1)
if [ ! -f "$BINARY_PATH" ]; then
    echo "❌ Binary not found in: $APP_PATH/Contents/MacOS/"
    exit 1
fi

echo "✓ Found app: $APP_PATH"
echo "✓ Found binary: $BINARY_PATH"
echo ""

# Check structure
echo "=== Checking .app structure ==="
ls -la "$APP_PATH/Contents/"
echo ""
echo "=== MacOS directory ==="
ls -la "$APP_PATH/Contents/MacOS/"
echo ""

# Check Info.plist
echo "=== Checking Info.plist ==="
if [ -f "$APP_PATH/Contents/Info.plist" ]; then
    plutil -p "$APP_PATH/Contents/Info.plist" | head -20
    echo "✓ Info.plist is valid"
else
    echo "❌ Info.plist not found"
    exit 1
fi
echo ""

# Check signature
echo "=== Checking code signature ==="
if codesign -vvv --deep --strict "$APP_PATH" 2>&1; then
    echo "✓ Signature is valid"
else
    echo "⚠️  Signature verification failed (this is OK for unsigned builds)"
fi
echo ""

# Check if binary is executable
echo "=== Checking binary permissions ==="
if [ -x "$BINARY_PATH" ]; then
    echo "✓ Binary is executable"
else
    echo "❌ Binary is not executable"
    chmod +x "$BINARY_PATH"
    echo "✓ Fixed permissions"
fi
echo ""

# Check binary architecture
echo "=== Checking binary architecture ==="
file "$BINARY_PATH"
lipo -info "$BINARY_PATH" 2>/dev/null || echo "Not a universal binary"
echo ""

# Test version command (if supported)
echo "=== Testing --version command ==="
if timeout "${TIMEOUT}s" "$BINARY_PATH" --version 2>&1; then
    echo "✓ Version command succeeded"
else
    EXIT_CODE=$?
    if [ $EXIT_CODE -eq 124 ]; then
        echo "⚠️  Command timed out (this might be normal if app opens a window)"
    else
        echo "❌ Command failed with exit code: $EXIT_CODE"
    fi
fi
echo ""

# Clear any existing logs
LOG_DIR="$HOME/Library/Logs/wx_video_download"
if [ -d "$LOG_DIR" ]; then
    echo "=== Clearing old logs ==="
    rm -f "$LOG_DIR"/*.log
fi

# Test app launch (background)
echo "=== Testing app launch ==="
echo "Starting app in background..."
"$BINARY_PATH" > /tmp/app_stdout.log 2> /tmp/app_stderr.log &
APP_PID=$!
echo "Started app with PID: $APP_PID"

# Wait for app to initialize
echo "Waiting for app to initialize..."
sleep 3

# Check if still running
if ps -p $APP_PID > /dev/null 2>&1; then
    echo "✓ App is running"
    
    # Wait a bit more to ensure it's stable
    sleep 2
    
    if ps -p $APP_PID > /dev/null 2>&1; then
        echo "✓ App is stable"
    else
        echo "❌ App crashed after initial startup"
        EXIT_CODE=1
    fi
    
    # Kill it
    echo "Terminating app..."
    kill $APP_PID 2>/dev/null || true
    sleep 1
    
    # Force kill if needed
    if ps -p $APP_PID > /dev/null 2>&1; then
        kill -9 $APP_PID 2>/dev/null || true
    fi
    
    echo "✓ App terminated"
else
    echo "❌ App exited immediately"
    EXIT_CODE=1
fi
echo ""

# Show stdout/stderr
echo "=== Application stdout ==="
cat /tmp/app_stdout.log 2>/dev/null || echo "(empty)"
echo ""
echo "=== Application stderr ==="
cat /tmp/app_stderr.log 2>/dev/null || echo "(empty)"
echo ""

# Check for log files
echo "=== Checking application log files ==="
if [ -d "$LOG_DIR" ]; then
    echo "Log directory: $LOG_DIR"
    ls -lh "$LOG_DIR" 2>/dev/null || echo "No logs yet"
    echo ""
    
    # Show latest log
    LATEST_LOG=$(ls -t "$LOG_DIR"/*.log 2>/dev/null | head -1)
    if [ -f "$LATEST_LOG" ]; then
        echo "=== Latest application log ==="
        cat "$LATEST_LOG"
        echo ""
    else
        echo "⚠️  No log files created"
    fi
else
    echo "⚠️  No log directory found at $LOG_DIR"
fi
echo ""

# Check for crash reports
echo "=== Checking for crash reports ==="
CRASH_DIR="$HOME/Library/Logs/DiagnosticReports"
BINARY_NAME=$(basename "$BINARY_PATH")
RECENT_CRASHES=$(find "$CRASH_DIR" -name "${BINARY_NAME}*.crash" -o -name "${BINARY_NAME}*.ips" -mmin -5 2>/dev/null)
if [ -n "$RECENT_CRASHES" ]; then
    echo "❌ Found recent crash reports:"
    echo "$RECENT_CRASHES"
    echo ""
    echo "=== Latest crash report ==="
    head -100 $(echo "$RECENT_CRASHES" | head -1)
    EXIT_CODE=1
else
    echo "✓ No crash reports found"
fi
echo ""

# Check system logs for errors
echo "=== Checking system logs for errors ==="
log show --predicate "process == \"$BINARY_NAME\"" --last 5m --style compact 2>/dev/null | tail -30 || echo "No system logs found"
echo ""

echo "=== Test complete ==="
exit ${EXIT_CODE:-0}
