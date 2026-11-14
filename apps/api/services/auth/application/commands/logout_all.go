package commands

import (
	"context"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/auth/application"
	"github.com/Briyantama/POS-Stery/services/auth/domain"
	"uuid"
)

type LogoutAllCommand struct {
	AccessToken string
}

type LogoutAllHandler struct {
	refreshRepo domain.RefreshTokenRepository
	signer      application.TokenSigner
}

func NewLogoutAllHandler(
	refreshRepo domain.RefreshTokenRepository,
	signer application.TokenSigner,
) *LogoutAllHandler {
	return &LogoutAllHandler{refreshRepo: refreshRepo, signer: signer}
}

func (h *LogoutAllHandler) Handle(ctx context.Context, cmd LogoutAllCommand) error {
	if cmd.AccessToken == "" {
		return fmt.Errorf("%w: access_token required", domain.ErrInvalidArgument)
	}
	claims, err := h.signer.Verify(cmd.AccessToken)
	if err != nil {
		return fmt.Errorf("%w: invalid token", domain.ErrUnauthenticated)
	}

	// Blacklist current access token by JTI.
	if err := h.signer.Blacklist(ctx, claims.JTI, claims.ExpiresAt); err != nil {
		return fmt.Errorf("blacklist access token: %w", err)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return fmt.Errorf("%w: invalid user_id in token", domain.ErrUnauthenticated)
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return fmt.Errorf("%w: invalid tenant_id in token", domain.ErrUnauthenticated)
	}

	if err := h.refreshRepo.RevokeAllForUser(ctx, userID, tenantID); err != nil {
		return fmt.Errorf("revoke all sessions: %w", err)
	}
	return nil
}
