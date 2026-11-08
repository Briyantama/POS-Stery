package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/auth/application"
	"github.com/Briyantama/POS-Stery/services/auth/domain"
)

type LogoutCommand struct {
	AccessToken  string
	RefreshToken string // optional; if empty only blacklists the access token
}

type LogoutHandler struct {
	refreshRepo domain.RefreshTokenRepository
	signer      application.TokenSigner
}

func NewLogoutHandler(
	refreshRepo domain.RefreshTokenRepository,
	signer application.TokenSigner,
) *LogoutHandler {
	return &LogoutHandler{refreshRepo: refreshRepo, signer: signer}
}

func (h *LogoutHandler) Handle(ctx context.Context, cmd LogoutCommand) error {
	if cmd.AccessToken == "" {
		return fmt.Errorf("%w: access_token required", domain.ErrInvalidArgument)
	}
	claims, err := h.signer.Verify(cmd.AccessToken)
	if err != nil {
		return fmt.Errorf("%w: invalid token", domain.ErrUnauthenticated)
	}
	if err := h.signer.Blacklist(ctx, claims.JTI, claims.ExpiresAt); err != nil {
		return fmt.Errorf("blacklist access token: %w", err)
	}
	if cmd.RefreshToken != "" {
		hash := hashToken(cmd.RefreshToken)
		rt, err := h.refreshRepo.FindByHash(ctx, hash)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return fmt.Errorf("find refresh token: %w", err)
		}
		if err == nil && !rt.IsRevoked() {
			if rErr := h.refreshRepo.Revoke(ctx, rt.ID, nil); rErr != nil {
				return fmt.Errorf("revoke refresh token: %w", rErr)
			}
		}
	}
	return nil
}
