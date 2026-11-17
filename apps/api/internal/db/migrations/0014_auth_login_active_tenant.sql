-- +goose Up
-- Login resolves the tenant by lower(email); index that lookup, and refuse tenants that are deactivated or deleted.
CREATE INDEX idx_users_email_lower ON auth.users (lower(email)) WHERE deleted_at IS NULL;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION auth.resolve_login_tenant(p_email TEXT)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = auth, pg_temp
AS $$
DECLARE
    v_tenants UUID[];
BEGIN
    SELECT array_agg(u.tenant_id) INTO v_tenants
    FROM auth.users u
    JOIN auth.tenants t ON t.id = u.tenant_id AND t.is_active AND t.deleted_at IS NULL
    WHERE lower(u.email) = lower(p_email)
      AND u.deleted_at IS NULL
      AND u.is_active = TRUE;

    -- Unknown or ambiguous email: caller treats as invalid credentials.
    IF cardinality(v_tenants) IS DISTINCT FROM 1 THEN
        RETURN NULL;
    END IF;
    RETURN v_tenants[1];
END;
$$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION auth.resolve_login_tenant(p_email TEXT)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = auth, pg_temp
AS $$
DECLARE
    v_count  INTEGER;
    v_tenant UUID;
BEGIN
    SELECT count(*) INTO v_count
    FROM auth.users
    WHERE lower(email) = lower(p_email)
      AND deleted_at IS NULL
      AND is_active = TRUE;

    IF v_count <> 1 THEN
        RETURN NULL;
    END IF;

    SELECT tenant_id INTO v_tenant
    FROM auth.users
    WHERE lower(email) = lower(p_email)
      AND deleted_at IS NULL
      AND is_active = TRUE;

    RETURN v_tenant;
END;
$$;
-- +goose StatementEnd

DROP INDEX IF EXISTS auth.idx_users_email_lower;
