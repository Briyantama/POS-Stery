-- Bootstrap roles and extensions for POS-Stery.
-- Executed once by docker-entrypoint-initdb.d
-- No application tables belong here. Only roles, extensions, and base privileges.

\connect pos_db

---

-- Extensions

---

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

---

-- Roles

---

DO $$
BEGIN
IF NOT EXISTS (
SELECT 1
FROM pg_roles
WHERE rolname = 'pos_admin'
) THEN
CREATE ROLE pos_admin
LOGIN
PASSWORD 'changeme'
CREATEDB
BYPASSRLS;
END IF;

IF NOT EXISTS (
SELECT 1
FROM pg_roles
WHERE rolname = 'pos_app'
) THEN
CREATE ROLE pos_app
LOGIN
PASSWORD 'changeme';
END IF;
END
$$;

---

-- Database privileges

---

GRANT CONNECT ON DATABASE pos_db TO pos_admin;
GRANT CONNECT ON DATABASE pos_db TO pos_app;

-- Required for service schema creation:
-- auth, product, inventory, sales, supplier, customer, etc.
GRANT CREATE ON DATABASE pos_db TO pos_admin;

---

-- Public schema

---

-- Migration metadata may still live in public.
GRANT USAGE, CREATE ON SCHEMA public TO pos_admin;

-- Runtime role should not create objects.
GRANT USAGE ON SCHEMA public TO pos_app;

---

-- Default privileges

---

ALTER DEFAULT PRIVILEGES FOR ROLE pos_admin IN SCHEMA public
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pos_app;

ALTER DEFAULT PRIVILEGES FOR ROLE pos_admin IN SCHEMA public
GRANT USAGE, SELECT ON SEQUENCES TO pos_app;

---

-- Metadata

---

COMMENT ON ROLE pos_admin IS
'Migration role. Used only by golang-migrate, bootstrap, and schema management.';

COMMENT ON ROLE pos_app IS
'Application runtime role. Subject to RLS and least-privilege access.';
