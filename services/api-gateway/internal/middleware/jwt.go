package middleware

import (
	"context"
	"net/http"
	"strings"

	gatewayjwt "github.com/pos-stery/pos-stery/services/api-gateway/internal/jwt"
)

// Verifier is the interface the JWT middleware depends on.
// It is satisfied by *jwt.Verifier.
type Verifier interface {
	Verify(tokenStr string) (*gatewayjwt.Claims, error)
}

// BlacklistChecker is the interface the JWT middleware depends on.
// It is satisfied by *jwt.Blacklist.
type BlacklistChecker interface {
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
}

// JWTDeps groups the dependencies of the JWT middleware.
type JWTDeps struct {
	Verifier  Verifier
	Blacklist BlacklistChecker
}

// JWT returns middleware that validates the Bearer token (or pos_token cookie),
// checks the Redis blacklist, and populates the request context with claims.
func JWT(deps JWTDeps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tok := extractToken(r)
			if tok == "" {
				writeJSONError(w, http.StatusUnauthorized, "Unauthenticated.")
				return
			}

			claims, err := deps.Verifier.Verify(tok)
			if err != nil {
				writeJSONError(w, http.StatusUnauthorized, "Unauthenticated.")
				return
			}

			blacklisted, err := deps.Blacklist.IsBlacklisted(r.Context(), claims.JTI)
			if err != nil || blacklisted {
				writeJSONError(w, http.StatusUnauthorized, "Unauthenticated.")
				return
			}

			ctx := r.Context()
			ctx = withRawToken(ctx, tok)
			ctx = withJTI(ctx, claims.JTI)
			ctx = withUserID(ctx, claims.UserID)
			ctx = withTenantID(ctx, claims.TenantID)
			ctx = withStoreID(ctx, claims.StoreID)
			ctx = withRole(ctx, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// extractToken tries Authorization: Bearer <tok>, then falls back to the pos_token cookie.
func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth != "" {
		if after, ok := strings.CutPrefix(auth, "Bearer "); ok && after != "" {
			return after
		}
	}
	if c, err := r.Cookie("pos_token"); err == nil {
		return c.Value
	}
	return ""
}
