package sales_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Briyantama/POS-Stery/services/sales"
	"github.com/Briyantama/POS-Stery/services/sales/application/commands"
	"github.com/Briyantama/POS-Stery/services/sales/domain"
	"github.com/Briyantama/stery-kit/go/authn"
	"uuid"
)

var (
	tenantID = uuid.NewV7()
	storeID  = uuid.NewV7()
	userID   = uuid.NewV7()
)

type stubVerifier struct{}

func (stubVerifier) Verify(_ context.Context, token string) (authn.Claims, error) {
	c := authn.Claims{UserID: userID.String(), TenantID: tenantID.String(), StoreID: storeID.String()}
	switch token {
	case "cashier":
		c.Roles = []string{"cashier"}
	case "admin":
		c.Roles = []string{"admin"}
	default:
		return authn.Claims{}, errors.New("bad token")
	}
	return c, nil
}

type stubCreate struct {
	got commands.CreateSaleCommand
	err error
}

func (s *stubCreate) Handle(_ context.Context, cmd commands.CreateSaleCommand) (*commands.CreateSaleResult, error) {
	s.got = cmd
	if s.err != nil {
		return nil, s.err
	}
	sale := &domain.Sale{
		ID: uuid.NewV7(), TenantID: tenantID, StoreID: storeID, CashierID: userID,
		Items:  []domain.SaleItem{{ID: uuid.NewV7(), ProductID: uuid.NewV7(), Quantity: 1, UnitPrice: 5, LineTotal: 5}},
		Status: "COMPLETED", Total: 5, Subtotal: 5, CompletedAt: time.Now(),
	}
	return &commands.CreateSaleResult{Sale: sale, Receipt: &domain.Receipt{ID: uuid.NewV7(), SaleID: sale.ID, Snapshot: []byte(`{}`)}}, nil
}

type stubRead struct{ err error }

func (s stubRead) FindByID(_ context.Context, _, _, id uuid.UUID) (*domain.Sale, []domain.SaleItem, *domain.Receipt, error) {
	if s.err != nil {
		return nil, nil, nil, s.err
	}
	return &domain.Sale{ID: id, Status: "COMPLETED"}, nil, nil, nil
}

func (s stubRead) GetReport(_ context.Context, _, _ *uuid.UUID, date string) (*domain.SalesReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &domain.SalesReport{Date: date, TotalRevenue: 10, TransactionCount: 2}, nil
}

const validBody = `{"items":[{"product_id":"` + "11111111-1111-1111-1111-111111111111" + `","quantity":1,"unit_price":5}]}`

func do(h *sales.HTTPHandler, method, path, token, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	h.Register(mux)
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

func newHandler(c *stubCreate, r stubRead) *sales.HTTPHandler {
	return sales.NewHTTPHandler(c, r, stubVerifier{}, nil)
}

func TestUnauthenticated401(t *testing.T) {
	rec := do(newHandler(&stubCreate{}, stubRead{}), "POST", "/api/v1/sales", "", validBody)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestWrongRole403(t *testing.T) {
	rec := do(newHandler(&stubCreate{}, stubRead{}), "POST", "/api/v1/sales", "admin", validBody)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestSuccessEnvelopeAndClaimScope(t *testing.T) {
	c := &stubCreate{}
	rec := do(newHandler(c, stubRead{}), "POST", "/api/v1/sales", "cashier", validBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("got %d: %s", rec.Code, rec.Body)
	}
	var env struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Error   *string         `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || !env.Success || env.Error != nil || len(env.Data) == 0 {
		t.Fatalf("bad envelope: %s", rec.Body)
	}
	if !strings.HasPrefix(rec.Header().Get("Location"), "/api/v1/sales/") {
		t.Errorf("Location = %q", rec.Header().Get("Location"))
	}
	if c.got.TenantID != tenantID.String() || c.got.StoreID != storeID.String() || c.got.CashierID != userID.String() {
		t.Errorf("command scope not taken from claims: %+v", c.got)
	}
}

// A body-supplied tenant_id is an unknown field: the strict decoder rejects it, so it can never override the claim.
func TestBodyTenantIDNeverReachesCommand(t *testing.T) {
	c := &stubCreate{}
	body := `{"tenant_id":"` + uuid.NewV7().String() + `","items":[]}`
	rec := do(newHandler(c, stubRead{}), "POST", "/api/v1/sales", "cashier", body)
	if rec.Code != http.StatusBadRequest || c.got.TenantID != "" {
		t.Fatalf("got %d, command tenant %q", rec.Code, c.got.TenantID)
	}
}

func TestStatusCodes(t *testing.T) {
	saleURL := "/api/v1/sales/" + uuid.NewV7().String()
	cases := []struct {
		name              string
		method, path, tok string
		body              string
		create            error
		read              error
		want              int
	}{
		{"create 201", "POST", "/api/v1/sales", "cashier", validBody, nil, nil, 201},
		{"get 200", "GET", saleURL, "cashier", "", nil, nil, 200},
		{"report 200", "GET", "/api/v1/sales/report?date=2026-01-02", "admin", "", nil, nil, 200},
		{"malformed json 400", "POST", "/api/v1/sales", "cashier", `{`, nil, nil, 400},
		{"no token 401", "GET", saleURL, "", "", nil, nil, 401},
		{"wrong role 403", "GET", "/api/v1/sales/report", "cashier", "", nil, nil, 403},
		{"missing sale 404", "GET", saleURL, "cashier", "", nil, domain.ErrNotFound, 404},
		{"stock conflict 409", "POST", "/api/v1/sales", "cashier", validBody, domain.ErrStockWouldGoNegative, nil, 409},
		{"validation 422", "POST", "/api/v1/sales", "cashier", validBody, domain.ErrInvalidArgument, nil, 422},
		{"bad sale id 422", "GET", "/api/v1/sales/not-a-uuid", "cashier", "", nil, nil, 422},
		{"bad date 422", "GET", "/api/v1/sales/report?date=nope", "admin", "", nil, nil, 422},
		{"internal 500", "POST", "/api/v1/sales", "cashier", validBody, errors.New("db password leaked"), nil, 500},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(newHandler(&stubCreate{err: tc.create}, stubRead{err: tc.read}), tc.method, tc.path, tc.tok, tc.body)
			if rec.Code != tc.want {
				t.Fatalf("got %d, want %d: %s", rec.Code, tc.want, rec.Body)
			}
			if tc.want == 500 && !strings.Contains(rec.Body.String(), `"internal error"`) {
				t.Errorf("500 must carry the fixed text: %s", rec.Body)
			}
		})
	}
}
