package commands

import (
	"context"
	"fmt"

	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
	"go.uber.org/zap"
)

type LogoutCommand struct {
	AccessToken  string
	RefreshToken string // optional; if empty only blacklists the access token
}

type LogoutHandler struct {
	refreshRepo domain.RefreshTokenRepository
	signer      application.TokenSigner
	log         *zap.Logger
}

func NewLogoutHandler(
	refreshRepo domain.RefreshTokenRepository,
	signer application.TokenSigner,
	log *zap.Logger,
) *LogoutHandler {
	return &LogoutHandler{refreshRepo: refreshRepo, signer: signer, log: log}
}

func (h *LogoutHandler) Handle(ctx context.Context, cmd LogoutCommand) error {
	if cmd.AccessToken == "" {
		return fmt.Errorf("%w: access_token required", sherrors.ErrInvalidArgument)
	}
	claims, err := h.signer.Verify(cmd.AccessToken)
	if err != nil {
		return fmt.Errorf("%w: invalid token", sherrors.ErrUnauthenticated)
	}
	if err := h.signer.Blacklist(ctx, claims.JTI); err != nil {
		h.log.Warn("failed to blacklist access token on logout", zap.String("jti", claims.JTI), zap.Error(err))
	}
	if cmd.RefreshToken != "" {
		hash := hashToken(cmd.RefreshToken)
		rt, err := h.refreshRepo.FindByHash(ctx, hash)
		if err == nil && !rt.IsRevoked() {
			if rErr := h.refreshRepo.Revoke(ctx, rt.ID, nil); rErr != nil {
				h.log.Warn("failed to revoke refresh token on logout", zap.Error(rErr))
			}
		}
	}
	return nil
}
