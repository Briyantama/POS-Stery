package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	productv1 "github.com/pos-stery/pos-stery/gen/go/pos/product/v1"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/product-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/product-service/internal/application/queries"
	"github.com/pos-stery/pos-stery/services/product-service/internal/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProductServiceServer implements productv1.ProductServiceServer.
type ProductServiceServer struct {
	productv1.UnimplementedProductServiceServer
	createHandler *commands.CreateProductHandler
	updateHandler *commands.UpdateProductHandler
	searchHandler *queries.SearchProductsHandler
}

// NewProductServiceServer constructs the gRPC server with all handlers.
func NewProductServiceServer(
	create *commands.CreateProductHandler,
	update *commands.UpdateProductHandler,
	search *queries.SearchProductsHandler,
) *ProductServiceServer {
	return &ProductServiceServer{
		createHandler: create,
		updateHandler: update,
		searchHandler: search,
	}
}

// CreateProduct handles the CreateProduct RPC.
func (s *ProductServiceServer) CreateProduct(
	ctx context.Context,
	req *productv1.CreateProductRequest,
) (*productv1.CreateProductResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid tenant_id: %w", sherrors.ErrInvalidArgument))
	}

	cmd := commands.CreateProductCommand{
		TenantID:    tenantID,
		Name:        req.Name,
		SKU:         req.Sku,
		Barcode:     req.Barcode,
		BasePrice:   req.BasePrice,
		Description: req.Description,
		Unit:        req.Unit,
	}

	if req.CategoryId != "" {
		catID, err := uuid.Parse(req.CategoryId)
		if err != nil {
			return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid category_id: %w", sherrors.ErrInvalidArgument))
		}
		cmd.CategoryID = &catID
	}

	p, err := s.createHandler.Handle(ctx, cmd)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &productv1.CreateProductResponse{Product: domainToProto(p)}, nil
}

// UpdateProduct handles the UpdateProduct RPC.
func (s *ProductServiceServer) UpdateProduct(
	ctx context.Context,
	req *productv1.UpdateProductRequest,
) (*productv1.UpdateProductResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid tenant_id: %w", sherrors.ErrInvalidArgument))
	}
	productID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid product_id: %w", sherrors.ErrInvalidArgument))
	}

	cmd := commands.UpdateProductCommand{
		TenantID:    tenantID,
		ProductID:   productID,
		Name:        req.Name,
		Barcode:     req.Barcode,
		BasePrice:   req.BasePrice,
		Description: req.Description,
		IsActive:    req.IsActive,
	}

	if req.SalePrice != 0 {
		sp := req.SalePrice
		cmd.SalePrice = &sp
	}

	if req.CategoryId != "" {
		catID, err := uuid.Parse(req.CategoryId)
		if err != nil {
			return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid category_id: %w", sherrors.ErrInvalidArgument))
		}
		cmd.CategoryID = &catID
	}

	p, err := s.updateHandler.Handle(ctx, cmd)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &productv1.UpdateProductResponse{Product: domainToProto(p)}, nil
}

// GetProduct handles the GetProduct RPC by delegating to a search with an exact ID match.
// We re-use the repository directly through a thin query path.
func (s *ProductServiceServer) GetProduct(
	ctx context.Context,
	req *productv1.GetProductRequest,
) (*productv1.GetProductResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid tenant_id: %w", sherrors.ErrInvalidArgument))
	}
	productID, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid product_id: %w", sherrors.ErrInvalidArgument))
	}

	// Re-use update handler's embedded FindByID via a no-op update path is wrong.
	// The server carries a dedicated getByID func via the search handler's repo pointer.
	// Cleanest: expose a GetByID query or delegate via the search handler with an exact barcode.
	// We use the search handler's repo reference via a dedicated query.
	p, err := s.searchHandler.GetByID(ctx, tenantID, productID)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &productv1.GetProductResponse{Product: domainToProto(p)}, nil
}

// SearchProducts handles the SearchProducts RPC.
func (s *ProductServiceServer) SearchProducts(
	ctx context.Context,
	req *productv1.SearchProductsRequest,
) (*productv1.SearchProductsResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, sherrors.ToGRPCStatus(fmt.Errorf("invalid tenant_id: %w", sherrors.ErrInvalidArgument))
	}

	result, err := s.searchHandler.Handle(ctx, queries.SearchProductsQuery{
		TenantID: tenantID,
		Query:    req.Query,
		Category: req.Category,
		Limit:    int(req.Limit),
		Offset:   int(req.Offset),
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	protos := make([]*productv1.Product, 0, len(result.Products))
	for _, p := range result.Products {
		protos = append(protos, domainToProto(p))
	}

	return &productv1.SearchProductsResponse{
		Products: protos,
		Total:    int32(result.Total),
	}, nil
}

// domainToProto converts a domain.Product to its protobuf representation.
func domainToProto(p *domain.Product) *productv1.Product {
	proto := &productv1.Product{
		ProductId:   p.ID.String(),
		TenantId:    p.TenantID.String(),
		Name:        p.Name,
		Sku:         p.SKU,
		Barcode:     p.Barcode,
		BasePrice:   p.BasePrice,
		Description: p.Description,
		Unit:        p.Unit,
		IsActive:    p.IsActive,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}

	if p.CategoryID != nil {
		proto.CategoryId = p.CategoryID.String()
	}
	if p.SalePrice != nil {
		proto.SalePrice = *p.SalePrice
	}

	return proto
}
