package application

import "context"

// EventPublisher abstracts outbound event stream publishing.
type EventPublisher interface {
	// PublishReplenished publishes an Inventory.Replenished event after stock receipt.
	PublishReplenished(
		ctx context.Context,
		tenantID, storeID, purchaseOrderID, productID string,
		quantity int64,
	) error
}
