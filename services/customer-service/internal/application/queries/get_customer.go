package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/domain"
)

// GetCustomerQuery carries the parameters for looking up a single customer.
type GetCustomerQuery struct {
	TenantID   uuid.UUID
	CustomerID uuid.UUID
}

// ListCustomersQuery carries the parameters for listing customers.
type ListCustomersQuery struct {
	TenantID uuid.UUID
	Query    string
	Limit    int
	Offset   int
}

// ListCustomersResult holds the paged list result.
type ListCustomersResult struct {
	Customers []*domain.Customer
	Total     int
}

// CustomerQueryHandler handles customer read queries.
type CustomerQueryHandler struct {
	repo        domain.CustomerRepository
	loyaltyRepo domain.LoyaltyRepository
}

// NewCustomerQueryHandler constructs a handler with the given repositories.
func NewCustomerQueryHandler(repo domain.CustomerRepository, loyaltyRepo domain.LoyaltyRepository) *CustomerQueryHandler {
	return &CustomerQueryHandler{repo: repo, loyaltyRepo: loyaltyRepo}
}

// GetCustomer returns a single customer by ID.
func (h *CustomerQueryHandler) GetCustomer(ctx context.Context, q GetCustomerQuery) (*domain.Customer, error) {
	c, err := h.repo.FindByID(ctx, q.TenantID, q.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	return c, nil
}

// GetTotalPoints returns the accumulated loyalty points for a customer.
func (h *CustomerQueryHandler) GetTotalPoints(ctx context.Context, tenantID, customerID uuid.UUID) (int, error) {
	total, err := h.loyaltyRepo.GetTotalPoints(ctx, tenantID, customerID)
	if err != nil {
		return 0, fmt.Errorf("get total points: %w", err)
	}
	return total, nil
}

// ListCustomers returns a paged list of customers.
func (h *CustomerQueryHandler) ListCustomers(ctx context.Context, q ListCustomersQuery) (*ListCustomersResult, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	customers, total, err := h.repo.List(ctx, q.TenantID, q.Query, q.Limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}

	return &ListCustomersResult{
		Customers: customers,
		Total:     total,
	}, nil
}
