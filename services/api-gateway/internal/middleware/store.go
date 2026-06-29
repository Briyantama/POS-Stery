package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// StoreOwnershipChecker verifies that a store belongs to the given tenant.
// Implemented by *clients.AuthClient.
type StoreOwnershipChecker interface {
	StoreExistsInTenant(ctx context.Context, tenantID, storeID string) (bool, error)
}

// StoreDeps groups the dependencies of the Store middleware.
type StoreDeps struct {
	// Validator confirms that the store_id header value belongs to the authenticated tenant.
	// Required for admin requests; cashier store_id comes from the JWT and is already trusted.
	Validator StoreOwnershipChecker
}

// Store resolves the store_id for each request:
//   - Cashiers: store_id is embedded in the JWT (populated by JWT middleware into ctx).
//   - Admins: store_id comes from the X-Store-ID header, verified against the auth-service.
//
// If role == "cashier" and JWT already has a storeID, the header is ignored entirely.
// Returns 403 if no valid store ID can be resolved or if the store does not belong to the tenant.
func Store(deps StoreDeps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			role := RoleFromCtx(ctx)
			jwtStoreID := StoreIDFromCtx(ctx)

			// Determine resolved store ID.
			storeID := jwtStoreID

			// Admins (or any role without a JWT storeID) may supply the header.
			if storeID == "" {
				storeID = r.Header.Get("X-Store-ID")
			}

			if storeID == "" {
				writeJSONError(w, http.StatusForbidden, "Store context required.")
				return
			}

			if _, err := uuid.Parse(storeID); err != nil {
				writeJSONError(w, http.StatusForbidden, "Invalid store context.")
				return
			}

			// For cashiers, the resolved store must equal the JWT claim.
			// This prevents a cashier from overriding their assigned store via a header.
			if role == "cashier" && jwtStoreID != "" && jwtStoreID != storeID {
				writeJSONError(w, http.StatusForbidden, "Store context mismatch.")
				return
			}

			// For non-cashier roles (admins), the header store_id must belong to the tenant.
			// Cashier store_id comes from the JWT and is already tenant-scoped at login.
			if role != "cashier" && jwtStoreID == "" {
				tenantID := TenantIDFromCtx(ctx)
				ok, err := deps.Validator.StoreExistsInTenant(ctx, tenantID, storeID)
				if err != nil || !ok {
					writeJSONError(w, http.StatusForbidden, "Store does not belong to tenant.")
					return
				}
			}

			ctx = withStoreID(ctx, storeID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
