-- name: CreateRefreshToken :one
INSERT INTO auth.refresh_tokens (
    id, user_id, tenant_id, family_id, token_hash, expires_at, user_agent, ip_address
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, tenant_id, family_id, token_hash, issued_at, expires_at,
          revoked_at, replaced_by, user_agent, ip_address, created_at, updated_at;

-- name: FindRefreshTokenByHash :one
SELECT id, user_id, tenant_id, family_id, token_hash, issued_at, expires_at,
       revoked_at, replaced_by, user_agent, ip_address, created_at, updated_at
FROM auth.refresh_tokens
WHERE token_hash = $1;

-- name: FindRefreshTokenByID :one
SELECT id, user_id, tenant_id, family_id, token_hash, issued_at, expires_at,
       revoked_at, replaced_by, user_agent, ip_address, created_at, updated_at
FROM auth.refresh_tokens
WHERE id = $1;

-- name: RevokeRefreshToken :exec
UPDATE auth.refresh_tokens
SET revoked_at = NOW(), replaced_by = $2, updated_at = NOW()
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeFamilyTokens :exec
UPDATE auth.refresh_tokens
SET revoked_at = NOW(), updated_at = NOW()
WHERE family_id = $1 AND revoked_at IS NULL;

-- name: ListActiveSessionsByUser :many
SELECT id, user_id, tenant_id, family_id, token_hash, issued_at, expires_at,
       revoked_at, replaced_by, user_agent, ip_address, created_at, updated_at
FROM auth.refresh_tokens
WHERE user_id = $1 AND tenant_id = $2
  AND revoked_at IS NULL AND expires_at > NOW()
ORDER BY created_at DESC;

-- name: RevokeAllUserTokens :exec
UPDATE auth.refresh_tokens
SET revoked_at = NOW(), updated_at = NOW()
WHERE user_id = $1 AND tenant_id = $2 AND revoked_at IS NULL;
