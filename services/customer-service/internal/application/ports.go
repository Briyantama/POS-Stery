package application

import "context"

// EventPublisher abstracts outbound NATS event publishing so the application
// layer remains free of infrastructure dependencies.
type EventPublisher interface {
	// PublishCustomerNew emits a Customer.New event after a customer is persisted.
	PublishCustomerNew(ctx context.Context, customerID, tenantID, name string) error
}
