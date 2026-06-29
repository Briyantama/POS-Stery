package handlers

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	authv1 "github.com/pos-stery/pos-stery/gen/go/pos/auth/v1"
	"github.com/pos-stery/pos-stery/services/api-gateway/internal/middleware"
)

// AuthServicer defines the auth-service operations used by AuthHandlers.
type AuthServicer interface {
	Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error)
	Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error)
}

// AuthHandlers holds the auth-related HTTP handlers.
type AuthHandlers struct {
	auth AuthServicer
}

// NewAuthHandlers constructs AuthHandlers with the given service client.
func NewAuthHandlers(auth AuthServicer) *AuthHandlers {
	return &AuthHandlers{auth: auth}
}

// loginBody is the JSON payload expected on POST /api/login.
type loginBody struct {
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
	StoreID  string `json:"store_id"`
}

// Login handles POST /api/login.
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		validationError(w, "body", "invalid JSON")
		return
	}

	if body.TenantID == "" {
		validationError(w, "tenant_id", "tenant_id is required")
		return
	}
	if body.Email == "" {
		validationError(w, "email", "email is required")
		return
	}
	if body.Password == "" {
		validationError(w, "password", "password is required")
		return
	}

	resp, err := h.auth.Login(r.Context(), &authv1.LoginRequest{
		TenantId:  body.TenantID,
		Email:     body.Email,
		Password:  body.Password,
		StoreId:   body.StoreID,
		UserAgent: r.UserAgent(),
		IpAddress: remoteIP(r),
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, resp)
}

// logoutBody optionally carries a refresh_token to revoke alongside the access token.
type logoutBody struct {
	RefreshToken string `json:"refresh_token"`
}

// remoteIP extracts the client IP from r.RemoteAddr (strips port).
// When the gateway is behind a trusted proxy, prefer X-Forwarded-For.
func remoteIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Logout handles POST /api/logout.
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	accessToken := middleware.RawTokenFromCtx(r.Context())

	var body logoutBody
	// Body is optional — ignore decode errors.
	_ = json.NewDecoder(r.Body).Decode(&body)

	resp, err := h.auth.Logout(r.Context(), &authv1.LogoutRequest{
		AccessToken:  accessToken,
		RefreshToken: body.RefreshToken,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, resp)
}
