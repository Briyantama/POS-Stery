package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Briyantama/POS-Stery/services/product/domain"
	"github.com/Briyantama/stery-kit/go/pgtenant"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ProductRepository is the PostgreSQL implementation of domain.ProductRepository.
type ProductRepository struct {
	pool *pgxpool.Pool
}

// NewProductRepository constructs a repository backed by the given pgxpool.
func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

// Create inserts a new product row inside a tenant-scoped transaction.
func (r *ProductRepository) Create(ctx context.Context, tenantID uuid.UUID, p *domain.Product) error {
	return pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		now := time.Now().UTC()
		p.CreatedAt = now
		p.UpdatedAt = now

		_, err := tx.Exec(ctx, `
			INSERT INTO product.products
				(id, tenant_id, category_id, name, sku, barcode, base_price, sale_price,
				 description, unit, is_active, created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		`,
			p.ID, p.TenantID, p.CategoryID, p.Name, p.SKU, nullableString(p.Barcode),
			p.BasePrice, p.SalePrice, nullableString(p.Description),
			nullableString(p.Unit), p.IsActive, p.CreatedAt, p.UpdatedAt,
		)
		if err != nil {
			if isDuplicateKey(err) {
				return fmt.Errorf("product with sku %q: %w", p.SKU, domain.ErrAlreadyExists)
			}
			if isForeignKeyViolation(err) {
				return fmt.Errorf("category_id does not exist: %w", domain.ErrInvalidArgument)
			}
			return fmt.Errorf("insert product: %w", err)
		}
		return nil
	})
}

// Update modifies an existing product row inside a tenant-scoped transaction.
func (r *ProductRepository) Update(ctx context.Context, tenantID uuid.UUID, p *domain.Product) error {
	return pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		p.UpdatedAt = time.Now().UTC()

		tag, err := tx.Exec(ctx, `
			UPDATE product.products SET
				category_id = $1,
				name        = $2,
				barcode     = $3,
				base_price  = $4,
				sale_price  = $5,
				description = $6,
				is_active   = $7,
				updated_at  = $8
			WHERE id = $9 AND tenant_id = $10 AND deleted_at IS NULL
		`,
			p.CategoryID, p.Name, nullableString(p.Barcode), p.BasePrice, p.SalePrice,
			nullableString(p.Description), p.IsActive, p.UpdatedAt,
			p.ID, p.TenantID,
		)
		if err != nil {
			if isForeignKeyViolation(err) {
				return fmt.Errorf("category_id does not exist: %w", domain.ErrInvalidArgument)
			}
			return fmt.Errorf("update product: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

// FindByID retrieves a single product by its ID within a tenant-scoped transaction.
func (r *ProductRepository) FindByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	var p *domain.Product

	err := pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			SELECT id, tenant_id, category_id, name, sku, barcode,
			       base_price, sale_price, description, unit, is_active,
			       created_at, updated_at
			FROM product.products
			WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
		`, id, tenantID)

		var prod domain.Product
		if err := scanProduct(row, &prod); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.ErrNotFound
			}
			return fmt.Errorf("scan product: %w", err)
		}
		p = &prod
		return nil
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

// CategoryExists reports whether the category belongs to tenantID (queried under RLS).
func (r *ProductRepository) CategoryExists(ctx context.Context, tenantID, categoryID uuid.UUID) (bool, error) {
	var ok bool
	err := pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM product.categories WHERE id = $1 AND tenant_id = $2)`,
			categoryID, tenantID).Scan(&ok)
	})
	if err != nil {
		return false, fmt.Errorf("check category: %w", err)
	}
	return ok, nil
}

// Search returns products matching the optional query/category with pagination.
// query is matched with ILIKE on name, or exact match on barcode.
// category is matched exactly on category_id (UUID string).
func (r *ProductRepository) Search(
	ctx context.Context,
	tenantID uuid.UUID,
	query, category string,
	limit, offset int,
) ([]*domain.Product, int, error) {
	var products []*domain.Product
	var total int

	err := pgtenant.WithTenantTx(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		args := []any{tenantID}
		conditions := []string{"tenant_id = $1", "deleted_at IS NULL"}
		argIdx := 2

		if query != "" {
			conditions = append(conditions,
				fmt.Sprintf("(name ILIKE $%d OR barcode = $%d)", argIdx, argIdx+1),
			)
			args = append(args, "%"+query+"%", query)
			argIdx += 2
		}

		if category != "" {
			conditions = append(conditions, fmt.Sprintf("category_id = $%d", argIdx))
			args = append(args, category)
			argIdx++
		}

		where := "WHERE " + strings.Join(conditions, " AND ")

		// Count query
		countSQL := fmt.Sprintf("SELECT COUNT(*) FROM product.products %s", where)
		if err := tx.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
			return fmt.Errorf("count products: %w", err)
		}

		// Data query
		dataSQL := fmt.Sprintf(`
			SELECT id, tenant_id, category_id, name, sku, barcode,
			       base_price, sale_price, description, unit, is_active,
			       created_at, updated_at
			FROM product.products %s
			ORDER BY created_at DESC
			LIMIT $%d OFFSET $%d
		`, where, argIdx, argIdx+1)
		args = append(args, limit, offset)

		rows, err := tx.Query(ctx, dataSQL, args...)
		if err != nil {
			return fmt.Errorf("query products: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var p domain.Product
			if err := scanProduct(rows, &p); err != nil {
				return fmt.Errorf("scan product row: %w", err)
			}
			products = append(products, &p)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

// scanProduct reads a product row from a pgx.Row or pgx.Rows into p.
type scanner interface {
	Scan(dest ...any) error
}

func scanProduct(s scanner, p *domain.Product) error {
	var barcode, description, unit *string
	if err := s.Scan(
		&p.ID, &p.TenantID, &p.CategoryID, &p.Name, &p.SKU, &barcode,
		&p.BasePrice, &p.SalePrice, &description, &unit, &p.IsActive,
		&p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return err
	}
	p.Barcode, p.Description, p.Unit = deref(barcode), deref(description), deref(unit)
	return nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// nullableString returns nil for empty strings so the DB stores NULL.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// isDuplicateKey checks for PostgreSQL unique-violation error code 23505.
func isDuplicateKey(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// isForeignKeyViolation checks for PostgreSQL foreign-key error code 23503.
func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
