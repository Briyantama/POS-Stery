package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/domain"
)

// AddLoyaltyPointsCommand carries the data to add loyalty points after a sale.
// Points MUST only be added after a fully committed, successful sale.
type AddLoyaltyPointsCommand struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
	SaleID     uuid.UUID
	Points     int
}

// AddLoyaltyPointsResult holds the outcome of the command.
type AddLoyaltyPointsResult struct {
	Customer    *domain.Customer
	TotalPoints int
}

// AddLoyaltyPointsHandler handles the AddLoyaltyPointsCommand.
type AddLoyaltyPointsHandler struct {
	customerRepo domain.CustomerRepository
	loyaltyRepo  domain.LoyaltyRepository
}

// NewAddLoyaltyPointsHandler constructs a handler with the required repositories.
func NewAddLoyaltyPointsHandler(
	customerRepo domain.CustomerRepository,
	loyaltyRepo domain.LoyaltyRepository,
) *AddLoyaltyPointsHandler {
	return &AddLoyaltyPointsHandler{
		customerRepo: customerRepo,
		loyaltyRepo:  loyaltyRepo,
	}
}

// Handle validates the command, verifies the customer exists, records the
// loyalty entry and returns the updated total.
func (h *AddLoyaltyPointsHandler) Handle(ctx context.Context, cmd AddLoyaltyPointsCommand) (*AddLoyaltyPointsResult, error) {
	if cmd.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.CustomerID == uuid.Nil {
		return nil, fmt.Errorf("customer_id is required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.SaleID == uuid.Nil {
		return nil, fmt.Errorf("sale_id is required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.Points <= 0 {
		return nil, fmt.Errorf("points must be positive: %w", sherrors.ErrInvalidArgument)
	}

	c, err := h.customerRepo.FindByID(ctx, cmd.TenantID, cmd.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("find customer: %w", err)
	}

	currentTotal, err := h.loyaltyRepo.GetTotalPoints(ctx, cmd.TenantID, cmd.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("get total points: %w", err)
	}

	entry := &domain.LoyaltyEntry{
		ID:          uuid.New(),
		TenantID:    cmd.TenantID,
		CustomerID:  cmd.CustomerID,
		Points:      cmd.Points,
		TotalPoints: currentTotal + cmd.Points,
		SaleID:      cmd.SaleID,
	}

	newTotal, err := h.loyaltyRepo.AddPoints(ctx, cmd.TenantID, entry)
	if err != nil {
		return nil, fmt.Errorf("add loyalty points: %w", err)
	}

	return &AddLoyaltyPointsResult{
		Customer:    c,
		TotalPoints: newTotal,
	}, nil
}
