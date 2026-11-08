package inventory_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Briyantama/POS-Stery/services/inventory"
	"github.com/Briyantama/POS-Stery/services/inventory/application/commands"
	"github.com/Briyantama/POS-Stery/services/inventory/application/queries"
	"github.com/Briyantama/POS-Stery/services/inventory/domain"
	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/google/uuid"
)

var (
	tenantID = uuid.NewString()
	storeID  = uuid.NewString()
	prodID   = uuid.NewString()
)

type stubVerifier struct{}

func (stubVerifier) Verify(_ context.Context, token string) (authn.Claims, error) {
	switch token {
	case "admin":
		return authn.Claims{UserID: "u1", TenantID: tenantID, StoreID: storeID, Roles: []string{"admin"}}, nil
	case "cashier":
		return authn.Claims{UserID: "u2", TenantID: tenantID, StoreID: storeID, Roles: []string{"cashier"}}, nil
	}
	return authn.Claims{}, errors.New("bad token")
}

// stub implements all five handler interfaces; err is returned by every call.
type stub struct {
	err        error
	gotTenant  uuid.UUID
	gotProduct uuid.UUID
}

func (s *stub) item() *domain.StockItem {
	return &domain.StockItem{ID: uuid.New(), TenantID: uuid.MustParse(tenantID), StoreID: uuid.MustParse(storeID), ProductID: uuid.MustParse(prodID), Quantity: 7}
}

func (s *stub) addOrGet(t uuid.UUID, p uuid.UUID) (*domain.StockItem, error) {
	s.gotTenant, s.gotProduct = t, p
	if s.err != nil {
		return nil, s.err
	}
	return s.item(), nil
}

type addStub struct{ *stub }

func (a addStub) Handle(ctx context.Context, c commands.AddItemCommand) (*domain.StockItem, error) {
	return a.addOrGet(c.TenantID, c.ProductID)
}

type updStub struct{ *stub }

func (u updStub) Handle(_ context.Context, _ commands.UpdateStockCommand) (commands.UpdateStockResult, error) {
	if u.err != nil {
		return commands.UpdateStockResult{}, u.err
	}
	return commands.UpdateStockResult{Item: u.item(), LowStockAlert: true}, nil
}

type thrStub struct{ *stub }

func (t thrStub) Handle(_ context.Context, _ commands.SetThresholdCommand) error { return t.err }

type getStub struct{ *stub }

func (g getStub) Handle(_ context.Context, q queries.GetItemQuery) (*domain.StockItem, error) {
	return g.addOrGet(q.TenantID, q.ProductID)
}

type listStub struct{ *stub }

func (l listStub) Handle(_ context.Context, _ queries.ListItemsQuery) (queries.ListItemsResult, error) {
	if l.err != nil {
		return queries.ListItemsResult{}, l.err
	}
	return queries.ListItemsResult{Items: []*domain.StockItem{l.item()}, Total: 1}, nil
}

func newMux(s *stub) *http.ServeMux {
	mux := http.NewServeMux()
	inventory.NewHTTPHandler(addStub{s}, updStub{s}, thrStub{s}, getStub{s}, listStub{s}, stubVerifier{}, nil).Register(mux)
	return mux
}

type lowListStub struct{}

func (lowListStub) Handle(context.Context, queries.ListItemsQuery) (queries.ListItemsResult, error) {
	min := int64(10)
	mk := func(name string, min *int64, low bool) *domain.StockItem {
		return &domain.StockItem{ID: uuid.New(), TenantID: uuid.MustParse(tenantID), StoreID: uuid.MustParse(storeID), ProductID: uuid.New(), Quantity: 7, ProductName: name, MinQuantity: min, IsLowStock: low}
	}
	return queries.ListItemsResult{Items: []*domain.StockItem{mk("Tea", &min, true), mk("Cup", nil, false)}, Total: 2}, nil
}

