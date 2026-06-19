package application

import "context"

// InventoryPort abstracts calls to the inventory-service so the application
// layer has no direct dependency on infrastructure (gRPC clients).
type InventoryPort interface {
	// UpdateStock adjusts the stock level for a product in the inventory-service.
	UpdateStock(
		ctx context.Context,
		tenantID, storeID, productID string,
		delta int64,
		reason string,
		refID string,
	) error
}

// EventPublisher abstracts outbound NATS event publishing.
type EventPublisher interface {
	// PublishReplenished publishes an Inventory.Replenished event after stock receipt.
	PublishReplenished(
		ctx context.Context,
		tenantID, storeID, purchaseOrderID, productID string,
		quantity int64,
	) error
}
