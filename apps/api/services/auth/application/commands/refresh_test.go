package commands_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Briyantama/POS-Stery/services/auth/application"
	"github.com/Briyantama/POS-Stery/services/auth/application/commands"
	"github.com/Briyantama/POS-Stery/services/auth/domain"
)

// ---- refresh token repo stub ----

type stubRefreshTokenRepo struct {
	beforeRotate  func()
	tokens        map[string]*domain.RefreshToken // keyed by token_hash
	byID          map[uuid.UUID]*domain.RefreshToken
	revokedIDs    []uuid.UUID
	familyRevoked []uuid.UUID
	allRevoked    bool
	createErr     error
	findErr       error
}

func newStubRefreshTokenRepo() *stubRefreshTokenRepo {
	return &stubRefreshTokenRepo{
		tokens: make(map[string]*domain.RefreshToken),
		byID:   make(map[uuid.UUID]*domain.RefreshToken),
	}
}

func (r *stubRefreshTokenRepo) Create(_ context.Context, rt *domain.RefreshToken) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.tokens[rt.TokenHash] = rt
	r.byID[rt.ID] = rt
	return nil
}

func (r *stubRefreshTokenRepo) FindByHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	rt, ok := r.tokens[hash]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return rt, nil
}

func (r *stubRefreshTokenRepo) Revoke(_ context.Context, id uuid.UUID, replacedBy *uuid.UUID) error {
	r.revokedIDs = append(r.revokedIDs, id)
	if rt, ok := r.byID[id]; ok {
		now := time.Now()
		rt.RevokedAt = &now
		rt.ReplacedBy = replacedBy
	}
	return nil
}

func (r *stubRefreshTokenRepo) RevokeFamily(_ context.Context, familyID uuid.UUID) error {
	r.familyRevoked = append(r.familyRevoked, familyID)
	for _, rt := range r.tokens {
		if rt.FamilyID == familyID {
			now := time.Now()
			rt.RevokedAt = &now
		}
	}
	return nil
}

func (r *stubRefreshTokenRepo) ListActiveSessions(_ context.Context, userID, tenantID uuid.UUID) ([]*domain.RefreshToken, error) {
	var result []*domain.RefreshToken
	for _, rt := range r.tokens {
		if rt.UserID == userID && rt.TenantID == tenantID && !rt.IsRevoked() && !rt.IsExpired() {
			result = append(result, rt)
		}
	}
	return result, nil
}

func (r *stubRefreshTokenRepo) Rotate(_ context.Context, oldID uuid.UUID, newRT *domain.RefreshToken) error {
	if r.beforeRotate != nil {
		r.beforeRotate()
	}
	// mirrors the SQL: 0 rows affected when the old token is already revoked
	if old, ok := r.byID[oldID]; ok && old.IsRevoked() {
		return fmt.Errorf("%w: refresh token already rotated", domain.ErrUnauthenticated)
	}
	r.tokens[newRT.TokenHash] = newRT
	r.byID[newRT.ID] = newRT
	r.revokedIDs = append(r.revokedIDs, oldID)
	if rt, ok := r.byID[oldID]; ok {
		now := time.Now()
		rt.RevokedAt = &now
		rt.ReplacedBy = &newRT.ID
	}
	return nil
}

func TestRefreshHandler_ConcurrentRotationRejected(t *testing.T) {
	user := adminUser(t)
	repo := newStubRefreshTokenRepo()
	raw, rt := makeActiveRefreshToken(user.ID, user.TenantID)
	repo.tokens[rt.TokenHash] = rt
	repo.byID[rt.ID] = rt
	// a parallel refresh revokes the token after FindByHash but before Rotate
	repo.beforeRotate = func() { now := time.Now(); rt.RevokedAt = &now }

	h := commands.NewRefreshHandler(repo, &stubUserRepo{user: user}, &stubSigner{}, discardLogger())
	res, err := h.Handle(context.Background(), commands.RefreshCommand{RefreshToken: raw})
	if !errors.Is(err, domain.ErrUnauthenticated) || res != nil {
		t.Fatalf("want ErrUnauthenticated and no tokens, got res=%v err=%v", res, err)
	}
	if len(repo.byID) != 1 {
		t.Error("replayed rotation must not store a new refresh token")
	}
}

func (r *stubRefreshTokenRepo) RevokeAllForUser(_ context.Context, userID, tenantID uuid.UUID) error {
	r.allRevoked = true
	for _, rt := range r.tokens {
		if rt.UserID == userID && rt.TenantID == tenantID {
			now := time.Now()
			rt.RevokedAt = &now
		}
	}
	return nil
}

