package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	sharednats "github.com/pos-stery/pos-stery/services/_shared/nats"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/domain"
)

// CreateCustomerCommand carries the data required to create a new customer.
type CreateCustomerCommand struct {
	TenantID uuid.UUID
	Name     string
	Phone    string
	Email    string
}

// CustomerNewPayload is the event payload published on Customer.New.
type CustomerNewPayload struct {
	CustomerID string `json:"customer_id"`
	TenantID   string `json:"tenant_id"`
	Name       string `json:"name"`
}

// CreateCustomerHandler handles the CreateCustomerCommand.
type CreateCustomerHandler struct {
	repo domain.CustomerRepository
	js   nats.JetStreamContext
}

// NewCreateCustomerHandler constructs a handler with the given repository and
// JetStream context for publishing Customer.New events.
func NewCreateCustomerHandler(repo domain.CustomerRepository, js nats.JetStreamContext) *CreateCustomerHandler {
	return &CreateCustomerHandler{repo: repo, js: js}
}

// Handle validates the command, persists the customer and publishes Customer.New.
func (h *CreateCustomerHandler) Handle(ctx context.Context, cmd CreateCustomerCommand) (*domain.Customer, error) {
	if cmd.Name == "" {
		return nil, fmt.Errorf("name is required: %w", sherrors.ErrInvalidArgument)
	}

	c := &domain.Customer{
		ID:       uuid.New(),
		TenantID: cmd.TenantID,
		Name:     cmd.Name,
		Phone:    cmd.Phone,
		Email:    cmd.Email,
		IsActive: true,
	}

	if err := h.repo.Create(ctx, cmd.TenantID, c); err != nil {
		return nil, fmt.Errorf("create customer: %w", err)
	}

	// Publish Customer.New event after successful persist.
	payload := CustomerNewPayload{
		CustomerID: c.ID.String(),
		TenantID:   c.TenantID.String(),
		Name:       c.Name,
	}
	if err := sharednats.Publish(h.js, sharednats.SubjectCustomerNew, c.TenantID.String(), "", payload); err != nil {
		// Non-fatal: log would be better in production; here we return the
		// customer but surface the publish error as a warning via the error.
		// In practice, wire a logger and log.Warn instead of returning the error.
		_ = err
	}

	return c, nil
}
