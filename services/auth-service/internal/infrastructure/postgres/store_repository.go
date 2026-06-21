package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pos-stery/pos-stery/services/_shared/database"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/infrastructure/postgres/db"
)

type StoreRepository struct {
	pool *pgxpool.Pool
}

func NewStoreRepository(pool *pgxpool.Pool) *StoreRepository {
	return &StoreRepository{pool: pool}
}

func (r *StoreRepository) FindByID(ctx context.Context, tenantID domain.TenantID, id domain.StoreID) (*domain.Store, error) {
	var s *domain.Store
	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.GetStoreByID(ctx, db.GetStoreByIDParams{
			ID:       id,
			TenantID: tenantID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return sherrors.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get store by id: %w", err)
		}
		s = &domain.Store{
			ID:        row.ID,
			TenantID:  row.TenantID,
			Name:      row.Name,
			Address:   row.Address.String,
			Phone:     row.Phone.String,
			IsActive:  row.IsActive,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
		return nil
	})
	return s, err
}

func (r *StoreRepository) ExistsInTenant(ctx context.Context, tenantID domain.TenantID, id domain.StoreID) (bool, error) {
	var exists bool
	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		q := db.New(tx)
		ok, err := q.StoreExistsInTenant(ctx, db.StoreExistsInTenantParams{
			ID:       id,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("store exists in tenant: %w", err)
		}
		exists = ok
		return nil
	})
	return exists, err
}
