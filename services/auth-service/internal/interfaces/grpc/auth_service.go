package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	authv1 "github.com/pos-stery/pos-stery/gen/go/pos/auth/v1"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// storeExistenceChecker is the subset of the store repository used by ValidateStore.
type storeExistenceChecker interface {
	ExistsInTenant(ctx context.Context, tenantID domain.TenantID, id domain.StoreID) (bool, error)
}

type AuthServiceServer struct {
	authv1.UnimplementedAuthServiceServer
	login        *commands.LoginHandler
	validate     *commands.ValidateHandler
	logout       *commands.LogoutHandler
	refresh      *commands.RefreshHandler
	logoutAll    *commands.LogoutAllHandler
	listSessions *commands.ListSessionsHandler
	signer       application.TokenSigner
	storeRepo    storeExistenceChecker
}

func NewAuthServiceServer(
	login *commands.LoginHandler,
	validate *commands.ValidateHandler,
	logout *commands.LogoutHandler,
	refresh *commands.RefreshHandler,
	logoutAll *commands.LogoutAllHandler,
	listSessions *commands.ListSessionsHandler,
	signer application.TokenSigner,
	storeRepo storeExistenceChecker,
) *AuthServiceServer {
	return &AuthServiceServer{
		login:        login,
		validate:     validate,
		logout:       logout,
		refresh:      refresh,
		logoutAll:    logoutAll,
		listSessions: listSessions,
		signer:       signer,
		storeRepo:    storeRepo,
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

func (s *AuthServiceServer) ValidateStore(ctx context.Context, req *authv1.ValidateStoreRequest) (*authv1.ValidateStoreResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	storeID, err := uuid.Parse(req.StoreId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid store_id: %v", err)
	}

	ok, err := s.storeRepo.ExistsInTenant(ctx, tenantID, storeID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, fmt.Sprintf("check store ownership: %v", err))
	}
	return &authv1.ValidateStoreResponse{Valid: ok}, nil
}
