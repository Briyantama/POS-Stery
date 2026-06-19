package domain

import (
	"time"

	"github.com/google/uuid"
)

// StockUpdateReason describes why a stock level changed.
type StockUpdateReason string

const (
	ReasonSale       StockUpdateReason = "SALE"
	ReasonReceipt    StockUpdateReason = "RECEIPT"
	ReasonAdjustment StockUpdateReason = "ADJUSTMENT"
)

// StockItem represents the quantity of a specific product held at a store.
type StockItem struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	StoreID   uuid.UUID
	ProductID uuid.UUID
	Quantity  int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StockThreshold holds the minimum acceptable quantity for a product at a store.
// When Quantity falls at or below MinQuantity a LowStock event is emitted.
type StockThreshold struct {
	TenantID    uuid.UUID
	StoreID     uuid.UUID
	ProductID   uuid.UUID
	MinQuantity int64
}

// LowStockDetected is a domain event published when stock reaches or falls below
// the configured threshold.
type LowStockDetected struct {
	TenantID    uuid.UUID
	StoreID     uuid.UUID
	ProductID   uuid.UUID
	Quantity    int64
	MinQuantity int64
}
