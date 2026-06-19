package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pos-stery/pos-stery/services/_shared/database"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/domain"
)

// ThresholdRepository implements domain.ThresholdRepository using PostgreSQL.
type ThresholdRepository struct {
	pool *pgxpool.Pool
}

// NewThresholdRepository creates a ThresholdRepository backed by the given pool.
func NewThresholdRepository(pool *pgxpool.Pool) *ThresholdRepository {
	return &ThresholdRepository{pool: pool}
}

// SetThreshold upserts a low-stock threshold.
func (r *ThresholdRepository) SetThreshold(ctx context.Context, t *domain.StockThreshold) error {
	return database.WithTenantContext(ctx, r.pool, t.TenantID.String(), t.StoreID.String(), func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO inventory.stock_thresholds
				(tenant_id, store_id, product_id, min_quantity)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (tenant_id, store_id, product_id)
			DO UPDATE SET
				min_quantity = EXCLUDED.min_quantity,
				updated_at   = NOW()
		`, t.TenantID, t.StoreID, t.ProductID, t.MinQuantity)
		if err != nil {
			return fmt.Errorf("upsert threshold: %w", err)
		}
		return nil
	})
}

// GetThreshold retrieves the threshold for a product. Returns ErrNotFound if absent.
func (r *ThresholdRepository) GetThreshold(
	ctx context.Context,
	tenantID, storeID, productID uuid.UUID,
) (*domain.StockThreshold, error) {
	var result *domain.StockThreshold

	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), storeID.String(), func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT tenant_id, store_id, product_id, min_quantity
			FROM inventory.stock_thresholds
			WHERE tenant_id  = $1
			  AND store_id   = $2
			  AND product_id = $3
		`, tenantID, storeID, productID)

		t := &domain.StockThreshold{}
		if err := row.Scan(&t.TenantID, &t.StoreID, &t.ProductID, &t.MinQuantity); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("threshold: %w", sherrors.ErrNotFound)
			}
			return fmt.Errorf("get threshold: %w", err)
		}
		result = t
		return nil
	})

	return result, err
}
