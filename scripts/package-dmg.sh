#!/bin/bash
# Packaging script for Symprint macOS Application -> DMG
set -euo pipefail

APP_PATH="${1:-client/build/Build/Products/Release/Symprint.app}"
DMG_PATH="${2:-dist/Symprint.dmg}"
VOL_NAME="Symprint"

echo "==> Packaging DMG..."
echo "    App:  $APP_PATH"
echo "    Dest: $DMG_PATH"

if [ ! -d "$APP_PATH" ]; then
    echo "Error: App bundle not found at $APP_PATH"
    exit 1
fi

# Ensure target directory exists
mkdir -p "$(dirname "$DMG_PATH")"
rm -f "$DMG_PATH"

# Check if create-dmg is installed
if command -v create-dmg >/dev/null 2>&1; then
    echo "==> create-dmg found. Packaging with visual styling..."
    create-dmg \
        --volname "$VOL_NAME" \
        --window-pos 200 120 \
        --window-size 600 400 \
        --icon-size 100 \
        --icon "Symprint.app" 175 190 \
        --hide-extension "Symprint.app" \
        --app-drop-link 425 190 \
        "$DMG_PATH" \
        "$(dirname "$APP_PATH")/Symprint.app"
else
    echo "==> create-dmg not found. Falling back to native hdiutil..."
    
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT
    
    echo "    Creating temporary directory structure..."
    cp -R "$APP_PATH" "$TMP_DIR/"
    ln -s /Applications "$TMP_DIR/Applications"
    
    echo "    Building DMG with hdiutil..."
    hdiutil create \
        -fs HFS+ \
        -srcfolder "$TMP_DIR" \
        -volname "$VOL_NAME" \
        -format UDZO \
        -ov \
        "$DMG_PATH"
fi

echo "✓ DMG created successfully at $DMG_PATH"
