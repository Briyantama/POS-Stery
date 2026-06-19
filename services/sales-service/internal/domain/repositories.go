package domain

import (
	"context"

	"github.com/google/uuid"
)

// SaleRepository persists and retrieves sales data.
// All implementations MUST use database.WithTenantContext to enforce RLS.
type SaleRepository interface {
	// Create persists the sale, its line items, and the receipt atomically.
	Create(ctx context.Context, sale *Sale, items []SaleItem, receipt *Receipt) error

	// FindByID retrieves a sale plus its items and receipt.
	FindByID(ctx context.Context, tenantID, storeID, saleID uuid.UUID) (*Sale, []SaleItem, *Receipt, error)

	// GetReport returns aggregated revenue metrics for the given tenant/store/date.
	GetReport(ctx context.Context, tenantID, storeID *uuid.UUID, date string) (*SalesReport, error)
}
