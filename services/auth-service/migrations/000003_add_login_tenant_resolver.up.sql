-- Public login tenant resolution.
--
-- The gateway's public POST /api/login knows only email/password (+ store_id for
-- cashiers); it cannot supply tenant_id pre-auth. auth.users is RLS-scoped by
-- tenant, so resolving the tenant requires a lookup that runs before any tenant
-- context exists — mirroring how refresh tokens are looked up by hash pre-tenant.
--
-- This SECURITY DEFINER function runs as its owner (pos_admin, BYPASSRLS), so it
-- can read across tenants. It returns a tenant ONLY when exactly one active,
-- non-deleted user owns the email (deterministic; fails closed on ambiguity).
-- It exposes nothing but the tenant_id, preserving isolation.

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

    -- Unknown or ambiguous email → caller treats as invalid credentials.
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

REVOKE ALL ON FUNCTION auth.resolve_login_tenant(TEXT) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION auth.resolve_login_tenant(TEXT) TO pos_app;
