package grpc

import (
	"context"

	authv1 "github.com/pos-stery/pos-stery/gen/go/pos/auth/v1"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application/commands"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthServiceServer struct {
	authv1.UnimplementedAuthServiceServer
	login        *commands.LoginHandler
	validate     *commands.ValidateHandler
	logout       *commands.LogoutHandler
	refresh      *commands.RefreshHandler
	logoutAll    *commands.LogoutAllHandler
	listSessions *commands.ListSessionsHandler
	signer       application.TokenSigner
}

func NewAuthServiceServer(
	login *commands.LoginHandler,
	validate *commands.ValidateHandler,
	logout *commands.LogoutHandler,
	refresh *commands.RefreshHandler,
	logoutAll *commands.LogoutAllHandler,
	listSessions *commands.ListSessionsHandler,
	signer application.TokenSigner,
) *AuthServiceServer {
	return &AuthServiceServer{
		login:        login,
		validate:     validate,
		logout:       logout,
		refresh:      refresh,
		logoutAll:    logoutAll,
		listSessions: listSessions,
		signer:       signer,
	}
}

func (s *AuthServiceServer) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	result, err := s.login.Handle(ctx, commands.LoginCommand{
		TenantID:  req.TenantId,
		Email:     req.Email,
		Password:  req.Password,
		StoreID:   req.StoreId,
		UserAgent: req.UserAgent,
		IPAddress: req.IpAddress,
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &authv1.LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    timestamppb.New(result.ExpiresAt),
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
	if err := s.logout.Handle(ctx, commands.LogoutCommand{
		AccessToken:  req.AccessToken,
		RefreshToken: req.RefreshToken,
	}); err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	return &authv1.LogoutResponse{Success: true}, nil
}

func (s *AuthServiceServer) Refresh(ctx context.Context, req *authv1.RefreshRequest) (*authv1.RefreshResponse, error) {
	result, err := s.refresh.Handle(ctx, commands.RefreshCommand{
		RefreshToken: req.RefreshToken,
		UserAgent:    req.UserAgent,
		IPAddress:    req.IpAddress,
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	return &authv1.RefreshResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    "Bearer",
		ExpiresAt:    timestamppb.New(result.ExpiresAt),
		Claims: &authv1.UserClaims{
			UserId:   result.Claims.UserID,
			TenantId: result.Claims.TenantID,
			StoreId:  result.Claims.StoreID,
			Role:     result.Claims.Role,
			Email:    result.Claims.Email,
		},
	}, nil
}

func (s *AuthServiceServer) LogoutAll(ctx context.Context, req *authv1.LogoutAllRequest) (*authv1.LogoutAllResponse, error) {
	if err := s.logoutAll.Handle(ctx, commands.LogoutAllCommand{
		AccessToken: req.AccessToken,
	}); err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}
	return &authv1.LogoutAllResponse{Success: true}, nil
}

func (s *AuthServiceServer) ListSessions(ctx context.Context, req *authv1.ListSessionsRequest) (*authv1.ListSessionsResponse, error) {
	result, err := s.listSessions.Handle(ctx, commands.ListSessionsQuery{
		AccessToken: req.AccessToken,
	})
	if err != nil {
		return nil, sherrors.ToGRPCStatus(err)
	}

	sessions := make([]*authv1.SessionInfo, len(result.Sessions))
	for i, s := range result.Sessions {
		sessions[i] = &authv1.SessionInfo{
			SessionId: s.ID.String(),
			FamilyId:  s.FamilyID.String(),
			IssuedAt:  timestamppb.New(s.IssuedAt),
			ExpiresAt: timestamppb.New(s.ExpiresAt),
			UserAgent: s.UserAgent,
			IpAddress: s.IPAddress,
		}
	}
	return &authv1.ListSessionsResponse{Sessions: sessions}, nil
}
