package commands_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/domain"
)

// ── stubs ──────────────────────────────────────────────────────────────────────

type stubStockRepo struct {
	current   *domain.StockItem
	getErr    error
	updateErr error
}

func (r *stubStockRepo) AddItem(_ context.Context, _ *domain.StockItem) error { return nil }

func (r *stubStockRepo) UpdateStock(
	_ context.Context,
	_, _, _ uuid.UUID,
	delta int64,
	_, _ string,
) (*domain.StockItem, error) {
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	updated := *r.current
	updated.Quantity += delta
	return &updated, nil
}

func (r *stubStockRepo) GetItem(_ context.Context, _, _, _ uuid.UUID) (*domain.StockItem, error) {
	return r.current, r.getErr
}

func (r *stubStockRepo) ListItems(_ context.Context, _, _ uuid.UUID, _ bool, _, _ int) ([]*domain.StockItem, int, error) {
	return nil, 0, nil
}

type stubThresholdRepo struct {
	threshold *domain.StockThreshold
	err       error
}

func (r *stubThresholdRepo) SetThreshold(_ context.Context, _ *domain.StockThreshold) error {
	return nil
}

func (r *stubThresholdRepo) GetThreshold(_ context.Context, _, _, _ uuid.UUID) (*domain.StockThreshold, error) {
	return r.threshold, r.err
}

type stubInvPublisher struct {
	lowStockCalls int
}

func (p *stubInvPublisher) PublishLowStock(_ context.Context, _ domain.LowStockDetected) error {
	p.lowStockCalls++
	return nil
}

func (p *stubInvPublisher) PublishReplenished(_ context.Context, _, _, _ string, _ int64) error {
	return nil
}

type failingPublisher struct{}

func (p *failingPublisher) PublishLowStock(_ context.Context, _ domain.LowStockDetected) error {
	return errors.New("nats down")
}

func (p *failingPublisher) PublishReplenished(_ context.Context, _, _, _ string, _ int64) error {
	return errors.New("nats down")
}

// ── helpers ────────────────────────────────────────────────────────────────────

func stockItem(qty int64) *domain.StockItem {
	return &domain.StockItem{
		ID:        uuid.New(),
		TenantID:  uuid.New(),
		StoreID:   uuid.New(),
		ProductID: uuid.New(),
		Quantity:  qty,
	}
}

func validUpdateCmd(tenantID, storeID, productID uuid.UUID, delta int64) commands.UpdateStockCommand {
	return commands.UpdateStockCommand{
		TenantID:  tenantID,
		StoreID:   storeID,
		ProductID: productID,
		Delta:     delta,
		Reason:    "ADJUSTMENT",
		RefID:     "ref-1",
	}
}

// ── tests ──────────────────────────────────────────────────────────────────────

