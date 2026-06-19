package domain

import (
	"context"

	"github.com/google/uuid"
)

// StockRepository handles persistence of stock items.
// All implementations must use database.WithTenantContext.
type StockRepository interface {
	// AddItem inserts a new stock item; returns ErrAlreadyExists if the
	// (tenant, store, product) triple already exists.
	AddItem(ctx context.Context, item *StockItem) error

	// UpdateStock atomically adjusts the quantity by delta (positive or negative)
	// and records the reason and reference ID. Returns the updated item.
	UpdateStock(
		ctx context.Context,
		tenantID, storeID, productID uuid.UUID,
		delta int64,
		reason string,
		refID string,
	) (*StockItem, error)

	// GetItem retrieves a single stock item. Returns ErrNotFound if absent.
	GetItem(ctx context.Context, tenantID, storeID, productID uuid.UUID) (*StockItem, error)

	// ListItems returns a page of stock items for a store, optionally filtered
	// to items at or below their threshold (lowStockOnly). Returns the items,
	// the total count (before pagination), and any error.
	ListItems(
		ctx context.Context,
		tenantID, storeID uuid.UUID,
		lowStockOnly bool,
		limit, offset int,
	) ([]*StockItem, int, error)
}

// ThresholdRepository handles persistence of low-stock thresholds.
type ThresholdRepository interface {
	// SetThreshold upserts a threshold record.
	SetThreshold(ctx context.Context, threshold *StockThreshold) error

	// GetThreshold retrieves the threshold for a product. Returns ErrNotFound
	// if no threshold has been configured.
	GetThreshold(ctx context.Context, tenantID, storeID, productID uuid.UUID) (*StockThreshold, error)
}
