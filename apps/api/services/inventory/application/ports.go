package application

import (
	"context"

	"github.com/Briyantama/POS-Stery/services/inventory/domain"
)

// EventPublisher abstracts outbound event publishing so the application layer
// remains free of infrastructure dependencies.
type EventPublisher interface {
	PublishLowStock(ctx context.Context, event domain.LowStockDetected) error
	PublishReplenished(ctx context.Context, tenantID, storeID, productID string, quantity int64) error
}
