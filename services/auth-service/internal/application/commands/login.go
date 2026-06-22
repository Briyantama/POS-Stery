package commands

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type LoginCommand struct {
	TenantID  string
	Email     string
	Password  string
	StoreID   string // required for cashier role; empty for admin
	UserAgent string
	IPAddress string
}

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	Claims       application.TokenClaims
}

type LoginHandler struct {
	users            domain.UserRepository
	stores           domain.StoreRepository
	signer           application.TokenSigner
	refreshTokenRepo domain.RefreshTokenRepository
}

func NewLoginHandler(
	users domain.UserRepository,
	stores domain.StoreRepository,
	signer application.TokenSigner,
	refreshTokenRepo domain.RefreshTokenRepository,
) *LoginHandler {
	return &LoginHandler{users: users, stores: stores, signer: signer, refreshTokenRepo: refreshTokenRepo}
}

func (h *LoginHandler) Handle(ctx context.Context, cmd LoginCommand) (*LoginResult, error) {
	if cmd.Email == "" || cmd.Password == "" {
		return nil, fmt.Errorf("%w: email and password required", sherrors.ErrInvalidArgument)
	}

	tenantID, err := h.resolveTenantID(ctx, cmd)
	if err != nil {
		return nil, err
	}

	user, err := h.users.FindByEmail(ctx, tenantID, cmd.Email)
	if errors.Is(err, sherrors.ErrNotFound) {
		return nil, fmt.Errorf("%w: invalid credentials", sherrors.ErrUnauthenticated)
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("%w: account disabled", sherrors.ErrUnauthenticated)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(cmd.Password)); err != nil {
		return nil, fmt.Errorf("%w: invalid credentials", sherrors.ErrUnauthenticated)
	}

	role, storeIDPtr := user.PrimaryRole()

	// Cashiers must log into a specific store
	if role == "cashier" {
		if cmd.StoreID == "" {
			return nil, fmt.Errorf("%w: store_id required for cashier login", sherrors.ErrInvalidArgument)
		}
		// Validate the requested store_id matches the assigned store
		if storeIDPtr != nil && storeIDPtr.String() != cmd.StoreID {
			return nil, fmt.Errorf("%w: store not assigned to this user", sherrors.ErrPermissionDenied)
		}
	}

	storeID := ""
	if cmd.StoreID != "" && role == "cashier" {
		storeID = cmd.StoreID
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
		return nil, fmt.Errorf("issue token: %w", err)
	}

	rawRefresh, hashRefresh, err := generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	now := time.Now().UTC()
	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TenantID:  user.TenantID,
		FamilyID:  uuid.New(), // new family per login
		TokenHash: hashRefresh,
		IssuedAt:  now,
		ExpiresAt: now.Add(domain.RefreshTokenTTL),
		UserAgent: cmd.UserAgent,
		IPAddress: cmd.IPAddress,
	}
	if err := h.refreshTokenRepo.Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresAt:    expiresAt,
		Claims:       claims,
	}, nil
}

// resolveTenantID determines the tenant for a login. An explicit tenant_id
// (service-to-service callers) is honored as-is and stays backwards-compatible.
// For public login the gateway cannot know the tenant pre-auth, so it is
// resolved from the email; unknown or ambiguous emails fail closed as invalid
// credentials to avoid tenant/user enumeration.
func (h *LoginHandler) resolveTenantID(ctx context.Context, cmd LoginCommand) (uuid.UUID, error) {
	if cmd.TenantID != "" {
		parsed, err := uuid.Parse(cmd.TenantID)
		if err != nil {
			return uuid.Nil, fmt.Errorf("%w: invalid tenant_id", sherrors.ErrInvalidArgument)
		}
		return parsed, nil
	}

	tenantID, err := h.users.ResolveTenantByEmail(ctx, cmd.Email)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: invalid credentials", sherrors.ErrUnauthenticated)
	}
	return tenantID, nil
}

// generateRefreshToken creates a cryptographically secure random token.
// Returns the raw (base64url) value sent to the client and its SHA-256 hex hash
// for storage.
func generateRefreshToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate refresh token bytes: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(raw))
	hash = hex.EncodeToString(h[:])
	return
}
