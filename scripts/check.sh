#!/usr/bin/env bash
# Local validation: Go (format, vet, lint, tests with coverage) and web type checks and unit tests.
# CHECK_STAGES picks stages (default "go web"); CI runs them as separate jobs, e.g. CHECK_STAGES=web.
# RACE=1 adds the race detector to go test (needs cgo and a C compiler). Run from the repo root.
set -euo pipefail

cd "$(dirname "$0")/.."

STAGES="${CHECK_STAGES:-go web}"
has() { [[ " $STAGES " == *" $1 "* ]]; }

if has go; then
  if [ -z "${TEST_DATABASE_URL:-}" ]; then
    echo "TEST_DATABASE_URL is not set, e.g. postgres://pos_admin:<password>@127.0.0.1:5432/postgres?sslmode=disable" >&2
    exit 1
  fi

  # Redis-protocol tests run against Dragonfly from deploy/docker-compose.yml; default them from .env.
  if [ -f .env ]; then
    REDIS_ADDR="$(sed -n 's/\r$//; s/^REDIS_ADDR=//p' .env)"
    REDIS_PASSWORD="$(sed -n 's/\r$//; s/^REDIS_PASSWORD=//p' .env)"
  fi
  export TEST_REDIS_ADDR="${TEST_REDIS_ADDR:-${REDIS_ADDR:-}}"
  export TEST_REDIS_PASSWORD="${TEST_REDIS_PASSWORD:-${REDIS_PASSWORD:-}}"
  if [ -z "$TEST_REDIS_ADDR" ]; then
    echo "TEST_REDIS_ADDR (or REDIS_ADDR in .env) is not set" >&2
    exit 1
  fi

  echo "==> Go (apps/api): format"
  unformatted="$(cd apps/api && gofmt -l .)"
  if [ -n "$unformatted" ]; then
    echo "gofmt needed on:" >&2
    echo "$unformatted" >&2
    exit 1
  fi

  echo "==> Go (apps/api): vet"
  (cd apps/api && go vet ./...)

  echo "==> Go (apps/api): lint"
  (cd apps/api && golangci-lint run --timeout 15m)

  echo "==> Go (apps/api): test"
  race_flag=""
  [ "${RACE:-0}" = "1" ] && race_flag="-race"
  (cd apps/api && go test $race_flag ./... -cover)
fi

if has web; then
  for app in apps/admin apps/cashier; do
    echo "==> Web ($app): check"
    (cd "$app" && npm run check)
    echo "==> Web ($app): lint"
    (cd "$app" && npx oxlint src && npx eslint .)
    if grep -q '"test:unit"' "$app/package.json"; then
      echo "==> Web ($app): test:unit"
      (cd "$app" && npm run test:unit)
    fi
  done
fi

echo "All checks passed (stages: $STAGES)."
