package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/sales-service/internal/application"
	"github.com/pos-stery/pos-stery/services/sales-service/internal/domain"
	"go.uber.org/zap"
)

// CreateSaleCommand carries the validated inputs for a new sale transaction.
type CreateSaleCommand struct {
	TenantID       string
	StoreID        string
	CashierID      string
	CustomerID     string // empty = no customer
	Items          []SaleItemInput
	DiscountAmount float64
	Notes          string
}

// SaleItemInput is the raw item data supplied by the caller.
type SaleItemInput struct {
	ProductID string
	Quantity  int
	UnitPrice float64
	Discount  float64
}

// CreateSaleResult is returned on success.
type CreateSaleResult struct {
	Sale    *domain.Sale
	Receipt *domain.Receipt
}

// CreateSaleHandler orchestrates the synchronous saga pattern for sale creation:
// validate → deduct stock (with compensation) → persist → publish event.
type CreateSaleHandler struct {
	saleRepo  domain.SaleRepository
	inventory application.InventoryUpdater
	publisher application.EventPublisher
	log       *zap.Logger
}

// NewCreateSaleHandler wires the handler with its dependencies.
func NewCreateSaleHandler(
	saleRepo domain.SaleRepository,
	inventory application.InventoryUpdater,
	publisher application.EventPublisher,
	log *zap.Logger,
) *CreateSaleHandler {
	return &CreateSaleHandler{
		saleRepo:  saleRepo,
		inventory: inventory,
		publisher: publisher,
		log:       log,
	}
}

// Handle executes the create-sale saga.
//
// Saga steps:
//  1. Validate inputs.
//  2. Calculate subtotal / total.
//  3. Deduct stock for each item; compensate on any failure.
//  4. Persist sale + items + receipt in one DB transaction.
//     On DB failure: compensate all previous deductions.
//  5. Publish Sale.Completed (best-effort).
func (h *CreateSaleHandler) Handle(ctx context.Context, cmd CreateSaleCommand) (*CreateSaleResult, error) {
	// ── 1. Validate ──────────────────────────────────────────────────────────
	if err := validateCreateSaleCommand(cmd); err != nil {
		return nil, err
	}

	tenantID, err := uuid.Parse(cmd.TenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tenant_id", sherrors.ErrInvalidArgument)
	}
	storeID, err := uuid.Parse(cmd.StoreID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid store_id", sherrors.ErrInvalidArgument)
	}
	cashierID, err := uuid.Parse(cmd.CashierID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cashier_id", sherrors.ErrInvalidArgument)
	}

	var customerID *uuid.UUID
	if cmd.CustomerID != "" {
		id, err := uuid.Parse(cmd.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid customer_id", sherrors.ErrInvalidArgument)
		}
		customerID = &id
	}

	// ── 2. Calculate amounts ─────────────────────────────────────────────────
	var subtotal float64
	items := make([]domain.SaleItem, 0, len(cmd.Items))

	for _, input := range cmd.Items {
		productID, err := uuid.Parse(input.ProductID)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid product_id %q", sherrors.ErrInvalidArgument, input.ProductID)
		}
		lineTotal := float64(input.Quantity)*input.UnitPrice - input.Discount
		subtotal += lineTotal

		items = append(items, domain.SaleItem{
			ID:        uuid.New(),
			TenantID:  tenantID,
			ProductID: productID,
			Quantity:  input.Quantity,
			UnitPrice: input.UnitPrice,
			Discount:  input.Discount,
			LineTotal: lineTotal,
		})
	}

	total := subtotal - cmd.DiscountAmount
	if total < 0 {
		total = 0
	}

	saleID := uuid.New()

	// Fill in SaleID on each item now that saleID is known.
	for i := range items {
		items[i].SaleID = saleID
	}

	sale := &domain.Sale{
		ID:             saleID,
		TenantID:       tenantID,
		StoreID:        storeID,
		CashierID:      cashierID,
		CustomerID:     customerID,
		Items:          items,
		Subtotal:       subtotal,
		DiscountAmount: cmd.DiscountAmount,
		Total:          total,
		Status:         "COMPLETED",
		Notes:          cmd.Notes,
		CompletedAt:    time.Now().UTC(),
	}

	// ── 3. Deduct stock (compensating saga) ──────────────────────────────────
	type compensated struct {
		productID string
		quantity  int64
	}
	var compensations []compensated

	compensate := func() {
		for _, c := range compensations {
			if err := h.inventory.RestoreStock(ctx, cmd.TenantID, cmd.StoreID, c.productID, c.quantity, saleID.String()); err != nil {
				h.log.Warn("stock restore failed during compensation",
					zap.String("product_id", c.productID),
					zap.Int64("quantity", c.quantity),
					zap.Error(err),
				)
			}
		}
	}

	for _, item := range cmd.Items {
		if err := h.inventory.DeductStock(ctx, cmd.TenantID, cmd.StoreID, item.ProductID, int64(item.Quantity), ""); err != nil {
			compensate()
			return nil, fmt.Errorf("deduct stock for product %s: %w", item.ProductID, err)
		}
		compensations = append(compensations, compensated{productID: item.ProductID, quantity: int64(item.Quantity)})
	}

	// ── 4. Persist sale atomically ────────────────────────────────────────────
	snapshot, err := buildSnapshot(sale, items)
	if err != nil {
		compensate()
		return nil, fmt.Errorf("build receipt snapshot: %w", err)
	}

	receipt := &domain.Receipt{
		ID:       uuid.New(),
		TenantID: tenantID,
		StoreID:  storeID,
		SaleID:   saleID,
		Snapshot: snapshot,
		IssuedAt: time.Now().UTC(),
	}

	if err := h.saleRepo.Create(ctx, sale, items, receipt); err != nil {
		compensate()
		return nil, fmt.Errorf("persist sale: %w", err)
	}

	// ── 5. Publish event (best-effort) ────────────────────────────────────────
	if err := h.publisher.PublishSaleCompleted(ctx, sale); err != nil {
		h.log.Warn("sale event publish failed (best-effort)",
			zap.String("sale_id", saleID.String()),
			zap.Error(err),
		)
	}

	return &CreateSaleResult{Sale: sale, Receipt: receipt}, nil
}

