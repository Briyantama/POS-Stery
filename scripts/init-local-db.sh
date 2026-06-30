#!/usr/bin/env bash
# Create pos_db and bootstrap pos_admin / pos_app roles on local postgres.
# Run once before `make dev` or `docker compose up`.
# Usage: bash scripts/init-local-db.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
ENV_FILE="$ROOT/.env"

# ── Parse .env ────────────────────────────────────────────────────────────────
env_val() {
  local key="$1" default="$2"
  local val
  val=$(grep -E "^${key}=" "$ENV_FILE" 2>/dev/null | head -1 | cut -d= -f2-)
  echo "${val:-$default}"
}

PG_HOST=$(env_val POSTGRES_HOST localhost)
PG_PORT=$(env_val POSTGRES_PORT 5432)
PG_DB=$(env_val POSTGRES_DB pos_db)
PG_SUPER_PASS=$(env_val POSTGRES_SUPERUSER_PASSWORD postgres)
PG_ADMIN_PASS=$(env_val POSTGRES_ADMIN_PASSWORD postgres)
PG_APP_PASS=$(env_val POSTGRES_APP_PASSWORD pos_db_stery)

echo ""
echo "POS-Stery — local DB init"
echo "─────────────────────────"
echo "  host : $PG_HOST:$PG_PORT"
echo "  db   : $PG_DB"

# ── Verify superuser connection; prompt if .env password is wrong ─────────────
export PGPASSWORD="$PG_SUPER_PASS"
PSQL="psql -h $PG_HOST -p $PG_PORT -U postgres"

if ! $PSQL -tAc "SELECT 1" &>/dev/null; then
  echo ""
  echo "  ! Could not connect with POSTGRES_SUPERUSER_PASSWORD from .env."
  read -r -s -p "  Enter postgres superuser password: " PG_SUPER_PASS
  echo ""
  export PGPASSWORD="$PG_SUPER_PASS"
  if ! $PSQL -tAc "SELECT 1" &>/dev/null; then
    echo "  ✗ Authentication failed. Check your postgres superuser password."
    exit 1
  fi
fi

# ── Create database ───────────────────────────────────────────────────────────
DB_EXISTS=$($PSQL -tAc "SELECT 1 FROM pg_database WHERE datname='$PG_DB'" 2>/dev/null || true)
if [ "$DB_EXISTS" = "1" ]; then
  echo "  ✓ database '$PG_DB' already exists"
else
  $PSQL -c "CREATE DATABASE $PG_DB"
  echo "  ✓ database '$PG_DB' created"
fi

# ── Bootstrap roles, extensions, and privileges ───────────────────────────────
$PSQL -d "$PG_DB" <<SQL
-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Roles (idempotent)
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pos_admin') THEN
    CREATE ROLE pos_admin LOGIN PASSWORD '$PG_ADMIN_PASS' CREATEDB BYPASSRLS;
    RAISE NOTICE 'role pos_admin created';
  ELSE
    ALTER ROLE pos_admin PASSWORD '$PG_ADMIN_PASS';
    RAISE NOTICE 'role pos_admin already exists — password updated';
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pos_app') THEN
    CREATE ROLE pos_app LOGIN PASSWORD '$PG_APP_PASS';
    RAISE NOTICE 'role pos_app created';
  ELSE
    ALTER ROLE pos_app PASSWORD '$PG_APP_PASS';
    RAISE NOTICE 'role pos_app already exists — password updated';
  END IF;
END
\$\$;

-- Database privileges
GRANT CONNECT ON DATABASE $PG_DB TO pos_admin;
GRANT CONNECT ON DATABASE $PG_DB TO pos_app;
GRANT CREATE   ON DATABASE $PG_DB TO pos_admin;

-- Schema privileges
GRANT USAGE, CREATE ON SCHEMA public TO pos_admin;
GRANT USAGE         ON SCHEMA public TO pos_app;

-- Default privileges (future tables/sequences created by pos_admin)
ALTER DEFAULT PRIVILEGES FOR ROLE pos_admin IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO pos_app;
ALTER DEFAULT PRIVILEGES FOR ROLE pos_admin IN SCHEMA public
  GRANT USAGE, SELECT                  ON SEQUENCES TO pos_app;
SQL

echo "  ✓ roles and privileges ready"
echo ""
echo "Next: make migrate-up"
