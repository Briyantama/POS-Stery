package grpc_clients

import (
	"context"
	"fmt"
	"os"

	inventoryv1 "github.com/pos-stery/pos-stery/gen/go/pos/inventory/v1"
	"github.com/pos-stery/pos-stery/services/_shared/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// InventoryClient wraps the inventory-service gRPC client with service-to-service
// authentication. It reads INVENTORY_SERVICE_URL from the environment and adds an
// X-Service-Token header computed from SERVICE_TOKEN_SECRET.
type InventoryClient struct {
	client inventoryv1.InventoryServiceClient
	secret string
}

// NewInventoryClient dials the inventory-service and returns an InventoryClient.
// It reads INVENTORY_SERVICE_URL (defaults to "localhost:8083") and
// SERVICE_TOKEN_SECRET (defaults to "changeme") from environment variables.
func NewInventoryClient() (*InventoryClient, error) {
	addr := os.Getenv("INVENTORY_SERVICE_URL")
	if addr == "" {
		addr = "localhost:8083"
	}

	secret := os.Getenv("SERVICE_TOKEN_SECRET")
	if secret == "" {
		secret = "changeme"
	}

	// Plaintext transport: inventory-service is only reachable within the pos-net
	// Docker bridge / Kubernetes overlay network. X-Service-Token HMAC-SHA256
	// verifies service identity on every call. See docs/security/grpc-transport-security.md.
	cc, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial inventory-service at %s: %w", addr, err)
	}

	return &InventoryClient{
		client: inventoryv1.NewInventoryServiceClient(cc),
		secret: secret,
	}, nil
}

// UpdateStock calls InventoryService.UpdateStock with a service token header.
// The reason is always RECEIPT for supplier-initiated stock additions.
func (c *InventoryClient) UpdateStock(
	ctx context.Context,
	tenantID, storeID, productID string,
	delta int64,
	_ string, // reason — always RECEIPT for supplier calls; ignored here
	refID string,
) error {
	ctx = c.withServiceToken(ctx)

	_, err := c.client.UpdateStock(ctx, &inventoryv1.UpdateStockRequest{
		TenantId:      tenantID,
		StoreId:       storeID,
		ProductId:     productID,
		QuantityDelta: delta,
		Reason:        inventoryv1.StockUpdateReason_STOCK_UPDATE_REASON_RECEIPT,
		ReferenceId:   refID,
	})
	if err != nil {
		return fmt.Errorf("inventory UpdateStock: %w", err)
	}
	return nil
}

// withServiceToken injects the X-Service-Token metadata into the outgoing context.
func (c *InventoryClient) withServiceToken(ctx context.Context) context.Context {
	token := middleware.ComputeServiceToken(c.secret)
	return metadata.AppendToOutgoingContext(ctx, "x-service-token", token)
}
