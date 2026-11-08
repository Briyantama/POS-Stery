package sales

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Briyantama/POS-Stery/services/sales/application/commands"
	"github.com/Briyantama/POS-Stery/services/sales/domain"
	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/httpx"
	"github.com/google/uuid"
)

const maxBody = 1 << 20

type saleCreator interface {
	Handle(ctx context.Context, cmd commands.CreateSaleCommand) (*commands.CreateSaleResult, error)
}

type saleReader interface {
	FindByID(ctx context.Context, tenantID, storeID, saleID uuid.UUID) (*domain.Sale, []domain.SaleItem, *domain.Receipt, error)
	GetReport(ctx context.Context, tenantID, storeID *uuid.UUID, date string) (*domain.SalesReport, error)
}

// HTTPHandler serves the /api/v1/sales routes.
type HTTPHandler struct {
	create       saleCreator
	read         saleReader
	verifier     authn.Verifier
	requireStore func(http.Handler) http.Handler
}

// NewHTTPHandler builds the handler. requireStore may be nil (tests); it runs after authentication.
func NewHTTPHandler(create saleCreator, read saleReader, v authn.Verifier, requireStore func(http.Handler) http.Handler) *HTTPHandler {
	if requireStore == nil {
		requireStore = func(next http.Handler) http.Handler { return next }
	}
	return &HTTPHandler{create: create, read: read, verifier: v, requireStore: requireStore}
}

// Register mounts the routes. The literal /report pattern wins over /{id} in ServeMux.
func (h *HTTPHandler) Register(mux *http.ServeMux) {
	mux.Handle("POST /api/v1/sales", h.guard(h.createSale, "cashier"))
	mux.Handle("GET /api/v1/sales/report", h.guard(h.report, "admin"))
	mux.Handle("GET /api/v1/sales/{id}", h.guard(h.getSale, "cashier"))
}

// guard = Authenticate -> RequireStore -> WithRole -> handler.
func (h *HTTPHandler) guard(fn http.HandlerFunc, roles ...string) http.Handler {
	return authn.Authenticate(h.verifier)(h.requireStore(authn.WithRole(roles...)(fn)))
}

type saleItemBody struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Discount  float64 `json:"discount"`
}

type createSaleBody struct {
	CustomerID     string         `json:"customer_id"`
	Items          []saleItemBody `json:"items"`
	DiscountAmount float64        `json:"discount_amount"`
	Notes          string         `json:"notes"`
}

type saleItemView struct {
	ID        string  `json:"id"`
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Discount  float64 `json:"discount"`
	LineTotal float64 `json:"line_total"`
}

type saleView struct {
	ID             string         `json:"id"`
	StoreID        string         `json:"store_id"`
	CashierID      string         `json:"cashier_id"`
	CustomerID     string         `json:"customer_id,omitempty"`
	Items          []saleItemView `json:"items"`
	Subtotal       float64        `json:"subtotal"`
	DiscountAmount float64        `json:"discount_amount"`
	Total          float64        `json:"total"`
	Status         string         `json:"status"`
	Notes          string         `json:"notes"`
	CompletedAt    time.Time      `json:"completed_at"`
}

type receiptView struct {
	ID       string          `json:"id"`
	SaleID   string          `json:"sale_id"`
	Snapshot json.RawMessage `json:"snapshot"`
	IssuedAt time.Time       `json:"issued_at"`
}

type saleResponse struct {
	Sale    saleView     `json:"sale"`
	Receipt *receiptView `json:"receipt"`
}

func toResponse(s *domain.Sale, items []domain.SaleItem, r *domain.Receipt) saleResponse {
	v := saleView{
		ID: s.ID.String(), StoreID: s.StoreID.String(), CashierID: s.CashierID.String(),
		Items:    make([]saleItemView, 0, len(items)),
		Subtotal: s.Subtotal, DiscountAmount: s.DiscountAmount, Total: s.Total,
		Status: s.Status, Notes: s.Notes, CompletedAt: s.CompletedAt,
	}
	if s.CustomerID != nil {
		v.CustomerID = s.CustomerID.String()
	}
	for _, it := range items {
		v.Items = append(v.Items, saleItemView{
			ID: it.ID.String(), ProductID: it.ProductID.String(), Quantity: it.Quantity,
			UnitPrice: it.UnitPrice, Discount: it.Discount, LineTotal: it.LineTotal,
		})
	}
	out := saleResponse{Sale: v}
	if r != nil {
		out.Receipt = &receiptView{ID: r.ID.String(), SaleID: r.SaleID.String(), Snapshot: r.Snapshot, IssuedAt: r.IssuedAt}
	}
	return out
}

