#!/bin/bash
# Generate manifest.json from GitHub Release
# Usage: ./generate_manifest.sh <owner/repo> <version> [github_token]
#
# Example: ./generate_manifest.sh myorg/myapp v1.2.3
# Example with token: ./generate_manifest.sh myorg/myapp v1.2.3 ghp_xxxxx

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check arguments
if [ $# -lt 2 ]; then
    echo -e "${RED}Error: Missing required arguments${NC}"
    echo "Usage: $0 <owner/repo> <version> [github_token]"
    echo ""
    echo "Example: $0 myorg/myapp v1.2.3"
    echo "Example with token: $0 myorg/myapp v1.2.3 ghp_xxxxx"
    exit 1
fi

REPO=$1
VERSION=$2
GITHUB_TOKEN=${3:-""}

# Remove 'v' prefix if present for API call
API_VERSION=${VERSION#v}

echo -e "${GREEN}Generating manifest for ${REPO} version ${VERSION}${NC}"

# Set up GitHub API headers
if [ -n "$GITHUB_TOKEN" ]; then
    AUTH_HEADER="Authorization: token $GITHUB_TOKEN"
    echo -e "${GREEN}Using provided GitHub token${NC}"
else
    AUTH_HEADER=""
    echo -e "${YELLOW}Warning: No GitHub token provided. API rate limits may apply.${NC}"
fi

# Fetch release information from GitHub API
echo -e "${GREEN}Fetching release information from GitHub...${NC}"
RELEASE_URL="https://api.github.com/repos/${REPO}/releases/tags/${VERSION}"

if [ -n "$AUTH_HEADER" ]; then
    RELEASE_JSON=$(curl -s -H "$AUTH_HEADER" "$RELEASE_URL")
else
    RELEASE_JSON=$(curl -s "$RELEASE_URL")
fi

# Check if release was found
if echo "$RELEASE_JSON" | grep -q '"message": "Not Found"'; then
    echo -e "${RED}Error: Release ${VERSION} not found for ${REPO}${NC}"
    exit 1
fi

# Extract release information
PUBLISHED_AT=$(echo "$RELEASE_JSON" | grep -o '"published_at": "[^"]*"' | cut -d'"' -f4)
RELEASE_NOTES=$(echo "$RELEASE_JSON" | grep -o '"body": "[^"]*"' | cut -d'"' -f4 | sed 's/\\n/\n/g' | sed 's/\\r//g')

echo -e "${GREEN}Release found:${NC}"
echo "  Version: $VERSION"
echo "  Published: $PUBLISHED_AT"

# Create temporary directory for downloads
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Download checksums file
echo -e "${GREEN}Downloading checksums file...${NC}"
PROJECT_NAME=$(basename "$REPO")
CHECKSUMS_FILE="${PROJECT_NAME}_${VERSION}_checksums.txt"
CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${VERSION}/${CHECKSUMS_FILE}"

if [ -n "$AUTH_HEADER" ]; then
    curl -sL -H "$AUTH_HEADER" -o "$TEMP_DIR/checksums.txt" "$CHECKSUMS_URL" || {
        echo -e "${YELLOW}Warning: Could not download checksums file from ${CHECKSUMS_URL}${NC}"
        echo -e "${YELLOW}Will calculate checksums by downloading assets${NC}"
        CHECKSUMS_FILE=""
    }
else
    curl -sL -o "$TEMP_DIR/checksums.txt" "$CHECKSUMS_URL" || {
        echo -e "${YELLOW}Warning: Could not download checksums file from ${CHECKSUMS_URL}${NC}"
        echo -e "${YELLOW}Will calculate checksums by downloading assets${NC}"
        CHECKSUMS_FILE=""
    }
fi

# Function to get checksum for a file
get_checksum() {
    local filename=$1
    local checksum=""
    
    if [ -n "$CHECKSUMS_FILE" ] && [ -f "$TEMP_DIR/checksums.txt" ]; then
        # Extract checksum from checksums file
        checksum=$(grep "$filename" "$TEMP_DIR/checksums.txt" | awk '{print $1}')
    fi
    
    if [ -z "$checksum" ]; then
        # Download file and calculate checksum
        echo -e "${YELLOW}Calculating checksum for ${filename}...${NC}"
        local asset_url="https://github.com/${REPO}/releases/download/${VERSION}/${filename}"
        if [ -n "$AUTH_HEADER" ]; then
            curl -sL -H "$AUTH_HEADER" -o "$TEMP_DIR/$filename" "$asset_url"
        else
            curl -sL -o "$TEMP_DIR/$filename" "$asset_url"
        fi
        checksum=$(sha256sum "$TEMP_DIR/$filename" | awk '{print $1}')
        rm -f "$TEMP_DIR/$filename"
    fi
    
    echo "$checksum"
}

# Function to get file size
get_file_size() {
    local filename=$1
    local size=""
    
    # Try to get size from GitHub API
    size=$(echo "$RELEASE_JSON" | grep -A 5 "\"name\": \"$filename\"" | grep '"size":' | head -1 | grep -o '[0-9]*')
    
    if [ -z "$size" ]; then
        # Download file to get size
        echo -e "${YELLOW}Getting size for ${filename}...${NC}"
        local asset_url="https://github.com/${REPO}/releases/download/${VERSION}/${filename}"
        if [ -n "$AUTH_HEADER" ]; then
            size=$(curl -sI -H "$AUTH_HEADER" "$asset_url" | grep -i content-length | awk '{print $2}' | tr -d '\r')
        else
            size=$(curl -sI "$asset_url" | grep -i content-length | awk '{print $2}' | tr -d '\r')
        fi
    fi
    
    echo "$size"
}

# Start building manifest JSON
echo -e "${GREEN}Building manifest...${NC}"

# Escape release notes for JSON
RELEASE_NOTES_ESCAPED=$(echo "$RELEASE_NOTES" | sed 's/\\/\\\\/g' | sed 's/"/\\"/g' | sed ':a;N;$!ba;s/\n/\\n/g')

# Start JSON
cat > manifest.json << EOF
{
  "version": "${VERSION#v}",
  "published_at": "$PUBLISHED_AT",
  "release_notes": "$RELEASE_NOTES_ESCAPED",
  "assets": {
EOF

# Define platform mappings
declare -A PLATFORMS=(
    ["windows_amd64"]="${PROJECT_NAME}_windows_amd64.zip"
    ["windows_arm64"]="${PROJECT_NAME}_windows_arm64.zip"
    ["linux_amd64"]="${PROJECT_NAME}_linux_amd64.tar.gz"
    ["linux_arm64"]="${PROJECT_NAME}_linux_arm64.tar.gz"
    ["darwin_amd64"]="${PROJECT_NAME}_darwin_amd64.zip"
    ["darwin_arm64"]="${PROJECT_NAME}_darwin_arm64.zip"
)

FIRST=true
for platform in "${!PLATFORMS[@]}"; do
    filename="${PLATFORMS[$platform]}"
    asset_url="https://github.com/${REPO}/releases/download/${VERSION}/${filename}"
    
    # Check if asset exists
    if [ -n "$AUTH_HEADER" ]; then
        http_code=$(curl -sI -H "$AUTH_HEADER" -o /dev/null -w "%{http_code}" "$asset_url")
    else
        http_code=$(curl -sI -o /dev/null -w "%{http_code}" "$asset_url")
    fi
    
    if [ "$http_code" != "200" ]; then
        echo -e "${YELLOW}Skipping ${platform}: asset not found${NC}"
        continue
    fi
    
    echo -e "${GREEN}Processing ${platform}...${NC}"
    
    # Get checksum and size
    checksum=$(get_checksum "$filename")
    size=$(get_file_size "$filename")
    
    # Add comma if not first entry
    if [ "$FIRST" = false ]; then
        echo "," >> manifest.json
    fi
    FIRST=false
    
    # Add asset entry
    cat >> manifest.json << EOF
    "$platform": {
      "url": "$asset_url",
      "size": $size,
      "checksum": "$checksum",
      "name": "$filename"
    }
EOF
done

# Close JSON
cat >> manifest.json << EOF

  }
}
EOF

echo -e "${GREEN}Manifest generated successfully: manifest.json${NC}"
echo ""
echo -e "${GREEN}Summary:${NC}"
echo "  Version: ${VERSION#v}"
echo "  Published: $PUBLISHED_AT"
echo "  Platforms: $(grep -c '"url":' manifest.json)"
echo ""
echo -e "${GREEN}You can now upload manifest.json to your update server${NC}"
