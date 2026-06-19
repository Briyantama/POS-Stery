package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/domain"
)

// SetThresholdCommand carries the data needed to configure a low-stock threshold.
type SetThresholdCommand struct {
	TenantID    uuid.UUID
	StoreID     uuid.UUID
	ProductID   uuid.UUID
	MinQuantity int64
}

// SetThresholdHandler processes SetThresholdCommand.
type SetThresholdHandler struct {
	thresholdRepo domain.ThresholdRepository
}

// NewSetThresholdHandler creates a SetThresholdHandler.
func NewSetThresholdHandler(thresholdRepo domain.ThresholdRepository) *SetThresholdHandler {
	return &SetThresholdHandler{thresholdRepo: thresholdRepo}
}

// Handle executes the SetThreshold use case.
func (h *SetThresholdHandler) Handle(ctx context.Context, cmd SetThresholdCommand) error {
	if cmd.TenantID == uuid.Nil {
		return fmt.Errorf("tenantID required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.StoreID == uuid.Nil {
		return fmt.Errorf("storeID required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.ProductID == uuid.Nil {
		return fmt.Errorf("productID required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.MinQuantity < 0 {
		return fmt.Errorf("min_quantity must be >= 0: %w", sherrors.ErrInvalidArgument)
	}

	threshold := &domain.StockThreshold{
		TenantID:    cmd.TenantID,
		StoreID:     cmd.StoreID,
		ProductID:   cmd.ProductID,
		MinQuantity: cmd.MinQuantity,
	}

	if err := h.thresholdRepo.SetThreshold(ctx, threshold); err != nil {
		return fmt.Errorf("set threshold: %w", err)
	}
	return nil
}
