package clients

import (
	"context"

	customerv1 "github.com/pos-stery/pos-stery/gen/go/pos/customer/v1"
	"google.golang.org/grpc"
)

// CustomerClient is the gateway's client for the customer-service.
type CustomerClient struct {
	client       customerv1.CustomerServiceClient
	serviceToken string
}

// NewCustomerClient returns a CustomerClient using the given connection.
func NewCustomerClient(conn *grpc.ClientConn, serviceToken string) *CustomerClient {
	return &CustomerClient{
		client:       customerv1.NewCustomerServiceClient(conn),
		serviceToken: serviceToken,
	}
}

// CreateCustomer registers a new customer.
func (c *CustomerClient) CreateCustomer(ctx context.Context, req *customerv1.CreateCustomerRequest) (*customerv1.CreateCustomerResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.CreateCustomer(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}

// ListCustomers retrieves customers for the given tenant.
func (c *CustomerClient) ListCustomers(ctx context.Context, req *customerv1.ListCustomersRequest) (*customerv1.ListCustomersResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.ListCustomers(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}
