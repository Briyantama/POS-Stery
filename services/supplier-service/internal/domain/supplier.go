package domain

import (
	"time"

	"github.com/google/uuid"
)

// PurchaseOrderStatus represents the lifecycle state of a purchase order.
type PurchaseOrderStatus string

const (
	StatusPending   PurchaseOrderStatus = "PENDING"
	StatusReceived  PurchaseOrderStatus = "RECEIVED"
	StatusCancelled PurchaseOrderStatus = "CANCELLED"
)

// Supplier is a vendor that supplies products to the business.
type Supplier struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	Contact   string
	Phone     string
	Email     string
	Address   string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// PurchaseOrderItem is a single line in a purchase order.
type PurchaseOrderItem struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	PurchaseOrderID uuid.UUID
	ProductID       uuid.UUID
	QuantityOrdered int32
	QuantityReceived int32
	UnitCost        float64
}

// PurchaseOrder is a formal request to a supplier to deliver products.
type PurchaseOrder struct {
	ID         uuid.UUID
	TenantID   uuid.UUID
	StoreID    uuid.UUID
	SupplierID uuid.UUID
	Status     PurchaseOrderStatus
	Notes      string
	Items      []*PurchaseOrderItem
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
