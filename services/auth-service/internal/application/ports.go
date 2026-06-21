package application

import (
	"context"
	"time"
)

// TokenSigner issues and validates RS256 JWTs.
type TokenSigner interface {
	Issue(claims TokenClaims) (token string, expiresAt time.Time, err error)
	Verify(token string) (*TokenClaims, error)
	Blacklist(ctx context.Context, jti string) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

// BlacklistStore persists revoked JTIs in Redis.
type BlacklistStore interface {
	Blacklist(ctx context.Context, jti string) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

// TokenClaims is the set of claims embedded in every JWT.
type TokenClaims struct {
	JTI      string // JWT ID — used for access token blacklisting
	UserID   string
	TenantID string
	StoreID  string // empty for admin tokens
	Role     string
	Email    string
}
