package infrastructure

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	sharednats "github.com/pos-stery/pos-stery/services/_shared/nats"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/domain"
)

// lowStockPayload is the JSON payload published for Product.LowStock events.
type lowStockPayload struct {
	ProductID   string `json:"product_id"`
	Quantity    int64  `json:"quantity"`
	MinQuantity int64  `json:"min_quantity"`
}

// replenishedPayload is the JSON payload published for Inventory.Replenished events.
type replenishedPayload struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

// NATSPublisher publishes inventory domain events to NATS JetStream.
type NATSPublisher struct {
	js nats.JetStreamContext
}

// NewNATSPublisher creates a NATSPublisher.
func NewNATSPublisher(js nats.JetStreamContext) *NATSPublisher {
	return &NATSPublisher{js: js}
}

// PublishLowStock publishes a Product.LowStock event.
func (p *NATSPublisher) PublishLowStock(_ context.Context, event domain.LowStockDetected) error {
	payload := lowStockPayload{
		ProductID:   event.ProductID.String(),
		Quantity:    event.Quantity,
		MinQuantity: event.MinQuantity,
	}
	if err := sharednats.Publish(
		p.js,
		sharednats.SubjectLowStock,
		event.TenantID.String(),
		event.StoreID.String(),
		payload,
	); err != nil {
		return fmt.Errorf("publish low stock: %w", err)
	}
	return nil
}

// PublishReplenished publishes an Inventory.Replenished event.
func (p *NATSPublisher) PublishReplenished(_ context.Context, tenantID, storeID, productID string, quantity int64) error {
	payload := replenishedPayload{
		ProductID: productID,
		Quantity:  quantity,
	}
	if err := sharednats.Publish(
		p.js,
		sharednats.SubjectInventoryReplenished,
		tenantID,
		storeID,
		payload,
	); err != nil {
		return fmt.Errorf("publish replenished: %w", err)
	}
	return nil
}
