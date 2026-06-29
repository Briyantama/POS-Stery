package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	inventoryv1 "github.com/pos-stery/pos-stery/gen/go/pos/inventory/v1"
	"github.com/pos-stery/pos-stery/services/api-gateway/internal/middleware"
)

// InventoryServicer defines the inventory-service operations used by InventoryHandlers.
type InventoryServicer interface {
	AddItem(ctx context.Context, req *inventoryv1.AddItemRequest) (*inventoryv1.AddItemResponse, error)
	ListItems(ctx context.Context, req *inventoryv1.ListItemsRequest) (*inventoryv1.ListItemsResponse, error)
}

// InventoryHandlers holds the inventory-related HTTP handlers.
type InventoryHandlers struct {
	inventory InventoryServicer
}

// NewInventoryHandlers constructs InventoryHandlers.
func NewInventoryHandlers(inventory InventoryServicer) *InventoryHandlers {
	return &InventoryHandlers{inventory: inventory}
}

// List handles GET /api/inventory.
func (h *InventoryHandlers) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	storeID := middleware.StoreIDFromCtx(r.Context())

	resp, err := h.inventory.ListItems(r.Context(), &inventoryv1.ListItemsRequest{
		TenantId: tenantID,
		StoreId:  storeID,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, resp)
}

type addItemBody struct {
	ProductID   string `json:"product_id"`
	Quantity    int64  `json:"quantity"`
	MinQuantity int64  `json:"min_quantity"`
}

// Add handles POST /api/inventory.
func (h *InventoryHandlers) Add(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())
	storeID := middleware.StoreIDFromCtx(r.Context())

	var body addItemBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		validationError(w, "body", "invalid JSON")
		return
	}
	if body.ProductID == "" {
		validationError(w, "product_id", "product_id is required")
		return
	}

	resp, err := h.inventory.AddItem(r.Context(), &inventoryv1.AddItemRequest{
		TenantId:    tenantID,
		StoreId:     storeID,
		ProductId:   body.ProductID,
		Quantity:    body.Quantity,
		MinQuantity: body.MinQuantity,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusCreated, resp)
}
