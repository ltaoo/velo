#!/bin/bash

# DMG Creation Script for macOS Applications
# This script creates a DMG installer with custom background and layout

set -e  # Exit on error
set -u  # Exit on undefined variable

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Global variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_FILE="$PROJECT_ROOT/app-config.json"
TEMP_DIR=""
OUTPUT_DIR="${OUTPUT_DIR:-$PROJECT_ROOT/dist}"

# Parse command line arguments and environment variables
APP_NAME="${APP_NAME:-}"
VERSION="${VERSION:-}"
ARCH="${ARCH:-}"
APP_PATH="${APP_PATH:-}"

# Function to print error messages and exit
error_exit() {
    local message=$1
    local exit_code=${2:-1}
    echo -e "${RED}Error: $message${NC}" >&2
    exit "$exit_code"
}

# Function to print info messages
info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

# Function to print warning messages
warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Function to check if required dependencies are installed
check_dependencies() {
    info "Checking dependencies..."
    
    # Check for create-dmg
    if ! command -v create-dmg &> /dev/null; then
        error_exit "create-dmg not found. Install with: brew install create-dmg" 1
    fi
    
    # Check for jq
    if ! command -v jq &> /dev/null; then
        error_exit "jq not found. Install with: brew install jq" 1
    fi
    
    info "All dependencies are installed"
}

# Function to parse and validate input parameters
parse_parameters() {
    info "Parsing parameters..."
    
    # Check required parameters
    if [ -z "$APP_NAME" ]; then
        error_exit "APP_NAME is required. Set via environment variable or command line." 2
    fi
    
    if [ -z "$VERSION" ]; then
        error_exit "VERSION is required. Set via environment variable or command line." 2
    fi
    
    if [ -z "$ARCH" ]; then
        error_exit "ARCH is required. Set via environment variable or command line." 2
    fi
    
    info "Parameters: APP_NAME=$APP_NAME, VERSION=$VERSION, ARCH=$ARCH"
}

# Function to read configuration from app-config.json
read_config() {
    info "Reading configuration from $CONFIG_FILE..."
    
    if [ ! -f "$CONFIG_FILE" ]; then
        error_exit "Configuration file not found: $CONFIG_FILE" 2
    fi
    
    # Read DMG configuration
    WINDOW_WIDTH=$(jq -r '.platforms.macos.dmg.window_size.width // 660' "$CONFIG_FILE")
    WINDOW_HEIGHT=$(jq -r '.platforms.macos.dmg.window_size.height // 400' "$CONFIG_FILE")
    ICON_SIZE=$(jq -r '.platforms.macos.dmg.icon_size // 128' "$CONFIG_FILE")
    APP_X=$(jq -r '.platforms.macos.dmg.positions.app.x // 180' "$CONFIG_FILE")
    APP_Y=$(jq -r '.platforms.macos.dmg.positions.app.y // 170' "$CONFIG_FILE")
    APPS_X=$(jq -r '.platforms.macos.dmg.positions.applications.x // 480' "$CONFIG_FILE")
    APPS_Y=$(jq -r '.platforms.macos.dmg.positions.applications.y // 170' "$CONFIG_FILE")
    
    # Read background configuration
    AUTO_GENERATE=$(jq -r '.platforms.macos.dmg.background.auto_generate // false' "$CONFIG_FILE")
    CUSTOM_PATH=$(jq -r '.platforms.macos.dmg.background.custom_path // empty' "$CONFIG_FILE")
    
    # Read app display name
    APP_DISPLAY_NAME=$(jq -r '.app.display_name // .app.name' "$CONFIG_FILE")
    
    info "Configuration loaded successfully"
}

