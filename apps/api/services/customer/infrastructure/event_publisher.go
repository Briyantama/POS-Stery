// Package infrastructure holds the customer service's outbound adapters.
package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Briyantama/stery-kit/go/events"
	"github.com/google/uuid"
)

// SubjectCustomerNew is the event type of a newly created customer.
const SubjectCustomerNew = "Customer.New"

type customerNewPayload struct {
	CustomerID string `json:"customer_id"`
	TenantID   string `json:"tenant_id"`
	Name       string `json:"name"`
}

// EventPublisher implements application.EventPublisher on the shared event stream.
type EventPublisher struct {
	pub *events.Publisher
}

// NewEventPublisher wraps the stream publisher.
func NewEventPublisher(pub *events.Publisher) *EventPublisher {
	return &EventPublisher{pub: pub}
}

// PublishCustomerNew publishes a Customer.New event.
func (p *EventPublisher) PublishCustomerNew(ctx context.Context, customerID, tenantID, name string) error {
	payload, err := json.Marshal(customerNewPayload{CustomerID: customerID, TenantID: tenantID, Name: name})
	if err != nil {
		return fmt.Errorf("marshal customer new: %w", err)
	}
	if err := p.pub.Publish(ctx, events.Envelope{
		EventID:       uuid.NewString(),
		EventType:     SubjectCustomerNew,
		SchemaVersion: 1,
		TenantID:      tenantID,
		OccurredAt:    time.Now().UTC(),
		Payload:       payload,
	}); err != nil {
		return fmt.Errorf("publish customer new: %w", err)
	}
	return nil
}
