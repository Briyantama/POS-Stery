package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Briyantama/POS-Stery/internal/testdb"
	"github.com/Briyantama/POS-Stery/services/supplier/domain"
	"github.com/Briyantama/POS-Stery/services/supplier/infrastructure/postgres"
)

// fixture seeds one PENDING order with an item per product, each with stock of 10.
// A product in noStock gets an order item but no inventory row.
type fixture struct {
	pool          *pgxpool.Pool
	repo          *postgres.PurchaseOrderRepository
	tenant, store uuid.UUID
	order         uuid.UUID
}

func newFixture(t *testing.T, products, noStock []uuid.UUID) *fixture {
	t.Helper()
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, testdb.NewMigratedURL(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	f := &fixture{pool: pool, repo: postgres.NewPurchaseOrderRepository(pool), tenant: uuid.New(), store: uuid.New(), order: uuid.New()}
	supplier := uuid.New()
	mustExec(t, pool, `INSERT INTO supplier.suppliers (id, tenant_id, name) VALUES ($1, $2, 'acme')`, supplier, f.tenant)
	mustExec(t, pool, `INSERT INTO supplier.purchase_orders (id, tenant_id, store_id, supplier_id, notes) VALUES ($1, $2, $3, $4, '')`, f.order, f.tenant, f.store, supplier)
	for _, p := range append(append([]uuid.UUID{}, products...), noStock...) {
		mustExec(t, pool, `INSERT INTO supplier.purchase_order_items (tenant_id, purchase_order_id, product_id, quantity_ordered, unit_cost) VALUES ($1, $2, $3, 100, 1)`, f.tenant, f.order, p)
	}
	for _, p := range products {
		mustExec(t, pool, `INSERT INTO inventory.stock_items (tenant_id, store_id, product_id, quantity) VALUES ($1, $2, $3, 10)`, f.tenant, f.store, p)
	}
	return f
}

func mustExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) stock(t *testing.T, product uuid.UUID) int64 {
	t.Helper()
	var q int64
	err := f.pool.QueryRow(context.Background(),
		`SELECT quantity FROM inventory.stock_items WHERE tenant_id = $1 AND store_id = $2 AND product_id = $3`,
		f.tenant, f.store, product).Scan(&q)
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func (f *fixture) status(t *testing.T) string {
	t.Helper()
	var s string
	if err := f.pool.QueryRow(context.Background(), `SELECT status FROM supplier.purchase_orders WHERE id = $1`, f.order).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *fixture) receive(lines ...domain.ReceivedLine) (*domain.PurchaseOrder, []domain.ReceivedLine, error) {
	return f.repo.ReceiveOrder(context.Background(), f.tenant, f.store, f.order, lines)
}

func TestReceiveOrder_IncrementsStockRelativelyAndMarksReceived(t *testing.T) {
	p := uuid.New()
	f := newFixture(t, []uuid.UUID{p}, nil)

	order, applied, err := f.receive(domain.ReceivedLine{ProductID: p, Quantity: 5})
	if err != nil {
		t.Fatal(err)
	}
	if order.Status != domain.StatusReceived || order.Items[0].QuantityReceived != 5 || len(applied) != 1 {
		t.Fatalf("unexpected result: status=%s received=%d applied=%d", order.Status, order.Items[0].QuantityReceived, len(applied))
	}
	if got := f.stock(t, p); got != 15 {
		t.Fatalf("stock = %d, want 15", got)
	}
}

func TestReceiveOrder_SecondReceiptConflictsAndDoesNotDoubleCount(t *testing.T) {
	p := uuid.New()
	f := newFixture(t, []uuid.UUID{p}, nil)
	line := domain.ReceivedLine{ProductID: p, Quantity: 5}

	if _, _, err := f.receive(line); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.receive(line); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second receipt err = %v, want ErrConflict", err)
	}
	if got := f.stock(t, p); got != 15 {
		t.Fatalf("stock = %d, want 15 (applied once)", got)
	}
}

func TestReceiveOrder_ConcurrentReceiptsApplyOnce(t *testing.T) {
	p := uuid.New()
	f := newFixture(t, []uuid.UUID{p}, nil)

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, errs[i] = f.receive(domain.ReceivedLine{ProductID: p, Quantity: 5})
		}()
	}
	wg.Wait()

	ok := 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case !errors.Is(err, domain.ErrConflict):
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Fatalf("%d receipts succeeded, want exactly 1", ok)
	}
	if got := f.stock(t, p); got != 15 {
		t.Fatalf("stock = %d, want 15", got)
	}
}

// A failure on a later line must roll back earlier lines: stock, quantity_received and status.
func TestReceiveOrder_FailureRollsBackEverything(t *testing.T) {
	withStock, withoutStock := uuid.New(), uuid.New()
	f := newFixture(t, []uuid.UUID{withStock}, []uuid.UUID{withoutStock})

	_, _, err := f.receive(
		domain.ReceivedLine{ProductID: withStock, Quantity: 5},
		domain.ReceivedLine{ProductID: withoutStock, Quantity: 5},
	)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if got := f.stock(t, withStock); got != 10 {
		t.Fatalf("stock = %d, want 10 (rolled back)", got)
	}
	if got := f.status(t); got != string(domain.StatusPending) {
		t.Fatalf("status = %s, want PENDING", got)
	}
	var received int
	mustScan(t, f.pool, &received, `SELECT quantity_received FROM supplier.purchase_order_items WHERE purchase_order_id = $1 AND product_id = $2`, f.order, withStock)
	if received != 0 {
		t.Fatalf("quantity_received = %d, want 0 (rolled back)", received)
	}
}

func TestReceiveOrder_CancelledOrderConflicts(t *testing.T) {
	p := uuid.New()
	f := newFixture(t, []uuid.UUID{p}, nil)
	mustExec(t, f.pool, `UPDATE supplier.purchase_orders SET status = 'CANCELLED' WHERE id = $1`, f.order)

	if _, _, err := f.receive(domain.ReceivedLine{ProductID: p, Quantity: 5}); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
}

func TestReceiveOrder_UnknownOrderNotFound(t *testing.T) {
	f := newFixture(t, nil, nil)
	_, _, err := f.repo.ReceiveOrder(context.Background(), f.tenant, f.store, uuid.New(), []domain.ReceivedLine{{ProductID: uuid.New(), Quantity: 1}})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func mustScan(t *testing.T, pool *pgxpool.Pool, dst any, sql string, args ...any) {
	t.Helper()
	if err := pool.QueryRow(context.Background(), sql, args...).Scan(dst); err != nil {
		t.Fatal(err)
	}
}