func (h *HTTPHandler) createSale(w http.ResponseWriter, r *http.Request) {
	claims, ok := authn.FromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var body createSaleBody
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	items := make([]commands.SaleItemInput, len(body.Items))
	for i, it := range body.Items {
		items[i] = commands.SaleItemInput{ProductID: it.ProductID, Quantity: it.Quantity, UnitPrice: it.UnitPrice, Discount: it.Discount}
	}
	res, err := h.create.Handle(r.Context(), commands.CreateSaleCommand{
		TenantID:       claims.TenantID,
		StoreID:        claims.StoreID,
		CashierID:      claims.UserID,
		CustomerID:     body.CustomerID,
		Items:          items,
		DiscountAmount: body.DiscountAmount,
		Notes:          body.Notes,
	})
	if err != nil {
		writeDomainError(w, r, claims.TenantID, err)
		return
	}
	w.Header().Set("Location", "/api/v1/sales/"+res.Sale.ID.String())
	httpx.WriteSuccess(w, http.StatusCreated, toResponse(res.Sale, res.Sale.Items, res.Receipt))
}

func (h *HTTPHandler) getSale(w http.ResponseWriter, r *http.Request) {
	claims, ok := authn.FromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID, storeID, ok := parseScope(w, claims)
	if !ok {
		return
	}
	saleID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid sale id")
		return
	}
	s, items, rec, err := h.read.FindByID(r.Context(), tenantID, storeID, saleID)
	if err != nil {
		writeDomainError(w, r, claims.TenantID, err)
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, toResponse(s, items, rec))
}

type topProductView struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	UnitsSold   int     `json:"units_sold"`
	Revenue     float64 `json:"revenue"`
}

type reportView struct {
	Date             string           `json:"date"`
	StoreID          string           `json:"store_id"`
	TotalRevenue     float64          `json:"total_revenue"`
	TransactionCount int              `json:"transaction_count"`
	TopProducts      []topProductView `json:"top_products"`
}

func (h *HTTPHandler) report(w http.ResponseWriter, r *http.Request) {
	claims, ok := authn.FromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var storeID *uuid.UUID
	if claims.StoreID != "" {
		id, err := uuid.Parse(claims.StoreID)
		if err != nil {
			httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		storeID = &id
	}
	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().UTC().Format(time.DateOnly)
	} else if _, err := time.Parse(time.DateOnly, date); err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "date must be YYYY-MM-DD")
		return
	}
	rep, err := h.read.GetReport(r.Context(), &tenantID, storeID, date)
	if err != nil {
		writeDomainError(w, r, claims.TenantID, err)
		return
	}
	out := reportView{
		Date: rep.Date, StoreID: rep.StoreID, TotalRevenue: rep.TotalRevenue,
		TransactionCount: rep.TransactionCount, TopProducts: make([]topProductView, 0, len(rep.TopProducts)),
	}
	for _, tp := range rep.TopProducts {
		out.TopProducts = append(out.TopProducts, topProductView{tp.ProductID, tp.ProductName, tp.UnitsSold, tp.Revenue})
	}
	httpx.WriteSuccess(w, http.StatusOK, out)
}

// parseScope turns the token's tenant and store claims into UUIDs; a malformed claim is a 401.
func parseScope(w http.ResponseWriter, c authn.Claims) (tenantID, storeID uuid.UUID, ok bool) {
	tenantID, err1 := uuid.Parse(c.TenantID)
	storeID, err2 := uuid.Parse(c.StoreID)
	if err1 != nil || err2 != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, storeID, true
}

func writeDomainError(w http.ResponseWriter, r *http.Request, tenantID string, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid sale data")
	case errors.Is(err, domain.ErrStockWouldGoNegative):
		httpx.WriteError(w, http.StatusConflict, "insufficient stock")
	case errors.Is(err, domain.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not found")
	default:
		slog.ErrorContext(r.Context(), "sales request failed", "tenant_id", tenantID, "path", r.URL.Path, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}
