// Package customer exposes the customer and loyalty HTTP API.
package customer

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Briyantama/POS-Stery/services/customer/application/commands"
	"github.com/Briyantama/POS-Stery/services/customer/application/queries"
	"github.com/Briyantama/POS-Stery/services/customer/domain"
	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/httpx"
	"github.com/google/uuid"
)

const (
	maxBody        = 1 << 20
	defaultPerPage = 20
	maxPerPage     = 100
)

type createHandler interface {
	Handle(ctx context.Context, cmd commands.CreateCustomerCommand) (*domain.Customer, error)
}

type loyaltyHandler interface {
	Handle(ctx context.Context, cmd commands.AddLoyaltyPointsCommand) (*commands.AddLoyaltyPointsResult, error)
}

type queryHandler interface {
	GetCustomer(ctx context.Context, q queries.GetCustomerQuery) (*domain.Customer, error)
	GetTotalPoints(ctx context.Context, tenantID, customerID uuid.UUID) (int, error)
	ListCustomers(ctx context.Context, q queries.ListCustomersQuery) (*queries.ListCustomersResult, error)
}

// HTTPHandler serves /api/v1/customers.
type HTTPHandler struct {
	create       createHandler
	loyalty      loyaltyHandler
	query        queryHandler
	verifier     authn.Verifier
	requireStore func(http.Handler) http.Handler
}

// NewHTTPHandler builds the handler. requireStore may be nil (no store check).
func NewHTTPHandler(c createHandler, l loyaltyHandler, q queryHandler, v authn.Verifier, requireStore func(http.Handler) http.Handler) *HTTPHandler {
	return &HTTPHandler{create: c, loyalty: l, query: q, verifier: v, requireStore: requireStore}
}

// Register mounts the routes; all are authenticated, roles admin and cashier.
func (h *HTTPHandler) Register(mux *http.ServeMux) {
	guard := func(f http.HandlerFunc) http.Handler {
		var next http.Handler = authn.WithRole("admin", "cashier")(f)
		if h.requireStore != nil {
			next = h.requireStore(next)
		}
		return authn.Authenticate(h.verifier)(next)
	}
	mux.Handle("GET /api/v1/customers", guard(h.list))
	mux.Handle("POST /api/v1/customers", guard(h.createCustomer))
	mux.Handle("GET /api/v1/customers/{customer_id}", guard(h.get))
	mux.Handle("POST /api/v1/customers/{customer_id}/loyalty", guard(h.addLoyalty))
}

type customerJSON struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	Name          string    `json:"name"`
	Phone         string    `json:"phone"`
	Email         string    `json:"email"`
	IsActive      bool      `json:"is_active"`
	LoyaltyPoints int       `json:"loyalty_points"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toJSON(c *domain.Customer, points int) customerJSON {
	return customerJSON{
		ID: c.ID.String(), TenantID: c.TenantID.String(), Name: c.Name, Phone: c.Phone, Email: c.Email,
		IsActive: c.IsActive, LoyaltyPoints: points, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

// tenant returns the tenant of the token; it writes 401 when the claim is unusable.
func tenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	cl, ok := authn.FromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(cl.TenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}
	return id, true
}

func customerID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("customer_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid customer id")
		return uuid.Nil, false
	}
	return id, true
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid request")
	case errors.Is(err, domain.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "customer not found")
	default:
		slog.ErrorContext(r.Context(), "customer request failed", "path", r.URL.Path, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}

func (h *HTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenant(w, r)
	if !ok {
		return
	}
	page, perPage, ok := paging(w, r)
	if !ok {
		return
	}
	res, err := h.query.ListCustomers(r.Context(), queries.ListCustomersQuery{
		TenantID: tid, Query: r.URL.Query().Get("q"), Limit: perPage, Offset: (page - 1) * perPage,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	items := make([]customerJSON, 0, len(res.Customers))
	for _, c := range res.Customers {
		items = append(items, toJSON(c, 0))
	}
	httpx.WriteSuccess(w, http.StatusOK, map[string]any{"items": items, "page": page, "per_page": perPage, "total": res.Total})
}

func paging(w http.ResponseWriter, r *http.Request) (page, perPage int, ok bool) {
	page, perPage = 1, defaultPerPage
	q := r.URL.Query()
	var err error
	if v := q.Get("page"); v != "" {
		if page, err = strconv.Atoi(v); err != nil || page < 1 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid pagination")
			return 0, 0, false
		}
	}
	if v := q.Get("per_page"); v != "" {
		if perPage, err = strconv.Atoi(v); err != nil || perPage < 1 || perPage > maxPerPage {
			httpx.WriteError(w, http.StatusBadRequest, "invalid pagination")
			return 0, 0, false
		}
	}
	return page, perPage, true
}

func (h *HTTPHandler) createCustomer(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenant(w, r)
	if !ok {
		return
	}
	var body struct {
		Name  string `json:"name"`
		Phone string `json:"phone"`
		Email string `json:"email"`
	}
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	c, err := h.create.Handle(r.Context(), commands.CreateCustomerCommand{
		TenantID: tid, Name: body.Name, Phone: body.Phone, Email: body.Email,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/customers/"+c.ID.String())
	httpx.WriteSuccess(w, http.StatusCreated, toJSON(c, 0))
}

func (h *HTTPHandler) get(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenant(w, r)
	if !ok {
		return
	}
	id, ok := customerID(w, r)
	if !ok {
		return
	}
	c, err := h.query.GetCustomer(r.Context(), queries.GetCustomerQuery{TenantID: tid, CustomerID: id})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	points, err := h.query.GetTotalPoints(r.Context(), tid, id)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, toJSON(c, points))
}

func (h *HTTPHandler) addLoyalty(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenant(w, r)
	if !ok {
		return
	}
	id, ok := customerID(w, r)
	if !ok {
		return
	}
	var body struct {
		SaleID string `json:"sale_id"`
		Points int    `json:"points"`
	}
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	saleID, err := uuid.Parse(body.SaleID)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid request")
		return
	}
	res, err := h.loyalty.Handle(r.Context(), commands.AddLoyaltyPointsCommand{
		TenantID: tid, CustomerID: id, SaleID: saleID, Points: body.Points,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, toJSON(res.Customer, res.TotalPoints))
}
