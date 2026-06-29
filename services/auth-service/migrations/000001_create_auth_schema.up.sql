-- Auth service schema: tenants, stores, users, roles, permissions
-- Runs as pos_admin (BYPASSRLS). Application code uses pos_app (subject to RLS).

CREATE SCHEMA IF NOT EXISTS auth;

GRANT USAGE ON SCHEMA auth TO pos_app;

-- ── Tenants ───────────────────────────────────────────────────────────────────

CREATE TABLE auth.tenants (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    slug        TEXT        NOT NULL UNIQUE,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

-- Tenants table has no RLS — the superadmin must be able to read all tenants.
-- Access is controlled at the application layer via the pos_superadmin role.

-- ── Stores ────────────────────────────────────────────────────────────────────

CREATE TABLE auth.stores (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL REFERENCES auth.tenants(id),
    name        TEXT        NOT NULL,
    address     TEXT,
    phone       TEXT,
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_stores_tenant_id ON auth.stores(tenant_id) WHERE deleted_at IS NULL;

ALTER TABLE auth.stores ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.stores FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON auth.stores
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

-- ── Roles ─────────────────────────────────────────────────────────────────────

CREATE TABLE auth.roles (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL UNIQUE,  -- admin, cashier, stock_manager
    description TEXT
);

INSERT INTO auth.roles (name, description) VALUES
    ('admin',         'Tenant administrator'),
    ('cashier',       'Cashier (store-scoped)'),
    ('stock_manager', 'Inventory and purchasing');

-- ── Users ─────────────────────────────────────────────────────────────────────

CREATE TABLE auth.users (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES auth.tenants(id),
    email           TEXT        NOT NULL,
    password_hash   TEXT        NOT NULL,
    name            TEXT        NOT NULL,
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE(tenant_id, email)
);

CREATE INDEX idx_users_tenant_id ON auth.users(tenant_id) WHERE deleted_at IS NULL;

ALTER TABLE auth.users ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.users FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON auth.users
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

-- ── User-Roles ────────────────────────────────────────────────────────────────

CREATE TABLE auth.user_roles (
    id         UUID NOT NULL PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES auth.roles(id),
    store_id   UUID,  -- NULL = role applies to all stores (admin), non-NULL = store-scoped (cashier)
    tenant_id  UUID NOT NULL REFERENCES auth.tenants(id)
);

CREATE UNIQUE INDEX ux_user_roles_user_role_tenant_store
    ON auth.user_roles (user_id, role_id, tenant_id, COALESCE(store_id, '00000000-0000-0000-0000-000000000000'));

CREATE INDEX idx_user_roles_user_id   ON auth.user_roles(user_id);
CREATE INDEX idx_user_roles_tenant_id ON auth.user_roles(tenant_id);

ALTER TABLE auth.user_roles ENABLE ROW LEVEL SECURITY;
ALTER TABLE auth.user_roles FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON auth.user_roles
    USING (tenant_id = current_setting('app.tenant_id', TRUE)::uuid)
    WITH CHECK (tenant_id = current_setting('app.tenant_id', TRUE)::uuid);

-- ── Audit Logs ────────────────────────────────────────────────────────────────

CREATE TABLE auth.audit_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL,
    user_id     UUID,
    action      TEXT        NOT NULL,
    entity      TEXT        NOT NULL,
    entity_id   TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Audit logs are append-only. No RLS — written by pos_admin trigger or service.
CREATE INDEX idx_audit_logs_tenant_id  ON auth.audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_created_at ON auth.audit_logs(created_at DESC);

-- ── Grants ────────────────────────────────────────────────────────────────────

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA auth TO pos_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO pos_app;
