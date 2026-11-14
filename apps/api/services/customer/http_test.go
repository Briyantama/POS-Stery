package customer_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Briyantama/POS-Stery/services/customer"
	"github.com/Briyantama/POS-Stery/services/customer/application/commands"
	"github.com/Briyantama/POS-Stery/services/customer/domain"
	"github.com/Briyantama/stery-kit/go/authn"
	"uuid"
)

var tenantA = uuid.NewV7()

type stubVerifier struct{}

func (stubVerifier) Verify(_ context.Context, tok string) (authn.Claims, error) {
	switch tok {
	case "admin":
		return authn.Claims{UserID: "u1", TenantID: tenantA.String(), Roles: []string{"admin"}}, nil
	case "viewer":
		return authn.Claims{UserID: "u2", TenantID: tenantA.String(), Roles: []string{"viewer"}}, nil
	case "cashier":
		return authn.Claims{UserID: "u3", TenantID: tenantA.String(), Roles: []string{"cashier"}}, nil
	}
	return authn.Claims{}, errors.New("bad token")
}

type stub struct{ err error }

func (s stub) cust(tenant uuid.UUID) *domain.Customer {
	return &domain.Customer{ID: uuid.NewV7(), TenantID: tenant, Name: "Jane", IsActive: true}
}

func (s stub) Handle(_ context.Context, cmd commands.CreateCustomerCommand) (*domain.Customer, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.cust(cmd.TenantID), nil
}

type stubLoyalty struct{ stub }

func (s stubLoyalty) Handle(_ context.Context, cmd commands.AddLoyaltyPointsCommand) (*commands.AddLoyaltyPointsResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &commands.AddLoyaltyPointsResult{Customer: s.cust(cmd.TenantID), TotalPoints: cmd.Points}, nil
}

type stubQuery struct{ stub }

func (s stubQuery) FindByID(_ context.Context, tenantID, _ uuid.UUID) (*domain.Customer, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.cust(tenantID), nil
}
func (stubQuery) GetTotalPoints(context.Context, uuid.UUID, uuid.UUID) (int, error) { return 5, nil }
func (s stubQuery) List(_ context.Context, tenantID uuid.UUID, _ string, _, _ int) ([]*domain.Customer, int, error) {
	if s.err != nil {
		return nil, 0, s.err
	}
	return []*domain.Customer{s.cust(tenantID)}, 1, nil
}

func newMux(err error) *http.ServeMux {
	mux := http.NewServeMux()
	customer.NewHTTPHandler(stub{err}, stubLoyalty{stub{err}}, stubQuery{stub{err}}, stubQuery{stub{err}}, stubVerifier{}, nil).Register(mux)
	return mux
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
	if got := do(newMux(nil), "GET", "/api/v1/customers", "", "").Code; got != 401 {
		t.Fatalf("no token: %d", got)
	}
	if got := do(newMux(nil), "GET", "/api/v1/customers", "viewer", "").Code; got != 403 {
		t.Fatalf("wrong role: %d", got)
	}
}

func TestEnvelopeAndTenantFromClaim(t *testing.T) {
	// A body-supplied tenant_id is rejected (unknown field), so it can never override the claim.
	if rec := do(newMux(nil), "POST", "/api/v1/customers", "admin", `{"name":"Jane","tenant_id":"`+uuid.NewV7().String()+`"}`); rec.Code != 400 {
		t.Fatalf("body tenant_id not rejected: %d", rec.Code)
	}
	rec := do(newMux(nil), "POST", "/api/v1/customers", "admin", `{"name":"Jane"}`)
	if rec.Code != 201 || !strings.HasPrefix(rec.Header().Get("Location"), "/api/v1/customers/") {
		t.Fatalf("create: %d %v %s", rec.Code, rec.Header(), rec.Body)
	}
	var env struct {
		Success bool `json:"success"`
		Data    struct {
			TenantID string `json:"tenant_id"`
		} `json:"data"`
		Error *string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	if !env.Success || env.Error != nil || env.Data.TenantID != tenantA.String() {
		t.Fatalf("envelope %s", rec.Body)
	}
}

func TestStatusTable(t *testing.T) {
	id := uuid.NewV7().String()
	loy := `{"sale_id":"` + uuid.NewV7().String() + `","points":10}`
	rows := []struct {
		name, method, path, body string
		err                      error
		want                     int
	}{
		{"list 200", "GET", "/api/v1/customers?page=2&per_page=5", "", nil, 200},
		{"get 200", "GET", "/api/v1/customers/" + id, "", nil, 200},
		{"loyalty 200", "POST", "/api/v1/customers/" + id + "/loyalty", loy, nil, 200},
		{"loyalty duplicate sale 409", "POST", "/api/v1/customers/" + id + "/loyalty", loy, domain.ErrAlreadyAwarded, 409},
		{"create 201", "POST", "/api/v1/customers", `{"name":"J"}`, nil, 201},
		{"malformed 400", "POST", "/api/v1/customers", `{`, nil, 400},
		{"bad page 400", "GET", "/api/v1/customers?page=0", "", nil, 400},
		{"bad id 400", "GET", "/api/v1/customers/x", "", nil, 400},
		{"validation 422", "POST", "/api/v1/customers", `{"name":""}`, domain.ErrInvalidArgument, 422},
		{"not found 404", "GET", "/api/v1/customers/" + id, "", domain.ErrNotFound, 404},
		{"internal 500", "GET", "/api/v1/customers/" + id, "", errors.New("db password leak"), 500},
	}
	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			rec := do(newMux(r.err), r.method, r.path, "admin", r.body)
			if rec.Code != r.want {
				t.Fatalf("got %d want %d: %s", rec.Code, r.want, rec.Body)
			}
			if r.want == 500 && !strings.Contains(rec.Body.String(), `"internal error"`) {
				t.Fatalf("leaked: %s", rec.Body)
			}
		})
	}
}

func TestLoyaltyAdminOnly(t *testing.T) {
	path := "/api/v1/customers/" + uuid.NewV7().String() + "/loyalty"
	body := `{"sale_id":"` + uuid.NewV7().String() + `","points":1000000}`
	if got := do(newMux(nil), "POST", path, "cashier", body).Code; got != 403 {
		t.Fatalf("cashier minted points: %d", got)
	}
	if got := do(newMux(nil), "GET", "/api/v1/customers", "cashier", "").Code; got != 200 {
		t.Fatalf("cashier read: %d", got)
	}
}
