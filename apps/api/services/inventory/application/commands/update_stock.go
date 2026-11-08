package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Briyantama/POS-Stery/services/inventory/application"
	"github.com/Briyantama/POS-Stery/services/inventory/domain"
	"github.com/google/uuid"
)

// UpdateStockCommand carries the data needed to adjust a product's stock level.
type UpdateStockCommand struct {
	TenantID  uuid.UUID
	StoreID   uuid.UUID
	ProductID uuid.UUID
	Delta     int64  // positive = add, negative = remove
	Reason    string // SALE | RECEIPT | ADJUSTMENT
	RefID     string // order ID, PO ID, etc.
}

// UpdateStockResult is the return value of UpdateStockHandler.Handle.
type UpdateStockResult struct {
	Item          *domain.StockItem
	LowStockAlert bool
}

// UpdateStockHandler processes UpdateStockCommand.
type UpdateStockHandler struct {
	stockRepo     domain.StockRepository
	thresholdRepo domain.ThresholdRepository
	publisher     application.EventPublisher
}

// NewUpdateStockHandler creates an UpdateStockHandler with the required dependencies.
func NewUpdateStockHandler(
	stockRepo domain.StockRepository,
	thresholdRepo domain.ThresholdRepository,
	publisher application.EventPublisher,
) *UpdateStockHandler {
	return &UpdateStockHandler{
		stockRepo:     stockRepo,
		thresholdRepo: thresholdRepo,
		publisher:     publisher,
	}
}

// Handle executes the UpdateStock use case.
//
// Business rules enforced:
//  1. Stock must not go negative.
//  2. If the updated quantity is at or below the configured threshold, a
//     LowStockDetected event is published.
func (h *UpdateStockHandler) Handle(ctx context.Context, cmd UpdateStockCommand) (UpdateStockResult, error) {
	if cmd.TenantID == uuid.Nil {
		return UpdateStockResult{}, fmt.Errorf("tenantID required: %w", domain.ErrInvalidArgument)
	}
	if cmd.StoreID == uuid.Nil {
		return UpdateStockResult{}, fmt.Errorf("storeID required: %w", domain.ErrInvalidArgument)
	}
	if cmd.ProductID == uuid.Nil {
		return UpdateStockResult{}, fmt.Errorf("productID required: %w", domain.ErrInvalidArgument)
	}
	if cmd.Delta == 0 {
		return UpdateStockResult{}, fmt.Errorf("delta must be non-zero: %w", domain.ErrInvalidArgument)
	}

	// 1. Atomically apply the delta in Postgres; the repo rejects a result below zero
	//    (a read-then-write pre-check here would race with concurrent sales).
	updated, err := h.stockRepo.UpdateStock(
		ctx,
		cmd.TenantID, cmd.StoreID, cmd.ProductID,
		cmd.Delta,
		cmd.Reason,
		cmd.RefID,
	)
	if err != nil {
		return UpdateStockResult{}, fmt.Errorf("update stock: %w", err)
	}

	// 4. Load threshold; if absent we skip the alert.
	threshold, err := h.thresholdRepo.GetThreshold(ctx, cmd.TenantID, cmd.StoreID, cmd.ProductID)
	if err != nil {
		// No threshold configured — not an error, just skip low-stock logic.
		return UpdateStockResult{Item: updated, LowStockAlert: false}, nil
	}

	// 5. Emit low-stock event when quantity is at or below the threshold.
	if updated.Quantity <= threshold.MinQuantity {
		event := domain.LowStockDetected{
			TenantID:    cmd.TenantID,
			StoreID:     cmd.StoreID,
			ProductID:   cmd.ProductID,
			Quantity:    updated.Quantity,
			MinQuantity: threshold.MinQuantity,
		}
		if pubErr := h.publisher.PublishLowStock(ctx, event); pubErr != nil {
			// Publishing failure is non-fatal: the stock change is already committed.
			slog.ErrorContext(ctx, "publish low stock event", "tenant_id", cmd.TenantID, "error", pubErr)
		}
		return UpdateStockResult{Item: updated, LowStockAlert: true}, nil
	}

	return UpdateStockResult{Item: updated, LowStockAlert: false}, nil
}
