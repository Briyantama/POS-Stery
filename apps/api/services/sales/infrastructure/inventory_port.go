package infrastructure

import (
	"context"
	"errors"
	"fmt"

	invcommands "github.com/Briyantama/POS-Stery/services/inventory/application/commands"
	invdomain "github.com/Briyantama/POS-Stery/services/inventory/domain"
	"github.com/Briyantama/POS-Stery/services/sales/domain"
	"uuid"
)

// InventoryPort implements application.InventoryUpdater by calling the inventory
// service's UpdateStockHandler in-process.
type InventoryPort struct {
	stock *invcommands.UpdateStockHandler
}

// NewInventoryPort wires the port to inventory's UpdateStock use case.
func NewInventoryPort(stock *invcommands.UpdateStockHandler) *InventoryPort {
	return &InventoryPort{stock: stock}
}

// DeductStock applies a negative delta with reason SALE.
// Returns domain.ErrStockWouldGoNegative when stock would drop below zero.
func (p *InventoryPort) DeductStock(ctx context.Context, tenantID, storeID, productID string, quantity int64, saleID string) error {
	return mapInventoryError(p.update(ctx, tenantID, storeID, productID, -quantity, "SALE", saleID))
}

// RestoreStock applies a positive delta with reason ADJUSTMENT (saga compensation).
func (p *InventoryPort) RestoreStock(ctx context.Context, tenantID, storeID, productID string, quantity int64, saleID string) error {
	if err := p.update(ctx, tenantID, storeID, productID, quantity, "ADJUSTMENT", saleID); err != nil {
		return fmt.Errorf("restore stock for product %s: %w", productID, err)
	}
	return nil
}

func (p *InventoryPort) update(ctx context.Context, tenantID, storeID, productID string, delta int64, reason, refID string) error {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("%w: invalid tenant_id", domain.ErrInvalidArgument)
	}
	sid, err := uuid.Parse(storeID)
	if err != nil {
		return fmt.Errorf("%w: invalid store_id", domain.ErrInvalidArgument)
	}
	pid, err := uuid.Parse(productID)
	if err != nil {
		return fmt.Errorf("%w: invalid product_id", domain.ErrInvalidArgument)
	}
	_, err = p.stock.Handle(ctx, invcommands.UpdateStockCommand{
		TenantID: tid, StoreID: sid, ProductID: pid, Delta: delta, Reason: reason, RefID: refID,
	})
	return err
}

// mapInventoryError translates inventory domain errors into sales domain errors.
func mapInventoryError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, invdomain.ErrStockWouldGoNegative):
		return fmt.Errorf("%w: %w", domain.ErrStockWouldGoNegative, err)
	case errors.Is(err, invdomain.ErrNotFound):
		return fmt.Errorf("%w: %w", domain.ErrNotFound, err)
	default:
		return fmt.Errorf("inventory call: %w", err)
	}
}
