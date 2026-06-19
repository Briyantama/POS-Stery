package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTenantContext executes fn inside a transaction with app.tenant_id and
// app.store_id set as session-local parameters. PostgreSQL RLS policies depend
// on these settings — never call repository methods without this wrapper.
//
// If storeID is empty (tenant-level admin operations), app.store_id is set to
// an empty string. RLS policies that gate on store_id must handle this case.
func WithTenantContext(
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID, storeID string,
	fn func(pgx.Tx) error,
) error {
	if tenantID == "" {
		return fmt.Errorf("tenantID must not be empty")
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if _, err := tx.Exec(ctx,
		"SET LOCAL app.tenant_id = $1; SET LOCAL app.store_id = $2",
		tenantID, storeID,
	); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("set tenant context: %w", err)
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
