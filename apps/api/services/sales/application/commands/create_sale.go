package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/Briyantama/POS-Stery/services/sales/application"
	"github.com/Briyantama/POS-Stery/services/sales/domain"
	"github.com/google/uuid"
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
	prices    application.PriceReader
	publisher application.EventPublisher
}

// NewCreateSaleHandler wires the handler with its dependencies.
func NewCreateSaleHandler(
	saleRepo domain.SaleRepository,
	inventory application.InventoryUpdater,
	prices application.PriceReader,
	publisher application.EventPublisher,
) *CreateSaleHandler {
	return &CreateSaleHandler{
		saleRepo:  saleRepo,
		inventory: inventory,
		prices:    prices,
		publisher: publisher,
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
		return nil, fmt.Errorf("%w: invalid tenant_id", domain.ErrInvalidArgument)
	}
	storeID, err := uuid.Parse(cmd.StoreID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid store_id", domain.ErrInvalidArgument)
	}
	cashierID, err := uuid.Parse(cmd.CashierID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cashier_id", domain.ErrInvalidArgument)
	}

	var customerID *uuid.UUID
	if cmd.CustomerID != "" {
		id, err := uuid.Parse(cmd.CustomerID)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid customer_id", domain.ErrInvalidArgument)
		}
		customerID = &id
	}

	// ── 2. Calculate amounts ─────────────────────────────────────────────────
	var subtotal float64
	items := make([]domain.SaleItem, 0, len(cmd.Items))

	for _, input := range cmd.Items {
		productID, err := uuid.Parse(input.ProductID)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid product_id %q", domain.ErrInvalidArgument, input.ProductID)
		}
		unitPrice, err := h.prices.UnitPrice(ctx, tenantID, productID)
		if err != nil {
			return nil, err
		}
		gross := float64(input.Quantity) * unitPrice
		if input.Discount > gross {
			return nil, fmt.Errorf("%w: item discount exceeds line total for product %s", domain.ErrInvalidArgument, productID)
		}
		lineTotal := gross - input.Discount
		subtotal += lineTotal

		items = append(items, domain.SaleItem{
			ID:        uuid.New(),
			TenantID:  tenantID,
			ProductID: productID,
			Quantity:  input.Quantity,
			UnitPrice: unitPrice,
			Discount:  input.Discount,
			LineTotal: lineTotal,
		})
	}

	if cmd.DiscountAmount > subtotal {
		return nil, fmt.Errorf("%w: discount_amount exceeds subtotal", domain.ErrInvalidArgument)
	}
	total := subtotal - cmd.DiscountAmount

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
		// Detached: a cancelled request must not strand the deducted stock.
		ctx := context.WithoutCancel(ctx)
		for _, c := range compensations {
			if err := h.inventory.RestoreStock(ctx, cmd.TenantID, cmd.StoreID, c.productID, c.quantity, saleID.String()); err != nil {
				slog.WarnContext(ctx, "stock restore failed during compensation",
					"tenant_id", cmd.TenantID, "product_id", c.productID, "quantity", c.quantity, "error", err,
				)
			}
		}
	}

	for _, item := range cmd.Items {
		if err := h.inventory.DeductStock(ctx, cmd.TenantID, cmd.StoreID, item.ProductID, int64(item.Quantity), saleID.String()); err != nil {
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
		slog.WarnContext(ctx, "sale event publish failed (best-effort)",
			"tenant_id", cmd.TenantID, "sale_id", saleID.String(), "error", err,
		)
	}

	return &CreateSaleResult{Sale: sale, Receipt: receipt}, nil
}

// validateCreateSaleCommand checks required fields and positive quantities.
func validateCreateSaleCommand(cmd CreateSaleCommand) error {
	if cmd.TenantID == "" {
		return fmt.Errorf("%w: tenant_id is required", domain.ErrInvalidArgument)
	}
	if cmd.StoreID == "" {
		return fmt.Errorf("%w: store_id is required", domain.ErrInvalidArgument)
	}
	if cmd.CashierID == "" {
		return fmt.Errorf("%w: cashier_id is required", domain.ErrInvalidArgument)
	}
	if len(cmd.Items) == 0 {
		return fmt.Errorf("%w: at least one item is required", domain.ErrInvalidArgument)
	}
	if len(cmd.Notes) > 500 {
		return fmt.Errorf("%w: notes must be at most 500 characters", domain.ErrInvalidArgument)
	}
	for i, item := range cmd.Items {
		if item.ProductID == "" {
			return fmt.Errorf("%w: item[%d] product_id is required", domain.ErrInvalidArgument, i)
		}
		if item.Quantity <= 0 {
			return fmt.Errorf("%w: item[%d] quantity must be positive", domain.ErrInvalidArgument, i)
		}
		if item.Discount < 0 {
			return fmt.Errorf("%w: item[%d] discount must be non-negative", domain.ErrInvalidArgument, i)
		}
	}
	if cmd.DiscountAmount < 0 {
		return fmt.Errorf("%w: discount_amount must be non-negative", domain.ErrInvalidArgument)
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
