// Package auth is the POS identity service: login, refresh, logout, sessions and store validation.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Briyantama/POS-Stery/services/auth/application/commands"
	"github.com/Briyantama/POS-Stery/services/auth/domain"
	"github.com/Briyantama/POS-Stery/services/auth/infrastructure/jwt"
	"github.com/Briyantama/POS-Stery/services/auth/infrastructure/postgres"
	"github.com/Briyantama/POS-Stery/services/auth/infrastructure/redisstore"
	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/httpx"
	"github.com/google/uuid"
)

// Build wires the repositories, command handlers and HTTP handler.
func Build(d Deps) (*HTTPHandler, error) {
	signer, err := jwt.NewRSASigner(d.Config.JWTPrivateKeyPath, d.Config.JWTPublicKeyPath, redisstore.NewTokenBlacklist(d.Redis))
	if err != nil {
		return nil, fmt.Errorf("jwt signer: %w", err)
	}
	users := postgres.NewUserRepository(d.Pool)
	stores := postgres.NewStoreRepository(d.Pool)
	refreshTokens := postgres.NewRefreshTokenRepository(d.Pool)

	return NewHTTPHandler(
		commands.NewLoginHandler(users, stores, signer, refreshTokens),
		commands.NewRefreshHandler(refreshTokens, users, signer, slog.Default()),
		commands.NewLogoutHandler(refreshTokens, signer),
		commands.NewLogoutAllHandler(refreshTokens, signer),
		commands.NewListSessionsHandler(refreshTokens, signer),
		commands.NewListStoresHandler(stores),
		d.Verifier,
		d.LoginLimit,
	), nil
}

// Service answers cross-service questions in-process (it replaces the old ValidateStore gRPC).
type Service struct {
	stores domain.StoreRepository
}

func NewService(stores domain.StoreRepository) *Service { return &Service{stores: stores} }

// ValidateStore returns nil when the store belongs to the tenant, domain.ErrPermissionDenied when it
// does not, and domain.ErrInvalidArgument when an id is not a UUID.
func (s *Service) ValidateStore(ctx context.Context, tenantID, storeID string) error {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("%w: tenant id", domain.ErrInvalidArgument)
	}
	sid, err := uuid.Parse(storeID)
	if err != nil {
		return fmt.Errorf("%w: store id", domain.ErrInvalidArgument)
	}
	ok, err := s.stores.ExistsInTenant(ctx, tid, sid)
	if err != nil {
		return fmt.Errorf("check store ownership: %w", err)
	}
	if !ok {
		return domain.ErrPermissionDenied
	}
	return nil
}

// RequireStore resolves the store of an authenticated request and puts it in the claims.
// Cashier tokens carry their store; otherwise the X-Store-ID header is used and must belong to
// the caller's tenant. It must run after authn.Authenticate.
func RequireStore(d Deps) func(http.Handler) http.Handler {
	svc := NewService(postgres.NewStoreRepository(d.Pool))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := authn.FromContext(r.Context())
			if !ok {
				httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			if claims.StoreID == "" {
				storeID := r.Header.Get("X-Store-ID")
				if storeID == "" {
					httpx.WriteError(w, http.StatusBadRequest, "store id required")
					return
				}
				switch err := svc.ValidateStore(r.Context(), claims.TenantID, storeID); {
				case err == nil:
					claims.StoreID = storeID
				case errors.Is(err, domain.ErrPermissionDenied):
					httpx.WriteError(w, http.StatusForbidden, "forbidden")
					return
				case errors.Is(err, domain.ErrInvalidArgument):
					httpx.WriteError(w, http.StatusBadRequest, "invalid store id")
					return
				default:
					slog.ErrorContext(r.Context(), "validate store", "error", err, "tenant_id", claims.TenantID)
					httpx.WriteError(w, http.StatusInternalServerError, "internal error")
					return
				}
			}
			next.ServeHTTP(w, r.WithContext(authn.WithClaims(r.Context(), claims)))
		})
	}
}
