package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pos-stery/pos-stery/services/_shared/database"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/sales-service/internal/domain"
)

// SaleRepository is the PostgreSQL implementation of domain.SaleRepository.
// All operations run inside WithTenantContext to activate RLS policies.
type SaleRepository struct {
	pool *pgxpool.Pool
}

// NewSaleRepository creates a SaleRepository backed by the given connection pool.
func NewSaleRepository(pool *pgxpool.Pool) *SaleRepository {
	return &SaleRepository{pool: pool}
}

// Create persists the sale, its line items, and the receipt in a single transaction.
func (r *SaleRepository) Create(ctx context.Context, sale *domain.Sale, items []domain.SaleItem, receipt *domain.Receipt) error {
	storeID := sale.StoreID.String()
	return database.WithTenantContext(ctx, r.pool, sale.TenantID.String(), storeID, func(tx pgx.Tx) error {
		// Insert sale
		var customerID *string
		if sale.CustomerID != nil {
			s := sale.CustomerID.String()
			customerID = &s
		}

		_, err := tx.Exec(ctx,
			`INSERT INTO sales.sales
				(id, tenant_id, store_id, cashier_id, customer_id,
				 subtotal, discount_amount, total, status, notes, completed_at, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			sale.ID, sale.TenantID, sale.StoreID, sale.CashierID, customerID,
			sale.Subtotal, sale.DiscountAmount, sale.Total, sale.Status, sale.Notes,
			sale.CompletedAt, time.Now().UTC(),
		)
		if err != nil {
			return fmt.Errorf("insert sale: %w", err)
		}

		// Insert line items
		for _, item := range items {
			_, err := tx.Exec(ctx,
				`INSERT INTO sales.sale_items
					(id, tenant_id, sale_id, product_id, quantity, unit_price, discount, line_total)
				 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
				item.ID, item.TenantID, item.SaleID, item.ProductID,
				item.Quantity, item.UnitPrice, item.Discount, item.LineTotal,
			)
			if err != nil {
				return fmt.Errorf("insert sale_item product=%s: %w", item.ProductID, err)
			}
		}

		// Insert receipt
		_, err = tx.Exec(ctx,
			`INSERT INTO sales.receipts
				(id, tenant_id, store_id, sale_id, snapshot, issued_at)
			 VALUES ($1,$2,$3,$4,$5,$6)`,
			receipt.ID, receipt.TenantID, receipt.StoreID, receipt.SaleID,
			receipt.Snapshot, receipt.IssuedAt,
		)
		if err != nil {
			return fmt.Errorf("insert receipt: %w", err)
		}

		return nil
	})
}

