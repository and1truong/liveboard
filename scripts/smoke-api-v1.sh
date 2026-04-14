#!/usr/bin/env bash
# Smoke test for /api/v1. Usage: BASE=http://localhost:7070 bash scripts/smoke-api-v1.sh
set -euo pipefail

BASE="${BASE:-http://localhost:7070}"
BOARD="${BOARD:-demo}"

pass() { printf "  \033[32m✓\033[0m %s\n" "$1"; }
fail() { printf "  \033[31m✗\033[0m %s\n" "$1"; exit 1; }

say() { printf "\n\033[1m%s\033[0m\n" "$1"; }

say "Versions probe"
curl -fsS "$BASE/api/versions" | grep -q '"current":"v1"' && pass "/api/versions" || fail "/api/versions"

say "Workspace"
curl -fsS "$BASE/api/v1/workspace" | grep -q '"dir"' && pass "/api/v1/workspace" || fail "/api/v1/workspace"

say "Boards list"
curl -fsS "$BASE/api/v1/boards" | grep -q '\[' && pass "/api/v1/boards" || fail "/api/v1/boards"

say "Board detail"
curl -fsS "$BASE/api/v1/boards/$BOARD" | grep -q '"columns"' && pass "/api/v1/boards/$BOARD" || fail "/api/v1/boards/$BOARD"

say "Board settings"
curl -fsS "$BASE/api/v1/boards/$BOARD/settings" > /dev/null && pass "GET settings" || fail "GET settings"

say "Mutation: add_card"
RESP=$(curl -fsS -X POST "$BASE/api/v1/boards/$BOARD/mutations" \
  -H 'Content-Type: application/json' \
  -d '{"client_version":-1,"op":{"type":"add_card","column":"Todo","title":"smoke-test"}}')
echo "$RESP" | grep -q '"version"' && pass "POST mutation" || fail "POST mutation"

say "Version conflict"
STATUS=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$BASE/api/v1/boards/$BOARD/mutations" \
  -H 'Content-Type: application/json' \
  -d '{"client_version":0,"op":{"type":"add_card","column":"Todo","title":"stale"}}')
[ "$STATUS" = "409" ] && pass "409 on stale version" || fail "expected 409, got $STATUS"

say "SSE connects"
timeout 2 curl -fsSN "$BASE/api/v1/events?board=$BOARD" | head -n 1 | grep -q 'event:' && pass "SSE connect" || fail "SSE connect"

say "All smoke tests passed."
