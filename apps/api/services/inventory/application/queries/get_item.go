package queries

import (
	"context"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/inventory/domain"
	"uuid"
)

// GetItemQuery identifies the stock item to retrieve.
type GetItemQuery struct {
	TenantID  uuid.UUID
	StoreID   uuid.UUID
	ProductID uuid.UUID
}

// GetItemHandler processes GetItemQuery.
type GetItemHandler struct {
	stockRepo domain.StockRepository
}

// NewGetItemHandler creates a GetItemHandler.
func NewGetItemHandler(stockRepo domain.StockRepository) *GetItemHandler {
	return &GetItemHandler{stockRepo: stockRepo}
}

// Handle executes the GetItem query.
func (h *GetItemHandler) Handle(ctx context.Context, qry GetItemQuery) (*domain.StockItem, error) {
	if qry.TenantID == uuid.Nil() {
		return nil, fmt.Errorf("tenantID required: %w", domain.ErrInvalidArgument)
	}
	if qry.StoreID == uuid.Nil() {
		return nil, fmt.Errorf("storeID required: %w", domain.ErrInvalidArgument)
	}
	if qry.ProductID == uuid.Nil() {
		return nil, fmt.Errorf("productID required: %w", domain.ErrInvalidArgument)
	}

	item, err := h.stockRepo.GetItem(ctx, qry.TenantID, qry.StoreID, qry.ProductID)
	if err != nil {
		return nil, fmt.Errorf("get item: %w", err)
	}
	return item, nil
}
