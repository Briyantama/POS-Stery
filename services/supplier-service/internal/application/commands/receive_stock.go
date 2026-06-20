package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/supplier-service/internal/application"
	"github.com/pos-stery/pos-stery/services/supplier-service/internal/domain"
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
	inventory application.InventoryPort
	publisher application.EventPublisher
}

// NewReceiveStockHandler creates a ReceiveStockHandler.
func NewReceiveStockHandler(
	orderRepo domain.PurchaseOrderRepository,
	inventory application.InventoryPort,
	publisher application.EventPublisher,
) *ReceiveStockHandler {
	return &ReceiveStockHandler{
		orderRepo: orderRepo,
		inventory: inventory,
		publisher: publisher,
	}
}

// Handle executes the ReceiveStock use case.
//
// For each received line item, the inventory-service is called via gRPC to add
// the received quantity with reason RECEIPT. An Inventory.Replenished event is
// published to NATS for each item. The purchase order status is set to RECEIVED.
func (h *ReceiveStockHandler) Handle(ctx context.Context, cmd ReceiveStockCommand) (*domain.PurchaseOrder, error) {
	if cmd.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenantID required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.StoreID == uuid.Nil {
		return nil, fmt.Errorf("storeID required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.PurchaseOrderID == uuid.Nil {
		return nil, fmt.Errorf("purchaseOrderID required: %w", sherrors.ErrInvalidArgument)
	}
	if len(cmd.ItemsReceived) == 0 {
		return nil, fmt.Errorf("at least one received item required: %w", sherrors.ErrInvalidArgument)
	}

	// Load the purchase order to validate ownership and current state.
	order, err := h.orderRepo.Get(ctx, cmd.TenantID, cmd.StoreID, cmd.PurchaseOrderID)
	if err != nil {
		return nil, fmt.Errorf("get purchase order: %w", err)
	}

	if order.Status == domain.StatusCancelled {
		return nil, fmt.Errorf("cannot receive against a cancelled order: %w", sherrors.ErrInvalidArgument)
	}

	// Build a map of received quantities keyed by product ID for easy lookup.
	receivedMap := make(map[uuid.UUID]int32, len(cmd.ItemsReceived))
	for _, line := range cmd.ItemsReceived {
		if line.ProductID == uuid.Nil {
			return nil, fmt.Errorf("received item product_id required: %w", sherrors.ErrInvalidArgument)
		}
		if line.QuantityReceived <= 0 {
			return nil, fmt.Errorf("received quantity must be > 0: %w", sherrors.ErrInvalidArgument)
		}
		receivedMap[line.ProductID] = line.QuantityReceived
	}

	// For each matched line item: update inventory via gRPC, then emit an event.
	for _, item := range order.Items {
		qty, ok := receivedMap[item.ProductID]
		if !ok || qty == 0 {
			continue
		}

		item.QuantityReceived += qty

		// Notify inventory-service of the stock addition.
		if err := h.inventory.UpdateStock(
			ctx,
			cmd.TenantID.String(),
			cmd.StoreID.String(),
			item.ProductID.String(),
			int64(qty),
			"RECEIPT",
			order.ID.String(),
		); err != nil {
			return nil, fmt.Errorf("update inventory for product %s: %w", item.ProductID, err)
		}

		// Publish Inventory.Replenished event; publishing failure is non-fatal.
		if pubErr := h.publisher.PublishReplenished(
			ctx,
			cmd.TenantID.String(),
			cmd.StoreID.String(),
			order.ID.String(),
			item.ProductID.String(),
			int64(qty),
		); pubErr != nil {
			_ = pubErr // log-worthy but must not abort the operation
		}
	}

	// Mark order as RECEIVED.
	order.Status = domain.StatusReceived

	if err := h.orderRepo.UpdateReceived(ctx, order); err != nil {
		return nil, fmt.Errorf("persist received order: %w", err)
	}

	return order, nil
}
