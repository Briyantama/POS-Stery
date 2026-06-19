package jwt

import (
	"crypto/rsa"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
)

const tokenTTL = 60 * time.Minute

type posClaims struct {
	jwt.RegisteredClaims
	TenantID string `json:"tid"`
	StoreID  string `json:"sid"`
	Role     string `json:"role"`
	Email    string `json:"email"`
}

type RSASigner struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	blacklist  application.BlacklistStore
}

func NewRSASigner(privateKeyPath, publicKeyPath string) (*RSASigner, error) {
	privBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	pubBytes, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	return &RSASigner{privateKey: privKey, publicKey: pubKey}, nil
}

func (s *RSASigner) Issue(claims application.TokenClaims) (string, error) {
	now := time.Now()
	jwtClaims := posClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
		TenantID: claims.TenantID,
		StoreID:  claims.StoreID,
		Role:     claims.Role,
		Email:    claims.Email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwtClaims)
	return token.SignedString(s.privateKey)
}

func (s *RSASigner) Verify(tokenStr string) (*application.TokenClaims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &posClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	c, ok := parsed.Claims.(*posClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &application.TokenClaims{
		UserID:   c.Subject,
		TenantID: c.TenantID,
		StoreID:  c.StoreID,
		Role:     c.Role,
		Email:    c.Email,
	}, nil
}

// Blacklist and IsBlacklisted are delegated to the Redis store.
// They are set via SetBlacklistStore after construction.
func (s *RSASigner) Blacklist(token string) error {
	if s.blacklist != nil {
		return s.blacklist.Blacklist(token)
	}
	return nil
}

func (s *RSASigner) IsBlacklisted(token string) (bool, error) {
	if s.blacklist != nil {
		return s.blacklist.IsBlacklisted(token)
	}
	return false, nil
}

func (s *RSASigner) SetBlacklistStore(store application.BlacklistStore) {
	s.blacklist = store
}