// FindByID retrieves a sale, its items, and its receipt.
// Returns ErrNotFound when the sale does not exist or belongs to a different tenant/store.
func (r *SaleRepository) FindByID(ctx context.Context, tenantID, storeID, saleID uuid.UUID) (*domain.Sale, []domain.SaleItem, *domain.Receipt, error) {
	var sale *domain.Sale
	var items []domain.SaleItem
	var receipt *domain.Receipt

	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), storeID.String(), func(tx pgx.Tx) error {
		// Fetch sale
		row := tx.QueryRow(ctx,
			`SELECT id, tenant_id, store_id, cashier_id, customer_id,
			        subtotal, discount_amount, total, status, notes, completed_at
			 FROM sales.sales
			 WHERE id = $1`,
			saleID,
		)

		s := &domain.Sale{}
		var custID *uuid.UUID
		if err := row.Scan(
			&s.ID, &s.TenantID, &s.StoreID, &s.CashierID, &custID,
			&s.Subtotal, &s.DiscountAmount, &s.Total, &s.Status, &s.Notes, &s.CompletedAt,
		); err != nil {
			if err == pgx.ErrNoRows {
				return fmt.Errorf("%w: sale %s", sherrors.ErrNotFound, saleID)
			}
			return fmt.Errorf("scan sale: %w", err)
		}
		s.CustomerID = custID
		sale = s

		// Fetch line items
		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, sale_id, product_id, quantity, unit_price, discount, line_total
			 FROM sales.sale_items
			 WHERE sale_id = $1`,
			saleID,
		)
		if err != nil {
			return fmt.Errorf("query sale_items: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var item domain.SaleItem
			if err := rows.Scan(
				&item.ID, &item.TenantID, &item.SaleID, &item.ProductID,
				&item.Quantity, &item.UnitPrice, &item.Discount, &item.LineTotal,
			); err != nil {
				return fmt.Errorf("scan sale_item: %w", err)
			}
			items = append(items, item)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate sale_items: %w", err)
		}

		// Fetch receipt
		rRow := tx.QueryRow(ctx,
			`SELECT id, tenant_id, store_id, sale_id, snapshot, issued_at
			 FROM sales.receipts
			 WHERE sale_id = $1`,
			saleID,
		)
		rec := &domain.Receipt{}
		if err := rRow.Scan(&rec.ID, &rec.TenantID, &rec.StoreID, &rec.SaleID, &rec.Snapshot, &rec.IssuedAt); err != nil {
			if err == pgx.ErrNoRows {
				return fmt.Errorf("%w: receipt for sale %s", sherrors.ErrNotFound, saleID)
			}
			return fmt.Errorf("scan receipt: %w", err)
		}
		receipt = rec

		return nil
	})
	if err != nil {
		return nil, nil, nil, err
	}

	return sale, items, receipt, nil
}

// GetReport returns aggregated daily revenue for the given tenant/store/date.
// If storeID is nil, revenue across all stores in the tenant is returned.
func (r *SaleRepository) GetReport(ctx context.Context, tenantID, storeID *uuid.UUID, date string) (*domain.SalesReport, error) {
	if tenantID == nil {
		return nil, fmt.Errorf("%w: tenantID is required for report", sherrors.ErrInvalidArgument)
	}

	// Determine storeID string for tenant context (empty = tenant-level admin operation).
	storeCtx := ""
	if storeID != nil {
		storeCtx = storeID.String()
	}

	var report domain.SalesReport

	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), storeCtx, func(tx pgx.Tx) error {
		var row pgx.Row
		if storeID != nil {
			row = tx.QueryRow(ctx,
				`SELECT COALESCE(date_trunc('day', completed_at)::text, $3),
				        COUNT(*),
				        COALESCE(SUM(total), 0)
				 FROM sales.sales
				 WHERE tenant_id = $1
				   AND store_id  = $2
				   AND date_trunc('day', completed_at) = $3::date`,
				tenantID, storeID, date,
			)
		} else {
			row = tx.QueryRow(ctx,
				`SELECT $2,
				        COUNT(*),
				        COALESCE(SUM(total), 0)
				 FROM sales.sales
				 WHERE tenant_id = $1
				   AND date_trunc('day', completed_at) = $2::date`,
				tenantID, date,
			)
		}

		var dateResult string
		if err := row.Scan(&dateResult, &report.TransactionCount, &report.TotalRevenue); err != nil {
			return fmt.Errorf("scan report: %w", err)
		}
		report.Date = date
		if storeID != nil {
			report.StoreID = storeID.String()
		}

		// Top products — aggregate from sale_items joined to sales.
		var topRows pgx.Rows
		var err error
		if storeID != nil {
			topRows, err = tx.Query(ctx,
				`SELECT si.product_id::text,
				        ''        AS product_name,
				        SUM(si.quantity)::int,
				        SUM(si.line_total)
				 FROM sales.sale_items si
				 JOIN sales.sales s ON s.id = si.sale_id
				 WHERE s.tenant_id = $1
				   AND s.store_id  = $2
				   AND date_trunc('day', s.completed_at) = $3::date
				 GROUP BY si.product_id
				 ORDER BY SUM(si.quantity) DESC
				 LIMIT 10`,
				tenantID, storeID, date,
			)
		} else {
			topRows, err = tx.Query(ctx,
				`SELECT si.product_id::text,
				        ''        AS product_name,
				        SUM(si.quantity)::int,
				        SUM(si.line_total)
				 FROM sales.sale_items si
				 JOIN sales.sales s ON s.id = si.sale_id
				 WHERE s.tenant_id = $1
				   AND date_trunc('day', s.completed_at) = $2::date
				 GROUP BY si.product_id
				 ORDER BY SUM(si.quantity) DESC
				 LIMIT 10`,
				tenantID, date,
			)
		}
		if err != nil {
			return fmt.Errorf("query top products: %w", err)
		}
		defer topRows.Close()

		for topRows.Next() {
			var tp domain.TopProduct
			if err := topRows.Scan(&tp.ProductID, &tp.ProductName, &tp.UnitsSold, &tp.Revenue); err != nil {
				return fmt.Errorf("scan top product: %w", err)
			}
			report.TopProducts = append(report.TopProducts, tp)
		}
		return topRows.Err()
	})
	if err != nil {
		return nil, err
	}

	return &report, nil
}
