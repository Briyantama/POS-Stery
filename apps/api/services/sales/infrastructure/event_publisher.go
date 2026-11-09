package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Briyantama/POS-Stery/services/sales/domain"
	"github.com/Briyantama/stery-kit/go/events"
	"uuid"
)

// SaleCompletedEventType is the stream `type` of the Sale.Completed event.
const SaleCompletedEventType = "Sale.Completed"

// EventPublisher implements application.EventPublisher on the Dragonfly event stream.
type EventPublisher struct {
	pub *events.Publisher
}

// NewEventPublisher wraps the shared stream publisher.
func NewEventPublisher(pub *events.Publisher) *EventPublisher {
	return &EventPublisher{pub: pub}
}

// PublishSaleCompleted emits Sale.Completed. Callers treat failure as best-effort.
func (p *EventPublisher) PublishSaleCompleted(ctx context.Context, sale *domain.Sale) error {
	payload, err := json.Marshal(struct {
		SaleID    string  `json:"sale_id"`
		StoreID   string  `json:"store_id"`
		CashierID string  `json:"cashier_id"`
		Total     float64 `json:"total"`
		Status    string  `json:"status"`
	}{sale.ID.String(), sale.StoreID.String(), sale.CashierID.String(), sale.Total, sale.Status})
	if err != nil {
		return fmt.Errorf("marshal Sale.Completed: %w", err)
	}
	if err := p.pub.Publish(ctx, events.Envelope{
		EventID:       uuid.NewV7().String(),
		EventType:     SaleCompletedEventType,
		SchemaVersion: 1,
		TenantID:      sale.TenantID.String(),
		ActorID:       sale.CashierID.String(),
		OccurredAt:    time.Now().UTC(),
		Payload:       payload,
	}); err != nil {
		return fmt.Errorf("publish Sale.Completed: %w", err)
	}
	return nil
}
