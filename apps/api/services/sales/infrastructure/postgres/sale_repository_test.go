package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"uuid"

	"github.com/Briyantama/POS-Stery/internal/testdb"
	"github.com/Briyantama/POS-Stery/services/sales/infrastructure/postgres"
)

// GetReport covers the half-open day [date 00:00, date+1 00:00) in the session time zone: a sale one
// second before midnight at either end is outside, the first and last second inside. Other stores
// and other tenants never count for the store-scoped report.
func TestGetReport_DayBoundaries(t *testing.T) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, testdb.NewMigratedURL(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	tenant, otherTenant, store, otherStore := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	const day = "2026-03-10"
	insert := func(tenantID, storeID uuid.UUID, completedAt string, total int) {
		t.Helper()
		sale := uuid.NewV7()
		if _, err := pool.Exec(ctx, `
			INSERT INTO sales.sales (id, tenant_id, store_id, cashier_id, subtotal, total, completed_at)
			VALUES ($1, $2, $3, $4, $5, $5, `+completedAt+`)`,
			sale, tenantID, storeID, uuid.NewV7(), total); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `
			INSERT INTO sales.sale_items (id, tenant_id, sale_id, product_id, quantity, unit_price, line_total)
			VALUES ($1, $2, $3, $4, 1, $5, $5)`,
			uuid.NewV7(), tenantID, sale, uuid.NewV7(), total); err != nil {
			t.Fatal(err)
		}
	}
	dayStart := `'` + day + `'::date::timestamptz`
	nextStart := `('` + day + `'::date + 1)::timestamptz`
	insert(tenant, store, dayStart+` - interval '1 second'`, 1)   // before: excluded
	insert(tenant, store, dayStart, 10)                           // first instant: included
	insert(tenant, store, nextStart+` - interval '1 second'`, 20) // last second: included
	insert(tenant, store, nextStart, 100)                         // next day: excluded
	insert(tenant, otherStore, dayStart, 1000)                    // other store: only in the tenant-wide report
	insert(otherTenant, store, dayStart, 10000)                   // other tenant: never

	repo := postgres.NewSaleRepository(pool)
	for _, tc := range []struct {
		name  string
		store *uuid.UUID
		count int
		total float64
	}{
		{"store", &store, 2, 30},
		{"all stores", nil, 3, 1030},
	} {
		rep, err := repo.GetReport(ctx, &tenant, tc.store, day)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if rep.TransactionCount != tc.count || rep.TotalRevenue != tc.total {
			t.Errorf("%s: count=%d revenue=%v, want %d and %v", tc.name, rep.TransactionCount, rep.TotalRevenue, tc.count, tc.total)
		}
		var units int
		for _, tp := range rep.TopProducts {
			units += tp.UnitsSold
		}
		if units != tc.count {
			t.Errorf("%s: top products sum %d units, want %d", tc.name, units, tc.count)
		}
	}
}
