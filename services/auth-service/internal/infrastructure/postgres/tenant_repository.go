package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/infrastructure/postgres/db"
)

// TenantRepository queries auth.tenants, which has no RLS policy.
// Queries run directly against the pool without a tenant context transaction.
type TenantRepository struct {
	pool *pgxpool.Pool
}

func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

func (r *TenantRepository) FindByID(ctx context.Context, id domain.TenantID) (*domain.Tenant, error) {
	q := db.New(r.pool)
	row, err := q.GetTenantByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, sherrors.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return &domain.Tenant{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}
