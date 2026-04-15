#!/usr/bin/env bash
# Renderer bundle size gate.
# Measured 2026-04-15: 142889 bytes gzipped (added marked lazy chunk in P6.1).
# Budget = measured + ~5 KB headroom, rounded to next 5 KB.
set -euo pipefail
MAX_BYTES="${MAX_BYTES:-148480}"  # 145 KB
if ! compgen -G "web/renderer/default/dist/assets/*.js" > /dev/null; then
  echo "ERROR: no built JS in web/renderer/default/dist/assets/. Run 'make renderer' first."
  exit 1
fi
TOTAL=$(gzip -c web/renderer/default/dist/assets/*.js | wc -c | tr -d ' ')
echo "renderer bundle (gz): ${TOTAL} bytes (max ${MAX_BYTES})"
if [ "$TOTAL" -gt "$MAX_BYTES" ]; then
  echo "ERROR: bundle exceeds budget"
  exit 1
fi