# Function to validate background image
validate_background() {
    info "Validating background image..."
    
    # Check if custom path is provided
    if [ "$AUTO_GENERATE" = "false" ] && [ -z "$CUSTOM_PATH" ]; then
        error_exit "Background image path not configured in app-config.json. Set platforms.macos.dmg.background.custom_path or enable auto_generate." 5
    fi
    
    # If custom path is provided, validate it exists
    if [ -n "$CUSTOM_PATH" ] && [ "$CUSTOM_PATH" != "null" ]; then
        # Handle relative paths
        if [[ "$CUSTOM_PATH" != /* ]]; then
            CUSTOM_PATH="$PROJECT_ROOT/$CUSTOM_PATH"
        fi
        
        if [ ! -f "$CUSTOM_PATH" ]; then
            error_exit "Background image not found at $CUSTOM_PATH" 5
        fi
        
        info "Using custom background image: $CUSTOM_PATH"
        BACKGROUND_PATH="$CUSTOM_PATH"
    elif [ "$AUTO_GENERATE" = "true" ]; then
        info "Auto-generate mode enabled, no custom background required"
        BACKGROUND_PATH=""
    else
        error_exit "Invalid background configuration" 5
    fi
}

# Function to validate app bundle
validate_app_bundle() {
    info "Validating app bundle..."
    
    # If APP_PATH not provided, try to find it in dist directory
    if [ -z "$APP_PATH" ]; then
        # Try to find the app bundle in dist directory
        local possible_path="$OUTPUT_DIR/${APP_NAME}_${VERSION}_darwin_${ARCH}/${APP_DISPLAY_NAME}.app"
        if [ -d "$possible_path" ]; then
            APP_PATH="$possible_path"
        else
            # Try alternative naming
            possible_path="$OUTPUT_DIR/${APP_NAME}.app"
            if [ -d "$possible_path" ]; then
                APP_PATH="$possible_path"
            else
                error_exit "App bundle not found. Please specify APP_PATH." 2
            fi
        fi
    fi
    
    # Validate app bundle exists
    if [ ! -d "$APP_PATH" ]; then
        error_exit "Invalid app bundle at $APP_PATH. Directory does not exist." 2
    fi
    
    # Validate Info.plist exists
    if [ ! -f "$APP_PATH/Contents/Info.plist" ]; then
        error_exit "Invalid app bundle at $APP_PATH. Info.plist not found." 2
    fi
    
    info "App bundle validated: $APP_PATH"
}

# Function to create temporary directory
create_temp_dir() {
    TEMP_DIR=$(mktemp -d)
    info "Created temporary directory: $TEMP_DIR"
}

# Function to cleanup temporary files
cleanup_temp_files() {
    if [ -n "$TEMP_DIR" ] && [ -d "$TEMP_DIR" ]; then
        info "Cleaning up temporary files..."
        rm -rf "$TEMP_DIR"
    fi
}

# Register cleanup on exit
trap cleanup_temp_files EXIT

# Function to create DMG package
create_dmg_package() {
    info "Creating DMG package..."
    
    # Create output directory if it doesn't exist
    mkdir -p "$OUTPUT_DIR"
    
    # Define output DMG filename
    local dmg_name="${APP_NAME}_${VERSION}_${ARCH}.dmg"
    local dmg_path="$OUTPUT_DIR/$dmg_name"
    
    # Remove existing DMG if it exists
    if [ -f "$dmg_path" ]; then
        warn "Removing existing DMG: $dmg_path"
        rm -f "$dmg_path"
    fi
    
    # Get the actual app bundle name from the path
    local app_bundle_name=$(basename "$APP_PATH")
    
    # Build create-dmg command
    local create_dmg_cmd=(
        create-dmg
        --volname "$APP_DISPLAY_NAME"
        --window-pos 200 120
        --window-size "$WINDOW_WIDTH" "$WINDOW_HEIGHT"
        --icon-size "$ICON_SIZE"
        --icon "$app_bundle_name" "$APP_X" "$APP_Y"
        --hide-extension "$app_bundle_name"
        --app-drop-link "$APPS_X" "$APPS_Y"
    )
    
    # Add background if provided
    if [ -n "$BACKGROUND_PATH" ]; then
        create_dmg_cmd+=(--background "$BACKGROUND_PATH")
    fi
    
    # Add output path and source
    create_dmg_cmd+=("$dmg_path" "$APP_PATH")
    
    # Execute create-dmg
    info "Executing: ${create_dmg_cmd[*]}"
    if ! "${create_dmg_cmd[@]}"; then
        error_exit "create-dmg failed. Check the error messages above." 6
    fi
    
    # Verify DMG was created
    if [ ! -f "$dmg_path" ]; then
        error_exit "DMG file was not created at $dmg_path" 6
    fi
    
    info "DMG created successfully: $dmg_path"
    echo "$dmg_path"
}

# Main execution
main() {
    info "Starting DMG creation process..."
    
    # Check dependencies
    check_dependencies
    
    # Parse parameters
    parse_parameters
    
    # Read configuration
    read_config
    
    # Validate background image
    validate_background
    
    # Validate app bundle
    validate_app_bundle
    
    # Create temporary directory
    create_temp_dir
    
    # Create DMG package
    create_dmg_package
    
    info "DMG creation completed successfully!"
}

# Run main function
main "$@"
