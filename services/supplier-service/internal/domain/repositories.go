package domain

import (
	"context"

	"github.com/google/uuid"
)

// SupplierRepository handles persistence of supplier records.
// All implementations must use database.WithTenantContext.
type SupplierRepository interface {
	// Add inserts a new supplier. Returns ErrAlreadyExists on duplicate name for the tenant.
	Add(ctx context.Context, supplier *Supplier) error

	// List returns a paginated list of active (non-deleted) suppliers for a tenant.
	List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Supplier, int, error)

	// Get retrieves a supplier by ID. Returns ErrNotFound if absent or soft-deleted.
	Get(ctx context.Context, tenantID, supplierID uuid.UUID) (*Supplier, error)
}

// PurchaseOrderRepository handles persistence of purchase orders.
// All implementations must use database.WithTenantContext.
type PurchaseOrderRepository interface {
	// Create inserts a new purchase order with its line items in a single transaction.
	Create(ctx context.Context, order *PurchaseOrder) error

	// Get retrieves a purchase order and its items. Returns ErrNotFound if absent.
	Get(ctx context.Context, tenantID, storeID, orderID uuid.UUID) (*PurchaseOrder, error)

	// List returns a paginated list of purchase orders filtered by optional status.
	List(ctx context.Context, tenantID, storeID uuid.UUID, status string, limit, offset int) ([]*PurchaseOrder, int, error)

	// UpdateStatus changes the order status and updates received quantities for items.
	UpdateReceived(ctx context.Context, order *PurchaseOrder) error
}
