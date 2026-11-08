package commands

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"

	"github.com/Briyantama/POS-Stery/services/customer/application"
	"github.com/Briyantama/POS-Stery/services/customer/domain"
	"github.com/google/uuid"
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
	if cmd.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required: %w", domain.ErrInvalidArgument)
	}
	cmd.Name, cmd.Phone, cmd.Email = strings.TrimSpace(cmd.Name), strings.TrimSpace(cmd.Phone), strings.TrimSpace(cmd.Email)
	if cmd.Name == "" {
		return nil, fmt.Errorf("name is required: %w", domain.ErrInvalidArgument)
	}
	if len(cmd.Name) > 200 || len(cmd.Phone) > 32 || len(cmd.Email) > 254 {
		return nil, fmt.Errorf("name, phone or email too long: %w", domain.ErrInvalidArgument)
	}
	if cmd.Email != "" {
		if a, err := mail.ParseAddress(cmd.Email); err != nil || a.Address != cmd.Email {
			return nil, fmt.Errorf("email is invalid: %w", domain.ErrInvalidArgument)
		}
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
	if err := h.publisher.PublishCustomerNew(ctx, c.ID.String(), c.TenantID.String(), c.Name); err != nil {
		slog.WarnContext(ctx, "publish Customer.New failed", "tenant_id", c.TenantID.String(), "customer_id", c.ID.String(), "error", err)
	}

	return c, nil
}
