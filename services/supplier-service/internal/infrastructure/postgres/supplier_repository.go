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
	"github.com/pos-stery/pos-stery/services/supplier-service/internal/domain"
)

// SupplierRepository implements domain.SupplierRepository using PostgreSQL.
type SupplierRepository struct {
	pool *pgxpool.Pool
}

// NewSupplierRepository creates a SupplierRepository.
func NewSupplierRepository(pool *pgxpool.Pool) *SupplierRepository {
	return &SupplierRepository{pool: pool}
}

// Add inserts a new supplier.
func (r *SupplierRepository) Add(ctx context.Context, s *domain.Supplier) error {
	// Supplier operations are tenant-scoped; storeID is empty for tenant-level admin operations.
	return database.WithTenantContext(ctx, r.pool, s.TenantID.String(), "", func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			INSERT INTO supplier.suppliers
				(tenant_id, name, contact, phone, email, address)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at, updated_at
		`, s.TenantID, s.Name, s.Contact, s.Phone, s.Email, s.Address)

		if err := row.Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			if isSupplierUniqueViolation(err) {
				return fmt.Errorf("supplier already exists: %w", sherrors.ErrAlreadyExists)
			}
			return fmt.Errorf("insert supplier: %w", err)
		}
		return nil
	})
}

// List returns a paginated list of active suppliers.
func (r *SupplierRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*domain.Supplier, int, error) {
	var suppliers []*domain.Supplier
	var total int

	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `
			SELECT COUNT(*) FROM supplier.suppliers
			WHERE tenant_id = $1 AND deleted_at IS NULL
		`, tenantID).Scan(&total); err != nil {
			return fmt.Errorf("count suppliers: %w", err)
		}

		if limit <= 0 || limit > 200 {
			limit = 50
		}

		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, name, contact, phone, email, address, created_at, updated_at
			FROM supplier.suppliers
			WHERE tenant_id = $1 AND deleted_at IS NULL
			ORDER BY name
			LIMIT $2 OFFSET $3
		`, tenantID, limit, offset)
		if err != nil {
			return fmt.Errorf("list suppliers: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			s := &domain.Supplier{}
			if err := rows.Scan(
				&s.ID, &s.TenantID, &s.Name, &s.Contact, &s.Phone,
				&s.Email, &s.Address, &s.CreatedAt, &s.UpdatedAt,
			); err != nil {
				return fmt.Errorf("scan supplier: %w", err)
			}
			suppliers = append(suppliers, s)
		}
		return rows.Err()
	})

	return suppliers, total, err
}

// Get retrieves a supplier by ID.
func (r *SupplierRepository) Get(ctx context.Context, tenantID, supplierID uuid.UUID) (*domain.Supplier, error) {
	var result *domain.Supplier

	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT id, tenant_id, name, contact, phone, email, address, created_at, updated_at
			FROM supplier.suppliers
			WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
		`, tenantID, supplierID)

		s := &domain.Supplier{}
		if err := row.Scan(
			&s.ID, &s.TenantID, &s.Name, &s.Contact, &s.Phone,
			&s.Email, &s.Address, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("supplier: %w", sherrors.ErrNotFound)
			}
			return fmt.Errorf("get supplier: %w", err)
		}
		result = s
		return nil
	})

	return result, err
}

func isSupplierUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
