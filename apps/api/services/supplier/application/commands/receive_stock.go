package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Briyantama/POS-Stery/services/supplier/application"
	"github.com/Briyantama/POS-Stery/services/supplier/domain"
	"uuid"
)

// ReceivedLineItem describes how many units of a product were physically received.
type ReceivedLineItem struct {
	ProductID        uuid.UUID
	QuantityReceived int32
}

// ReceiveStockCommand carries the data needed to receive goods against a purchase order.
type ReceiveStockCommand struct {
	TenantID        uuid.UUID
	StoreID         uuid.UUID
	PurchaseOrderID uuid.UUID
	ItemsReceived   []ReceivedLineItem
}

// ReceiveStockHandler processes ReceiveStockCommand.
type ReceiveStockHandler struct {
	orderRepo domain.PurchaseOrderRepository
	publisher application.EventPublisher
}

// NewReceiveStockHandler creates a ReceiveStockHandler.
func NewReceiveStockHandler(
	orderRepo domain.PurchaseOrderRepository,
	publisher application.EventPublisher,
) *ReceiveStockHandler {
	return &ReceiveStockHandler{orderRepo: orderRepo, publisher: publisher}
}

// Handle executes the ReceiveStock use case.
//
// The repository books the whole receipt (order items, store stock, order status RECEIVED) in one
// transaction and refuses an order that is not PENDING, so a receipt cannot be applied twice.
// An Inventory.Replenished event is published after commit for each applied line.
func (h *ReceiveStockHandler) Handle(ctx context.Context, cmd ReceiveStockCommand) (*domain.PurchaseOrder, error) {
	if cmd.TenantID == uuid.Nil() {
		return nil, fmt.Errorf("tenantID required: %w", domain.ErrInvalidArgument)
	}
	if cmd.StoreID == uuid.Nil() {
		return nil, fmt.Errorf("storeID required: %w", domain.ErrInvalidArgument)
	}
	if cmd.PurchaseOrderID == uuid.Nil() {
		return nil, fmt.Errorf("purchaseOrderID required: %w", domain.ErrInvalidArgument)
	}
	if len(cmd.ItemsReceived) == 0 {
		return nil, fmt.Errorf("at least one received item required: %w", domain.ErrInvalidArgument)
	}

	// Validate all line items before any IO; merge repeated products so each is booked once.
	lines := make([]domain.ReceivedLine, 0, len(cmd.ItemsReceived))
	index := make(map[uuid.UUID]int, len(cmd.ItemsReceived))
	for _, line := range cmd.ItemsReceived {
		if line.ProductID == uuid.Nil() {
			return nil, fmt.Errorf("received item product_id required: %w", domain.ErrInvalidArgument)
		}
		if line.QuantityReceived <= 0 {
			return nil, fmt.Errorf("received quantity must be > 0: %w", domain.ErrInvalidArgument)
		}
		if i, ok := index[line.ProductID]; ok {
			lines[i].Quantity += line.QuantityReceived
			continue
		}
		index[line.ProductID] = len(lines)
		lines = append(lines, domain.ReceivedLine{ProductID: line.ProductID, Quantity: line.QuantityReceived})
	}

	order, applied, err := h.orderRepo.ReceiveOrder(ctx, cmd.TenantID, cmd.StoreID, cmd.PurchaseOrderID, lines)
	if err != nil {
		return nil, fmt.Errorf("receive purchase order: %w", err)
	}

	for _, line := range applied {
		// Publishing failure is non-fatal: the receipt is already committed.
		if err := h.publisher.PublishReplenished(
			ctx,
			cmd.TenantID.String(),
			cmd.StoreID.String(),
			order.ID.String(),
			line.ProductID.String(),
			int64(line.Quantity),
		); err != nil {
			slog.ErrorContext(ctx, "publish replenished event", "purchase_order_id", order.ID, "error", err)
		}
	}

	return order, nil
}
