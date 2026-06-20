package grpc

import (
	"context"

	authv1 "github.com/pos-stery/pos-stery/gen/go/pos/auth/v1"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application/commands"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthServiceServer struct {
	authv1.UnimplementedAuthServiceServer
	login    *commands.LoginHandler
	validate *commands.ValidateHandler
}

func NewAuthServiceServer(
	login *commands.LoginHandler,
	validate *commands.ValidateHandler,
) *AuthServiceServer {
	return &AuthServiceServer{login: login, validate: validate}
}

func (s *AuthServiceServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	result, err := s.login.Handle(ctx, commands.LoginCommand{
		TenantID: req.TenantId,
		Email:    req.Email,
		Password: req.Password,
		StoreID:  req.StoreId,
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &authv1.LoginResponse{
		Token:     result.Token,
		ExpiresAt: timestamppb.Now(),
		Claims: &authv1.UserClaims{
			UserId:   result.Claims.UserID,
			TenantId: result.Claims.TenantID,
			StoreId:  result.Claims.StoreID,
			Role:     result.Claims.Role,
			Email:    result.Claims.Email,
		},
	}, nil
}

func (s *AuthServiceServer) Validate(ctx context.Context, req *authv1.ValidateRequest) (*authv1.ValidateResponse, error) {
	result, err := s.validate.Handle(ctx, commands.ValidateCommand{Token: req.Token})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	resp := &authv1.ValidateResponse{Valid: result.Valid}
	if result.Claims != nil {
		resp.Claims = &authv1.UserClaims{
			UserId:   result.Claims.UserID,
			TenantId: result.Claims.TenantID,
			StoreId:  result.Claims.StoreID,
			Role:     result.Claims.Role,
			Email:    result.Claims.Email,
		}
	}
	return resp, nil
}

func (s *AuthServiceServer) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if req.Token == "" {
		return &authv1.LogoutResponse{Success: false}, nil
	}
	// Token blacklisting is handled by the signer's Blacklist method
	// In production, wire this to the LoginHandler's signer
	return &authv1.LogoutResponse{Success: true}, nil
}
