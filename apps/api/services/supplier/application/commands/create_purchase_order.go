package commands

import (
	"context"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/supplier/domain"
	"github.com/google/uuid"
)

// PurchaseOrderLineItem is an individual product line within a purchase order.
type PurchaseOrderLineItem struct {
	ProductID       uuid.UUID
	QuantityOrdered int32
	UnitCost        float64
}

// CreatePurchaseOrderCommand carries the data needed to create a new purchase order.
type CreatePurchaseOrderCommand struct {
	TenantID   uuid.UUID
	StoreID    uuid.UUID
	SupplierID uuid.UUID
	Items      []PurchaseOrderLineItem
	Notes      string
}

// CreatePurchaseOrderHandler processes CreatePurchaseOrderCommand.
type CreatePurchaseOrderHandler struct {
	orderRepo    domain.PurchaseOrderRepository
	supplierRepo domain.SupplierRepository
}

// NewCreatePurchaseOrderHandler creates a CreatePurchaseOrderHandler.
func NewCreatePurchaseOrderHandler(
	orderRepo domain.PurchaseOrderRepository,
	supplierRepo domain.SupplierRepository,
) *CreatePurchaseOrderHandler {
	return &CreatePurchaseOrderHandler{
		orderRepo:    orderRepo,
		supplierRepo: supplierRepo,
	}
}

// Handle executes the CreatePurchaseOrder use case.
func (h *CreatePurchaseOrderHandler) Handle(ctx context.Context, cmd CreatePurchaseOrderCommand) (*domain.PurchaseOrder, error) {
	if cmd.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenantID required: %w", domain.ErrInvalidArgument)
	}
	if cmd.StoreID == uuid.Nil {
		return nil, fmt.Errorf("storeID required: %w", domain.ErrInvalidArgument)
	}
	if cmd.SupplierID == uuid.Nil {
		return nil, fmt.Errorf("supplierID required: %w", domain.ErrInvalidArgument)
	}
	if len(cmd.Items) == 0 {
		return nil, fmt.Errorf("purchase order must have at least one item: %w", domain.ErrInvalidArgument)
	}

	// Verify the supplier belongs to this tenant.
	if _, err := h.supplierRepo.Get(ctx, cmd.TenantID, cmd.SupplierID); err != nil {
		return nil, fmt.Errorf("validate supplier: %w", err)
	}

	orderID := uuid.New()
	items := make([]*domain.PurchaseOrderItem, 0, len(cmd.Items))
	for _, line := range cmd.Items {
		if line.ProductID == uuid.Nil {
			return nil, fmt.Errorf("item product_id required: %w", domain.ErrInvalidArgument)
		}
		if line.QuantityOrdered <= 0 {
			return nil, fmt.Errorf("item quantity_ordered must be > 0: %w", domain.ErrInvalidArgument)
		}
		if !(line.UnitCost >= 0 && line.UnitCost < 1e10) {
			return nil, fmt.Errorf("item unit_cost must be between 0 and 1e10: %w", domain.ErrInvalidArgument)
		}
		items = append(items, &domain.PurchaseOrderItem{
			TenantID:        cmd.TenantID,
			PurchaseOrderID: orderID,
			ProductID:       line.ProductID,
			QuantityOrdered: line.QuantityOrdered,
			UnitCost:        line.UnitCost,
		})
	}

	order := &domain.PurchaseOrder{
		ID:         orderID,
		TenantID:   cmd.TenantID,
		StoreID:    cmd.StoreID,
		SupplierID: cmd.SupplierID,
		Status:     domain.StatusPending,
		Notes:      cmd.Notes,
		Items:      items,
	}

	if err := h.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("create purchase order: %w", err)
	}
	return order, nil
}
