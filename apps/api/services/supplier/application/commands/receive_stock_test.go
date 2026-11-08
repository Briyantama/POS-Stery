package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Briyantama/POS-Stery/services/supplier/application/commands"
	"github.com/Briyantama/POS-Stery/services/supplier/domain"
	"github.com/google/uuid"
)

// ── stubs ──────────────────────────────────────────────────────────────────────

type stubPORepo struct {
	order      *domain.PurchaseOrder
	applied    []domain.ReceivedLine
	receiveErr error
	gotLines   []domain.ReceivedLine
	calls      int
}

func (r *stubPORepo) Create(_ context.Context, _ *domain.PurchaseOrder) error { return nil }

func (r *stubPORepo) Get(_ context.Context, _, _, _ uuid.UUID) (*domain.PurchaseOrder, error) {
	return r.order, nil
}

func (r *stubPORepo) List(_ context.Context, _, _ uuid.UUID, _ string, _, _ int) ([]*domain.PurchaseOrder, int, error) {
	return nil, 0, nil
}

func (r *stubPORepo) ReceiveOrder(_ context.Context, _, _, _ uuid.UUID, lines []domain.ReceivedLine) (*domain.PurchaseOrder, []domain.ReceivedLine, error) {
	r.calls++
	r.gotLines = lines
	if r.receiveErr != nil {
		return nil, nil, r.receiveErr
	}
	return r.order, r.applied, nil
}

type stubSupplierPublisher struct {
	calls int
}

func (p *stubSupplierPublisher) PublishReplenished(_ context.Context, _, _, _, _ string, _ int64) error {
	p.calls++
	return nil
}

// ── helpers ────────────────────────────────────────────────────────────────────

func receiveCmd(items ...commands.ReceivedLineItem) commands.ReceiveStockCommand {
	return commands.ReceiveStockCommand{
		TenantID:        uuid.New(),
		StoreID:         uuid.New(),
		PurchaseOrderID: uuid.New(),
		ItemsReceived:   items,
	}
}

func line(productID uuid.UUID, qty int32) commands.ReceivedLineItem {
	return commands.ReceivedLineItem{ProductID: productID, QuantityReceived: qty}
}

// ── tests ──────────────────────────────────────────────────────────────────────

func TestReceiveStock_InvalidArguments(t *testing.T) {
	tests := map[string]func(*commands.ReceiveStockCommand){
		"missing tenant": func(c *commands.ReceiveStockCommand) { c.TenantID = uuid.Nil },
		"missing store":  func(c *commands.ReceiveStockCommand) { c.StoreID = uuid.Nil },
		"missing order":  func(c *commands.ReceiveStockCommand) { c.PurchaseOrderID = uuid.Nil },
		"no items":       func(c *commands.ReceiveStockCommand) { c.ItemsReceived = nil },
		"nil product": func(c *commands.ReceiveStockCommand) {
			c.ItemsReceived = []commands.ReceivedLineItem{line(uuid.Nil, 5)}
		},
		"zero quantity": func(c *commands.ReceiveStockCommand) {
			c.ItemsReceived = []commands.ReceivedLineItem{line(uuid.New(), 0)}
		},
		"negative quantity": func(c *commands.ReceiveStockCommand) {
			c.ItemsReceived = []commands.ReceivedLineItem{line(uuid.New(), -1)}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			cmd := receiveCmd(line(uuid.New(), 5))
			mutate(&cmd)
			repo := &stubPORepo{}
			_, err := commands.NewReceiveStockHandler(repo, &stubSupplierPublisher{}).Handle(context.Background(), cmd)
			if !errors.Is(err, domain.ErrInvalidArgument) {
				t.Fatalf("want ErrInvalidArgument, got %v", err)
			}
			if repo.calls != 0 {
				t.Fatal("repository must not be called for invalid input")
			}
		})
	}
}

// The repository refuses a non-PENDING order (already received or cancelled). The handler must
// surface the conflict and publish nothing.
func TestReceiveStock_RepoConflict_NoEvents(t *testing.T) {
	repo := &stubPORepo{receiveErr: domain.ErrConflict}
	pub := &stubSupplierPublisher{}

	_, err := commands.NewReceiveStockHandler(repo, pub).Handle(context.Background(), receiveCmd(line(uuid.New(), 5)))
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	if pub.calls != 0 {
		t.Errorf("want 0 publish calls, got %d", pub.calls)
	}
}

// A failure inside the receipt transaction (e.g. missing stock row) propagates; no events.
func TestReceiveStock_RepoFails_NoEvents(t *testing.T) {
	repo := &stubPORepo{receiveErr: domain.ErrNotFound}
	pub := &stubSupplierPublisher{}

	_, err := commands.NewReceiveStockHandler(repo, pub).Handle(context.Background(), receiveCmd(line(uuid.New(), 5)))
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if pub.calls != 0 {
		t.Errorf("want 0 publish calls, got %d", pub.calls)
	}
}

func TestReceiveStock_HappyPath(t *testing.T) {
	productID := uuid.New()
	order := &domain.PurchaseOrder{ID: uuid.New(), Status: domain.StatusReceived}
	repo := &stubPORepo{order: order, applied: []domain.ReceivedLine{{ProductID: productID, Quantity: 5}}}
	pub := &stubSupplierPublisher{}

	result, err := commands.NewReceiveStockHandler(repo, pub).Handle(context.Background(), receiveCmd(line(productID, 5)))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != order {
		t.Error("want the order returned by the repository")
	}
	if repo.calls != 1 {
		t.Errorf("want 1 repository receipt, got %d", repo.calls)
	}
	if pub.calls != 1 {
		t.Errorf("want 1 publish call, got %d", pub.calls)
	}
}

// Lines the repository skipped (product not on the order) must not produce events.
func TestReceiveStock_ProductNotInOrder_NoEvent(t *testing.T) {
	repo := &stubPORepo{order: &domain.PurchaseOrder{ID: uuid.New()}, applied: nil}
	pub := &stubSupplierPublisher{}

	if _, err := commands.NewReceiveStockHandler(repo, pub).Handle(context.Background(), receiveCmd(line(uuid.New(), 3))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pub.calls != 0 {
		t.Errorf("want 0 publish calls for unmatched product, got %d", pub.calls)
	}
}

// Repeated lines for one product are merged so stock and quantity_received are booked once.
func TestReceiveStock_DuplicateProductLinesMerged(t *testing.T) {
	productID := uuid.New()
	repo := &stubPORepo{order: &domain.PurchaseOrder{ID: uuid.New()}}

	if _, err := commands.NewReceiveStockHandler(repo, &stubSupplierPublisher{}).Handle(context.Background(), receiveCmd(line(productID, 2), line(productID, 3))); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.gotLines) != 1 || repo.gotLines[0].Quantity != 5 {
		t.Fatalf("want one merged line of 5, got %+v", repo.gotLines)
	}
}
