package jwt_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/infrastructure/jwt"
)

func newTestSigner(t *testing.T) *jwt.RSASigner {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key: %v", err)
	}

	dir := t.TempDir()

	privPath := filepath.Join(dir, "private.pem")
	privFile, _ := os.Create(privPath)
	_ = pem.Encode(privFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	privFile.Close()

	pubPath := filepath.Join(dir, "public.pem")
	pubFile, _ := os.Create(pubPath)
	pubDER, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	_ = pem.Encode(pubFile, &pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})
	pubFile.Close()

	signer, err := jwt.NewRSASigner(privPath, pubPath)
	if err != nil {
		t.Fatalf("new signer: %v", err)
	}
	return signer
}

func TestRSASigner_IssueAndVerify(t *testing.T) {
	s := newTestSigner(t)

	claims := application.TokenClaims{
		UserID:   "user-1",
		TenantID: "tenant-1",
		StoreID:  "store-1",
		Role:     "cashier",
		Email:    "cashier@example.com",
	}

	token, expiresAt, err := s.Issue(claims)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if token == "" {
		t.Fatal("token must not be empty")
	}
	if expiresAt.IsZero() {
		t.Fatal("expiresAt must not be zero")
	}

	got, err := s.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.UserID != claims.UserID {
		t.Errorf("UserID: want %q got %q", claims.UserID, got.UserID)
	}
	if got.TenantID != claims.TenantID {
		t.Errorf("TenantID: want %q got %q", claims.TenantID, got.TenantID)
	}
	if got.StoreID != claims.StoreID {
		t.Errorf("StoreID: want %q got %q", claims.StoreID, got.StoreID)
	}
	if got.Role != claims.Role {
		t.Errorf("Role: want %q got %q", claims.Role, got.Role)
	}
	if got.Email != claims.Email {
		t.Errorf("Email: want %q got %q", claims.Email, got.Email)
	}
}

func TestRSASigner_InvalidToken(t *testing.T) {
	s := newTestSigner(t)

	_, err := s.Verify("not.a.token")
	if err == nil {
		t.Fatal("want error for invalid token, got nil")
	}
}

func TestRSASigner_AdminTokenHasNoStore(t *testing.T) {
	s := newTestSigner(t)

	claims := application.TokenClaims{
		UserID:   "admin-1",
		TenantID: "tenant-1",
		StoreID:  "",
		Role:     "admin",
		Email:    "admin@example.com",
	}

	token, _, err := s.Issue(claims)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	got, err := s.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.StoreID != "" {
		t.Errorf("admin token must have empty StoreID, got %q", got.StoreID)
	}
}
