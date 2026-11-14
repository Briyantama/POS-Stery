package auth_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Briyantama/POS-Stery/services/auth"
	"github.com/Briyantama/POS-Stery/services/auth/application"
	"github.com/Briyantama/POS-Stery/services/auth/application/commands"
	"github.com/Briyantama/POS-Stery/services/auth/domain"
	"github.com/Briyantama/stery-kit/go/authn"
	"uuid"
)

type stubVerifier struct{}

func (stubVerifier) Verify(_ context.Context, token string) (authn.Claims, error) {
	switch token {
	case "admin":
		return authn.Claims{UserID: "u1", TenantID: "t1", Roles: []string{"admin"}}, nil
	case "stock":
		return authn.Claims{UserID: "u2", TenantID: "t1", Roles: []string{"stock_manager"}}, nil
	}
	return authn.Claims{}, errors.New("bad token")
}

type stub struct {
	loginErr error
	calls    int
}

func (s *stub) claims() application.TokenClaims {
	return application.TokenClaims{UserID: "u1", TenantID: "t1", Role: "admin", Email: "a@x.io"}
}

func (s *stub) Handle(_ context.Context, _ commands.LoginCommand) (*commands.LoginResult, error) {
	s.calls++
	if s.loginErr != nil {
		return nil, s.loginErr
	}
	return &commands.LoginResult{AccessToken: "at", RefreshToken: "rt", ExpiresAt: time.Unix(1, 0).UTC(), Claims: s.claims()}, nil
}

type refreshStub struct{}

func (refreshStub) Handle(context.Context, commands.RefreshCommand) (*commands.RefreshResult, error) {
	return &commands.RefreshResult{AccessToken: "at2", RefreshToken: "rt2"}, nil
}

type logoutStub struct{}

func (logoutStub) Handle(context.Context, commands.LogoutCommand) error { return nil }

type logoutAllStub struct{}

func (logoutAllStub) Handle(context.Context, commands.LogoutAllCommand) error { return nil }

type sessionsStub struct{}

func (sessionsStub) Handle(context.Context, commands.ListSessionsQuery) (*commands.ListSessionsResult, error) {
	return &commands.ListSessionsResult{Sessions: []commands.SessionInfo{{ID: uuid.NewV7(), FamilyID: uuid.NewV7()}}}, nil
}

type storesStub struct{}

func (storesStub) ListByTenant(context.Context, domain.TenantID) ([]*domain.Store, error) {
	return []*domain.Store{{ID: uuid.MustParse("11111111-1111-1111-1111-111111111111"), Name: "Main"}}, nil
}

func newMux(s *stub) *http.ServeMux {
	mux := http.NewServeMux()
	auth.NewHTTPHandler(s, refreshStub{}, logoutStub{}, logoutAllStub{}, sessionsStub{}, storesStub{}, stubVerifier{}, nil).Register(mux)
	return mux
}

func TestStoresAdminOnly(t *testing.T) {
	mux := newMux(&stub{})
	for _, tc := range []struct {
		token string
		want  int
	}{{"", 401}, {"stock", 403}} {
		if got := do(mux, "GET", "/api/v1/stores", tc.token, "").Code; got != tc.want {
			t.Errorf("token %q: status = %d, want %d", tc.token, got, tc.want)
		}
	}
}

func TestStoresList(t *testing.T) {
	mux := http.NewServeMux()
	auth.NewHTTPHandler(&stub{}, refreshStub{}, logoutStub{}, logoutAllStub{}, sessionsStub{}, storesStub{}, tenantVerifier{}, nil).Register(mux)
	rec := do(mux, "GET", "/api/v1/stores", "admin", "")
	want := `"data":{"items":[{"id":"11111111-1111-1111-1111-111111111111","name":"Main"}]}`
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
}

type tenantVerifier struct{}

func (tenantVerifier) Verify(context.Context, string) (authn.Claims, error) {
	return authn.Claims{UserID: "u1", TenantID: uuid.NewV7().String(), Roles: []string{"admin"}}, nil
}

func TestLoginStoreRequiredMessage(t *testing.T) {
	rec := do(newMux(&stub{loginErr: domain.ErrStoreRequired}), "POST", "/api/v1/login", "", `{"email":"a","password":"p"}`)
	if rec.Code != 422 || !strings.Contains(rec.Body.String(), "store_id is required for cashier login") {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
}

func do(mux *http.ServeMux, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestLoginEnvelope(t *testing.T) {
	rec := do(newMux(&stub{}), "POST", "/api/v1/login", "", `{"email":"a@x.io","password":"p"}`)
	if rec.Code != 200 {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var env struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
		Error   *string        `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	claims, _ := env.Data["claims"].(map[string]any)
	if !env.Success || env.Error != nil || env.Data["access_token"] != "at" || env.Data["refresh_token"] != "rt" || claims["role"] != "admin" {
		t.Fatalf("unexpected envelope: %s", rec.Body)
	}
}

func TestLoginRejectsBodyTenantID(t *testing.T) {
	s := &stub{}
	rec := do(newMux(s), "POST", "/api/v1/login", "", `{"tenant_id":"other","email":"a@x.io","password":"p"}`)
	if rec.Code != 400 || s.calls != 0 {
		t.Fatalf("status = %d calls = %d, want 400 and no login call", rec.Code, s.calls)
	}
}

func TestAuthorization(t *testing.T) {
	mux := newMux(&stub{})
	for _, tc := range []struct {
		name, method, path, token string
		want                      int
	}{
		{"logout without token", "POST", "/api/v1/logout", "", 401},
		{"logout invalid token", "POST", "/api/v1/logout", "nope", 401},
		{"logout wrong role", "POST", "/api/v1/logout", "stock", 403},
		{"logout ok", "POST", "/api/v1/logout", "admin", 200},
		{"logout-all wrong role", "POST", "/api/v1/auth/logout-all", "stock", 403},
		{"logout-all ok", "POST", "/api/v1/auth/logout-all", "admin", 200},
		{"sessions without token", "GET", "/api/v1/auth/sessions", "", 401},
		{"sessions ok", "GET", "/api/v1/auth/sessions", "admin", 200},
		{"refresh is public", "POST", "/api/v1/auth/refresh", "", 415},
	} {
		if got := do(mux, tc.method, tc.path, tc.token, "").Code; got != tc.want {
			t.Errorf("%s: status = %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestStatusCodes(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		body string
		want int
	}{
		{"ok", nil, `{"email":"a","password":"p"}`, 200},
		{"malformed json", nil, `{`, 400},
		{"missing password", nil, `{"email":"a"}`, 422},
		{"invalid argument", domain.ErrInvalidArgument, `{"email":"a","password":"p"}`, 422},
		{"bad credentials", domain.ErrUnauthenticated, `{"email":"a","password":"p"}`, 401},
		{"store not assigned", domain.ErrPermissionDenied, `{"email":"a","password":"p"}`, 403},
		{"infrastructure error", errors.New("pq: connection refused"), `{"email":"a","password":"p"}`, 500},
	} {
		rec := do(newMux(&stub{loginErr: tc.err}), "POST", "/api/v1/login", "", tc.body)
		if rec.Code != tc.want {
			t.Errorf("%s: status = %d, want %d (%s)", tc.name, rec.Code, tc.want, rec.Body)
		}
		if tc.want == 500 && strings.Contains(rec.Body.String(), "connection refused") {
			t.Errorf("%s: infrastructure error leaked: %s", tc.name, rec.Body)
		}
	}
}
