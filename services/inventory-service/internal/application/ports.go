package application

import (
	"context"

	"github.com/pos-stery/pos-stery/services/inventory-service/internal/domain"
)

// EventPublisher abstracts outbound event publishing so the application layer
// remains free of infrastructure dependencies.
type EventPublisher interface {
	PublishLowStock(ctx context.Context, event domain.LowStockDetected) error
	PublishReplenished(ctx context.Context, tenantID, storeID, productID string, quantity int64) error
}
