package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"

	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/sales-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/sales-service/internal/domain"
)

// ---- stubs ----

type stubInventory struct {
	deductErr    error
	restoreErr   error
	deductCalls  int
	restoreCalls int
}

func (s *stubInventory) DeductStock(_ context.Context, _, _, _ string, _ int64, _ string) error {
	s.deductCalls++
	return s.deductErr
}

func (s *stubInventory) RestoreStock(_ context.Context, _, _, _ string, _ int64, _ string) error {
	s.restoreCalls++
	return s.restoreErr
}

type stubSaleRepo struct {
	createErr error
}

func (s *stubSaleRepo) Create(_ context.Context, _ *domain.Sale, _ []domain.SaleItem, _ *domain.Receipt) error {
	return s.createErr
}

func (s *stubSaleRepo) FindByID(_ context.Context, _, _, _ uuid.UUID) (*domain.Sale, []domain.SaleItem, *domain.Receipt, error) {
	return nil, nil, nil, nil
}

func (s *stubSaleRepo) GetReport(_ context.Context, _, _ *uuid.UUID, _ string) (*domain.SalesReport, error) {
	return nil, nil
}

type stubPublisher struct{ published bool }

func (s *stubPublisher) PublishSaleCompleted(_ context.Context, _ *domain.Sale) error {
	s.published = true
	return nil
}

// ---- helpers ----

func validCmd() commands.CreateSaleCommand {
	return commands.CreateSaleCommand{
		TenantID:  uuid.New().String(),
		StoreID:   uuid.New().String(),
		CashierID: uuid.New().String(),
		Items: []commands.SaleItemInput{
			{ProductID: uuid.New().String(), Quantity: 2, UnitPrice: 10.0},
		},
	}
}

func newHandler(inv *stubInventory, repo *stubSaleRepo, pub *stubPublisher) *commands.CreateSaleHandler {
	return commands.NewCreateSaleHandler(repo, inv, pub, zap.NewNop())
}

// ---- tests ----

func TestCreateSaleHandler_MissingTenantID(t *testing.T) {
	cmd := validCmd()
	cmd.TenantID = ""

	_, err := newHandler(&stubInventory{}, &stubSaleRepo{}, &stubPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestCreateSaleHandler_NoItems(t *testing.T) {
	cmd := validCmd()
	cmd.Items = nil

	_, err := newHandler(&stubInventory{}, &stubSaleRepo{}, &stubPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestCreateSaleHandler_NegativeQuantity(t *testing.T) {
	cmd := validCmd()
	cmd.Items[0].Quantity = -1

	_, err := newHandler(&stubInventory{}, &stubSaleRepo{}, &stubPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestCreateSaleHandler_StockShortage(t *testing.T) {
	cmd := validCmd()
	inv := &stubInventory{deductErr: sherrors.ErrStockWouldGoNegative}
	pub := &stubPublisher{}
	repo := &stubSaleRepo{}

	_, err := newHandler(inv, repo, pub).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrStockWouldGoNegative) {
		t.Fatalf("want ErrStockWouldGoNegative, got %v", err)
	}
	if pub.published {
		t.Error("event must not be published on stock failure")
	}
}

func TestCreateSaleHandler_DBFailureTriggersCompensation(t *testing.T) {
	cmd := validCmd()
	cmd.Items = append(cmd.Items, commands.SaleItemInput{
		ProductID: uuid.New().String(), Quantity: 1, UnitPrice: 5.0,
	})

	inv := &stubInventory{}
	repo := &stubSaleRepo{createErr: errors.New("db down")}
	pub := &stubPublisher{}

	_, err := newHandler(inv, repo, pub).Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("want error on DB failure, got nil")
	}
	if inv.restoreCalls != len(cmd.Items) {
		t.Errorf("want %d compensation calls, got %d", len(cmd.Items), inv.restoreCalls)
	}
	if pub.published {
		t.Error("event must not be published after DB failure")
	}
}

func TestCreateSaleHandler_HappyPath(t *testing.T) {
	cmd := validCmd()
	cmd.Items = []commands.SaleItemInput{
		{ProductID: uuid.New().String(), Quantity: 3, UnitPrice: 10.0, Discount: 1.0},
	}

	inv := &stubInventory{}
	repo := &stubSaleRepo{}
	pub := &stubPublisher{}

	res, err := newHandler(inv, repo, pub).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Sale == nil || res.Receipt == nil {
		t.Fatal("expected sale and receipt in result")
	}
	if res.Sale.Status != "COMPLETED" {
		t.Errorf("want status=COMPLETED, got %q", res.Sale.Status)
	}
	// lineTotal = 3 * 10 - 1 = 29
	if res.Sale.Subtotal != 29.0 {
		t.Errorf("want subtotal=29, got %v", res.Sale.Subtotal)
	}
	if inv.deductCalls != 1 {
		t.Errorf("want 1 deduct call, got %d", inv.deductCalls)
	}
	if !pub.published {
		t.Error("event must be published on success")
	}
}

func TestCreateSaleHandler_NegativeTotalClampedToZero(t *testing.T) {
	cmd := validCmd()
	cmd.Items = []commands.SaleItemInput{
		{ProductID: uuid.New().String(), Quantity: 1, UnitPrice: 5.0},
	}
	cmd.DiscountAmount = 100.0 // discount > subtotal

	res, err := newHandler(&stubInventory{}, &stubSaleRepo{}, &stubPublisher{}).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Sale.Total != 0.0 {
		t.Errorf("want total=0 when discount exceeds subtotal, got %v", res.Sale.Total)
	}
}
