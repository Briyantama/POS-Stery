package application

import (
	"context"

	"github.com/pos-stery/pos-stery/services/sales-service/internal/domain"
)

// InventoryUpdater is the outbound port for stock adjustments.
// Implementations call the inventory-service gRPC endpoint.
type InventoryUpdater interface {
	// DeductStock reduces stock for a product in a store.
	// Returns ErrStockWouldGoNegative when the deduction would make stock < 0.
	DeductStock(ctx context.Context, tenantID, storeID, productID string, quantity int64, saleID string) error

	// RestoreStock increases stock to compensate a failed deduction.
	RestoreStock(ctx context.Context, tenantID, storeID, productID string, quantity int64, saleID string) error
}

// EventPublisher is the outbound port for domain event publishing.
type EventPublisher interface {
	// PublishSaleCompleted emits a Sale.Completed event to NATS JetStream.
	// Callers treat publish failures as best-effort — the sale is not rolled back.
	PublishSaleCompleted(ctx context.Context, sale *domain.Sale) error
}