// validateCreateSaleCommand checks required fields and positive quantities.
func validateCreateSaleCommand(cmd CreateSaleCommand) error {
	if cmd.TenantID == "" {
		return fmt.Errorf("%w: tenant_id is required", sherrors.ErrInvalidArgument)
	}
	if cmd.StoreID == "" {
		return fmt.Errorf("%w: store_id is required", sherrors.ErrInvalidArgument)
	}
	if cmd.CashierID == "" {
		return fmt.Errorf("%w: cashier_id is required", sherrors.ErrInvalidArgument)
	}
	if len(cmd.Items) == 0 {
		return fmt.Errorf("%w: at least one item is required", sherrors.ErrInvalidArgument)
	}
	for i, item := range cmd.Items {
		if item.ProductID == "" {
			return fmt.Errorf("%w: item[%d] product_id is required", sherrors.ErrInvalidArgument, i)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("%w: item[%d] quantity must be positive", sherrors.ErrInvalidArgument, i)
		}
		if item.UnitPrice < 0 {
			return fmt.Errorf("%w: item[%d] unit_price must be non-negative", sherrors.ErrInvalidArgument, i)
		}
	}
	if cmd.DiscountAmount < 0 {
		return fmt.Errorf("%w: discount_amount must be non-negative", sherrors.ErrInvalidArgument)
	}
	return nil
}

// buildSnapshot serialises the sale and items into the receipt JSON snapshot.
func buildSnapshot(sale *domain.Sale, items []domain.SaleItem) ([]byte, error) {
	type itemSnap struct {
		ProductID string  `json:"product_id"`
		Quantity  int     `json:"quantity"`
		UnitPrice float64 `json:"unit_price"`
		Discount  float64 `json:"discount"`
		LineTotal float64 `json:"line_total"`
	}
	type snap struct {
		SaleID         string     `json:"sale_id"`
		TenantID       string     `json:"tenant_id"`
		StoreID        string     `json:"store_id"`
		CashierID      string     `json:"cashier_id"`
		Items          []itemSnap `json:"items"`
		Subtotal       float64    `json:"subtotal"`
		DiscountAmount float64    `json:"discount_amount"`
		Total          float64    `json:"total"`
		Status         string     `json:"status"`
		CompletedAt    time.Time  `json:"completed_at"`
	}

	snaps := make([]itemSnap, len(items))
	for i, it := range items {
		snaps[i] = itemSnap{
			ProductID: it.ProductID.String(),
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice,
			Discount:  it.Discount,
			LineTotal: it.LineTotal,
		}
	}

	return json.Marshal(snap{
		SaleID:         sale.ID.String(),
		TenantID:       sale.TenantID.String(),
		StoreID:        sale.StoreID.String(),
		CashierID:      sale.CashierID.String(),
		Items:          snaps,
		Subtotal:       sale.Subtotal,
		DiscountAmount: sale.DiscountAmount,
		Total:          sale.Total,
		Status:         sale.Status,
		CompletedAt:    sale.CompletedAt,
	})
}
