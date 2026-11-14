package supplier

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/httpx"
	"uuid"

	"github.com/Briyantama/POS-Stery/services/supplier/application/commands"
	"github.com/Briyantama/POS-Stery/services/supplier/domain"
)

const (
	maxBody        = 1 << 20
	defaultPerPage = 20
	maxPerPage     = 100
)

type supplierAdder interface {
	Handle(ctx context.Context, cmd commands.AddSupplierCommand) (*domain.Supplier, error)
}

type orderCreator interface {
	Handle(ctx context.Context, cmd commands.CreatePurchaseOrderCommand) (*domain.PurchaseOrder, error)
}

type stockReceiver interface {
	Handle(ctx context.Context, cmd commands.ReceiveStockCommand) (*domain.PurchaseOrder, error)
}

type supplierLister interface {
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Supplier, int, error)
}

type orderLister interface {
	List(ctx context.Context, tenantID, storeID uuid.UUID, status string, limit, offset int) ([]*domain.PurchaseOrder, int, error)
}

// HTTPHandler serves the supplier and purchase-order routes.
type HTTPHandler struct {
	addSupplier  supplierAdder
	createOrder  orderCreator
	receiveStock stockReceiver
	suppliers    supplierLister
	orders       orderLister
	verifier     authn.Verifier
	requireStore func(http.Handler) http.Handler
}

// NewHTTPHandler creates an HTTPHandler. requireStore validates the store claim on store-scoped routes.
func NewHTTPHandler(
	addSupplier supplierAdder,
	createOrder orderCreator,
	receiveStock stockReceiver,
	suppliers supplierLister,
	orders orderLister,
	verifier authn.Verifier,
	requireStore func(http.Handler) http.Handler,
) *HTTPHandler {
	return &HTTPHandler{addSupplier, createOrder, receiveStock, suppliers, orders, verifier, requireStore}
}

// Register mounts the routes on mux.
func (h *HTTPHandler) Register(mux *http.ServeMux) {
	auth := authn.Authenticate(h.verifier)
	tenantScoped := func(f http.HandlerFunc, roles ...string) http.Handler {
		return auth(authn.WithRole(roles...)(f))
	}
	storeScoped := func(f http.HandlerFunc, roles ...string) http.Handler {
		return auth(h.requireStore(authn.WithRole(roles...)(f)))
	}
	mux.Handle("GET /api/v1/suppliers", tenantScoped(h.listSuppliers, "admin", "stock_manager"))
	mux.Handle("POST /api/v1/suppliers", tenantScoped(h.createSupplier, "admin"))
	mux.Handle("GET /api/v1/purchase-orders", storeScoped(h.listOrders, "admin", "stock_manager"))
	mux.Handle("POST /api/v1/purchase-orders", storeScoped(h.createPurchaseOrder, "admin", "stock_manager"))
	mux.Handle("POST /api/v1/purchase-orders/{id}/receive", storeScoped(h.receive, "admin", "stock_manager"))
}

// --- request / response shapes ---

