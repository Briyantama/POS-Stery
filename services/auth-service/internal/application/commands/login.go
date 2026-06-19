package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"golang.org/x/crypto/bcrypt"
)

type LoginCommand struct {
	TenantID string
	Email    string
	Password string
	StoreID  string // required for cashier role; empty for admin
}

type LoginResult struct {
	Token  string
	Claims application.TokenClaims
}

type LoginHandler struct {
	users   domain.UserRepository
	stores  domain.StoreRepository
	signer  application.TokenSigner
}

func NewLoginHandler(
	users domain.UserRepository,
	stores domain.StoreRepository,
	signer application.TokenSigner,
) *LoginHandler {
	return &LoginHandler{users: users, stores: stores, signer: signer}
}

func (h *LoginHandler) Handle(ctx context.Context, cmd LoginCommand) (*LoginResult, error) {
	if cmd.Email == "" || cmd.Password == "" {
		return nil, fmt.Errorf("%w: email and password required", sherrors.ErrInvalidArgument)
	}

	tenantID, err := uuid.Parse(cmd.TenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tenant_id", sherrors.ErrInvalidArgument)
	}

	user, err := h.users.FindByEmail(ctx, tenantID, cmd.Email)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid credentials", sherrors.ErrUnauthenticated)
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

	token, err := h.signer.Issue(claims)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &LoginResult{Token: token, Claims: claims}, nil
}
