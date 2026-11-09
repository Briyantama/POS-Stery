package postgres

import (
	"context"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/customer/domain"
	"github.com/Briyantama/stery-kit/go/pgtenant"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"uuid"
)

// LoyaltyRepository is the PostgreSQL implementation of domain.LoyaltyRepository.
type LoyaltyRepository struct {
	pool *pgxpool.Pool
}

// NewLoyaltyRepository constructs a repository backed by the given pgxpool.
func NewLoyaltyRepository(pool *pgxpool.Pool) *LoyaltyRepository {
	return &LoyaltyRepository{pool: pool}
}

// AddPoints inserts a loyalty_points row and returns the new running total.
// It returns domain.ErrAlreadyAwarded when the sale already awarded points.
func (r *LoyaltyRepository) AddPoints(ctx context.Context, tenantID uuid.UUID, entry *domain.LoyaltyEntry) (int, error) {
	var newTotal int

	err := pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			INSERT INTO customer.loyalty_points
				(id, tenant_id, customer_id, points, total_points, sale_id)
			VALUES ($1,$2,$3,$4,$5,$6)
			ON CONFLICT (tenant_id, sale_id) DO NOTHING
		`,
			entry.ID, entry.TenantID, entry.CustomerID,
			entry.Points, entry.TotalPoints, entry.SaleID,
		)
		if err != nil {
			return fmt.Errorf("insert loyalty entry: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrAlreadyAwarded
		}
		newTotal = entry.TotalPoints
		return nil
	})
	if err != nil {
		return 0, err
	}
	return newTotal, nil
}

// GetTotalPoints returns the sum of all points rows for the given customer.
func (r *LoyaltyRepository) GetTotalPoints(ctx context.Context, tenantID, customerID uuid.UUID) (int, error) {
	var total int

	err := pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(points), 0)
			FROM customer.loyalty_points
			WHERE customer_id = $1 AND tenant_id = $2
		`, customerID, tenantID).Scan(&total)
		if err != nil {
			return fmt.Errorf("sum loyalty points: %w", err)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return total, nil
}
