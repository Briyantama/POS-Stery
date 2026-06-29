package clients

import (
	"context"

	salesv1 "github.com/pos-stery/pos-stery/gen/go/pos/sales/v1"
	"google.golang.org/grpc"
)

// SalesClient is the gateway's client for the sales-service.
type SalesClient struct {
	client       salesv1.SalesServiceClient
	serviceToken string
}

// NewSalesClient returns a SalesClient using the given connection.
func NewSalesClient(conn *grpc.ClientConn, serviceToken string) *SalesClient {
	return &SalesClient{
		client:       salesv1.NewSalesServiceClient(conn),
		serviceToken: serviceToken,
	}
}

// CreateSale submits a new sale transaction.
func (c *SalesClient) CreateSale(ctx context.Context, req *salesv1.CreateSaleRequest) (*salesv1.CreateSaleResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.CreateSale(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}

// GetSalesReport retrieves a sales report for the given tenant/store/period.
func (c *SalesClient) GetSalesReport(ctx context.Context, req *salesv1.GetSalesReportRequest) (*salesv1.GetSalesReportResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.GetSalesReport(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}