func TestListIncludesProductAndThreshold(t *testing.T) {
	s := &stub{}
	mux := http.NewServeMux()
	inventory.NewHTTPHandler(addStub{s}, updStub{s}, thrStub{s}, getStub{s}, lowListStub{}, stubVerifier{}, nil).Register(mux)
	rec := do(mux, "GET", "/api/v1/inventory", "admin", "")
	var env struct {
		Data struct {
			Items []map[string]any `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || len(env.Data.Items) != 2 {
		t.Fatalf("bad body: %d %s", rec.Code, rec.Body)
	}
	a, b := env.Data.Items[0], env.Data.Items[1]
	if a["product_name"] != "Tea" || a["min_quantity"] != float64(10) || a["is_low_stock"] != true {
		t.Errorf("first item: %v", a)
	}
	if v, present := b["min_quantity"]; !present || v != nil || b["is_low_stock"] != false {
		t.Errorf("second item must have null min_quantity and is_low_stock false: %v", b)
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

func TestAuth(t *testing.T) {
	mux := newMux(&stub{})
	if rec := do(mux, "GET", "/api/v1/inventory", "", ""); rec.Code != 401 {
		t.Fatalf("no token: want 401, got %d", rec.Code)
	}
	if rec := do(mux, "GET", "/api/v1/inventory", "cashier", ""); rec.Code != 403 {
		t.Fatalf("wrong role: want 403, got %d", rec.Code)
	}
}

func TestEnvelopeAndTenantFromClaim(t *testing.T) {
	s := &stub{}
	body := `{"product_id":"` + prodID + `","quantity":5,"tenant_id":"` + uuid.NewString() + `"}`
	rec := do(newMux(s), "POST", "/api/v1/inventory", "admin", body)
	// tenant_id is not a field of the body struct, so DecodeJSON rejects it; send without it for the claim check.
	if rec.Code != 400 {
		t.Fatalf("unknown body field: want 400, got %d", rec.Code)
	}
	rec = do(newMux(s), "POST", "/api/v1/inventory", "admin", `{"product_id":"`+prodID+`","quantity":5}`)
	if rec.Code != 201 || rec.Header().Get("Location") != "/api/v1/inventory/"+prodID {
		t.Fatalf("create: got %d, Location %q", rec.Code, rec.Header().Get("Location"))
	}
	var env struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   *string         `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || !env.Success || env.Error != nil || len(env.Data) == 0 {
		t.Fatalf("bad envelope: %s", rec.Body.String())
	}
	if s.gotTenant.String() != tenantID {
		t.Fatalf("tenant must come from the claim, got %s", s.gotTenant)
	}
}

func TestStatusCodes(t *testing.T) {
	bad := "/api/v1/inventory/not-a-uuid"
	good := "/api/v1/inventory/" + prodID
	cases := []struct {
		name, method, path, body string
		err                      error
		want                     int
	}{
		{"list ok", "GET", "/api/v1/inventory?page=1&per_page=20", "", nil, 200},
		{"list bad page", "GET", "/api/v1/inventory?page=0", "", nil, 422},
		{"get ok", "GET", good, "", nil, 200},
		{"get bad id", "GET", bad, "", nil, 422},
		{"get missing", "GET", good, "", domain.ErrNotFound, 404},
		{"add ok", "POST", "/api/v1/inventory", `{"product_id":"` + prodID + `"}`, nil, 201},
		{"add malformed", "POST", "/api/v1/inventory", `{`, nil, 400},
		{"add invalid", "POST", "/api/v1/inventory", `{"product_id":"x"}`, nil, 422},
		{"add duplicate", "POST", "/api/v1/inventory", `{"product_id":"` + prodID + `"}`, domain.ErrAlreadyExists, 409},
		{"threshold ok", "PUT", good + "/threshold", `{"min_quantity":3}`, nil, 200},
		{"threshold domain invalid", "PUT", good + "/threshold", `{"min_quantity":-1}`, domain.ErrInvalidArgument, 422},
		{"stock ok", "PATCH", good + "/stock", `{"quantity_delta":-2,"reason":"SALE"}`, nil, 200},
		{"stock bad reason", "PATCH", good + "/stock", `{"quantity_delta":1,"reason":"X"}`, nil, 422},
		{"stock negative", "PATCH", good + "/stock", `{"quantity_delta":-99}`, domain.ErrStockWouldGoNegative, 409},
		{"stock internal", "PATCH", good + "/stock", `{"quantity_delta":1}`, errors.New("db secret"), 500},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := do(newMux(&stub{err: c.err}), c.method, c.path, "admin", c.body)
			if rec.Code != c.want {
				t.Fatalf("want %d, got %d: %s", c.want, rec.Code, rec.Body.String())
			}
			if c.want == 500 && !strings.Contains(rec.Body.String(), `"internal error"`) {
				t.Fatalf("500 must hide the cause: %s", rec.Body.String())
			}
		})
	}
}
