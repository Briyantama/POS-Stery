package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
)

// ---- stubs ----

type stubUserRepo struct {
	user *domain.User
	err  error
}

func (s *stubUserRepo) FindByEmail(_ context.Context, _ domain.TenantID, _ string) (*domain.User, error) {
	return s.user, s.err
}

func (s *stubUserRepo) FindByID(_ context.Context, _ domain.TenantID, _ domain.UserID) (*domain.User, error) {
	return s.user, s.err
}

type stubStoreRepo struct{}

func (s *stubStoreRepo) FindByID(_ context.Context, _ domain.TenantID, _ domain.StoreID) (*domain.Store, error) {
	return nil, nil
}

func (s *stubStoreRepo) ExistsInTenant(_ context.Context, _ domain.TenantID, _ domain.StoreID) (bool, error) {
	return true, nil
}

type stubSigner struct{ issued string }

func (s *stubSigner) Issue(claims application.TokenClaims) (string, time.Time, error) {
	s.issued = "tok:" + claims.Role
	return s.issued, time.Now().Add(time.Hour), nil
}

func (s *stubSigner) Verify(_ string) (*application.TokenClaims, error) {
	return &application.TokenClaims{JTI: uuid.New().String()}, nil
}
func (s *stubSigner) Blacklist(_ context.Context, _ string) error             { return nil }
func (s *stubSigner) IsBlacklisted(_ context.Context, _ string) (bool, error) { return false, nil }

// ---- helpers ----

func hashPassword(t *testing.T, pw string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return string(hash)
}

func validTenant() string { return uuid.New().String() }

func adminUser(t *testing.T) *domain.User {
	t.Helper()
	return &domain.User{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Email:        "admin@example.com",
		PasswordHash: hashPassword(t, "secret"),
		IsActive:     true,
		Roles:        []domain.UserRole{{RoleID: uuid.New(), Role: "admin", StoreID: nil}},
	}
}

func cashierUser(t *testing.T, storeID uuid.UUID) *domain.User {
	t.Helper()
	return &domain.User{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		Email:        "cashier@example.com",
		PasswordHash: hashPassword(t, "secret"),
		IsActive:     true,
		Roles:        []domain.UserRole{{RoleID: uuid.New(), Role: "cashier", StoreID: &storeID}},
	}
}

// newLoginHandler constructs a LoginHandler with a no-op refresh token repo.
func newLoginHandler(users domain.UserRepository, stores domain.StoreRepository, signer application.TokenSigner) *commands.LoginHandler {
	return commands.NewLoginHandler(users, stores, signer, newStubRefreshTokenRepo())
}

// ---- tests ----

