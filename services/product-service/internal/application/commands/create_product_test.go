package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/product-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/product-service/internal/domain"
)

// ── stubs ──────────────────────────────────────────────────────────────────────

type stubProductRepo struct {
	createErr error
}

func (r *stubProductRepo) Create(_ context.Context, _ uuid.UUID, _ *domain.Product) error {
	return r.createErr
}

func (r *stubProductRepo) Update(_ context.Context, _ uuid.UUID, _ *domain.Product) error {
	return nil
}

func (r *stubProductRepo) FindByID(_ context.Context, _, _ uuid.UUID) (*domain.Product, error) {
	return nil, nil
}

func (r *stubProductRepo) Search(_ context.Context, _ uuid.UUID, _, _ string, _, _ int) ([]*domain.Product, int, error) {
	return nil, 0, nil
}

// ── helpers ────────────────────────────────────────────────────────────────────

func validCreateProductCmd() commands.CreateProductCommand {
	return commands.CreateProductCommand{
		TenantID:  uuid.New(),
		Name:      "Widget A",
		SKU:       "WGT-001",
		BasePrice: 9.99,
	}
}

// ── tests ──────────────────────────────────────────────────────────────────────

func TestCreateProduct_MissingTenantID(t *testing.T) {
	cmd := validCreateProductCmd()
	cmd.TenantID = uuid.Nil

	_, err := commands.NewCreateProductHandler(&stubProductRepo{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument for nil tenant_id, got %v", err)
	}
}

func TestCreateProduct_MissingName(t *testing.T) {
	cmd := validCreateProductCmd()
	cmd.Name = ""

	_, err := commands.NewCreateProductHandler(&stubProductRepo{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument for missing name, got %v", err)
	}
}

func TestCreateProduct_MissingSKU(t *testing.T) {
	cmd := validCreateProductCmd()
	cmd.SKU = ""

	_, err := commands.NewCreateProductHandler(&stubProductRepo{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument for missing SKU, got %v", err)
	}
}

func TestCreateProduct_NegativeBasePrice(t *testing.T) {
	cmd := validCreateProductCmd()
	cmd.BasePrice = -1.0

	_, err := commands.NewCreateProductHandler(&stubProductRepo{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument for negative base_price, got %v", err)
	}
}

func TestCreateProduct_DuplicateSKU(t *testing.T) {
	repo := &stubProductRepo{createErr: sherrors.ErrAlreadyExists}
	cmd := validCreateProductCmd()

	_, err := commands.NewCreateProductHandler(repo).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrAlreadyExists) {
		t.Fatalf("want ErrAlreadyExists for duplicate SKU, got %v", err)
	}
}

func TestCreateProduct_HappyPath(t *testing.T) {
	cmd := validCreateProductCmd()
	catID := uuid.New()
	cmd.CategoryID = &catID
	cmd.Barcode = "123456789"
	cmd.Description = "A fine widget"
	cmd.Unit = "each"

	p, err := commands.NewCreateProductHandler(&stubProductRepo{}).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected product, got nil")
	}
	if p.ID == uuid.Nil {
		t.Error("product ID must be set")
	}
	if p.Name != cmd.Name {
		t.Errorf("want name=%q, got %q", cmd.Name, p.Name)
	}
	if p.SKU != cmd.SKU {
		t.Errorf("want SKU=%q, got %q", cmd.SKU, p.SKU)
	}
	if !p.IsActive {
		t.Error("new product must be active")
	}
}
