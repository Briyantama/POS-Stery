package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	inventoryv1 "github.com/pos-stery/pos-stery/gen/go/pos/inventory/v1"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/application/queries"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InventoryServiceServer implements inventoryv1.InventoryServiceServer.
type InventoryServiceServer struct {
	inventoryv1.UnimplementedInventoryServiceServer
	addItem      *commands.AddItemHandler
	updateStock  *commands.UpdateStockHandler
	setThreshold *commands.SetThresholdHandler
	getItem      *queries.GetItemHandler
	listItems    *queries.ListItemsHandler
}

// NewInventoryServiceServer creates an InventoryServiceServer.
func NewInventoryServiceServer(
	addItem *commands.AddItemHandler,
	updateStock *commands.UpdateStockHandler,
	setThreshold *commands.SetThresholdHandler,
	getItem *queries.GetItemHandler,
	listItems *queries.ListItemsHandler,
) *InventoryServiceServer {
	return &InventoryServiceServer{
		addItem:      addItem,
		updateStock:  updateStock,
		setThreshold: setThreshold,
		getItem:      getItem,
		listItems:    listItems,
	}
}

// AddItem handles the AddItem RPC.
func (s *InventoryServiceServer) AddItem(ctx context.Context, req *inventoryv1.AddItemRequest) (*inventoryv1.AddItemResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	storeID, err := parseUUID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	productID, err := parseUUID(req.GetProductId(), "product_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	item, err := s.addItem.Handle(ctx, commands.AddItemCommand{
		TenantID:        tenantID,
		StoreID:         storeID,
		ProductID:       productID,
		InitialQuantity: req.GetQuantity(),
		MinQuantity:     req.GetMinQuantity(),
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &inventoryv1.AddItemResponse{Item: domainItemToProto(item, 0, false)}, nil
}

// UpdateStock handles the UpdateStock RPC.
func (s *InventoryServiceServer) UpdateStock(ctx context.Context, req *inventoryv1.UpdateStockRequest) (*inventoryv1.UpdateStockResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	storeID, err := parseUUID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	productID, err := parseUUID(req.GetProductId(), "product_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	result, err := s.updateStock.Handle(ctx, commands.UpdateStockCommand{
		TenantID:  tenantID,
		StoreID:   storeID,
		ProductID: productID,
		Delta:     req.GetQuantityDelta(),
		Reason:    protoReasonToString(req.GetReason()),
		RefID:     req.GetReferenceId(),
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &inventoryv1.UpdateStockResponse{
		Item:          domainItemToProto(result.Item, 0, result.LowStockAlert),
		LowStockAlert: result.LowStockAlert,
	}, nil
}

// ListItems handles the ListItems RPC.
func (s *InventoryServiceServer) ListItems(ctx context.Context, req *inventoryv1.ListItemsRequest) (*inventoryv1.ListItemsResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	storeID, err := parseUUID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	result, err := s.listItems.Handle(ctx, queries.ListItemsQuery{
		TenantID:     tenantID,
		StoreID:      storeID,
		LowStockOnly: req.GetLowStockOnly(),
		Limit:        int(req.GetLimit()),
		Offset:       int(req.GetOffset()),
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	protoItems := make([]*inventoryv1.StockItem, 0, len(result.Items))
	for _, item := range result.Items {
		protoItems = append(protoItems, domainItemToProto(item, 0, false))
	}

	return &inventoryv1.ListItemsResponse{
		Items: protoItems,
		Total: int32(result.Total),
	}, nil
}

// GetItem handles the GetItem RPC.
func (s *InventoryServiceServer) GetItem(ctx context.Context, req *inventoryv1.GetItemRequest) (*inventoryv1.GetItemResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	storeID, err := parseUUID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	productID, err := parseUUID(req.GetProductId(), "product_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	item, err := s.getItem.Handle(ctx, queries.GetItemQuery{
		TenantID:  tenantID,
		StoreID:   storeID,
		ProductID: productID,
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &inventoryv1.GetItemResponse{Item: domainItemToProto(item, 0, false)}, nil
}

// SetThreshold handles the SetThreshold RPC.
func (s *InventoryServiceServer) SetThreshold(ctx context.Context, req *inventoryv1.SetThresholdRequest) (*inventoryv1.SetThresholdResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	storeID, err := parseUUID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	productID, err := parseUUID(req.GetProductId(), "product_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	if err := s.setThreshold.Handle(ctx, commands.SetThresholdCommand{
		TenantID:    tenantID,
		StoreID:     storeID,
		ProductID:   productID,
		MinQuantity: req.GetMinQuantity(),
	}); err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &inventoryv1.SetThresholdResponse{}, nil
}

// domainItemToProto converts a domain StockItem to the proto representation.
func domainItemToProto(item *domain.StockItem, minQuantity int64, isLowStock bool) *inventoryv1.StockItem {
	if item == nil {
		return nil
	}
	return &inventoryv1.StockItem{
		ItemId:      item.ID.String(),
		TenantId:    item.TenantID.String(),
		StoreId:     item.StoreID.String(),
		ProductId:   item.ProductID.String(),
		Quantity:    item.Quantity,
		MinQuantity: minQuantity,
		IsLowStock:  isLowStock,
		UpdatedAt:   timestamppb.New(item.UpdatedAt),
	}
}

// parseUUID parses a UUID string, wrapping errors as ErrInvalidArgument.
func parseUUID(s, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s is not a valid UUID: %w", field, sherrors.ErrInvalidArgument)
	}
	return id, nil
}

// protoReasonToString converts the proto StockUpdateReason enum to its domain string.
func protoReasonToString(r inventoryv1.StockUpdateReason) string {
	switch r {
	case inventoryv1.StockUpdateReason_STOCK_UPDATE_REASON_SALE:
		return string(domain.ReasonSale)
	case inventoryv1.StockUpdateReason_STOCK_UPDATE_REASON_RECEIPT:
		return string(domain.ReasonReceipt)
	case inventoryv1.StockUpdateReason_STOCK_UPDATE_REASON_ADJUSTMENT:
		return string(domain.ReasonAdjustment)
	default:
		return string(domain.ReasonAdjustment)
	}
}
