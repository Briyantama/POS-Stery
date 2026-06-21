CREATE TABLE auth.refresh_tokens (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    tenant_id   UUID        NOT NULL REFERENCES auth.tenants(id),
    family_id   UUID        NOT NULL,
    token_hash  TEXT        NOT NULL UNIQUE,
    issued_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ,
    replaced_by UUID        REFERENCES auth.refresh_tokens(id),
    user_agent  TEXT        NOT NULL DEFAULT '',
    ip_address  TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_tenant ON auth.refresh_tokens(user_id, tenant_id);
CREATE INDEX idx_refresh_tokens_family_id   ON auth.refresh_tokens(family_id);

-- No RLS: refresh token lookup is by cryptographic hash (pre-tenant-identification).
-- Tenant isolation is enforced in application logic after hash lookup.

GRANT SELECT, INSERT, UPDATE ON auth.refresh_tokens TO pos_app;
