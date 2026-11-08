package commands

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Briyantama/POS-Stery/services/auth/application"
	"github.com/Briyantama/POS-Stery/services/auth/domain"
	"github.com/google/uuid"
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
	log         *slog.Logger
}

func NewRefreshHandler(
	refreshRepo domain.RefreshTokenRepository,
	users domain.UserRepository,
	signer application.TokenSigner,
	log *slog.Logger,
) *RefreshHandler {
	return &RefreshHandler{refreshRepo: refreshRepo, users: users, signer: signer, log: log}
}

func (h *RefreshHandler) Handle(ctx context.Context, cmd RefreshCommand) (*RefreshResult, error) {
	if cmd.RefreshToken == "" {
		return nil, fmt.Errorf("%w: refresh_token required", domain.ErrInvalidArgument)
	}

	hash := hashToken(cmd.RefreshToken)

	existing, err := h.refreshRepo.FindByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("%w: refresh token not found", domain.ErrUnauthenticated)
		}
		return nil, fmt.Errorf("find refresh token: %w", err)
	}

	// Reuse detection: if already revoked, the token was stolen or replayed.
	if existing.IsRevoked() {
		h.log.WarnContext(ctx, "refresh token reuse detected, revoking entire family",
			"family_id", existing.FamilyID.String(),
			"token_id", existing.ID.String(),
			"user_id", existing.UserID.String(),
			"tenant_id", existing.TenantID.String(),
		)
		if rErr := h.refreshRepo.RevokeFamily(ctx, existing.FamilyID); rErr != nil {
			h.log.ErrorContext(ctx, "revoke token family after reuse detection failed", "error", rErr)
		}
		return nil, fmt.Errorf("%w: refresh token reuse detected", domain.ErrUnauthenticated)
	}

	if existing.IsExpired() {
		return nil, fmt.Errorf("%w: refresh token expired", domain.ErrUnauthenticated)
	}

	// Look up user to get current state (user may have been deactivated since login).
	user, err := h.users.FindByID(ctx, existing.TenantID, existing.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if !user.IsActive {
		return nil, fmt.Errorf("%w: account disabled", domain.ErrUnauthenticated)
	}

	role, storeIDPtr := user.PrimaryRole()
	storeID := ""
	// Same claim shape as login: only cashiers carry a store.
	if role == "cashier" && storeIDPtr != nil {
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
	if err := h.refreshRepo.Rotate(ctx, existing.ID, newRT); err != nil {
		return nil, fmt.Errorf("rotate refresh token: %w", err)
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
