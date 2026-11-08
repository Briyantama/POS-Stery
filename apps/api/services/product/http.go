package product

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/httpx"
	"github.com/google/uuid"

	"github.com/Briyantama/POS-Stery/services/product/application/commands"
	"github.com/Briyantama/POS-Stery/services/product/application/queries"
	"github.com/Briyantama/POS-Stery/services/product/domain"
)

const maxBody = 1 << 20

type creator interface {
	Handle(context.Context, commands.CreateProductCommand) (*domain.Product, error)
}
type updater interface {
	Handle(context.Context, commands.UpdateProductCommand) (*domain.Product, error)
}
type searcher interface {
	Handle(context.Context, queries.SearchProductsQuery) (*queries.SearchProductsResult, error)
}
type getter interface {
	GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error)
}

// HTTPHandler serves /api/v1/products.
type HTTPHandler struct {
	create   creator
	update   updater
	search   searcher
	get      getter
	verifier authn.Verifier
}

func NewHTTPHandler(c creator, u updater, s searcher, g getter, v authn.Verifier) *HTTPHandler {
	return &HTTPHandler{create: c, update: u, search: s, get: g, verifier: v}
}

func (h *HTTPHandler) Register(mux *http.ServeMux) {
	auth := authn.Authenticate(h.verifier)
	read := authn.WithRole("admin", "cashier")
	write := authn.WithRole("admin")
	mux.Handle("GET /api/v1/products", auth(read(http.HandlerFunc(h.list))))
	mux.Handle("GET /api/v1/products/{id}", auth(read(http.HandlerFunc(h.getOne))))
	mux.Handle("POST /api/v1/products", auth(write(http.HandlerFunc(h.createProduct))))
	mux.Handle("PUT /api/v1/products/{id}", auth(write(http.HandlerFunc(h.updateProduct))))
}

type productJSON struct {
	ID          uuid.UUID  `json:"id"`
	CategoryID  *uuid.UUID `json:"category_id"`
	Name        string     `json:"name"`
	SKU         string     `json:"sku"`
	Barcode     string     `json:"barcode"`
	BasePrice   float64    `json:"base_price"`
	SalePrice   *float64   `json:"sale_price"`
	Description string     `json:"description"`
	Unit        string     `json:"unit"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   string     `json:"created_at"`
	UpdatedAt   string     `json:"updated_at"`
}

func toJSON(p *domain.Product) productJSON {
	return productJSON{p.ID, p.CategoryID, p.Name, p.SKU, p.Barcode, p.BasePrice, p.SalePrice,
		p.Description, p.Unit, p.IsActive, p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"), p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")}
}

// tenant returns the tenant id from the token claims; writes 401 when absent.
func tenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	c, ok := authn.FromContext(r.Context())
	id, err := uuid.Parse(c.TenantID)
	if !ok || err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return uuid.Nil, false
	}
	return id, true
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidArgument):
		httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid product data")
	case errors.Is(err, domain.ErrAlreadyExists):
		httpx.WriteError(w, http.StatusConflict, "product sku already exists")
	case errors.Is(err, domain.ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "product not found")
	default:
		slog.ErrorContext(r.Context(), "product request failed", "error", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal error")
	}
}

func (h *HTTPHandler) list(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenant(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	category := q.Get("category_id")
	if category != "" {
		if _, err := uuid.Parse(category); err != nil {
			httpx.WriteError(w, http.StatusUnprocessableEntity, "invalid category_id")
			return
		}
	}
	page, per := atoiOr(q.Get("page"), 1), atoiOr(q.Get("per_page"), 20)
	if page < 1 {
		page = 1
	}
	if per < 1 {
		per = 20
	}
	if per > 100 {
		per = 100
	}
	res, err := h.search.Handle(r.Context(), queries.SearchProductsQuery{
		TenantID: tid, Query: q.Get("q"), Category: category, Limit: per, Offset: (page - 1) * per,
	})
	if err != nil {
		fail(w, r, err)
		return
	}
	items := make([]productJSON, 0, len(res.Products))
	for _, p := range res.Products {
		items = append(items, toJSON(p))
	}
	httpx.WriteSuccess(w, http.StatusOK, map[string]any{"items": items, "page": page, "per_page": per, "total": res.Total})
}

func atoiOr(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func (h *HTTPHandler) getOne(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "product not found")
		return
	}
	p, err := h.get.GetByID(r.Context(), tid, id)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, toJSON(p))
}

func (h *HTTPHandler) createProduct(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenant(w, r)
	if !ok {
		return
	}
	var body struct {
		CategoryID  *uuid.UUID `json:"category_id"`
		Name        string     `json:"name"`
		SKU         string     `json:"sku"`
		Barcode     string     `json:"barcode"`
		BasePrice   float64    `json:"base_price"`
		Description string     `json:"description"`
		Unit        string     `json:"unit"`
	}
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	p, err := h.create.Handle(r.Context(), commands.CreateProductCommand{
		TenantID: tid, CategoryID: body.CategoryID, Name: body.Name, SKU: body.SKU,
		Barcode: body.Barcode, BasePrice: body.BasePrice, Description: body.Description, Unit: body.Unit,
	})
	if err != nil {
		fail(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/products/"+p.ID.String())
	httpx.WriteSuccess(w, http.StatusCreated, toJSON(p))
}

func (h *HTTPHandler) updateProduct(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenant(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, "product not found")
		return
	}
	var body struct {
		CategoryID  *uuid.UUID `json:"category_id"`
		Name        string     `json:"name"`
		Barcode     string     `json:"barcode"`
		BasePrice   float64    `json:"base_price"`
		SalePrice   *float64   `json:"sale_price"`
		Description string     `json:"description"`
		IsActive    bool       `json:"is_active"`
	}
	if !httpx.DecodeJSON(w, r, &body, maxBody) {
		return
	}
	p, err := h.update.Handle(r.Context(), commands.UpdateProductCommand{
		TenantID: tid, ProductID: id, CategoryID: body.CategoryID, Name: body.Name, Barcode: body.Barcode,
		BasePrice: body.BasePrice, SalePrice: body.SalePrice, Description: body.Description, IsActive: body.IsActive,
	})
	if err != nil {
		fail(w, r, err)
		return
	}
	httpx.WriteSuccess(w, http.StatusOK, toJSON(p))
}
