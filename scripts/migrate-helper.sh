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

if [[ -z "$ACTION" || -z "$SERVICE" ]]; then
  echo "Usage: migrate-helper.sh <up|down|create|drop> <service> [name|steps]"
  exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Load .env if present
if [[ -f "$REPO_ROOT/.env" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$REPO_ROOT/.env"
  set +a
fi

POSTGRES_ADMIN_PASSWORD="${POSTGRES_ADMIN_PASSWORD:-changeme}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_DB="${POSTGRES_DB:-pos_db}"
MIGRATE_BIN="${MIGRATE_BIN:-migrate}"

# Map service folder -> schema name
# auth-service -> auth
# product-service -> product
# inventory-service -> inventory
# sales-service -> sales
# supplier-service -> supplier
# customer-service -> customer
SCHEMA="${SERVICE%-service}"

MIGRATIONS_DIR="$REPO_ROOT/services/$SERVICE/migrations"

if [[ ! -d "$MIGRATIONS_DIR" ]]; then
  echo "Error: migrations directory not found: $MIGRATIONS_DIR"
  exit 1
fi

# Admin DSN used for schema bootstrap and migrations
BASE_DSN="postgres://pos_admin:${POSTGRES_ADMIN_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"
SCHEMA_DSN="${BASE_DSN}&search_path=${SCHEMA}"

bootstrap_schema() {
  local sql="
    CREATE SCHEMA IF NOT EXISTS \"${SCHEMA}\" AUTHORIZATION pos_admin;
    GRANT USAGE, CREATE ON SCHEMA \"${SCHEMA}\" TO pos_admin;
    GRANT USAGE ON SCHEMA \"${SCHEMA}\" TO pos_app;
  "

  psql "$BASE_DSN" -v ON_ERROR_STOP=1 -c "$sql" >/dev/null
}

case "$ACTION" in
  up)
    bootstrap_schema
    "$MIGRATE_BIN" -path "$MIGRATIONS_DIR" -database "$SCHEMA_DSN" up
    ;;
  down)
    bootstrap_schema
    STEPS="${3:-1}"
    "$MIGRATE_BIN" -path "$MIGRATIONS_DIR" -database "$SCHEMA_DSN" down "$STEPS"
    ;;
  create)
    NAME="${3:-}"
    [[ -n "$NAME" ]] || { echo "Usage: migrate-helper.sh create <service> <name>"; exit 1; }
    mkdir -p "$MIGRATIONS_DIR"
    "$MIGRATE_BIN" create -ext sql -dir "$MIGRATIONS_DIR" -seq "$NAME"
    ;;
  drop)
    "$MIGRATE_BIN" -path "$MIGRATIONS_DIR" -database "$SCHEMA_DSN" drop -f
    ;;
  *)
    echo "Unknown action: $ACTION"
    exit 1
    ;;
esac