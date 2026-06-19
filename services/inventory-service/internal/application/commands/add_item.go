package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/domain"
)

// AddItemCommand carries the data needed to register a product in inventory.
type AddItemCommand struct {
	TenantID        uuid.UUID
	StoreID         uuid.UUID
	ProductID       uuid.UUID
	InitialQuantity int64
	MinQuantity     int64 // set threshold when > 0
}

// AddItemHandler processes AddItemCommand.
type AddItemHandler struct {
	stockRepo     domain.StockRepository
	thresholdRepo domain.ThresholdRepository
}

// NewAddItemHandler creates an AddItemHandler with the required dependencies.
func NewAddItemHandler(
	stockRepo domain.StockRepository,
	thresholdRepo domain.ThresholdRepository,
) *AddItemHandler {
	return &AddItemHandler{
		stockRepo:     stockRepo,
		thresholdRepo: thresholdRepo,
	}
}

// Handle executes the AddItem use case.
func (h *AddItemHandler) Handle(ctx context.Context, cmd AddItemCommand) (*domain.StockItem, error) {
	if cmd.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenantID required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.StoreID == uuid.Nil {
		return nil, fmt.Errorf("storeID required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.ProductID == uuid.Nil {
		return nil, fmt.Errorf("productID required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.InitialQuantity < 0 {
		return nil, fmt.Errorf("initial quantity must be >= 0: %w", sherrors.ErrInvalidArgument)
	}

	item := &domain.StockItem{
		TenantID:  cmd.TenantID,
		StoreID:   cmd.StoreID,
		ProductID: cmd.ProductID,
		Quantity:  cmd.InitialQuantity,
	}

	if err := h.stockRepo.AddItem(ctx, item); err != nil {
		return nil, fmt.Errorf("add stock item: %w", err)
	}

	if cmd.MinQuantity > 0 {
		threshold := &domain.StockThreshold{
			TenantID:    cmd.TenantID,
			StoreID:     cmd.StoreID,
			ProductID:   cmd.ProductID,
			MinQuantity: cmd.MinQuantity,
		}
		if err := h.thresholdRepo.SetThreshold(ctx, threshold); err != nil {
			return nil, fmt.Errorf("set threshold: %w", err)
		}
	}

	return item, nil
}
