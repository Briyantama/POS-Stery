package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Customer is the core aggregate for the customer bounded context.
type Customer struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	Phone     string
	Email     string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// LoyaltyEntry represents a single loyalty-points transaction.
type LoyaltyEntry struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	CustomerID  uuid.UUID
	Points      int
	TotalPoints int
	SaleID      uuid.UUID
	CreatedAt   time.Time
}

// CustomerRepository defines the port the application layer uses to persist
// and retrieve customers.
type CustomerRepository interface {
	// Create persists a new customer.
	Create(ctx context.Context, tenantID uuid.UUID, c *Customer) error

	// FindByID returns the customer with the given id that belongs to tenantID.
	// Returns ErrNotFound when no matching row exists.
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*Customer, error)

	// List returns customers for the given tenant with optional search and paging.
	List(ctx context.Context, tenantID uuid.UUID, query string, limit, offset int) ([]*Customer, int, error)
}

// LoyaltyRepository defines the port for loyalty-points operations.
type LoyaltyRepository interface {
	// AddPoints inserts a loyalty entry and returns the new running total.
	AddPoints(ctx context.Context, tenantID uuid.UUID, entry *LoyaltyEntry) (int, error)

	// GetTotalPoints returns the sum of all points for the customer.
	GetTotalPoints(ctx context.Context, tenantID, customerID uuid.UUID) (int, error)
}
