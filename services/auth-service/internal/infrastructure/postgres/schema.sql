-- Schema snapshot for sqlc — kept in sync with migrations/000001_create_auth_schema.up.sql.
-- Only DDL that sqlc needs for type inference; policies, indexes and grants are omitted.

CREATE SCHEMA IF NOT EXISTS auth;

CREATE TABLE auth.tenants (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL,
    slug       TEXT        NOT NULL UNIQUE,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE auth.roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT
);

CREATE TABLE auth.stores (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES auth.tenants(id),
    name       TEXT        NOT NULL,
    address    TEXT,
    phone      TEXT,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE auth.users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL REFERENCES auth.tenants(id),
    email         TEXT        NOT NULL,
    password_hash TEXT        NOT NULL,
    name          TEXT        NOT NULL,
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,
    UNIQUE (tenant_id, email)
);

CREATE TABLE auth.user_roles (
    user_id   UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role_id   UUID NOT NULL REFERENCES auth.roles(id),
    store_id  UUID,
    tenant_id UUID NOT NULL REFERENCES auth.tenants(id)
);
