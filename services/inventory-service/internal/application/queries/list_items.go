package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/domain"
)

// ListItemsQuery describes the filtering and pagination parameters.
type ListItemsQuery struct {
	TenantID     uuid.UUID
	StoreID      uuid.UUID
	LowStockOnly bool
	Limit        int
	Offset       int
}

// ListItemsResult holds the paged result set.
type ListItemsResult struct {
	Items []*domain.StockItem
	Total int
}

// ListItemsHandler processes ListItemsQuery.
type ListItemsHandler struct {
	stockRepo domain.StockRepository
}

// NewListItemsHandler creates a ListItemsHandler.
func NewListItemsHandler(stockRepo domain.StockRepository) *ListItemsHandler {
	return &ListItemsHandler{stockRepo: stockRepo}
}

// Handle executes the ListItems query.
func (h *ListItemsHandler) Handle(ctx context.Context, qry ListItemsQuery) (ListItemsResult, error) {
	if qry.TenantID == uuid.Nil {
		return ListItemsResult{}, fmt.Errorf("tenantID required: %w", sherrors.ErrInvalidArgument)
	}
	if qry.StoreID == uuid.Nil {
		return ListItemsResult{}, fmt.Errorf("storeID required: %w", sherrors.ErrInvalidArgument)
	}

	limit := qry.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	items, total, err := h.stockRepo.ListItems(ctx, qry.TenantID, qry.StoreID, qry.LowStockOnly, limit, qry.Offset)
	if err != nil {
		return ListItemsResult{}, fmt.Errorf("list items: %w", err)
	}
	return ListItemsResult{Items: items, Total: total}, nil
}