type supplierBody struct {
	Name    string `json:"name"`
	Contact string `json:"contact"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

type supplierDTO struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Contact   string    `json:"contact"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type orderLineBody struct {
	ProductID       string  `json:"product_id"`
	QuantityOrdered int32   `json:"quantity_ordered"`
	UnitCost        float64 `json:"unit_cost"`
}

type orderBody struct {
	SupplierID string          `json:"supplier_id"`
	Items      []orderLineBody `json:"items"`
	Notes      string          `json:"notes"`
}

type receiveLineBody struct {
	ProductID        string `json:"product_id"`
	QuantityReceived int32  `json:"quantity_received"`
}

type receiveBody struct {
	ItemsReceived []receiveLineBody `json:"items_received"`
}

type orderItemDTO struct {
	ProductID        string  `json:"product_id"`
	QuantityOrdered  int32   `json:"quantity_ordered"`
	QuantityReceived int32   `json:"quantity_received"`
	UnitCost         float64 `json:"unit_cost"`
}

type orderDTO struct {
	ID         string         `json:"id"`
	SupplierID string         `json:"supplier_id"`
	Status     string         `json:"status"`
	Notes      string         `json:"notes"`
	Items      []orderItemDTO `json:"items"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

type page struct {
	Items   any `json:"items"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

func toSupplierDTO(s *domain.Supplier) supplierDTO {
	return supplierDTO{s.ID.String(), s.Name, s.Contact, s.Phone, s.Email, s.Address, s.CreatedAt, s.UpdatedAt}
}

func toOrderDTO(o *domain.PurchaseOrder) orderDTO {
	items := make([]orderItemDTO, 0, len(o.Items))
	for _, it := range o.Items {
		items = append(items, orderItemDTO{it.ProductID.String(), it.QuantityOrdered, it.QuantityReceived, it.UnitCost})
	}
	return orderDTO{o.ID.String(), o.SupplierID.String(), string(o.Status), o.Notes, items, o.CreatedAt, o.UpdatedAt}
}

// --- handlers ---

func (h *HTTPHandler) createSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := scope(w, r, false)
	if !ok {
		return
	}
	var body supplierBody
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	s, err := h.addSupplier.Handle(r.Context(), commands.AddSupplierCommand{
		TenantID: tenantID, Name: body.Name, Contact: body.Contact, Phone: body.Phone, Email: body.Email, Address: body.Address,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/suppliers/"+s.ID.String())
	httpx.WriteSuccess(w, http.StatusCreated, toSupplierDTO(s))
}

func (h *HTTPHandler) listSuppliers(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := scope(w, r, false)
	if !ok {
		return
	}
	pg, perPage, err := httpx.PageParams(r, defaultPerPage, maxPerPage)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	list, total, err := h.suppliers.List(r.Context(), tenantID, perPage, (pg-1)*perPage)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	items := make([]supplierDTO, 0, len(list))
	for _, s := range list {
		items = append(items, toSupplierDTO(s))
	}
	httpx.WriteSuccess(w, http.StatusOK, page{items, pg, perPage, total})
}

func (h *HTTPHandler) createPurchaseOrder(w http.ResponseWriter, r *http.Request) {
	tenantID, storeID, ok := scope(w, r, true)
	if !ok {
		return
	}
	var body orderBody
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	supplierID, err := uuid.Parse(body.SupplierID)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation failed")
		return
	}
	lines := make([]commands.PurchaseOrderLineItem, 0, len(body.Items))
	for _, it := range body.Items {
		pid, err := uuid.Parse(it.ProductID)
		if err != nil {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation failed")
			return
		}
		lines = append(lines, commands.PurchaseOrderLineItem{ProductID: pid, QuantityOrdered: it.QuantityOrdered, UnitCost: it.UnitCost})
	}
	o, err := h.createOrder.Handle(r.Context(), commands.CreatePurchaseOrderCommand{
		TenantID: tenantID, StoreID: storeID, SupplierID: supplierID, Items: lines, Notes: body.Notes,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/purchase-orders/"+o.ID.String())
	httpx.WriteSuccess(w, http.StatusCreated, toOrderDTO(o))
}

func (h *HTTPHandler) listOrders(w http.ResponseWriter, r *http.Request) {
	tenantID, storeID, ok := scope(w, r, true)
	if !ok {
		return
	}
	pg, perPage, err := httpx.PageParams(r, defaultPerPage, maxPerPage)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	list, total, err := h.orders.List(r.Context(), tenantID, storeID, r.URL.Query().Get("status"), perPage, (pg-1)*perPage)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	items := make([]orderDTO, 0, len(list))
	for _, o := range list {
		items = append(items, toOrderDTO(o))
	}
	httpx.WriteSuccess(w, http.StatusOK, page{items, pg, perPage, total})
}

func (h *HTTPHandler) receive(w http.ResponseWriter, r *http.Request) {
	tenantID, storeID, ok := scope(w, r, true)
	if !ok {
		return
	}
	orderID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	var body receiveBody
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	lines := make([]commands.ReceivedLineItem, 0, len(body.ItemsReceived))
	for _, it := range body.ItemsReceived {
		pid, err := uuid.Parse(it.ProductID)
		if err != nil {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "validation failed")
			return
		}
		lines = append(lines, commands.ReceivedLineItem{ProductID: pid, QuantityReceived: it.QuantityReceived})
	}
	o, err := h.receiveStock.Handle(r.Context(), commands.ReceiveStockCommand{
		TenantID: tenantID, StoreID: storeID, PurchaseOrderID: orderID, ItemsReceived: lines,
	})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, toOrderDTO(o))
}

// --- helpers ---

// scope reads tenant (and, when needStore, store) ids from the token claims; it writes the error response itself.
func scope(w http.ResponseWriter, r *http.Request, needStore bool) (tenantID, storeID uuid.UUID, ok bool) {
	c, found := authn.FromContext(r.Context())
	if !found {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantID, err := uuid.Parse(c.TenantID)
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if needStore {
		if storeID, err = uuid.Parse(c.StoreID); err != nil {
			httpx.WriteError(w, http.StatusForbidden, "forbidden")
			return
		}
	}
	return tenantID, storeID, true
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "validation failed")
	case errors.Is(err, domain.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		httpx.WriteError(w, http.StatusConflict, "already exists")
	case errors.Is(err, domain.ErrConflict):
		httpx.WriteError(w, http.StatusConflict, "conflict")
	default:
		slog.ErrorContext(r.Context(), "supplier request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}
