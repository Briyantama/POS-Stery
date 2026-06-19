#!/usr/bin/env bash
# Wrapper around golang-migrate for per-service migrations.
# Usage:
#   migrate-helper.sh up     <service>          Apply all pending
#   migrate-helper.sh down   <service> [steps]  Roll back N steps (default 1)
#   migrate-helper.sh create <service> <name>   Create up/down pair
#   migrate-helper.sh drop   <service>          Drop all (DESTRUCTIVE)
set -euo pipefail

ACTION="${1:-}"
SERVICE="${2:-}"

if [ -z "$ACTION" ] || [ -z "$SERVICE" ]; then
  echo "Usage: migrate-helper.sh <up|down|create|drop> <service> [name|steps]"
  exit 1
fi

# Load .env if present
if [ -f "$(pwd)/.env" ]; then
  set -a; source "$(pwd)/.env"; set +a
fi

POSTGRES_ADMIN_PASSWORD="${POSTGRES_ADMIN_PASSWORD:-changeme}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-pos_db}"

DB_URL="postgres://pos_admin:${POSTGRES_ADMIN_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"
MIGRATIONS_DIR="services/${SERVICE}/migrations"

if [ ! -d "$MIGRATIONS_DIR" ]; then
  echo "Error: migrations directory not found: $MIGRATIONS_DIR"
  exit 1
fi

case "$ACTION" in
  up)
    migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" up
    ;;
  down)
    STEPS="${3:-1}"
    migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" down "$STEPS"
    ;;
  create)
    NAME="${3:-}"
    [ -n "$NAME" ] || (echo "Usage: migrate-helper.sh create <service> <name>"; exit 1)
    migrate create -ext sql -dir "$MIGRATIONS_DIR" -seq "$NAME"
    ;;
  drop)
    migrate -path "$MIGRATIONS_DIR" -database "$DB_URL" drop -f
    ;;
  *)
    echo "Unknown action: $ACTION"
    exit 1
    ;;
esac
