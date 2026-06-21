package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/supplier-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/supplier-service/internal/domain"
)

// ── stubs ──────────────────────────────────────────────────────────────────────

type stubPORepo struct {
	order        *domain.PurchaseOrder
	getErr       error
	updateErr    error
	updatedOrder *domain.PurchaseOrder
}

func (r *stubPORepo) Create(_ context.Context, _ *domain.PurchaseOrder) error { return nil }

func (r *stubPORepo) Get(_ context.Context, _, _, _ uuid.UUID) (*domain.PurchaseOrder, error) {
	return r.order, r.getErr
}

func (r *stubPORepo) List(_ context.Context, _, _ uuid.UUID, _ string, _, _ int) ([]*domain.PurchaseOrder, int, error) {
	return nil, 0, nil
}

func (r *stubPORepo) UpdateReceived(_ context.Context, order *domain.PurchaseOrder) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updatedOrder = order
	return nil
}

type stubInventoryPort struct {
	updateErr   error
	updateCalls int
}

func (p *stubInventoryPort) UpdateStock(_ context.Context, _, _, _ string, _ int64, _, _ string) error {
	p.updateCalls++
	return p.updateErr
}

type stubSupplierPublisher struct {
	calls int
}

func (p *stubSupplierPublisher) PublishReplenished(_ context.Context, _, _, _, _ string, _ int64) error {
	p.calls++
	return nil
}

// ── helpers ────────────────────────────────────────────────────────────────────

func pendingOrder(tenantID, storeID, productID uuid.UUID, qty int32) *domain.PurchaseOrder {
	return &domain.PurchaseOrder{
		ID:         uuid.New(),
		TenantID:   tenantID,
		StoreID:    storeID,
		SupplierID: uuid.New(),
		Status:     domain.StatusPending,
		Items: []*domain.PurchaseOrderItem{
			{
				ID:              uuid.New(),
				TenantID:        tenantID,
				PurchaseOrderID: uuid.New(),
				ProductID:       productID,
				QuantityOrdered: qty,
			},
		},
	}
}

func newReceiveHandler(repo *stubPORepo, inv *stubInventoryPort, pub *stubSupplierPublisher) *commands.ReceiveStockHandler {
	return commands.NewReceiveStockHandler(repo, inv, pub)
}

// ── tests ──────────────────────────────────────────────────────────────────────

