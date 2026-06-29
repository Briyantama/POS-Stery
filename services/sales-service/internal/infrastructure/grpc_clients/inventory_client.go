package grpc_clients

import (
	"context"
	"fmt"
	"os"

	inventoryv1 "github.com/pos-stery/pos-stery/gen/go/pos/inventory/v1"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/_shared/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// InventoryClient is the gRPC client adapter for the inventory-service.
// It satisfies the application.InventoryUpdater port.
type InventoryClient struct {
	client       inventoryv1.InventoryServiceClient
	serviceToken string
}

// InventoryClientConfig holds configuration for the inventory gRPC client.
type InventoryClientConfig struct {
	// ServiceURL is read from the INVENTORY_SERVICE_URL environment variable.
	ServiceURL   string
	ServiceToken string
}

// NewInventoryClient dials the inventory-service and returns an InventoryClient.
func NewInventoryClient(cfg InventoryClientConfig) (*InventoryClient, error) {
	url := cfg.ServiceURL
	if url == "" {
		url = os.Getenv("INVENTORY_SERVICE_URL")
	}
	if url == "" {
		return nil, fmt.Errorf("INVENTORY_SERVICE_URL is not set")
	}

	// Plaintext transport: inventory-service is only reachable within the pos-net
	// Docker bridge / Kubernetes overlay network. X-Service-Token HMAC-SHA256
	// verifies service identity on every call. See docs/security/grpc-transport-security.md.
	conn, err := grpc.NewClient(url,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("dial inventory-service at %s: %w", url, err)
	}

	return &InventoryClient{
		client:       inventoryv1.NewInventoryServiceClient(conn),
		serviceToken: cfg.ServiceToken,
	}, nil
}

// DeductStock calls inventory UpdateStock with a negative quantity_delta.
// Returns ErrStockWouldGoNegative when the inventory-service rejects the deduction.
func (c *InventoryClient) DeductStock(ctx context.Context, tenantID, storeID, productID string, quantity int64, saleID string) error {
	ctx = c.withServiceToken(ctx)

	_, err := c.client.UpdateStock(ctx, &inventoryv1.UpdateStockRequest{
		TenantId:      tenantID,
		StoreId:       storeID,
		ProductId:     productID,
		QuantityDelta: -quantity,
		Reason:        inventoryv1.StockUpdateReason_STOCK_UPDATE_REASON_SALE,
		ReferenceId:   saleID,
	})
	if err != nil {
		return mapInventoryError(err)
	}
	return nil
}

// RestoreStock calls inventory UpdateStock with a positive quantity_delta.
// This is the compensation action for a failed saga step.
func (c *InventoryClient) RestoreStock(ctx context.Context, tenantID, storeID, productID string, quantity int64, saleID string) error {
	ctx = c.withServiceToken(ctx)

	_, err := c.client.UpdateStock(ctx, &inventoryv1.UpdateStockRequest{
		TenantId:      tenantID,
		StoreId:       storeID,
		ProductId:     productID,
		QuantityDelta: quantity,
		Reason:        inventoryv1.StockUpdateReason_STOCK_UPDATE_REASON_ADJUSTMENT,
		ReferenceId:   saleID,
	})
	if err != nil {
		return fmt.Errorf("restore stock for product %s: %w", productID, err)
	}
	return nil
}

// withServiceToken attaches the HMAC service token to the outgoing metadata.
func (c *InventoryClient) withServiceToken(ctx context.Context) context.Context {
	if c.serviceToken == "" {
		return ctx
	}
	token := middleware.ComputeServiceToken(c.serviceToken)
	return metadata.AppendToOutgoingContext(ctx, "x-service-token", token)
}

// mapInventoryError translates gRPC status codes from the inventory-service
// into shared domain errors.
func mapInventoryError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return fmt.Errorf("inventory call: %w", err)
	}
	switch st.Code() {
	case codes.FailedPrecondition:
		return fmt.Errorf("%w: %s", sherrors.ErrStockWouldGoNegative, st.Message())
	case codes.NotFound:
		return fmt.Errorf("%w: %s", sherrors.ErrNotFound, st.Message())
	default:
		return fmt.Errorf("inventory call (%s): %s", st.Code(), st.Message())
	}
}
