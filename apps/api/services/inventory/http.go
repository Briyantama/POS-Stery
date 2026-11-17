package inventory

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/Briyantama/POS-Stery/services/inventory/application/commands"
	"github.com/Briyantama/POS-Stery/services/inventory/application/queries"
	"github.com/Briyantama/POS-Stery/services/inventory/domain"
	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/httpx"
	"uuid"
)

const (
	maxBody        = 1 << 20
	defaultPerPage = 20
	maxPerPage     = queries.MaxPageSize
)

type addItemHandler interface {
	Handle(ctx context.Context, cmd commands.AddItemCommand) (*domain.StockItem, error)
}

type updateStockHandler interface {
	Handle(ctx context.Context, cmd commands.UpdateStockCommand) (commands.UpdateStockResult, error)
}

type setThresholdHandler interface {
	Handle(ctx context.Context, cmd commands.SetThresholdCommand) error
}

type getItemHandler interface {
	Handle(ctx context.Context, qry queries.GetItemQuery) (*domain.StockItem, error)
}

type listItemsHandler interface {
	Handle(ctx context.Context, qry queries.ListItemsQuery) (queries.ListItemsResult, error)
}

// HTTPHandler serves the /api/v1/inventory routes.
type HTTPHandler struct {
	addItem      addItemHandler
	updateStock  updateStockHandler
	setThreshold setThresholdHandler
	getItem      getItemHandler
	listItems    listItemsHandler
	verifier     authn.Verifier
	requireStore func(http.Handler) http.Handler
}

// NewHTTPHandler creates the handler; requireStore may be nil (no store validation).
func NewHTTPHandler(
	addItem addItemHandler,
	updateStock updateStockHandler,
	setThreshold setThresholdHandler,
	getItem getItemHandler,
	listItems listItemsHandler,
	verifier authn.Verifier,
	requireStore func(http.Handler) http.Handler,
) *HTTPHandler {
	return &HTTPHandler{
		addItem: addItem, updateStock: updateStock, setThreshold: setThreshold,
		getItem: getItem, listItems: listItems, verifier: verifier, requireStore: requireStore,
	}
}

// Register mounts the routes on mux. Every route needs a token with role admin or stock_manager
// and a store validated by requireStore.
func (h *HTTPHandler) Register(mux *http.ServeMux) {
	wrap := func(f http.HandlerFunc) http.Handler {
		var next http.Handler = f
		if h.requireStore != nil {
			next = h.requireStore(next)
		}
		return authn.Authenticate(h.verifier)(authn.WithRole("admin", "stock_manager")(next))
	}
	mux.Handle("GET /api/v1/inventory", wrap(h.list))
	mux.Handle("POST /api/v1/inventory", wrap(h.add))
	mux.Handle("GET /api/v1/inventory/{product_id}", wrap(h.get))
	mux.Handle("PUT /api/v1/inventory/{product_id}/threshold", wrap(h.putThreshold))
	mux.Handle("PATCH /api/v1/inventory/{product_id}/stock", wrap(h.patchStock))
}

