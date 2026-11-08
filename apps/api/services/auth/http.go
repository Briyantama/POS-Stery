package auth

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/Briyantama/POS-Stery/services/auth/application"
	"github.com/Briyantama/POS-Stery/services/auth/application/commands"
	"github.com/Briyantama/POS-Stery/services/auth/domain"
	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/httpx"
	"github.com/google/uuid"
)

const maxBody = 1 << 20

type loginer interface {
	Handle(ctx context.Context, cmd commands.LoginCommand) (*commands.LoginResult, error)
}

type refresher interface {
	Handle(ctx context.Context, cmd commands.RefreshCommand) (*commands.RefreshResult, error)
}

type logouter interface {
	Handle(ctx context.Context, cmd commands.LogoutCommand) error
}

type logoutAller interface {
	Handle(ctx context.Context, cmd commands.LogoutAllCommand) error
}

type sessionLister interface {
	Handle(ctx context.Context, q commands.ListSessionsQuery) (*commands.ListSessionsResult, error)
}

type storeLister interface {
	ListByTenant(ctx context.Context, tenantID domain.TenantID) ([]*domain.Store, error)
}

// HTTPHandler serves the auth routes.
type HTTPHandler struct {
	login      loginer
	refresh    refresher
	logout     logouter
	logoutAll  logoutAller
	sessions   sessionLister
	stores     storeLister
	verifier   authn.Verifier
	loginLimit func(http.Handler) http.Handler
}

func NewHTTPHandler(
	login loginer,
	refresh refresher,
	logout logouter,
	logoutAll logoutAller,
	sessions sessionLister,
	stores storeLister,
	verifier authn.Verifier,
	loginLimit func(http.Handler) http.Handler,
) *HTTPHandler {
	return &HTTPHandler{login: login, refresh: refresh, logout: logout, logoutAll: logoutAll, sessions: sessions, stores: stores, verifier: verifier, loginLimit: loginLimit}
}

// Register mounts the routes. Login and refresh are public (the credentials are in the body).
func (h *HTTPHandler) Register(mux *http.ServeMux) {
	authenticated := func(fn http.HandlerFunc) http.Handler {
		return authn.Authenticate(h.verifier)(authn.WithRole("admin", "cashier")(fn))
	}

	var login http.Handler = http.HandlerFunc(h.handleLogin)
	if h.loginLimit != nil {
		login = h.loginLimit(login)
	}
	mux.Handle("POST /api/v1/login", login)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.handleRefresh)
	mux.Handle("POST /api/v1/logout", authenticated(h.handleLogout))
	mux.Handle("POST /api/v1/auth/logout-all", authenticated(h.handleLogoutAll))
	mux.Handle("GET /api/v1/auth/sessions", authenticated(h.handleSessions))
	mux.Handle("GET /api/v1/stores", authn.Authenticate(h.verifier)(authn.WithRole("admin")(http.HandlerFunc(h.handleStores))))
}

type claimsResponse struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	StoreID  string `json:"store_id"`
	Role     string `json:"role"`
	Email    string `json:"email"`
}

type tokenResponse struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	TokenType    string         `json:"token_type"`
	ExpiresAt    time.Time      `json:"expires_at"`
	Claims       claimsResponse `json:"claims"`
}

func newTokenResponse(access, refresh string, expiresAt time.Time, c application.TokenClaims) tokenResponse {
	return tokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresAt:    expiresAt,
		Claims:       claimsResponse{UserID: c.UserID, TenantID: c.TenantID, StoreID: c.StoreID, Role: c.Role, Email: c.Email},
	}
}

func (h *HTTPHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		StoreID  string `json:"store_id"`
	}
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	if body.Email == "" || body.Password == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "email and password are required")
		return
	}
	res, err := h.login.Handle(r.Context(), commands.LoginCommand{
		Email:     body.Email,
		Password:  body.Password,
		StoreID:   body.StoreID,
		UserAgent: r.UserAgent(),
		IPAddress: clientIP(r),
	})
	if err != nil {
		writeAuthError(w, r, err, "invalid credentials")
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, newTokenResponse(res.AccessToken, res.RefreshToken, res.ExpiresAt, res.Claims))
}

