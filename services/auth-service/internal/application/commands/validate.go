package commands

import (
	"context"
	"fmt"

	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
)

type ValidateCommand struct {
	Token string
}

type ValidateResult struct {
	Valid  bool
	Claims *application.TokenClaims
}

type ValidateHandler struct {
	signer application.TokenSigner
}

func NewValidateHandler(signer application.TokenSigner) *ValidateHandler {
	return &ValidateHandler{signer: signer}
}

func (h *ValidateHandler) Handle(ctx context.Context, cmd ValidateCommand) (*ValidateResult, error) {
	if cmd.Token == "" {
		return nil, fmt.Errorf("%w: token required", sherrors.ErrInvalidArgument)
	}

	claims, err := h.signer.Verify(cmd.Token)
	if err != nil {
		return &ValidateResult{Valid: false}, nil
	}

	blacklisted, err := h.signer.IsBlacklisted(claims.JTI)
	if err != nil {
		return nil, fmt.Errorf("check blacklist: %w", err)
	}
	if blacklisted {
		return &ValidateResult{Valid: false}, nil
	}

	return &ValidateResult{Valid: true, Claims: claims}, nil
}