func TestUpdateStock_MissingTenantID(t *testing.T) {
	cmd := validUpdateCmd(uuid.Nil, uuid.New(), uuid.New(), 5)
	_, err := commands.NewUpdateStockHandler(&stubStockRepo{}, &stubThresholdRepo{}, &stubInvPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestUpdateStock_MissingStoreID(t *testing.T) {
	cmd := validUpdateCmd(uuid.New(), uuid.Nil, uuid.New(), 5)
	_, err := commands.NewUpdateStockHandler(&stubStockRepo{}, &stubThresholdRepo{}, &stubInvPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestUpdateStock_MissingProductID(t *testing.T) {
	cmd := validUpdateCmd(uuid.New(), uuid.New(), uuid.Nil, 5)
	_, err := commands.NewUpdateStockHandler(&stubStockRepo{}, &stubThresholdRepo{}, &stubInvPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestUpdateStock_GetItemError(t *testing.T) {
	repo := &stubStockRepo{getErr: errors.New("db down")}
	cmd := validUpdateCmd(uuid.New(), uuid.New(), uuid.New(), -3)
	_, err := commands.NewUpdateStockHandler(repo, &stubThresholdRepo{}, &stubInvPublisher{}).Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("want error when GetItem fails")
	}
}

func TestUpdateStock_WouldGoNegative(t *testing.T) {
	item := stockItem(2)
	repo := &stubStockRepo{current: item}
	cmd := validUpdateCmd(item.TenantID, item.StoreID, item.ProductID, -5)

	_, err := commands.NewUpdateStockHandler(repo, &stubThresholdRepo{}, &stubInvPublisher{}).Handle(context.Background(), cmd)
	if !errors.Is(err, sherrors.ErrStockWouldGoNegative) {
		t.Fatalf("want ErrStockWouldGoNegative, got %v", err)
	}
}

func TestUpdateStock_UpdateRepoError(t *testing.T) {
	item := stockItem(10)
	repo := &stubStockRepo{current: item, updateErr: errors.New("db error")}
	cmd := validUpdateCmd(item.TenantID, item.StoreID, item.ProductID, -3)

	_, err := commands.NewUpdateStockHandler(repo, &stubThresholdRepo{}, &stubInvPublisher{}).Handle(context.Background(), cmd)
	if err == nil {
		t.Fatal("want error when UpdateStock repo fails")
	}
}

func TestUpdateStock_AddStock_NoThreshold(t *testing.T) {
	item := stockItem(5)
	repo := &stubStockRepo{current: item}
	thresh := &stubThresholdRepo{err: errors.New("not found")}
	cmd := validUpdateCmd(item.TenantID, item.StoreID, item.ProductID, 10)
	pub := &stubInvPublisher{}

	res, err := commands.NewUpdateStockHandler(repo, thresh, pub).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Item.Quantity != 15 {
		t.Errorf("want quantity=15, got %d", res.Item.Quantity)
	}
	if res.LowStockAlert || pub.lowStockCalls != 0 {
		t.Error("no alert expected when no threshold is configured")
	}
}

func TestUpdateStock_DeductStock(t *testing.T) {
	item := stockItem(20)
	repo := &stubStockRepo{current: item}
	thresh := &stubThresholdRepo{err: errors.New("not found")}
	cmd := validUpdateCmd(item.TenantID, item.StoreID, item.ProductID, -8)

	res, err := commands.NewUpdateStockHandler(repo, thresh, &stubInvPublisher{}).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Item.Quantity != 12 {
		t.Errorf("want quantity=12, got %d", res.Item.Quantity)
	}
}

func TestUpdateStock_LowStockAlertTriggered(t *testing.T) {
	item := stockItem(3)
	repo := &stubStockRepo{current: item}
	thresh := &stubThresholdRepo{threshold: &domain.StockThreshold{MinQuantity: 5}}
	cmd := validUpdateCmd(item.TenantID, item.StoreID, item.ProductID, -1)
	pub := &stubInvPublisher{}

	res, err := commands.NewUpdateStockHandler(repo, thresh, pub).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.LowStockAlert {
		t.Error("want LowStockAlert=true when quantity <= threshold")
	}
	if pub.lowStockCalls != 1 {
		t.Errorf("want 1 low-stock event, got %d", pub.lowStockCalls)
	}
}

func TestUpdateStock_NoAlertWhenAboveThreshold(t *testing.T) {
	item := stockItem(10)
	repo := &stubStockRepo{current: item}
	thresh := &stubThresholdRepo{threshold: &domain.StockThreshold{MinQuantity: 5}}
	cmd := validUpdateCmd(item.TenantID, item.StoreID, item.ProductID, -1)
	pub := &stubInvPublisher{}

	res, err := commands.NewUpdateStockHandler(repo, thresh, pub).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.LowStockAlert || pub.lowStockCalls != 0 {
		t.Error("no alert expected when quantity still above threshold")
	}
}

func TestUpdateStock_PublishFailureIsNonFatal(t *testing.T) {
	item := stockItem(2)
	repo := &stubStockRepo{current: item}
	thresh := &stubThresholdRepo{threshold: &domain.StockThreshold{MinQuantity: 5}}
	cmd := validUpdateCmd(item.TenantID, item.StoreID, item.ProductID, -1)

	res, err := commands.NewUpdateStockHandler(repo, thresh, &failingPublisher{}).Handle(context.Background(), cmd)
	if err != nil {
		t.Fatalf("publisher failure must not propagate: %v", err)
	}
	if res.Item == nil {
		t.Fatal("expected updated item on success")
	}
}
