package infrastructure

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	sharednats "github.com/pos-stery/pos-stery/services/_shared/nats"
)

// replenishedPayload is the JSON payload published for Inventory.Replenished events.
type replenishedPayload struct {
	PurchaseOrderID string `json:"purchase_order_id"`
	ProductID       string `json:"product_id"`
	Quantity        int64  `json:"quantity"`
}

// NATSPublisher publishes supplier domain events to NATS JetStream.
type NATSPublisher struct {
	js nats.JetStreamContext
}

// NewNATSPublisher creates a NATSPublisher.
func NewNATSPublisher(js nats.JetStreamContext) *NATSPublisher {
	return &NATSPublisher{js: js}
}

// PublishReplenished publishes an Inventory.Replenished event after stock is received.
func (p *NATSPublisher) PublishReplenished(
	_ context.Context,
	tenantID, storeID, purchaseOrderID, productID string,
	quantity int64,
) error {
	payload := replenishedPayload{
		PurchaseOrderID: purchaseOrderID,
		ProductID:       productID,
		Quantity:        quantity,
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
