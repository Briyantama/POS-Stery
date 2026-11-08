package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Briyantama/stery-kit/go/events"
	"github.com/google/uuid"
)

// SubjectInventoryReplenished is the event type of the stock-receipt event.
const SubjectInventoryReplenished = "Inventory.Replenished"

// replenishedPayload is the JSON payload published for Inventory.Replenished events.
type replenishedPayload struct {
	StoreID         string `json:"store_id"`
	PurchaseOrderID string `json:"purchase_order_id"`
	ProductID       string `json:"product_id"`
	Quantity        int64  `json:"quantity"`
}

// EventPublisher publishes supplier domain events to the event stream.
type EventPublisher struct {
	pub *events.Publisher
}

// NewEventPublisher creates an EventPublisher.
func NewEventPublisher(pub *events.Publisher) *EventPublisher {
	return &EventPublisher{pub: pub}
}

// PublishReplenished publishes an Inventory.Replenished event after stock is received.
func (p *EventPublisher) PublishReplenished(
	ctx context.Context,
	tenantID, storeID, purchaseOrderID, productID string,
	quantity int64,
) error {
	payload, err := json.Marshal(replenishedPayload{
		StoreID:         storeID,
		PurchaseOrderID: purchaseOrderID,
		ProductID:       productID,
		Quantity:        quantity,
	})
	if err != nil {
		return fmt.Errorf("marshal replenished payload: %w", err)
	}
	if err := p.pub.Publish(ctx, events.Envelope{
		EventID:       uuid.NewString(),
		EventType:     SubjectInventoryReplenished,
		SchemaVersion: 1,
		TenantID:      tenantID,
		OccurredAt:    time.Now().UTC(),
		Payload:       payload,
	}); err != nil {
		return fmt.Errorf("publish replenished: %w", err)
	}
	return nil
}
