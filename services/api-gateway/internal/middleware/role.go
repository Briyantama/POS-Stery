package middleware

import "net/http"

// WithRole returns middleware that enforces role-based access control.
// The request proceeds only if the authenticated role matches one of the
// allowed roles. Returns 403 if the role is not in the allowed list.
func WithRole(allowed ...string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, r := range allowed {
		allowedSet[r] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := RoleFromCtx(r.Context())

			if _, ok := allowedSet[role]; !ok {
				writeJSONError(w, http.StatusForbidden, "Forbidden.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
