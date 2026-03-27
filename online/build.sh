#!/usr/bin/env bash
# Build LiveBoard Online — copies shared assets and compiles CSS.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DIST="${PROJECT_ROOT}/dist/online"

echo "Building LiveBoard Online..."

# Clean
rm -rf "$DIST"
mkdir -p "$DIST/js" "$DIST/css"

# Copy online-specific files
cp "$SCRIPT_DIR/index.html" "$DIST/"
cp "$SCRIPT_DIR/js/"*.js "$DIST/js/"

# Copy Alpine.js from server version
cp "$PROJECT_ROOT/web/js/alpine.min.js" "$DIST/js/"

# Build CSS with Tailwind (scan online HTML for classes)
command -v tailwindcss >/dev/null 2>&1 || { echo "Error: tailwindcss not installed"; exit 1; }
tailwindcss -i "$PROJECT_ROOT/web/css/input.css" -o "$DIST/css/liveboard.css" --minify \
  --content "$DIST/index.html,$DIST/js/*.js"

echo "Built to $DIST"
echo "Open dist/online/index.html in your browser to use LiveBoard Online."