func (h *HTTPHandler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	if body.RefreshToken == "" {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "refresh token is required")
		return
	}
	res, err := h.refresh.Handle(r.Context(), commands.RefreshCommand{
		RefreshToken: body.RefreshToken,
		UserAgent:    r.UserAgent(),
		IPAddress:    clientIP(r),
	})
	if err != nil {
		writeAuthError(w, r, err, "invalid refresh token")
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, newTokenResponse(res.AccessToken, res.RefreshToken, res.ExpiresAt, res.Claims))
}

// handleLogout revokes the presented access token and, when given, a refresh token.
// The body is optional.
func (h *HTTPHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if r.ContentLength != 0 && !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	if err := h.logout.Handle(r.Context(), commands.LogoutCommand{AccessToken: authn.Bearer(r), RefreshToken: body.RefreshToken}); err != nil {
		writeAuthError(w, r, err, "unauthorized")
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, map[string]bool{"logged_out": true})
}

func (h *HTTPHandler) handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	if err := h.logoutAll.Handle(r.Context(), commands.LogoutAllCommand{AccessToken: authn.Bearer(r)}); err != nil {
		writeAuthError(w, r, err, "unauthorized")
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, map[string]bool{"logged_out": true})
}

func (h *HTTPHandler) handleSessions(w http.ResponseWriter, r *http.Request) {
	res, err := h.sessions.Handle(r.Context(), commands.ListSessionsQuery{AccessToken: authn.Bearer(r)})
	if err != nil {
		writeAuthError(w, r, err, "unauthorized")
		return
	}
	type session struct {
		SessionID string    `json:"session_id"`
		FamilyID  string    `json:"family_id"`
		IssuedAt  time.Time `json:"issued_at"`
		ExpiresAt time.Time `json:"expires_at"`
		UserAgent string    `json:"user_agent"`
		IPAddress string    `json:"ip_address"`
	}
	out := make([]session, len(res.Sessions))
	for i, s := range res.Sessions {
		out[i] = session{s.ID.String(), s.FamilyID.String(), s.IssuedAt, s.ExpiresAt, s.UserAgent, s.IPAddress}
	}
	httpx.WriteSuccess(w, http.StatusOK, out)
}

// handleStores lists the stores of the token tenant (admin only).
func (h *HTTPHandler) handleStores(w http.ResponseWriter, r *http.Request) {
	c, ok := authn.FromContext(r.Context())
	tenant, err := uuid.Parse(c.TenantID)
	if !ok || err != nil {
		httpx.WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	stores, err := h.stores.ListByTenant(r.Context(), tenant)
	if err != nil {
		writeAuthError(w, r, err, "unauthorized")
		return
	}
	type item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	items := make([]item, len(stores))
	for i, s := range stores {
		items[i] = item{s.ID.String(), s.Name}
	}
	httpx.WriteSuccess(w, http.StatusOK, map[string]any{"items": items})
}

// writeAuthError maps domain errors to a status and a fixed message; unauthenticatedMsg is the text
// for 401 so that login never says which credential was wrong.
func writeAuthError(w http.ResponseWriter, r *http.Request, err error, unauthenticatedMsg string) {
	switch {
	case errors.Is(err, domain.ErrStoreRequired):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "store_id is required for cashier login")
	case errors.Is(err, domain.ErrInvalidArgument):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid request")
	case errors.Is(err, domain.ErrUnauthenticated), errors.Is(err, domain.ErrNotFound):
		httpx.WriteError(w, http.StatusUnauthorized, unauthenticatedMsg)
	case errors.Is(err, domain.ErrPermissionDenied):
		httpx.WriteError(w, http.StatusForbidden, "forbidden")
	default:
		attrs := []any{"error", err, "request_id", httpx.RequestIDFrom(r.Context())}
		if c, ok := authn.FromContext(r.Context()); ok {
			attrs = append(attrs, "tenant_id", c.TenantID)
		}
		slog.ErrorContext(r.Context(), "auth request failed", attrs...)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}

// clientIP is r.RemoteAddr without the port; httpx.TrustedProxies has already resolved proxies.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
