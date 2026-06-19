package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pos-stery/pos-stery/services/_shared/database"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/domain"
)

// CustomerRepository is the PostgreSQL implementation of domain.CustomerRepository.
type CustomerRepository struct {
	pool *pgxpool.Pool
}

// NewCustomerRepository constructs a repository backed by the given pgxpool.
func NewCustomerRepository(pool *pgxpool.Pool) *CustomerRepository {
	return &CustomerRepository{pool: pool}
}

// Create inserts a new customer row inside a tenant-scoped transaction.
func (r *CustomerRepository) Create(ctx context.Context, tenantID uuid.UUID, c *domain.Customer) error {
	return database.WithTenantContext(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		now := time.Now().UTC()
		c.CreatedAt = now
		c.UpdatedAt = now

		_, err := tx.Exec(ctx, `
			INSERT INTO customer.customers
				(id, tenant_id, name, phone, email, is_active, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		`,
			c.ID, c.TenantID, c.Name,
			nullableString(c.Phone), nullableString(c.Email),
			c.IsActive, c.CreatedAt, c.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert customer: %w", err)
		}
		return nil
	})
}

// FindByID retrieves a single customer by its ID within a tenant-scoped transaction.
func (r *CustomerRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Customer, error) {
	var c *domain.Customer

	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT id, tenant_id, name, phone, email, is_active, created_at, updated_at
			FROM customer.customers
			WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		`, id, tenantID)

		var cust domain.Customer
		if err := scanCustomer(row, &cust); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return sherrors.ErrNotFound
			}
			return fmt.Errorf("scan customer: %w", err)
		}
		c = &cust
		return nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// List returns customers for the given tenant with optional name search and paging.
func (r *CustomerRepository) List(
	ctx context.Context,
	tenantID uuid.UUID,
	query string,
	limit, offset int,
) ([]*domain.Customer, int, error) {
	var customers []*domain.Customer
	var total int

	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		args := []any{tenantID}
		conditions := []string{"tenant_id = $1", "deleted_at IS NULL"}
		argIdx := 2

		if query != "" {
			conditions = append(conditions,
				fmt.Sprintf("(name ILIKE $%d OR phone = $%d OR email ILIKE $%d)", argIdx, argIdx+1, argIdx+2),
			)
			args = append(args, "%"+query+"%", query, "%"+query+"%")
			argIdx += 3
		}

		where := "WHERE " + strings.Join(conditions, " AND ")

		countSQL := fmt.Sprintf("SELECT COUNT(*) FROM customer.customers %s", where)
		if err := tx.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
			return fmt.Errorf("count customers: %w", err)
		}

		dataSQL := fmt.Sprintf(`
			SELECT id, tenant_id, name, phone, email, is_active, created_at, updated_at
			FROM customer.customers %s
			ORDER BY created_at DESC
			LIMIT $%d OFFSET $%d
		`, where, argIdx, argIdx+1)
		args = append(args, limit, offset)

		rows, err := tx.Query(ctx, dataSQL, args...)
		if err != nil {
			return fmt.Errorf("query customers: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var c domain.Customer
			if err := scanCustomer(rows, &c); err != nil {
				return fmt.Errorf("scan customer row: %w", err)
			}
			customers = append(customers, &c)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return customers, total, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCustomer(s scanner, c *domain.Customer) error {
	var phone, email *string
	if err := s.Scan(
		&c.ID, &c.TenantID, &c.Name, &phone, &email,
		&c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return err
	}
	if phone != nil {
		c.Phone = *phone
	}
	if email != nil {
		c.Email = *email
	}
	return nil
}

func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
