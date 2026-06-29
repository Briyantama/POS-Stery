package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	supplierv1 "github.com/pos-stery/pos-stery/gen/go/pos/supplier/v1"
	"github.com/pos-stery/pos-stery/services/api-gateway/internal/middleware"
)

// SupplierServicer defines the supplier-service operations used by SupplierHandlers.
type SupplierServicer interface {
	AddSupplier(ctx context.Context, req *supplierv1.AddSupplierRequest) (*supplierv1.AddSupplierResponse, error)
	ListSuppliers(ctx context.Context, req *supplierv1.ListSuppliersRequest) (*supplierv1.ListSuppliersResponse, error)
	CreatePurchaseOrder(ctx context.Context, req *supplierv1.CreatePurchaseOrderRequest) (*supplierv1.CreatePurchaseOrderResponse, error)
}

// SupplierHandlers holds the supplier-related HTTP handlers.
type SupplierHandlers struct {
	supplier SupplierServicer
}

// NewSupplierHandlers constructs SupplierHandlers.
func NewSupplierHandlers(supplier SupplierServicer) *SupplierHandlers {
	return &SupplierHandlers{supplier: supplier}
}

// List handles GET /api/suppliers.
func (h *SupplierHandlers) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())

	resp, err := h.supplier.ListSuppliers(r.Context(), &supplierv1.ListSuppliersRequest{
		TenantId: tenantID,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, resp)
}

type addSupplierBody struct {
	Name    string `json:"name"`
	Contact string `json:"contact"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

// Add handles POST /api/suppliers.
func (h *SupplierHandlers) Add(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())

	var body addSupplierBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		validationError(w, "body", "invalid JSON")
		return
	}
	if body.Name == "" {
		validationError(w, "name", "name is required")
		return
	}

	resp, err := h.supplier.AddSupplier(r.Context(), &supplierv1.AddSupplierRequest{
		TenantId: tenantID,
		Name:     body.Name,
		Contact:  body.Contact,
		Phone:    body.Phone,
		Email:    body.Email,
		Address:  body.Address,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusCreated, resp)
}

type purchaseOrderItemBody struct {
	ProductID       string  `json:"product_id"`
	QuantityOrdered int32   `json:"quantity_ordered"`
	UnitCost        float64 `json:"unit_cost"`
}

type createPurchaseOrderBody struct {
	SupplierID string                  `json:"supplier_id"`
	Items      []purchaseOrderItemBody `json:"items"`
	Notes      string                  `json:"notes"`
}

// CreatePurchaseOrder handles POST /api/purchase-orders.
func (h *SupplierHandlers) CreatePurchaseOrder(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromCtx(ctx)
	storeID := middleware.StoreIDFromCtx(ctx)

	var body createPurchaseOrderBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		validationError(w, "body", "invalid JSON")
		return
	}
	if body.SupplierID == "" {
		validationError(w, "supplier_id", "supplier_id is required")
		return
	}
	if len(body.Items) == 0 {
		validationError(w, "items", "at least one item is required")
		return
	}

	items := make([]*supplierv1.PurchaseOrderItem, len(body.Items))
	for i, it := range body.Items {
		items[i] = &supplierv1.PurchaseOrderItem{
			ProductId:       it.ProductID,
			QuantityOrdered: it.QuantityOrdered,
			UnitCost:        it.UnitCost,
		}
	}

	resp, err := h.supplier.CreatePurchaseOrder(ctx, &supplierv1.CreatePurchaseOrderRequest{
		TenantId:   tenantID,
		StoreId:    storeID,
		SupplierId: body.SupplierID,
		Items:      items,
		Notes:      body.Notes,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusCreated, resp)
}
