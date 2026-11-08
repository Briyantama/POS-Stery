package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Briyantama/POS-Stery/services/auth/domain"
	"github.com/Briyantama/POS-Stery/services/auth/infrastructure/postgres/db"
	"github.com/Briyantama/stery-kit/go/pgtenant"
)

type StoreRepository struct {
	pool *pgxpool.Pool
}

func NewStoreRepository(pool *pgxpool.Pool) *StoreRepository {
	return &StoreRepository{pool: pool}
}

func (r *StoreRepository) FindByID(ctx context.Context, tenantID domain.TenantID, id domain.StoreID) (*domain.Store, error) {
	var s *domain.Store
	err := pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.GetStoreByID(ctx, db.GetStoreByIDParams{
			ID:       id,
			TenantID: tenantID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
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
	err := pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
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

func (r *StoreRepository) ListByTenant(ctx context.Context, tenantID domain.TenantID) ([]*domain.Store, error) {
	stores := []*domain.Store{}
	err := pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id, name FROM auth.stores WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY name, id`, tenantID)
		if err != nil {
			return fmt.Errorf("list stores: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			s := &domain.Store{TenantID: tenantID}
			if err := rows.Scan(&s.ID, &s.Name); err != nil {
				return fmt.Errorf("scan store: %w", err)
			}
			stores = append(stores, s)
		}
		return rows.Err()
	})
	return stores, err
}
