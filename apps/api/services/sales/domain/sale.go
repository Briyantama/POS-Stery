package domain

import (
	"time"

	"github.com/google/uuid"
)

// Sale represents a completed point-of-sale transaction.
type Sale struct {
	ID             uuid.UUID
	TenantID       uuid.UUID
	StoreID        uuid.UUID
	CashierID      uuid.UUID
	CustomerID     *uuid.UUID // optional
	Items          []SaleItem
	Subtotal       float64
	DiscountAmount float64
	Total          float64
	Status         string
	Notes          string
	CompletedAt    time.Time
}

// SaleItem is a single line item within a Sale.
type SaleItem struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	SaleID    uuid.UUID
	ProductID uuid.UUID
	Quantity  int
	UnitPrice float64
	Discount  float64
	LineTotal float64
}

// Receipt is a JSON snapshot of the sale issued to the customer.
type Receipt struct {
	ID       uuid.UUID
	TenantID uuid.UUID
	StoreID  uuid.UUID
	SaleID   uuid.UUID
	Snapshot []byte // JSON
	IssuedAt time.Time
}

// SalesReport aggregates revenue and product metrics for a given date and store.
type SalesReport struct {
	Date             string
	StoreID          string
	TotalRevenue     float64
	TransactionCount int
	TopProducts      []TopProduct
}

// TopProduct contains aggregated sales data for a single product.
type TopProduct struct {
	ProductID   string
	ProductName string
	UnitsSold   int
	Revenue     float64
}
