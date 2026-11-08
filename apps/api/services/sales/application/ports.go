package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/Briyantama/POS-Stery/services/sales/domain"
)

// InventoryUpdater is the outbound port for stock adjustments.
// Implementations call inventory in-process.
type InventoryUpdater interface {
	// DeductStock reduces stock for a product in a store.
	// Returns ErrStockWouldGoNegative when the deduction would make stock < 0.
	DeductStock(ctx context.Context, tenantID, storeID, productID string, quantity int64, saleID string) error

	// RestoreStock increases stock to compensate a failed deduction.
	RestoreStock(ctx context.Context, tenantID, storeID, productID string, quantity int64, saleID string) error
}

// PriceReader is the outbound port for the authoritative selling price of a product.
type PriceReader interface {
	// UnitPrice returns the current price of an active product.
	// Returns ErrNotFound for unknown products and ErrInvalidArgument for inactive ones.
	UnitPrice(ctx context.Context, tenantID, productID uuid.UUID) (float64, error)
}

// EventPublisher is the outbound port for domain event publishing.
type EventPublisher interface {
	// PublishSaleCompleted emits a Sale.Completed event to the Dragonfly event stream.
	// Callers treat publish failures as best-effort — the sale is not rolled back.
	PublishSaleCompleted(ctx context.Context, sale *domain.Sale) error
}
