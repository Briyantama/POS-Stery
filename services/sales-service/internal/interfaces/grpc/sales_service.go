package grpc

import (
	"context"

	"github.com/google/uuid"
	salesv1 "github.com/pos-stery/pos-stery/gen/go/pos/sales/v1"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/sales-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/sales-service/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// SalesServiceServer implements salesv1.SalesServiceServer.
type SalesServiceServer struct {
	salesv1.UnimplementedSalesServiceServer
	createSale *commands.CreateSaleHandler
	saleRepo   saleReader
}

// saleReader is the narrow read interface needed by the gRPC handler.
type saleReader interface {
	FindByID(ctx context.Context, tenantID, storeID, saleID uuid.UUID) (*domain.Sale, []domain.SaleItem, *domain.Receipt, error)
	GetReport(ctx context.Context, tenantID, storeID *uuid.UUID, date string) (*domain.SalesReport, error)
}

// NewSalesServiceServer wires the gRPC server with its command handlers and repo.
func NewSalesServiceServer(
	createSale *commands.CreateSaleHandler,
	saleRepo saleReader,
) *SalesServiceServer {
	return &SalesServiceServer{
		createSale: createSale,
		saleRepo:   saleRepo,
	}
}

// CreateSale handles the CreateSale RPC.
func (s *SalesServiceServer) CreateSale(ctx context.Context, req *salesv1.CreateSaleRequest) (*salesv1.CreateSaleResponse, error) {
	items := make([]commands.SaleItemInput, 0, len(req.Items))
	for _, it := range req.Items {
		items = append(items, commands.SaleItemInput{
			ProductID: it.ProductId,
			Quantity:  int(it.Quantity),
			UnitPrice: it.UnitPrice,
			Discount:  it.Discount,
		})
	}

	result, err := s.createSale.Handle(ctx, commands.CreateSaleCommand{
		TenantID:       req.TenantId,
		StoreID:        req.StoreId,
		CashierID:      req.CashierId,
		CustomerID:     req.CustomerId,
		Items:          items,
		DiscountAmount: req.DiscountAmount,
		Notes:          req.Notes,
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &salesv1.CreateSaleResponse{
		Sale:    domainSaleToProto(result.Sale),
		Receipt: domainReceiptToProto(result.Receipt),
	}, nil
}

// GetSale handles the GetSale RPC.
func (s *SalesServiceServer) GetSale(ctx context.Context, req *salesv1.GetSaleRequest) (*salesv1.GetSaleResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(sherrors.ErrInvalidArgument)
	}
	storeID, err := uuid.Parse(req.StoreId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(sherrors.ErrInvalidArgument)
	}
	saleID, err := uuid.Parse(req.SaleId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(sherrors.ErrInvalidArgument)
	}

	sale, items, receipt, err := s.saleRepo.FindByID(ctx, tenantID, storeID, saleID)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	sale.Items = items

	return &salesv1.GetSaleResponse{
		Sale:    domainSaleToProto(sale),
		Receipt: domainReceiptToProto(receipt),
	}, nil
}

// GetSalesReport handles the GetSalesReport RPC.
func (s *SalesServiceServer) GetSalesReport(ctx context.Context, req *salesv1.GetSalesReportRequest) (*salesv1.GetSalesReportResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(sherrors.ErrInvalidArgument)
	}

	var storeIDPtr *uuid.UUID
	if req.StoreId != "" {
		id, err := uuid.Parse(req.StoreId)
		if err != nil {
			return nil, sherrors.ToGRPCStatus(sherrors.ErrInvalidArgument)
		}
		storeIDPtr = &id
	}

	report, err := s.saleRepo.GetReport(ctx, &tenantID, storeIDPtr, req.Date)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	topProducts := make([]*salesv1.TopProduct, 0, len(report.TopProducts))
	for _, tp := range report.TopProducts {
		topProducts = append(topProducts, &salesv1.TopProduct{
			ProductId:   tp.ProductID,
			ProductName: tp.ProductName,
			UnitsSold:   int32(tp.UnitsSold),
			Revenue:     tp.Revenue,
		})
	}

	return &salesv1.GetSalesReportResponse{
		Date:             report.Date,
		StoreId:          report.StoreID,
		TotalRevenue:     report.TotalRevenue,
		TransactionCount: int32(report.TransactionCount),
		TopProducts:      topProducts,
	}, nil
}

// ── mapping helpers ────────────────────────────────────────────────────────────

func domainSaleToProto(s *domain.Sale) *salesv1.Sale {
	if s == nil {
		return nil
	}

	items := make([]*salesv1.SaleItem, 0, len(s.Items))
	for _, it := range s.Items {
		items = append(items, &salesv1.SaleItem{
			ProductId: it.ProductID.String(),
			Quantity:  int32(it.Quantity),
			UnitPrice: it.UnitPrice,
			Discount:  it.Discount,
		})
	}

	customerID := ""
	if s.CustomerID != nil {
		customerID = s.CustomerID.String()
	}

	return &salesv1.Sale{
		SaleId:         s.ID.String(),
		TenantId:       s.TenantID.String(),
		StoreId:        s.StoreID.String(),
		CashierId:      s.CashierID.String(),
		CustomerId:     customerID,
		Items:          items,
		Subtotal:       s.Subtotal,
		DiscountAmount: s.DiscountAmount,
		Total:          s.Total,
		Status:         s.Status,
		CompletedAt:    timestamppb.New(s.CompletedAt),
	}
}

func domainReceiptToProto(r *domain.Receipt) *salesv1.Receipt {
	if r == nil {
		return nil
	}
	return &salesv1.Receipt{
		ReceiptId: r.ID.String(),
		SaleId:    r.SaleID.String(),
		TenantId:  r.TenantID.String(),
		StoreId:   r.StoreID.String(),
		Snapshot:  r.Snapshot,
		IssuedAt:  timestamppb.New(r.IssuedAt),
	}
}
