#!/usr/bin/env bash
# Verify every service has matching .up.sql and .down.sql migration pairs.
set -euo pipefail

SERVICES=(auth-service product-service inventory-service sales-service supplier-service customer-service)
PASS=0
FAIL=0

for svc in "${SERVICES[@]}"; do
  dir="services/$svc/migrations"
  if [ ! -d "$dir" ]; then
    echo "  WARN: $svc — no migrations/ directory"
    continue
  fi

  ups=$(find "$dir" -name "*.up.sql" | sort)
  downs=$(find "$dir" -name "*.down.sql" | sort)
  up_count=$(echo "$ups" | grep -c '.' || true)
  down_count=$(echo "$downs" | grep -c '.' || true)

  if [ "$up_count" -ne "$down_count" ]; then
    echo "  FAIL: $svc — $up_count up files, $down_count down files (must match)"
    FAIL=$((FAIL+1))
  else
    echo "  OK:   $svc — $up_count migration pair(s)"
    PASS=$((PASS+1))
  fi
done

echo ""
if [ "$FAIL" -gt 0 ]; then
  echo "Migration validation failed: $FAIL service(s) have mismatched pairs."
  exit 1
else
  echo "Migration validation passed: $PASS service(s) checked."
fi