func TestReceiveStock_MissingTenantID(t *testing.T) {
	cmd := commands.ReceiveStockCommand{
		TenantID:        uuid.Nil,
		StoreID:         uuid.New(),
		PurchaseOrderID: uuid.New(),
		ItemsReceived:   []commands.ReceivedLineItem{{ProductID: uuid.New(), QuantityReceived: 5}},
	}
	_, err := newReceiveHandler(&stubPORepo{}, &stubInventoryPort{}, &stubSupplierPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestReceiveStock_MissingStoreID(t *testing.T) {
	cmd := commands.ReceiveStockCommand{
		TenantID:        uuid.New(),
		StoreID:         uuid.Nil,
		PurchaseOrderID: uuid.New(),
		ItemsReceived:   []commands.ReceivedLineItem{{ProductID: uuid.New(), QuantityReceived: 5}},
	}
	_, err := newReceiveHandler(&stubPORepo{}, &stubInventoryPort{}, &stubSupplierPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestReceiveStock_MissingPurchaseOrderID(t *testing.T) {
	cmd := commands.ReceiveStockCommand{
		TenantID:        uuid.New(),
		StoreID:         uuid.New(),
		PurchaseOrderID: uuid.Nil,
		ItemsReceived:   []commands.ReceivedLineItem{{ProductID: uuid.New(), QuantityReceived: 5}},
	}
	_, err := newReceiveHandler(&stubPORepo{}, &stubInventoryPort{}, &stubSupplierPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestReceiveStock_EmptyItems(t *testing.T) {
	cmd := commands.ReceiveStockCommand{
		TenantID:        uuid.New(),
		StoreID:         uuid.New(),
		PurchaseOrderID: uuid.New(),
		ItemsReceived:   nil,
	}
	_, err := newReceiveHandler(&stubPORepo{}, &stubInventoryPort{}, &stubSupplierPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument for empty items, got %v", err)
	}
}

func TestReceiveStock_AlreadyReceivedOrder(t *testing.T) {
	tenantID, storeID, productID := uuid.New(), uuid.New(), uuid.New()
	order := pendingOrder(tenantID, storeID, productID, 10)
	order.Status = domain.StatusReceived

	cmd := commands.ReceiveStockCommand{
		TenantID:        tenantID,
		StoreID:         storeID,
		PurchaseOrderID: order.ID,
		ItemsReceived:   []commands.ReceivedLineItem{{ProductID: productID, QuantityReceived: 5}},
	}
	_, err := newReceiveHandler(&stubPORepo{order: order}, &stubInventoryPort{}, &stubSupplierPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument for already-received order, got %v", err)
	}
}

func TestReceiveStock_CancelledOrder(t *testing.T) {
	tenantID, storeID, productID := uuid.New(), uuid.New(), uuid.New()
	order := pendingOrder(tenantID, storeID, productID, 10)
	order.Status = domain.StatusCancelled

	repo := &stubPORepo{order: order}
	cmd := commands.ReceiveStockCommand{
		TenantID:        tenantID,
		StoreID:         storeID,
		PurchaseOrderID: order.ID,
		ItemsReceived:   []commands.ReceivedLineItem{{ProductID: productID, QuantityReceived: 5}},
	}
	_, err := newReceiveHandler(repo, &stubInventoryPort{}, &stubSupplierPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument for cancelled order, got %v", err)
	}
}

func TestReceiveStock_InventoryUpdateFails(t *testing.T) {
	tenantID, storeID, productID := uuid.New(), uuid.New(), uuid.New()
	order := pendingOrder(tenantID, storeID, productID, 10)

	repo := &stubPORepo{order: order}
	inv := &stubInventoryPort{updateErr: errors.New("inventory unavailable")}
	cmd := commands.ReceiveStockCommand{
		TenantID:        tenantID,
		StoreID:         storeID,
		PurchaseOrderID: order.ID,
		ItemsReceived:   []commands.ReceivedLineItem{{ProductID: productID, QuantityReceived: 5}},
	}
	_, err := newReceiveHandler(repo, inv, &stubSupplierPublisher{}).Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("want error when inventory update fails")
	}
}

func TestReceiveStock_HappyPath(t *testing.T) {
	tenantID, storeID, productID := uuid.New(), uuid.New(), uuid.New()
	order := pendingOrder(tenantID, storeID, productID, 10)

	repo := &stubPORepo{order: order}
	inv := &stubInventoryPort{}
	pub := &stubSupplierPublisher{}
	cmd := commands.ReceiveStockCommand{
		TenantID:        tenantID,
		StoreID:         storeID,
		PurchaseOrderID: order.ID,
		ItemsReceived:   []commands.ReceivedLineItem{{ProductID: productID, QuantityReceived: 5}},
	}

	result, err := newReceiveHandler(repo, inv, pub).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != domain.StatusReceived {
		t.Errorf("want status=RECEIVED, got %s", result.Status)
	}
	if inv.updateCalls != 1 {
		t.Errorf("want 1 inventory update call, got %d", inv.updateCalls)
	}
	if pub.calls != 1 {
		t.Errorf("want 1 publish call, got %d", pub.calls)
	}
	if repo.updatedOrder == nil {
		t.Error("order must be persisted after receipt")
	}
}

func TestReceiveStock_ProductNotInOrder_Skipped(t *testing.T) {
	tenantID, storeID, productID := uuid.New(), uuid.New(), uuid.New()
	order := pendingOrder(tenantID, storeID, productID, 10)

	repo := &stubPORepo{order: order}
	inv := &stubInventoryPort{}
	unknownProduct := uuid.New()
	cmd := commands.ReceiveStockCommand{
		TenantID:        tenantID,
		StoreID:         storeID,
		PurchaseOrderID: order.ID,
		ItemsReceived:   []commands.ReceivedLineItem{{ProductID: unknownProduct, QuantityReceived: 3}},
	}

	_, err := newReceiveHandler(repo, inv, &stubSupplierPublisher{}).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if inv.updateCalls != 0 {
		t.Errorf("want 0 inventory calls for unmatched product, got %d", inv.updateCalls)
	}
}
