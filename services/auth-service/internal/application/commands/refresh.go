package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
	"go.uber.org/zap"
)

type RefreshCommand struct {
	RefreshToken string
	UserAgent    string
	IPAddress    string
}

type RefreshResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	Claims       application.TokenClaims
}

type RefreshHandler struct {
	refreshRepo domain.RefreshTokenRepository
	users       domain.UserRepository
	signer      application.TokenSigner
	log         *zap.Logger
}

func NewRefreshHandler(
	refreshRepo domain.RefreshTokenRepository,
	users domain.UserRepository,
	signer application.TokenSigner,
	log *zap.Logger,
) *RefreshHandler {
	return &RefreshHandler{refreshRepo: refreshRepo, users: users, signer: signer, log: log}
}

func (h *RefreshHandler) Handle(ctx context.Context, cmd RefreshCommand) (*RefreshResult, error) {
	if cmd.RefreshToken == "" {
		return nil, fmt.Errorf("%w: refresh_token required", sherrors.ErrInvalidArgument)
	}

	hash := hashToken(cmd.RefreshToken)

	existing, err := h.refreshRepo.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sherrors.ErrNotFound) {
			return nil, fmt.Errorf("%w: refresh token not found", sherrors.ErrUnauthenticated)
		}
		return nil, fmt.Errorf("find refresh token: %w", err)
	}

	// Reuse detection: if already revoked, the token was stolen or replayed.
	if existing.IsRevoked() {
		h.log.Warn("refresh token reuse detected — revoking entire family",
			zap.String("family_id", existing.FamilyID.String()),
			zap.String("token_id", existing.ID.String()),
			zap.String("user_id", existing.UserID.String()),
		)
		if rErr := h.refreshRepo.RevokeFamily(ctx, existing.FamilyID); rErr != nil {
			h.log.Error("failed to revoke token family after reuse detection", zap.Error(rErr))
		}
		return nil, fmt.Errorf("%w: refresh token reuse detected", sherrors.ErrUnauthenticated)
	}

	if existing.IsExpired() {
		return nil, fmt.Errorf("%w: refresh token expired", sherrors.ErrUnauthenticated)
	}

	// Look up user to get current state (user may have been deactivated since login).
	user, err := h.users.FindByID(ctx, existing.TenantID, existing.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if !user.IsActive {
		return nil, fmt.Errorf("%w: account disabled", sherrors.ErrUnauthenticated)
	}

	role, storeIDPtr := user.PrimaryRole()
	storeID := ""
	if storeIDPtr != nil {
		storeID = storeIDPtr.String()
	}

	// Generate new token pair.
	newRawRefresh, newHashRefresh, err := generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	newRTID := uuid.New()
	now := time.Now().UTC()
	newRT := &domain.RefreshToken{
		ID:        newRTID,
		UserID:    existing.UserID,
		TenantID:  existing.TenantID,
		FamilyID:  existing.FamilyID, // same family — preserves the chain
		TokenHash: newHashRefresh,
		IssuedAt:  now,
		ExpiresAt: now.Add(domain.RefreshTokenTTL),
		UserAgent: cmd.UserAgent,
		IPAddress: cmd.IPAddress,
	}
	if err := h.refreshRepo.Create(ctx, newRT); err != nil {
		return nil, fmt.Errorf("store new refresh token: %w", err)
	}

	// Revoke old token, point replaced_by to the new one.
	if err := h.refreshRepo.Revoke(ctx, existing.ID, &newRTID); err != nil {
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}

	claims := application.TokenClaims{
		UserID:   user.ID.String(),
		TenantID: user.TenantID.String(),
		StoreID:  storeID,
		Role:     role,
		Email:    user.Email,
	}
	accessToken, expiresAt, err := h.signer.Issue(claims)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	return &RefreshResult{
		AccessToken:  accessToken,
		RefreshToken: newRawRefresh,
		ExpiresAt:    expiresAt,
		Claims:       claims,
	}, nil
}

// hashToken returns the SHA-256 hex digest of the raw opaque token.
// This is the value stored in the database.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
