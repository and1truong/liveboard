#!/bin/bash
set -euo pipefail

VERSION="${1:?Usage: $0 <version>}"
ZIP="LiveBoard-${VERSION}-macos-universal.zip"
TAP_REPO="and1truong/homebrew-tap"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [ ! -f "$ZIP" ]; then
  echo "Error: $ZIP not found" >&2
  exit 1
fi

SHA=$(shasum -a 256 "$ZIP" | awk '{print $1}')
echo "SHA256: $SHA"

# Generate cask from template
CASK=$(sed -e "s/VERSION/$VERSION/g" -e "s/SHA256/$SHA/g" "$SCRIPT_DIR/liveboard-desktop.rb.tmpl")

# Clone tap, update cask, push
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

gh repo clone "$TAP_REPO" "$TMPDIR/tap" -- --depth 1
mkdir -p "$TMPDIR/tap/Casks"
echo "$CASK" > "$TMPDIR/tap/Casks/liveboard-desktop.rb"

cd "$TMPDIR/tap"
git add Casks/liveboard-desktop.rb
git commit -m "Update liveboard-desktop cask to $VERSION"
git push

echo "Cask updated to $VERSION"
