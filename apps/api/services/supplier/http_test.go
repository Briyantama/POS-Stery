package supplier_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/google/uuid"

	"github.com/Briyantama/POS-Stery/services/supplier"
	"github.com/Briyantama/POS-Stery/services/supplier/application/commands"
	"github.com/Briyantama/POS-Stery/services/supplier/domain"
)

var (
	tenantID = uuid.New()
	storeID  = uuid.New()
)

type stubVerifier struct{}

func (stubVerifier) Verify(_ context.Context, token string) (authn.Claims, error) {
	if token == "" || token == "bad" {
		return authn.Claims{}, errors.New("invalid token")
	}
	return authn.Claims{UserID: "u1", TenantID: tenantID.String(), StoreID: storeID.String(), Roles: []string{token}}, nil
}

// stub is one value implementing every handler interface; err is returned by all of them.
type stub struct {
	err    error
	tenant uuid.UUID
}

func (s *stub) Handle(_ context.Context, cmd commands.AddSupplierCommand) (*domain.Supplier, error) {
	s.tenant = cmd.TenantID
	return &domain.Supplier{ID: uuid.New(), TenantID: cmd.TenantID, Name: cmd.Name}, s.err
}

type orderStub struct{ *stub }

func (s orderStub) Handle(_ context.Context, cmd commands.CreatePurchaseOrderCommand) (*domain.PurchaseOrder, error) {
	return &domain.PurchaseOrder{ID: uuid.New(), SupplierID: cmd.SupplierID, Status: domain.StatusPending}, s.err
}

type receiveStub struct{ *stub }

func (s receiveStub) Handle(_ context.Context, cmd commands.ReceiveStockCommand) (*domain.PurchaseOrder, error) {
	return &domain.PurchaseOrder{ID: cmd.PurchaseOrderID, Status: domain.StatusReceived}, s.err
}

type listStub struct{ *stub }

func (s listStub) List(_ context.Context, _ uuid.UUID, _, _ int) ([]*domain.Supplier, int, error) {
	return []*domain.Supplier{{ID: uuid.New(), Name: "acme"}}, 1, s.err
}

type orderListStub struct{ *stub }

func (s orderListStub) List(_ context.Context, _, _ uuid.UUID, _ string, _, _ int) ([]*domain.PurchaseOrder, int, error) {
	return []*domain.PurchaseOrder{{ID: uuid.New(), Status: domain.StatusPending}}, 1, s.err
}

func newMux(s *stub) *http.ServeMux {
	h := supplier.NewHTTPHandler(s, orderStub{s}, receiveStub{s}, listStub{s}, orderListStub{s}, stubVerifier{},
		func(next http.Handler) http.Handler { return next })
	mux := http.NewServeMux()
	h.Register(mux)
	return mux
}

func do(mux *http.ServeMux, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestAuth(t *testing.T) {
	mux := newMux(&stub{})
	if got := do(mux, "GET", "/api/v1/suppliers", "", "").Code; got != 401 {
		t.Fatalf("no token: want 401, got %d", got)
	}
	if got := do(mux, "POST", "/api/v1/suppliers", "stock_manager", `{"name":"a"}`).Code; got != 403 {
		t.Fatalf("wrong role: want 403, got %d", got)
	}
}

func TestEnvelopeAndTenantFromClaim(t *testing.T) {
	s := &stub{}
	mux := newMux(s)
	rec := do(mux, "POST", "/api/v1/suppliers", "admin", `{"name":"acme"}`)
	var env struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   *string         `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || !env.Success || env.Error != nil || len(env.Data) == 0 {
		t.Fatalf("bad envelope %s", rec.Body)
	}
	if s.tenant != tenantID {
		t.Fatalf("tenant must come from claims, got %s", s.tenant)
	}
	// A body-supplied tenant_id is rejected (strict decoding), never used.
	s.tenant = uuid.Nil
	rec = do(mux, "POST", "/api/v1/suppliers", "admin", fmt.Sprintf(`{"name":"acme","tenant_id":%q}`, uuid.New()))
	if rec.Code != 400 || s.tenant != uuid.Nil {
		t.Fatalf("body tenant_id: want 400 and no call, got %d tenant=%s", rec.Code, s.tenant)
	}
}

func TestStatusCodes(t *testing.T) {
	pid, oid := uuid.NewString(), uuid.NewString()
	order := fmt.Sprintf(`{"supplier_id":%q,"items":[{"product_id":%q,"quantity_ordered":1,"unit_cost":2}]}`, uuid.NewString(), pid)
	recv := fmt.Sprintf(`{"items_received":[{"product_id":%q,"quantity_received":1}]}`, pid)
	tests := []struct {
		name         string
		err          error
		method, path string
		body         string
		want         int
	}{
		{"list suppliers", nil, "GET", "/api/v1/suppliers?page=2&per_page=5", "", 200},
		{"list orders", nil, "GET", "/api/v1/purchase-orders", "", 200},
		{"create supplier", nil, "POST", "/api/v1/suppliers", `{"name":"a"}`, 201},
		{"create order", nil, "POST", "/api/v1/purchase-orders", order, 201},
		{"receive", nil, "POST", "/api/v1/purchase-orders/" + oid + "/receive", recv, 200},
		{"malformed json", nil, "POST", "/api/v1/suppliers", `{`, 400},
		{"bad per_page", nil, "GET", "/api/v1/suppliers?per_page=101", "", 400},
		{"validation", domain.ErrInvalidArgument, "POST", "/api/v1/suppliers", `{"name":""}`, 422},
		{"bad uuid", nil, "POST", "/api/v1/purchase-orders", `{"supplier_id":"x","items":[]}`, 422},
		{"duplicate", domain.ErrAlreadyExists, "POST", "/api/v1/suppliers", `{"name":"a"}`, 409},
		{"already received", domain.ErrConflict, "POST", "/api/v1/purchase-orders/" + oid + "/receive", recv, 409},
		{"not found", domain.ErrNotFound, "POST", "/api/v1/purchase-orders/" + oid + "/receive", recv, 404},
		{"internal", errors.New("db down: secret"), "POST", "/api/v1/suppliers", `{"name":"a"}`, 500},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(newMux(&stub{err: tc.err}), tc.method, tc.path, "admin", tc.body)
			if rec.Code != tc.want {
				t.Fatalf("want %d, got %d: %s", tc.want, rec.Code, rec.Body)
			}
			if tc.want == 201 && rec.Header().Get("Location") == "" {
				t.Fatal("missing Location")
			}
			if tc.want == 500 && strings.Contains(rec.Body.String(), "secret") {
				t.Fatal("internal error text leaked")
			}
		})
	}
}
