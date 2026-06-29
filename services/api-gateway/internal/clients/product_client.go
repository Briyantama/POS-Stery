package clients

import (
	"context"

	productv1 "github.com/pos-stery/pos-stery/gen/go/pos/product/v1"
	"google.golang.org/grpc"
)

// ProductClient is the gateway's client for the product-service.
type ProductClient struct {
	client       productv1.ProductServiceClient
	serviceToken string
}

// NewProductClient dials the product-service and returns a ProductClient.
func NewProductClient(conn *grpc.ClientConn, serviceToken string) *ProductClient {
	return &ProductClient{
		client:       productv1.NewProductServiceClient(conn),
		serviceToken: serviceToken,
	}
}

// CreateProduct forwards a create product request to the product-service.
func (c *ProductClient) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.CreateProductResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.CreateProduct(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}

// UpdateProduct forwards an update product request to the product-service.
func (c *ProductClient) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.UpdateProductResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.UpdateProduct(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}

// SearchProducts lists products matching the search criteria.
func (c *ProductClient) SearchProducts(ctx context.Context, req *productv1.SearchProductsRequest) (*productv1.SearchProductsResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.SearchProducts(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}
