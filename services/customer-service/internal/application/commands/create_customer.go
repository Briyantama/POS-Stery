package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/application"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/domain"
)

// CreateCustomerCommand carries the data required to create a new customer.
type CreateCustomerCommand struct {
	TenantID uuid.UUID
	Name     string
	Phone    string
	Email    string
}


// CreateCustomerHandler handles the CreateCustomerCommand.
type CreateCustomerHandler struct {
	repo      domain.CustomerRepository
	publisher application.EventPublisher
}

// NewCreateCustomerHandler constructs a handler with the given repository and
// event publisher.
func NewCreateCustomerHandler(repo domain.CustomerRepository, publisher application.EventPublisher) *CreateCustomerHandler {
	return &CreateCustomerHandler{repo: repo, publisher: publisher}
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

	// Publish Customer.New event; non-fatal on failure.
	_ = h.publisher.PublishCustomerNew(ctx, c.ID.String(), c.TenantID.String(), c.Name)

	return c, nil
}
