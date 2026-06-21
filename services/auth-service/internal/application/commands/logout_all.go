package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
	"go.uber.org/zap"
)

type LogoutAllCommand struct {
	AccessToken string
}

type LogoutAllHandler struct {
	refreshRepo domain.RefreshTokenRepository
	signer      application.TokenSigner
	log         *zap.Logger
}

func NewLogoutAllHandler(
	refreshRepo domain.RefreshTokenRepository,
	signer application.TokenSigner,
	log *zap.Logger,
) *LogoutAllHandler {
	return &LogoutAllHandler{refreshRepo: refreshRepo, signer: signer, log: log}
}

func (h *LogoutAllHandler) Handle(ctx context.Context, cmd LogoutAllCommand) error {
	if cmd.AccessToken == "" {
		return fmt.Errorf("%w: access_token required", sherrors.ErrInvalidArgument)
	}
	claims, err := h.signer.Verify(cmd.AccessToken)
	if err != nil {
		return fmt.Errorf("%w: invalid token", sherrors.ErrUnauthenticated)
	}

	// Blacklist current access token by JTI.
	if err := h.signer.Blacklist(ctx, claims.JTI); err != nil {
		h.log.Warn("failed to blacklist access token on logout-all", zap.String("jti", claims.JTI), zap.Error(err))
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return fmt.Errorf("%w: invalid user_id in token", sherrors.ErrUnauthenticated)
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return fmt.Errorf("%w: invalid tenant_id in token", sherrors.ErrUnauthenticated)
	}

	if err := h.refreshRepo.RevokeAllForUser(ctx, userID, tenantID); err != nil {
		return fmt.Errorf("revoke all sessions: %w", err)
	}
	return nil
}
