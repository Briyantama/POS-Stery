package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Product is the core aggregate for the product bounded context.
type Product struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	CategoryID  *uuid.UUID
	Name        string
	SKU         string
	Barcode     string
	BasePrice   float64
	SalePrice   *float64
	Description string
	Unit        string
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ProductRepository defines the port the application layer uses to persist and
// retrieve products. Concrete implementations live in internal/infrastructure.
type ProductRepository interface {
	// Create persists a new product.
	Create(ctx context.Context, tenantID uuid.UUID, p *Product) error

	// Update persists changes to an existing product.
	Update(ctx context.Context, tenantID uuid.UUID, p *Product) error

	// FindByID returns the product with the given id that belongs to tenantID.
	// Returns ErrNotFound when no matching row exists.
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*Product, error)

	// Search returns products matching the optional query string (ILIKE on name /
	// exact on barcode) and optional category filter. It also returns the total
	// count before paging so callers can build pagination responses.
	Search(
		ctx context.Context,
		tenantID uuid.UUID,
		query, category string,
		limit, offset int,
	) ([]*Product, int, error)
}
