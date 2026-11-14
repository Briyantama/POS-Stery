package application

import (
	"context"
	"time"
)

// TokenSigner issues and verifies RS256 JWTs and revokes access tokens.
type TokenSigner interface {
	Issue(claims TokenClaims) (token string, expiresAt time.Time, err error)
	Verify(token string) (*TokenClaims, error)
	// Blacklist revokes the jti until the token would have expired anyway.
	Blacklist(ctx context.Context, jti string, expiresAt time.Time) error
}

// BlacklistStore persists revoked JTIs in Redis.
type BlacklistStore interface {
	Blacklist(ctx context.Context, jti string, expiresAt time.Time) error
}

// TokenClaims is the set of claims embedded in every JWT.
type TokenClaims struct {
	JTI      string // JWT ID — used for access token blacklisting
	UserID   string
	TenantID string
	StoreID  string // empty for admin tokens
	Role     string
	Email    string
	// ExpiresAt is set by Verify; Issue ignores it.
	ExpiresAt time.Time
}
