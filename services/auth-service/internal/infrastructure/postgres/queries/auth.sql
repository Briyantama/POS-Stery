-- Auth service queries managed by sqlc.
-- Run: cd services/auth-service && sqlc generate

-- name: GetUserByEmail :one
SELECT id, tenant_id, email, password_hash, name, is_active, created_at, updated_at
FROM auth.users
WHERE email = $1 AND tenant_id = $2 AND deleted_at IS NULL;

-- name: GetUserByID :one
SELECT id, tenant_id, email, password_hash, name, is_active, created_at, updated_at
FROM auth.users
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL;

-- name: GetUserRoles :many
SELECT ur.role_id, r.name AS role_name, ur.store_id
FROM auth.user_roles ur
JOIN auth.roles r ON r.id = ur.role_id
WHERE ur.user_id = $1 AND ur.tenant_id = $2;

-- name: GetTenantByID :one
SELECT id, name, slug, is_active, created_at, updated_at
FROM auth.tenants
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetStoreByID :one
SELECT id, tenant_id, name, address, phone, is_active, created_at, updated_at
FROM auth.stores
WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL;

-- name: StoreExistsInTenant :one
SELECT EXISTS(
    SELECT 1 FROM auth.stores
    WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
) AS store_exists;
