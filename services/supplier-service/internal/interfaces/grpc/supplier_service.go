package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	supplierv1 "github.com/pos-stery/pos-stery/gen/go/pos/supplier/v1"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/supplier-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/supplier-service/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SupplierServiceServer implements supplierv1.SupplierServiceServer.
type SupplierServiceServer struct {
	supplierv1.UnimplementedSupplierServiceServer
	addSupplier         *commands.AddSupplierHandler
	createPurchaseOrder *commands.CreatePurchaseOrderHandler
	receiveStock        *commands.ReceiveStockHandler
	supplierRepo        domain.SupplierRepository
	orderRepo           domain.PurchaseOrderRepository
}

// NewSupplierServiceServer creates a SupplierServiceServer.
func NewSupplierServiceServer(
	addSupplier *commands.AddSupplierHandler,
	createPurchaseOrder *commands.CreatePurchaseOrderHandler,
	receiveStock *commands.ReceiveStockHandler,
	supplierRepo domain.SupplierRepository,
	orderRepo domain.PurchaseOrderRepository,
) *SupplierServiceServer {
	return &SupplierServiceServer{
		addSupplier:         addSupplier,
		createPurchaseOrder: createPurchaseOrder,
		receiveStock:        receiveStock,
		supplierRepo:        supplierRepo,
		orderRepo:           orderRepo,
	}
}

