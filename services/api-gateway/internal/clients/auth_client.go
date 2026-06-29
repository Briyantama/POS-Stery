package clients

import (
	"context"

	authv1 "github.com/pos-stery/pos-stery/gen/go/pos/auth/v1"
	"google.golang.org/grpc"
)

// AuthClient is the gateway's client for the auth-service.
type AuthClient struct {
	client       authv1.AuthServiceClient
	serviceToken string
}

// NewAuthClient dials the auth-service and returns an AuthClient.
func NewAuthClient(conn *grpc.ClientConn, serviceToken string) *AuthClient {
	return &AuthClient{
		client:       authv1.NewAuthServiceClient(conn),
		serviceToken: serviceToken,
	}
}

// Login forwards a login request to the auth-service.
func (c *AuthClient) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.Login(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}

// Logout forwards a logout request (JWT blacklisting) to the auth-service.
func (c *AuthClient) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.Logout(ctx, req)
	if err != nil {
		return nil, wrapGrpcError(err)
	}
	return resp, nil
}

// StoreExistsInTenant checks whether store_id belongs to tenant_id in the auth-service.
func (c *AuthClient) StoreExistsInTenant(ctx context.Context, tenantID, storeID string) (bool, error) {
	ctx, cancel := callCtx(ctx)
	defer cancel()
	ctx = withServiceToken(ctx, c.serviceToken)
	resp, err := c.client.ValidateStore(ctx, &authv1.ValidateStoreRequest{
		TenantId: tenantID,
		StoreId:  storeID,
	})
	if err != nil {
		return false, wrapGrpcError(err)
	}
	return resp.Valid, nil
}
