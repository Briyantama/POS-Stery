package product_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Briyantama/stery-kit/go/authn"
	"uuid"

	"github.com/Briyantama/POS-Stery/services/product"
	"github.com/Briyantama/POS-Stery/services/product/application/commands"
	"github.com/Briyantama/POS-Stery/services/product/application/queries"
	"github.com/Briyantama/POS-Stery/services/product/domain"
)

var tenantID = uuid.NewV7()

type verifier struct{}

func (verifier) Verify(_ context.Context, tok string) (authn.Claims, error) {
	switch tok {
	case "admin", "cashier":
		return authn.Claims{UserID: "u1", TenantID: tenantID.String(), Roles: []string{tok}}, nil
	}
	return authn.Claims{}, errors.New("bad token")
}

type stub struct {
	err    error
	gotTID uuid.UUID
}

func (s *stub) prod() *domain.Product {
	return &domain.Product{ID: uuid.NewV7(), Name: "w", SKU: "s", CreatedAt: time.Now(), UpdatedAt: time.Now()}
}
func (s *stub) Handle(_ context.Context, c commands.CreateProductCommand) (*domain.Product, error) {
	s.gotTID = c.TenantID
	return s.prod(), s.err
}

type upd struct{ *stub }

func (u upd) Handle(_ context.Context, c commands.UpdateProductCommand) (*domain.Product, error) {
	return u.prod(), u.err
}

type srch struct{ *stub }

func (u srch) Handle(context.Context, queries.SearchProductsQuery) (*queries.SearchProductsResult, error) {
	return &queries.SearchProductsResult{Products: []*domain.Product{u.prod()}, Total: 1}, u.err
}

type get struct{ *stub }

func (g get) FindByID(context.Context, uuid.UUID, uuid.UUID) (*domain.Product, error) {
	return g.prod(), g.err
}

func serve(s *stub, method, path, token, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	product.NewHTTPHandler(s, upd{s}, srch{s}, get{s}, verifier{}).Register(mux)
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func TestStatuses(t *testing.T) {
	id := uuid.NewV7().String()
	ok := `{"name":"w","sku":"s"}`
	tests := []struct {
		name, method, path, token, body string
		err                             error
		want                            int
	}{
		{"list 200", "GET", "/api/v1/products", "cashier", "", nil, 200},
		{"get 200", "GET", "/api/v1/products/" + id, "cashier", "", nil, 200},
		{"create 201", "POST", "/api/v1/products", "admin", ok, nil, 201},
		{"update 200", "PUT", "/api/v1/products/" + id, "admin", `{"name":"w","is_active":true}`, nil, 200},
		{"bad json 400", "POST", "/api/v1/products", "admin", "{", nil, 400},
		{"no token 401", "GET", "/api/v1/products", "", "", nil, 401},
		{"wrong role 403", "POST", "/api/v1/products", "cashier", ok, nil, 403},
		{"missing 404", "GET", "/api/v1/products/" + id, "admin", "", domain.ErrNotFound, 404},
		{"duplicate 409", "POST", "/api/v1/products", "admin", ok, domain.ErrAlreadyExists, 409},
		{"invalid 422", "POST", "/api/v1/products", "admin", ok, domain.ErrInvalidArgument, 422},
		{"internal 500", "GET", "/api/v1/products", "admin", "", errors.New("db down"), 500},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := serve(&stub{err: tc.err}, tc.method, tc.path, tc.token, tc.body)
			if w.Code != tc.want {
				t.Fatalf("got %d want %d: %s", w.Code, tc.want, w.Body)
			}
		})
	}
}

func TestEnvelopeAndTenantFromClaim(t *testing.T) {
	// A body-supplied tenant_id is rejected (unknown field), so it can never override the claim.
	if w := serve(&stub{}, "POST", "/api/v1/products", "admin", `{"name":"w","sku":"s","tenant_id":"`+uuid.NewV7().String()+`"}`); w.Code != 400 {
		t.Fatalf("body tenant_id not rejected: %d", w.Code)
	}
	s := &stub{}
	w := serve(s, "POST", "/api/v1/products", "admin", `{"name":"w","sku":"s"}`)
	if s.gotTID != tenantID {
		t.Fatalf("tenant not from claim: %s", s.gotTID)
	}
	var env struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   *string         `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil || !env.Success || env.Error != nil || len(env.Data) == 0 {
		t.Fatalf("bad envelope: %s", w.Body)
	}
	if w.Header().Get("Location") == "" {
		t.Fatal("missing Location")
	}
}
