#!/usr/bin/env bash
# wait-for-services.sh — polls all 6 gRPC services until they respond or timeout

set -euo pipefail

TIMEOUT=${WAIT_TIMEOUT:-120}
INTERVAL=3

services=(
  "auth-service:8081"
  "product-service:8082"
  "inventory-service:8083"
  "sales-service:8084"
  "supplier-service:8085"
  "customer-service:8086"
)

wait_for() {
  local name=$1
  local host=${2%%:*}
  local port=${2##*:}
  local elapsed=0

  echo "Waiting for ${name} at ${host}:${port}..."
  until nc -z "$host" "$port" 2>/dev/null; do
    if (( elapsed >= TIMEOUT )); then
      echo "ERROR: ${name} did not become ready within ${TIMEOUT}s" >&2
      exit 1
    fi
    sleep "$INTERVAL"
    (( elapsed += INTERVAL ))
  done
  echo "${name} is ready."
}

for svc in "${services[@]}"; do
  name="${svc%%:*}"
  addr="${svc#*:}"
  wait_for "$name" "localhost:${addr}"
done

echo "All services ready."