func TestLoginHandler_EmptyCredentials(t *testing.T) {
	h := newLoginHandler(&stubUserRepo{}, &stubStoreRepo{}, &stubSigner{})

	_, err := h.Handle(context.Background(), commands.LoginCommand{
		TenantID: validTenant(),
		Email:    "",
		Password: "secret",
	})
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestLoginHandler_InvalidTenantID(t *testing.T) {
	h := newLoginHandler(&stubUserRepo{}, &stubStoreRepo{}, &stubSigner{})

	_, err := h.Handle(context.Background(), commands.LoginCommand{
		TenantID: "not-a-uuid",
		Email:    "a@b.com",
		Password: "secret",
	})
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestLoginHandler_UserNotFound(t *testing.T) {
	h := newLoginHandler(
		&stubUserRepo{err: sherrors.ErrNotFound},
		&stubStoreRepo{},
		&stubSigner{},
	)

	_, err := h.Handle(context.Background(), commands.LoginCommand{
		TenantID: validTenant(),
		Email:    "nobody@example.com",
		Password: "secret",
	})
	if !errors.Is(err, sherrors.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated, got %v", err)
	}
}

func TestLoginHandler_RepoInfraError(t *testing.T) {
	dbErr := errors.New("connection refused")
	h := newLoginHandler(
		&stubUserRepo{err: dbErr},
		&stubStoreRepo{},
		&stubSigner{},
	)

	_, err := h.Handle(context.Background(), commands.LoginCommand{
		TenantID: validTenant(),
		Email:    "a@b.com",
		Password: "secret",
	})
	if errors.Is(err, sherrors.ErrUnauthenticated) {
		t.Fatal("infra error must not be mapped to ErrUnauthenticated")
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("want wrapped dbErr, got %v", err)
	}
}

func TestLoginHandler_WrongPassword(t *testing.T) {
	user := adminUser(t)
	h := newLoginHandler(&stubUserRepo{user: user}, &stubStoreRepo{}, &stubSigner{})

	_, err := h.Handle(context.Background(), commands.LoginCommand{
		TenantID: validTenant(),
		Email:    user.Email,
		Password: "wrong",
	})
	if !errors.Is(err, sherrors.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated, got %v", err)
	}
}

func TestLoginHandler_InactiveUser(t *testing.T) {
	user := adminUser(t)
	user.IsActive = false
	h := newLoginHandler(&stubUserRepo{user: user}, &stubStoreRepo{}, &stubSigner{})

	_, err := h.Handle(context.Background(), commands.LoginCommand{
		TenantID: validTenant(),
		Email:    user.Email,
		Password: "secret",
	})
	if !errors.Is(err, sherrors.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated, got %v", err)
	}
}

func TestLoginHandler_CashierMissingStoreID(t *testing.T) {
	storeID := uuid.New()
	user := cashierUser(t, storeID)
	h := newLoginHandler(&stubUserRepo{user: user}, &stubStoreRepo{}, &stubSigner{})

	_, err := h.Handle(context.Background(), commands.LoginCommand{
		TenantID: validTenant(),
		Email:    user.Email,
		Password: "secret",
		StoreID:  "",
	})
	if !errors.Is(err, sherrors.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestLoginHandler_CashierWrongStore(t *testing.T) {
	storeID := uuid.New()
	user := cashierUser(t, storeID)
	h := newLoginHandler(&stubUserRepo{user: user}, &stubStoreRepo{}, &stubSigner{})

	_, err := h.Handle(context.Background(), commands.LoginCommand{
		TenantID: validTenant(),
		Email:    user.Email,
		Password: "secret",
		StoreID:  uuid.New().String(), // different store
	})
	if !errors.Is(err, sherrors.ErrPermissionDenied) {
		t.Fatalf("want ErrPermissionDenied, got %v", err)
	}
}

func TestLoginHandler_AdminSuccess(t *testing.T) {
	user := adminUser(t)
	signer := &stubSigner{}
	h := newLoginHandler(&stubUserRepo{user: user}, &stubStoreRepo{}, signer)

	res, err := h.Handle(context.Background(), commands.LoginCommand{
		TenantID: validTenant(),
		Email:    user.Email,
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Claims.Role != "admin" {
		t.Errorf("want role=admin, got %q", res.Claims.Role)
	}
	if res.Claims.StoreID != "" {
		t.Errorf("admin token must not carry store_id, got %q", res.Claims.StoreID)
	}
	if res.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if res.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
}

func TestLoginHandler_CashierSuccess(t *testing.T) {
	storeID := uuid.New()
	user := cashierUser(t, storeID)
	signer := &stubSigner{}
	h := newLoginHandler(&stubUserRepo{user: user}, &stubStoreRepo{}, signer)

	res, err := h.Handle(context.Background(), commands.LoginCommand{
		TenantID: validTenant(),
		Email:    user.Email,
		Password: "secret",
		StoreID:  storeID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Claims.Role != "cashier" {
		t.Errorf("want role=cashier, got %q", res.Claims.Role)
	}
	if res.Claims.StoreID != storeID.String() {
		t.Errorf("want store_id=%s, got %q", storeID, res.Claims.StoreID)
	}
	if res.RefreshToken == "" {
		t.Error("expected non-empty refresh token on cashier login")
	}
}
