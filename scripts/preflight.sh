#!/usr/bin/env bash
# Verify required tools before development begins.
set -euo pipefail

PASS=0
FAIL=0

check() {
  local name="$1"
  local cmd="$2"
  local min_version="$3"
  if command -v "$cmd" &>/dev/null; then
    local ver
    ver=$($cmd version 2>/dev/null || $cmd --version 2>/dev/null || echo "unknown" | head -1)
    echo "  ✓ $name ($ver)"
    ((PASS++))
  else
    echo "  ✗ $name — not found (required: $min_version)"
    ((FAIL++))
  fi
}

echo ""
echo "POS-Stery preflight check"
echo "─────────────────────────"
check "Go"             go          "1.24+"
check "buf"            buf         "1.x"
check "sqlc"           sqlc        "1.25+"
check "migrate"        migrate     "4.x"
check "Docker"         docker      "24+"
check "docker compose" "docker compose" "v2"
check "PHP"            php         "8.3+"
check "Composer"       composer    "2.x"
check "Node.js"        node        "20+"
check "pnpm"           pnpm        "9+"

echo ""
if [ "$FAIL" -gt 0 ]; then
  echo "⚠ $FAIL tool(s) missing. Install them before running make all."
  exit 1
else
  echo "✓ All $PASS tools found."
fi