// ---- signer stub that returns valid UUID claims ----

type stubSignerWithClaims struct {
	userID   string
	tenantID string
}

func (s *stubSignerWithClaims) Issue(claims application.TokenClaims) (string, time.Time, error) {
	return "tok:" + claims.Role, time.Now().Add(time.Hour), nil
}

func (s *stubSignerWithClaims) Verify(_ string) (*application.TokenClaims, error) {
	return &application.TokenClaims{
		JTI:      uuid.New().String(),
		UserID:   s.userID,
		TenantID: s.tenantID,
		Role:     "admin",
	}, nil
}

func (s *stubSignerWithClaims) Blacklist(_ context.Context, _ string, _ time.Time) error { return nil }

// ---- helpers ----

// testHash mirrors the sha256-hex hash used internally by the commands package.
func testHash(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func makeActiveRefreshToken(userID, tenantID uuid.UUID) (raw string, rt *domain.RefreshToken) {
	raw = "test-refresh-token-raw-value-abc"
	rt = &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TenantID:  tenantID,
		FamilyID:  uuid.New(),
		TokenHash: testHash(raw),
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(domain.RefreshTokenTTL),
	}
	return
}

func discardLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// ---- RefreshHandler tests ----

func TestRefreshHandler_Success(t *testing.T) {
	user := adminUser(t)
	repo := newStubRefreshTokenRepo()
	raw, rt := makeActiveRefreshToken(user.ID, user.TenantID)
	repo.tokens[rt.TokenHash] = rt
	repo.byID[rt.ID] = rt

	signer := &stubSigner{}
	h := commands.NewRefreshHandler(repo, &stubUserRepo{user: user}, signer, discardLogger())

	result, err := h.Handle(context.Background(), commands.RefreshCommand{
		RefreshToken: raw,
		UserAgent:    "TestAgent/1.0",
		IPAddress:    "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if result.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if result.RefreshToken == raw {
		t.Error("rotated refresh token must differ from the original")
	}
	if len(repo.revokedIDs) == 0 {
		t.Error("expected old token to be revoked after rotation")
	}
}

func TestRefreshHandler_TokenNotFound(t *testing.T) {
	h := commands.NewRefreshHandler(newStubRefreshTokenRepo(), &stubUserRepo{}, &stubSigner{}, discardLogger())
	_, err := h.Handle(context.Background(), commands.RefreshCommand{
		RefreshToken: "nonexistent-token",
	})
	if !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated, got %v", err)
	}
}

func TestRefreshHandler_RevokedToken_RevokesFamily(t *testing.T) {
	user := adminUser(t)
	repo := newStubRefreshTokenRepo()
	raw, rt := makeActiveRefreshToken(user.ID, user.TenantID)
	// Pre-mark as revoked to simulate reuse detection.
	now := time.Now()
	rt.RevokedAt = &now
	repo.tokens[rt.TokenHash] = rt
	repo.byID[rt.ID] = rt

	h := commands.NewRefreshHandler(repo, &stubUserRepo{user: user}, &stubSigner{}, discardLogger())

	_, err := h.Handle(context.Background(), commands.RefreshCommand{RefreshToken: raw})
	if !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated on reuse, got %v", err)
	}
	if len(repo.familyRevoked) == 0 {
		t.Error("expected entire family to be revoked on token reuse")
	}
}

func TestRefreshHandler_ExpiredToken(t *testing.T) {
	user := adminUser(t)
	repo := newStubRefreshTokenRepo()
	raw, rt := makeActiveRefreshToken(user.ID, user.TenantID)
	rt.ExpiresAt = time.Now().Add(-time.Hour) // already expired
	repo.tokens[rt.TokenHash] = rt
	repo.byID[rt.ID] = rt

	h := commands.NewRefreshHandler(repo, &stubUserRepo{user: user}, &stubSigner{}, discardLogger())

	_, err := h.Handle(context.Background(), commands.RefreshCommand{RefreshToken: raw})
	if !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated for expired token, got %v", err)
	}
}

func TestRefreshHandler_InactiveUser(t *testing.T) {
	user := adminUser(t)
	user.IsActive = false
	repo := newStubRefreshTokenRepo()
	raw, rt := makeActiveRefreshToken(user.ID, user.TenantID)
	repo.tokens[rt.TokenHash] = rt
	repo.byID[rt.ID] = rt

	h := commands.NewRefreshHandler(repo, &stubUserRepo{user: user}, &stubSigner{}, discardLogger())

	_, err := h.Handle(context.Background(), commands.RefreshCommand{RefreshToken: raw})
	if !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated for inactive user, got %v", err)
	}
}

