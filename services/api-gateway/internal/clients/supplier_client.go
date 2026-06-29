package clients

import (
	"context"

	supplierv1 "github.com/pos-stery/pos-stery/gen/go/pos/supplier/v1"
	"google.golang.org/grpc"
)

// SupplierClient is the gateway's client for the supplier-service.
type SupplierClient struct {
	client       supplierv1.SupplierServiceClient
	serviceToken string
}

// NewSupplierClient returns a SupplierClient using the given connection.
func NewSupplierClient(conn *grpc.ClientConn, serviceToken string) *SupplierClient {
	return &SupplierClient{
		client:       supplierv1.NewSupplierServiceClient(conn),
		serviceToken: serviceToken,
	}
}

// AddSupplier registers a new supplier for the given tenant.
func (c *SupplierClient) AddSupplier(ctx context.Context, req *supplierv1.AddSupplierRequest) (*supplierv1.AddSupplierResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.AddSupplier(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}

// ListSuppliers retrieves suppliers for the given tenant.
func (c *SupplierClient) ListSuppliers(ctx context.Context, req *supplierv1.ListSuppliersRequest) (*supplierv1.ListSuppliersResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.ListSuppliers(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}

// CreatePurchaseOrder creates a new purchase order.
func (c *SupplierClient) CreatePurchaseOrder(ctx context.Context, req *supplierv1.CreatePurchaseOrderRequest) (*supplierv1.CreatePurchaseOrderResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.CreatePurchaseOrder(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}
