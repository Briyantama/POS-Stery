package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pos-stery/pos-stery/services/_shared/database"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/domain"
)

// StockRepository implements domain.StockRepository using PostgreSQL.
type StockRepository struct {
	pool *pgxpool.Pool
}

// NewStockRepository creates a StockRepository backed by the given pool.
func NewStockRepository(pool *pgxpool.Pool) *StockRepository {
	return &StockRepository{pool: pool}
}

// AddItem inserts a new stock item. Returns ErrAlreadyExists on unique violation.
func (r *StockRepository) AddItem(ctx context.Context, item *domain.StockItem) error {
	return database.WithTenantContext(ctx, r.pool, item.TenantID.String(), item.StoreID.String(), func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO inventory.stock_items
				(tenant_id, store_id, product_id, quantity)
			VALUES ($1, $2, $3, $4)
			RETURNING id, created_at, updated_at
		`, item.TenantID, item.StoreID, item.ProductID, item.Quantity)

		if err := row.Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			if isUniqueViolation(err) {
				return fmt.Errorf("stock item already exists: %w", sherrors.ErrAlreadyExists)
			}
			return fmt.Errorf("insert stock item: %w", err)
		}
		return nil
	})
}

// UpdateStock atomically applies delta to the quantity using a single UPDATE … RETURNING.
func (r *StockRepository) UpdateStock(
	ctx context.Context,
	tenantID, storeID, productID uuid.UUID,
	delta int64,
	_ string,
	_ string,
) (*domain.StockItem, error) {
	var result *domain.StockItem

	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), storeID.String(), func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			UPDATE inventory.stock_items
			SET
				quantity   = quantity + $1,
				updated_at = NOW()
			WHERE tenant_id  = $2
			  AND store_id   = $3
			  AND product_id = $4
			RETURNING id, tenant_id, store_id, product_id, quantity, created_at, updated_at
		`, delta, tenantID, storeID, productID)

		item, err := scanStockItem(row)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("stock item: %w", sherrors.ErrNotFound)
			}
			return fmt.Errorf("update stock: %w", err)
		}
		result = item
		return nil
	})

	return result, err
}

// GetItem retrieves a single stock item. Returns ErrNotFound if absent.
func (r *StockRepository) GetItem(
	ctx context.Context,
	tenantID, storeID, productID uuid.UUID,
) (*domain.StockItem, error) {
	var result *domain.StockItem

	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), storeID.String(), func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT id, tenant_id, store_id, product_id, quantity, created_at, updated_at
			FROM inventory.stock_items
			WHERE tenant_id  = $1
			  AND store_id   = $2
			  AND product_id = $3
		`, tenantID, storeID, productID)

		item, err := scanStockItem(row)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("stock item: %w", sherrors.ErrNotFound)
			}
			return fmt.Errorf("get stock item: %w", err)
		}
		result = item
		return nil
	})

	return result, err
}

// ListItems returns a page of stock items, optionally filtered to low-stock entries.
func (r *StockRepository) ListItems(
	ctx context.Context,
	tenantID, storeID uuid.UUID,
	lowStockOnly bool,
	limit, offset int,
) ([]*domain.StockItem, int, error) {
	var items []*domain.StockItem
	var total int

	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), storeID.String(), func(tx pgx.Tx) error {
		baseFilter := `
			WHERE si.tenant_id = $1
			  AND si.store_id  = $2
		`
		lowStockJoin := `
			JOIN inventory.stock_thresholds st
				ON  st.tenant_id   = si.tenant_id
				AND st.store_id    = si.store_id
				AND st.product_id  = si.product_id
				AND si.quantity   <= st.min_quantity
		`

		fromClause := "FROM inventory.stock_items si"
		if lowStockOnly {
			fromClause += " " + lowStockJoin
		}

		countSQL := "SELECT COUNT(*) " + fromClause + " " + baseFilter
		if err := tx.QueryRow(ctx, countSQL, tenantID, storeID).Scan(&total); err != nil {
			return fmt.Errorf("count stock items: %w", err)
		}

		selectSQL := `SELECT si.id, si.tenant_id, si.store_id, si.product_id,
			si.quantity, si.created_at, si.updated_at ` +
			fromClause + " " + baseFilter +
			` ORDER BY si.product_id LIMIT $3 OFFSET $4`

		rows, err := tx.Query(ctx, selectSQL, tenantID, storeID, limit, offset)
		if err != nil {
			return fmt.Errorf("list stock items: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			item := &domain.StockItem{}
			if err := rows.Scan(
				&item.ID, &item.TenantID, &item.StoreID, &item.ProductID,
				&item.Quantity, &item.CreatedAt, &item.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan stock item: %w", err)
			}
			items = append(items, item)
		}
		return rows.Err()
	})

	return items, total, err
}

// scanStockItem reads a single pgx.Row into a StockItem.
func scanStockItem(row pgx.Row) (*domain.StockItem, error) {
	item := &domain.StockItem{}
	err := row.Scan(
		&item.ID, &item.TenantID, &item.StoreID, &item.ProductID,
		&item.Quantity, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return item, nil
}

// isUniqueViolation returns true for Postgres SQLSTATE 23505 (unique_violation).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