func TestRefreshHandler_EmptyToken(t *testing.T) {
	h := commands.NewRefreshHandler(newStubRefreshTokenRepo(), &stubUserRepo{}, &stubSigner{}, discardLogger())
	_, err := h.Handle(context.Background(), commands.RefreshCommand{RefreshToken: ""})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

// ---- LogoutHandler tests ----

func TestLogoutHandler_EmptyToken(t *testing.T) {
	h := commands.NewLogoutHandler(newStubRefreshTokenRepo(), &stubSigner{})
	err := h.Handle(context.Background(), commands.LogoutCommand{AccessToken: ""})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestLogoutHandler_InvalidToken(t *testing.T) {
	failSigner := &stubFailSigner{}
	h := commands.NewLogoutHandler(newStubRefreshTokenRepo(), failSigner)
	err := h.Handle(context.Background(), commands.LogoutCommand{AccessToken: "bad-token"})
	if !errors.Is(err, domain.ErrUnauthenticated) {
		t.Fatalf("want ErrUnauthenticated, got %v", err)
	}
}

func TestLogoutHandler_RevokesBothTokens(t *testing.T) {
	user := adminUser(t)
	repo := newStubRefreshTokenRepo()
	raw, rt := makeActiveRefreshToken(user.ID, user.TenantID)
	repo.tokens[rt.TokenHash] = rt
	repo.byID[rt.ID] = rt

	h := commands.NewLogoutHandler(repo, &stubSigner{})
	err := h.Handle(context.Background(), commands.LogoutCommand{
		AccessToken:  "any-access-token",
		RefreshToken: raw,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.revokedIDs) == 0 {
		t.Error("expected refresh token to be revoked on logout")
	}
}

// ---- LogoutAllHandler tests ----

func TestLogoutAllHandler_EmptyToken(t *testing.T) {
	h := commands.NewLogoutAllHandler(newStubRefreshTokenRepo(), &stubSigner{})
	err := h.Handle(context.Background(), commands.LogoutAllCommand{AccessToken: ""})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestLogoutAllHandler_RevokesAllSessions(t *testing.T) {
	user := adminUser(t)
	repo := newStubRefreshTokenRepo()
	_, rt := makeActiveRefreshToken(user.ID, user.TenantID)
	repo.tokens[rt.TokenHash] = rt
	repo.byID[rt.ID] = rt

	signer := &stubSignerWithClaims{userID: user.ID.String(), tenantID: user.TenantID.String()}
	h := commands.NewLogoutAllHandler(repo, signer)

	err := h.Handle(context.Background(), commands.LogoutAllCommand{AccessToken: "any-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.allRevoked {
		t.Error("expected RevokeAllForUser to be called")
	}
}

// ---- ListSessionsHandler tests ----

func TestListSessionsHandler_EmptyToken(t *testing.T) {
	h := commands.NewListSessionsHandler(newStubRefreshTokenRepo(), &stubSigner{})
	_, err := h.Handle(context.Background(), commands.ListSessionsQuery{AccessToken: ""})
	if !errors.Is(err, domain.ErrInvalidArgument) {
		t.Fatalf("want ErrInvalidArgument, got %v", err)
	}
}

func TestListSessionsHandler_ReturnsSessions(t *testing.T) {
	user := adminUser(t)
	repo := newStubRefreshTokenRepo()
	_, rt := makeActiveRefreshToken(user.ID, user.TenantID)
	repo.tokens[rt.TokenHash] = rt
	repo.byID[rt.ID] = rt

	signer := &stubSignerWithClaims{userID: user.ID.String(), tenantID: user.TenantID.String()}
	h := commands.NewListSessionsHandler(repo, signer)

	result, err := h.Handle(context.Background(), commands.ListSessionsQuery{AccessToken: "any-token"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Sessions) != 1 {
		t.Errorf("want 1 active session, got %d", len(result.Sessions))
	}
}

// stubFailSigner rejects all tokens.
type stubFailSigner struct{}

func (s *stubFailSigner) Issue(claims application.TokenClaims) (string, time.Time, error) {
	return "", time.Time{}, errors.New("not implemented")
}
func (s *stubFailSigner) Verify(_ string) (*application.TokenClaims, error) {
	return nil, errors.New("invalid token")
}
func (s *stubFailSigner) Blacklist(_ context.Context, _ string, _ time.Time) error { return nil }
