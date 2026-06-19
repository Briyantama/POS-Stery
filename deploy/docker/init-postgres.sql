-- Bootstrap roles and extensions for POS-Stery.
-- Runs once via docker-entrypoint-initdb.d when the container is first created.
-- Do not add schema/table DDL here — that belongs in service migrations.

\c pos_db

-- Extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Application roles
-- pos_admin: BYPASSRLS, used only by golang-migrate during make migrate-up
-- pos_app:   subject to RLS, used by all service processes at runtime
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pos_admin') THEN
    CREATE ROLE pos_admin WITH LOGIN PASSWORD 'changeme' CREATEDB BYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'pos_app') THEN
    CREATE ROLE pos_app WITH LOGIN PASSWORD 'changeme';
  END IF;
END
$$;

GRANT CONNECT ON DATABASE pos_db TO pos_admin, pos_app;

-- Service schemas are created in each service's 001 migration.
-- When a service migration runs as pos_admin, it must also:
--   GRANT USAGE ON SCHEMA <name> TO pos_app;
--   GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA <name> TO pos_app;
--   ALTER DEFAULT PRIVILEGES IN SCHEMA <name>
--     GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pos_app;
