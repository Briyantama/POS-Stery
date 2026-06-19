package application

// TokenSigner issues and validates RS256 JWTs.
type TokenSigner interface {
	Issue(claims TokenClaims) (token string, err error)
	Verify(token string) (*TokenClaims, error)
	Blacklist(token string) error
	IsBlacklisted(token string) (bool, error)
}

// BlacklistStore persists revoked tokens (Redis implementation).
type BlacklistStore interface {
	Blacklist(token string) error
	IsBlacklisted(token string) (bool, error)
}

// TokenClaims is the set of claims embedded in every JWT.
type TokenClaims struct {
	UserID   string
	TenantID string
	StoreID  string // empty for admin tokens
	Role     string
	Email    string
}
