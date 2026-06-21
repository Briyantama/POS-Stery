package infrastructure

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	sharednats "github.com/pos-stery/pos-stery/services/_shared/nats"
)

type customerNewPayload struct {
	CustomerID string `json:"customer_id"`
	TenantID   string `json:"tenant_id"`
	Name       string `json:"name"`
}

// NATSPublisher implements application.EventPublisher using NATS JetStream.
type NATSPublisher struct {
	js nats.JetStreamContext
}

// NewNATSPublisher creates a NATSPublisher backed by the given JetStream context.
func NewNATSPublisher(js nats.JetStreamContext) *NATSPublisher {
	return &NATSPublisher{js: js}
}

// PublishCustomerNew publishes a Customer.New event.
func (p *NATSPublisher) PublishCustomerNew(_ context.Context, customerID, tenantID, name string) error {
	payload := customerNewPayload{
		CustomerID: customerID,
		TenantID:   tenantID,
		Name:       name,
	}
	if err := sharednats.Publish(p.js, sharednats.SubjectCustomerNew, tenantID, "", payload); err != nil {
		return fmt.Errorf("publish customer new: %w", err)
	}
	return nil
}
