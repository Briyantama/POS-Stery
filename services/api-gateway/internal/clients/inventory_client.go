package clients

import (
	"context"

	inventoryv1 "github.com/pos-stery/pos-stery/gen/go/pos/inventory/v1"
	"google.golang.org/grpc"
)

// InventoryClient is the gateway's client for the inventory-service.
type InventoryClient struct {
	client       inventoryv1.InventoryServiceClient
	serviceToken string
}

// NewInventoryClient returns an InventoryClient using the given connection.
func NewInventoryClient(conn *grpc.ClientConn, serviceToken string) *InventoryClient {
	return &InventoryClient{
		client:       inventoryv1.NewInventoryServiceClient(conn),
		serviceToken: serviceToken,
	}
}

// AddItem adds a new item to inventory for a given store.
func (c *InventoryClient) AddItem(ctx context.Context, req *inventoryv1.AddItemRequest) (*inventoryv1.AddItemResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.AddItem(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}

// ListItems lists inventory items for a given store.
func (c *InventoryClient) ListItems(ctx context.Context, req *inventoryv1.ListItemsRequest) (*inventoryv1.ListItemsResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.ListItems(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}