type itemJSON struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	StoreID     string    `json:"store_id"`
	ProductID   string    `json:"product_id"`
	ProductName string    `json:"product_name"`
	Quantity    int64     `json:"quantity"`
	MinQuantity *int64    `json:"min_quantity"`
	IsLowStock  bool      `json:"is_low_stock"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toItem(i *domain.StockItem) itemJSON {
	return itemJSON{
		ID: i.ID.String(), TenantID: i.TenantID.String(), StoreID: i.StoreID.String(),
		ProductID: i.ProductID.String(), ProductName: i.ProductName, Quantity: i.Quantity,
		MinQuantity: i.MinQuantity, IsLowStock: i.IsLowStock, CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt,
	}
}

// scope reads tenant and store from the token claims; it writes the error response itself.
func scope(w http.ResponseWriter, r *http.Request) (tenant, store uuid.UUID, ok bool) {
	c, found := authn.FromContext(r.Context())
	if !found {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenant, terr := uuid.Parse(c.TenantID)
	store, serr := uuid.Parse(c.StoreID)
	if terr != nil || serr != nil {
		httpx.WriteError(w, http.StatusForbidden, "forbidden")
		return uuid.Nil(), uuid.Nil(), false
	}
	return tenant, store, true
}

func productID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("product_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid product id")
		return uuid.Nil(), false
	}
	return id, true
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid request")
	case errors.Is(err, domain.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrAlreadyExists):
		httpx.WriteError(w, http.StatusConflict, "item already exists")
	case errors.Is(err, domain.ErrStockWouldGoNegative):
		httpx.WriteError(w, http.StatusConflict, "stock would go negative")
	default:
		slog.ErrorContext(r.Context(), "inventory request failed", "path", r.URL.Path, "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}

func (h *HTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	tenant, store, ok := scope(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	page, perPage := 1, defaultPerPage
	var err error
	if v := q.Get("page"); v != "" {
		if page, err = strconv.Atoi(v); err != nil || page < 1 {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid page")
			return
		}
	}
	if v := q.Get("per_page"); v != "" {
		if perPage, err = strconv.Atoi(v); err != nil || perPage < 1 || perPage > maxPerPage {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid per_page")
			return
		}
	}
	low := false
	if v := q.Get("low_stock_only"); v != "" {
		if low, err = strconv.ParseBool(v); err != nil {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid low_stock_only")
			return
		}
	}
	res, err := h.listItems.Handle(r.Context(), queries.ListItemsQuery{
		TenantID: tenant, StoreID: store, LowStockOnly: low, Limit: perPage, Offset: (page - 1) * perPage,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	items := make([]itemJSON, 0, len(res.Items))
	for _, i := range res.Items {
		items = append(items, toItem(i))
	}
	httpx.WriteSuccess(w, http.StatusOK, map[string]any{
		"items": items, "page": page, "per_page": perPage, "total": res.Total,
	})
}

func (h *HTTPHandler) add(w http.ResponseWriter, r *http.Request) {
	tenant, store, ok := scope(w, r)
	if !ok {
		return
	}
	var body struct {
		ProductID   string `json:"product_id"`
		Quantity    int64  `json:"quantity"`
		MinQuantity int64  `json:"min_quantity"`
	}
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	pid, err := uuid.Parse(body.ProductID)
	if err != nil {
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid product id")
		return
	}
	item, err := h.addItem.Handle(r.Context(), commands.AddItemCommand{
		TenantID: tenant, StoreID: store, ProductID: pid,
		InitialQuantity: body.Quantity, MinQuantity: body.MinQuantity,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/inventory/"+item.ProductID.String())
	httpx.WriteSuccess(w, http.StatusCreated, toItem(item))
}

func (h *HTTPHandler) get(w http.ResponseWriter, r *http.Request) {
	tenant, store, ok := scope(w, r)
	if !ok {
		return
	}
	pid, ok := productID(w, r)
	if !ok {
		return
	}
	item, err := h.getItem.Handle(r.Context(), queries.GetItemQuery{TenantID: tenant, StoreID: store, ProductID: pid})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, toItem(item))
}

func (h *HTTPHandler) putThreshold(w http.ResponseWriter, r *http.Request) {
	tenant, store, ok := scope(w, r)
	if !ok {
		return
	}
	pid, ok := productID(w, r)
	if !ok {
		return
	}
	var body struct {
		MinQuantity int64 `json:"min_quantity"`
	}
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	if err := h.setThreshold.Handle(r.Context(), commands.SetThresholdCommand{
		TenantID: tenant, StoreID: store, ProductID: pid, MinQuantity: body.MinQuantity,
	}); err != nil {
		writeErr(w, r, err)
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, map[string]any{"product_id": pid.String(), "min_quantity": body.MinQuantity})
}

func (h *HTTPHandler) patchStock(w http.ResponseWriter, r *http.Request) {
	tenant, store, ok := scope(w, r)
	if !ok {
		return
	}
	pid, ok := productID(w, r)
	if !ok {
		return
	}
	var body struct {
		QuantityDelta int64  `json:"quantity_delta"`
		Reason        string `json:"reason"`
		ReferenceID   string `json:"reference_id"`
	}
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	switch body.Reason {
	case "":
		body.Reason = string(domain.ReasonAdjustment)
	case string(domain.ReasonSale), string(domain.ReasonReceipt), string(domain.ReasonAdjustment):
	default:
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid reason")
		return
	}
	res, err := h.updateStock.Handle(r.Context(), commands.UpdateStockCommand{
		TenantID: tenant, StoreID: store, ProductID: pid,
		Delta: body.QuantityDelta, Reason: body.Reason, RefID: body.ReferenceID,
	})
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, map[string]any{"item": toItem(res.Item), "low_stock_alert": res.LowStockAlert})
}
