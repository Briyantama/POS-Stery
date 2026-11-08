package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Briyantama/POS-Stery/services/inventory/domain"
	"github.com/Briyantama/stery-kit/go/events"
	"github.com/google/uuid"
)

// Event types published to the pos:events stream.
const (
	TypeLowStock      = "Product.LowStock"
	TypeReplenished   = "Inventory.Replenished"
	envelopeSchemaVer = 1
)

type lowStockPayload struct {
	StoreID     string `json:"store_id"`
	ProductID   string `json:"product_id"`
	Quantity    int64  `json:"quantity"`
	MinQuantity int64  `json:"min_quantity"`
}

type replenishedPayload struct {
	StoreID   string `json:"store_id"`
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

// EventPublisher publishes inventory domain events to the shared event stream.
type EventPublisher struct {
	pub *events.Publisher
}

// NewEventPublisher creates an EventPublisher.
func NewEventPublisher(pub *events.Publisher) *EventPublisher {
	return &EventPublisher{pub: pub}
}

// PublishLowStock publishes a Product.LowStock event.
func (p *EventPublisher) PublishLowStock(ctx context.Context, e domain.LowStockDetected) error {
	return p.publish(ctx, TypeLowStock, e.TenantID.String(), lowStockPayload{
		StoreID:     e.StoreID.String(),
		ProductID:   e.ProductID.String(),
		Quantity:    e.Quantity,
		MinQuantity: e.MinQuantity,
	})
}

// PublishReplenished publishes an Inventory.Replenished event.
func (p *EventPublisher) PublishReplenished(ctx context.Context, tenantID, storeID, productID string, quantity int64) error {
	return p.publish(ctx, TypeReplenished, tenantID, replenishedPayload{
		StoreID:   storeID,
		ProductID: productID,
		Quantity:  quantity,
	})
}

func (p *EventPublisher) publish(ctx context.Context, eventType, tenantID string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", eventType, err)
	}
	if err := p.pub.Publish(ctx, events.Envelope{
		EventID:       uuid.NewString(),
		EventType:     eventType,
		SchemaVersion: envelopeSchemaVer,
		TenantID:      tenantID,
		OccurredAt:    time.Now().UTC(),
		Payload:       data,
	}); err != nil {
		return fmt.Errorf("publish %s: %w", eventType, err)
	}
	return nil
}
