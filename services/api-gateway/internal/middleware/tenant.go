package middleware

import (
	"net/http"

	"github.com/google/uuid"
)

// Tenant validates that a non-empty, well-formed UUID tenant ID is present in
// the request context (populated by the JWT middleware).
// Returns 403 if missing or malformed.
func Tenant() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := TenantIDFromCtx(r.Context())

			if tenantID == "" {
				writeJSONError(w, http.StatusForbidden, "Tenant context required.")
				return
			}

			if _, err := uuid.Parse(tenantID); err != nil {
				writeJSONError(w, http.StatusForbidden, "Invalid tenant context.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
