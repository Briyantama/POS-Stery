package infrastructure

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	invcommands "github.com/Briyantama/POS-Stery/services/inventory/application/commands"
	invdomain "github.com/Briyantama/POS-Stery/services/inventory/domain"
	"github.com/Briyantama/POS-Stery/services/supplier/domain"
)

// InventoryPort implements application.InventoryPort by calling the inventory service in-process.
type InventoryPort struct {
	stock *invcommands.UpdateStockHandler
}

// NewInventoryPort wraps inventory's UpdateStockHandler.
func NewInventoryPort(stock *invcommands.UpdateStockHandler) *InventoryPort {
	return &InventoryPort{stock: stock}
}

// UpdateStock applies delta (positive for a RECEIPT) to the product's stock.
func (p *InventoryPort) UpdateStock(
	ctx context.Context,
	tenantID, storeID, productID string,
	delta int64,
	reason, refID string,
) error {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return fmt.Errorf("tenant id: %w", domain.ErrInvalidArgument)
	}
	sid, err := uuid.Parse(storeID)
	if err != nil {
		return fmt.Errorf("store id: %w", domain.ErrInvalidArgument)
	}
	pid, err := uuid.Parse(productID)
	if err != nil {
		return fmt.Errorf("product id: %w", domain.ErrInvalidArgument)
	}
	if _, err := p.stock.Handle(ctx, invcommands.UpdateStockCommand{
		TenantID: tid, StoreID: sid, ProductID: pid, Delta: delta, Reason: reason, RefID: refID,
	}); err != nil {
		if errors.Is(err, invdomain.ErrNotFound) {
			return fmt.Errorf("inventory item: %w", domain.ErrNotFound)
		}
		return fmt.Errorf("inventory update stock: %w", err)
	}
	return nil
}
