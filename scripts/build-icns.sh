#!/usr/bin/env bash
set -euo pipefail

# Build macOS .icns from the 1024×1024 SVG source
# Requires: rsvg-convert (librsvg), iconutil (macOS built-in)

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

SVG_SOURCE="$ROOT_DIR/web/img/liveboard-icon-macos.svg"
ICONSET_DIR="$ROOT_DIR/build/LiveBoard.iconset"
ICNS_OUTPUT="$ROOT_DIR/cmd/liveboard-desktop/icon.icns"

if ! command -v rsvg-convert &>/dev/null; then
  echo "Error: rsvg-convert not found. Install with: brew install librsvg"
  exit 1
fi

echo "Building .icns from $SVG_SOURCE"

rm -rf "$ICONSET_DIR"
mkdir -p "$ICONSET_DIR"

# All required sizes for macOS .icns
# Standard sizes + @2x retina variants
declare -a SIZES=(
  "icon_16x16:16"
  "icon_16x16@2x:32"
  "icon_32x32:32"
  "icon_32x32@2x:64"
  "icon_64x64:64"
  "icon_64x64@2x:128"
  "icon_128x128:128"
  "icon_128x128@2x:256"
  "icon_256x256:256"
  "icon_256x256@2x:512"
  "icon_512x512:512"
  "icon_512x512@2x:1024"
)

for entry in "${SIZES[@]}"; do
  name="${entry%%:*}"
  size="${entry##*:}"
  echo "  ${name}.png (${size}×${size})"
  rsvg-convert -w "$size" -h "$size" "$SVG_SOURCE" -o "$ICONSET_DIR/${name}.png"
done

echo "Compiling .icns..."
iconutil -c icns "$ICONSET_DIR" -o "$ICNS_OUTPUT"

echo "Cleaning up..."
rm -rf "$ICONSET_DIR"

echo "Done: $ICNS_OUTPUT ($(du -h "$ICNS_OUTPUT" | cut -f1))"