// AddSupplier handles the AddSupplier RPC.
func (s *SupplierServiceServer) AddSupplier(ctx context.Context, req *supplierv1.AddSupplierRequest) (*supplierv1.AddSupplierResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	supplier, err := s.addSupplier.Handle(ctx, commands.AddSupplierCommand{
		TenantID: tenantID,
		Name:     req.GetName(),
		Contact:  req.GetContact(),
		Phone:    req.GetPhone(),
		Email:    req.GetEmail(),
		Address:  req.GetAddress(),
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &supplierv1.AddSupplierResponse{Supplier: domainSupplierToProto(supplier)}, nil
}

// ListSuppliers handles the ListSuppliers RPC.
func (s *SupplierServiceServer) ListSuppliers(ctx context.Context, req *supplierv1.ListSuppliersRequest) (*supplierv1.ListSuppliersResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 50
	}

	suppliers, total, err := s.supplierRepo.List(ctx, tenantID, limit, int(req.GetOffset()))
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	protoSuppliers := make([]*supplierv1.Supplier, 0, len(suppliers))
	for _, sup := range suppliers {
		protoSuppliers = append(protoSuppliers, domainSupplierToProto(sup))
	}

	return &supplierv1.ListSuppliersResponse{
		Suppliers: protoSuppliers,
		Total:     int32(total),
	}, nil
}

// CreatePurchaseOrder handles the CreatePurchaseOrder RPC.
func (s *SupplierServiceServer) CreatePurchaseOrder(ctx context.Context, req *supplierv1.CreatePurchaseOrderRequest) (*supplierv1.CreatePurchaseOrderResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	storeID, err := parseUUID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	supplierID, err := parseUUID(req.GetSupplierId(), "supplier_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	lines := make([]commands.PurchaseOrderLineItem, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		productID, err := parseUUID(item.GetProductId(), "product_id")
		if err != nil {
			return nil, sherrors.ToGRPCStatus(err)
		}
		lines = append(lines, commands.PurchaseOrderLineItem{
			ProductID:       productID,
			QuantityOrdered: item.GetQuantityOrdered(),
			UnitCost:        item.GetUnitCost(),
		})
	}

	order, err := s.createPurchaseOrder.Handle(ctx, commands.CreatePurchaseOrderCommand{
		TenantID:   tenantID,
		StoreID:    storeID,
		SupplierID: supplierID,
		Items:      lines,
		Notes:      req.GetNotes(),
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &supplierv1.CreatePurchaseOrderResponse{Order: domainOrderToProto(order)}, nil
}

// ListPurchaseOrders handles the ListPurchaseOrders RPC.
func (s *SupplierServiceServer) ListPurchaseOrders(ctx context.Context, req *supplierv1.ListPurchaseOrdersRequest) (*supplierv1.ListPurchaseOrdersResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	storeID, err := parseUUID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 50
	}

	orders, total, err := s.orderRepo.List(ctx, tenantID, storeID, req.GetStatus(), limit, int(req.GetOffset()))
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	protoOrders := make([]*supplierv1.PurchaseOrder, 0, len(orders))
	for _, order := range orders {
		protoOrders = append(protoOrders, domainOrderToProto(order))
	}

	return &supplierv1.ListPurchaseOrdersResponse{
		Orders: protoOrders,
		Total:  int32(total),
	}, nil
}

// ReceiveStock handles the ReceiveStock RPC.
func (s *SupplierServiceServer) ReceiveStock(ctx context.Context, req *supplierv1.ReceiveStockRequest) (*supplierv1.ReceiveStockResponse, error) {
	tenantID, err := parseUUID(req.GetTenantId(), "tenant_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	storeID, err := parseUUID(req.GetStoreId(), "store_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	orderID, err := parseUUID(req.GetPurchaseOrderId(), "purchase_order_id")
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	received := make([]commands.ReceivedLineItem, 0, len(req.GetItemsReceived()))
	for _, item := range req.GetItemsReceived() {
		productID, err := parseUUID(item.GetProductId(), "product_id")
		if err != nil {
			return nil, sherrors.ToGRPCStatus(err)
		}
		received = append(received, commands.ReceivedLineItem{
			ProductID:        productID,
			QuantityReceived: item.GetQuantityReceived(),
		})
	}

	order, err := s.receiveStock.Handle(ctx, commands.ReceiveStockCommand{
		TenantID:        tenantID,
		StoreID:         storeID,
		PurchaseOrderID: orderID,
		ItemsReceived:   received,
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &supplierv1.ReceiveStockResponse{Order: domainOrderToProto(order)}, nil
}

// domainSupplierToProto maps a domain Supplier to the proto wire type.
func domainSupplierToProto(s *domain.Supplier) *supplierv1.Supplier {
	if s == nil {
		return nil
	}
	return &supplierv1.Supplier{
		SupplierId: s.ID.String(),
		TenantId:   s.TenantID.String(),
		Name:       s.Name,
		Contact:    s.Contact,
		Phone:      s.Phone,
		Email:      s.Email,
		Address:    s.Address,
		CreatedAt:  timestamppb.New(s.CreatedAt),
	}
}

// domainOrderToProto maps a domain PurchaseOrder to the proto wire type.
func domainOrderToProto(o *domain.PurchaseOrder) *supplierv1.PurchaseOrder {
	if o == nil {
		return nil
	}

	items := make([]*supplierv1.PurchaseOrderItem, 0, len(o.Items))
	for _, item := range o.Items {
		items = append(items, &supplierv1.PurchaseOrderItem{
			ProductId:       item.ProductID.String(),
			QuantityOrdered: item.QuantityOrdered,
			UnitCost:        item.UnitCost,
		})
	}

	return &supplierv1.PurchaseOrder{
		OrderId:    o.ID.String(),
		TenantId:   o.TenantID.String(),
		StoreId:    o.StoreID.String(),
		SupplierId: o.SupplierID.String(),
		Items:      items,
		Status:     string(o.Status),
		Notes:      o.Notes,
		CreatedAt:  timestamppb.New(o.CreatedAt),
		UpdatedAt:  timestamppb.New(o.UpdatedAt),
	}
}

// parseUUID parses a UUID string and wraps parse errors as ErrInvalidArgument.
func parseUUID(s, field string) (uuid.UUID, error) {
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%s is not a valid UUID: %w", field, sherrors.ErrInvalidArgument)
	}
	return id, nil
}
