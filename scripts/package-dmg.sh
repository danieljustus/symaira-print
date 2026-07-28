#!/bin/bash
# Packaging script for Symprint macOS Application -> DMG
set -euo pipefail

APP_PATH="${1:-client/build/Build/Products/Release/Symprint.app}"
DMG_PATH="${2:-dist/Symprint.dmg}"
VOL_NAME="Symaira Print"

echo "==> Packaging DMG..."
echo "    App:  $APP_PATH"
echo "    Dest: $DMG_PATH"

if [ ! -d "$APP_PATH" ]; then
    echo "Error: App bundle not found at $APP_PATH"
    exit 1
fi

mkdir -p "$(dirname "$DMG_PATH")"
rm -f "$DMG_PATH"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
"${SCRIPT_DIR}/create-symaira-dmg.sh" "$APP_PATH" "$DMG_PATH" "$VOL_NAME"

echo "✓ DMG created successfully at $DMG_PATH"
