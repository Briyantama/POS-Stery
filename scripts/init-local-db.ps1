# Create pos_db and bootstrap pos_admin / pos_app roles on local postgres.
# Run once before `make dev` or `docker compose up`.
# Usage: pwsh scripts/init-local-db.ps1
#        (requires psql.exe in PATH — ships with PostgreSQL)

$ErrorActionPreference = 'Stop'

$Root    = Split-Path $PSScriptRoot -Parent
$EnvFile = Join-Path $Root '.env'

# ── Parse .env ────────────────────────────────────────────────────────────────
function Get-EnvVal([string]$Key, [string]$Default) {
    $line = Get-Content $EnvFile -ErrorAction SilentlyContinue |
            Where-Object { $_ -match "^${Key}=(.*)$" } |
            Select-Object -First 1
    if ($line -match "^${Key}=(.*)$") { return $matches[1].Trim() }
    return $Default
}

$PgHost      = Get-EnvVal 'POSTGRES_HOST'               'localhost'
$PgPort      = Get-EnvVal 'POSTGRES_PORT'               '5432'
$PgDb        = Get-EnvVal 'POSTGRES_DB'                 'pos_db'
$PgSuperPass = Get-EnvVal 'POSTGRES_SUPERUSER_PASSWORD'  'postgres'
$PgAdminPass = Get-EnvVal 'POSTGRES_ADMIN_PASSWORD'     'postgres'
$PgAppPass   = Get-EnvVal 'POSTGRES_APP_PASSWORD'       'pos_db_stery'

$env:PGPASSWORD = $PgSuperPass
$psqlArgs = @('-h', $PgHost, '-p', $PgPort, '-U', 'postgres')

Write-Host ""
Write-Host "POS-Stery — local DB init"
Write-Host "-------------------------"
Write-Host "  host : ${PgHost}:${PgPort}"
Write-Host "  db   : $PgDb"

# ── Create database ───────────────────────────────────────────────────────────
$dbExists = psql @psqlArgs -tAc "SELECT 1 FROM pg_database WHERE datname='$PgDb'" 2>$null
if ($dbExists -eq '1') {
    Write-Host "  [ok] database '$PgDb' already exists"
} else {
    psql @psqlArgs -c "CREATE DATABASE $PgDb" | Out-Null
    Write-Host "  [ok] database '$PgDb' created"
}

# ── Bootstrap roles, extensions, and privileges ───────────────────────────────
$sql = @"
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DO `$`$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pos_admin') THEN
    CREATE ROLE pos_admin LOGIN PASSWORD '$PgAdminPass' CREATEDB BYPASSRLS;
    RAISE NOTICE 'role pos_admin created';
  ELSE
    ALTER ROLE pos_admin PASSWORD '$PgAdminPass';
    RAISE NOTICE 'role pos_admin already exists -- password updated';
  END IF;

  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pos_app') THEN
    CREATE ROLE pos_app LOGIN PASSWORD '$PgAppPass';
    RAISE NOTICE 'role pos_app created';
  ELSE
    ALTER ROLE pos_app PASSWORD '$PgAppPass';
    RAISE NOTICE 'role pos_app already exists -- password updated';
  END IF;
END
`$`$;

GRANT CONNECT ON DATABASE $PgDb TO pos_admin;
GRANT CONNECT ON DATABASE $PgDb TO pos_app;
GRANT CREATE   ON DATABASE $PgDb TO pos_admin;

GRANT USAGE, CREATE ON SCHEMA public TO pos_admin;
GRANT USAGE         ON SCHEMA public TO pos_app;

ALTER DEFAULT PRIVILEGES FOR ROLE pos_admin IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO pos_app;
ALTER DEFAULT PRIVILEGES FOR ROLE pos_admin IN SCHEMA public
  GRANT USAGE, SELECT                  ON SEQUENCES TO pos_app;
"@

$sql | psql @psqlArgs -d $PgDb

Write-Host "  [ok] roles and privileges ready"
Write-Host ""
Write-Host "Next: make migrate-up"
