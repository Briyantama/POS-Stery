package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	productv1 "github.com/pos-stery/pos-stery/gen/go/pos/product/v1"
	"github.com/pos-stery/pos-stery/services/api-gateway/internal/middleware"
	"github.com/go-chi/chi/v5"
)

// ProductServicer defines the product-service operations used by ProductHandlers.
type ProductServicer interface {
	CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.CreateProductResponse, error)
	UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.UpdateProductResponse, error)
	SearchProducts(ctx context.Context, req *productv1.SearchProductsRequest) (*productv1.SearchProductsResponse, error)
}

// ProductHandlers holds the product-related HTTP handlers.
type ProductHandlers struct {
	product ProductServicer
}

// NewProductHandlers constructs ProductHandlers.
func NewProductHandlers(product ProductServicer) *ProductHandlers {
	return &ProductHandlers{product: product}
}

// List handles GET /api/products.
func (h *ProductHandlers) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	q := r.URL.Query().Get("q")

	resp, err := h.product.SearchProducts(r.Context(), &productv1.SearchProductsRequest{
		TenantId: tenantID,
		Query:    q,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, resp)
}

type createProductBody struct {
	Name        string  `json:"name"`
	SKU         string  `json:"sku"`
	Barcode     string  `json:"barcode"`
	CategoryID  string  `json:"category_id"`
	BasePrice   float64 `json:"base_price"`
	Description string  `json:"description"`
	Unit        string  `json:"unit"`
}

// Create handles POST /api/products.
func (h *ProductHandlers) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())

	var body createProductBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		validationError(w, "body", "invalid JSON")
		return
	}
	if body.Name == "" {
		validationError(w, "name", "name is required")
		return
	}

	resp, err := h.product.CreateProduct(r.Context(), &productv1.CreateProductRequest{
		TenantId:    tenantID,
		Name:        body.Name,
		Sku:         body.SKU,
		Barcode:     body.Barcode,
		CategoryId:  body.CategoryID,
		BasePrice:   body.BasePrice,
		Description: body.Description,
		Unit:        body.Unit,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusCreated, resp)
}

type updateProductBody struct {
	Name        string  `json:"name"`
	BasePrice   float64 `json:"base_price"`
	Description string  `json:"description"`
}

// Update handles PUT /api/products/{id}.
func (h *ProductHandlers) Update(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	id := chi.URLParam(r, "id")

	var body updateProductBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		validationError(w, "body", "invalid JSON")
		return
	}

	resp, err := h.product.UpdateProduct(r.Context(), &productv1.UpdateProductRequest{
		TenantId:    tenantID,
		ProductId:   id,
		Name:        body.Name,
		BasePrice:   body.BasePrice,
		Description: body.Description,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, resp)
}
