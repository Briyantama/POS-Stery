package postgres_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Briyantama/POS-Stery/internal/testdb"
	"github.com/Briyantama/POS-Stery/services/product/domain"
	"github.com/Briyantama/POS-Stery/services/product/infrastructure/postgres"
)

// Two tenants, one category each. The tests run as the database owner, so RLS is not enforced;
// they exercise the explicit tenant filter and the composite FK, which must hold either way.
func setup(t *testing.T) (repo *postgres.ProductRepository, pool *pgxpool.Pool, tenantA, tenantB, catA, catB uuid.UUID) {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), testdb.NewMigratedURL(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	tenantA, tenantB, catA, catB = uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, c := range []struct{ id, tenant uuid.UUID }{{catA, tenantA}, {catB, tenantB}} {
		if _, err := pool.Exec(context.Background(), `INSERT INTO product.categories (id, tenant_id, name) VALUES ($1, $2, 'c')`, c.id, c.tenant); err != nil {
			t.Fatal(err)
		}
	}
	return postgres.NewProductRepository(pool), pool, tenantA, tenantB, catA, catB
}

func TestCategoryExists_OnlyOwnTenant(t *testing.T) {
	repo, _, tenantA, _, catA, catB := setup(t)

	for _, tc := range []struct {
		name string
		id   uuid.UUID
		want bool
	}{{"own", catA, true}, {"other tenant", catB, false}, {"unknown", uuid.New(), false}} {
		got, err := repo.CategoryExists(context.Background(), tenantA, tc.id)
		if err != nil || got != tc.want {
			t.Errorf("%s: got %v, %v; want %v", tc.name, got, err, tc.want)
		}
	}
}

func TestCreateUpdate_CategoryOfOtherTenantRejected(t *testing.T) {
	repo, _, tenantA, _, catA, catB := setup(t)
	ctx := context.Background()
	p := &domain.Product{ID: uuid.New(), TenantID: tenantA, CategoryID: &catB, Name: "w", SKU: "s", IsActive: true}

	if err := repo.Create(ctx, tenantA, p); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("create with foreign category: err = %v, want ErrInvalidArgument", err)
	}

	p.CategoryID = &catA
	if err := repo.Create(ctx, tenantA, p); err != nil {
		t.Fatalf("create with own category: %v", err)
	}
	p.CategoryID = &catB
	if err := repo.Update(ctx, tenantA, p); !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("update to foreign category: err = %v, want ErrInvalidArgument", err)
	}
}
