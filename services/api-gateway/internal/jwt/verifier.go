package jwt

import (
	"crypto/rsa"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds the parsed JWT fields that the gateway cares about.
type Claims struct {
	JTI      string // jti (JWT ID)
	UserID   string // sub
	TenantID string // tid
	StoreID  string // sid
	Role     string // role
	Email    string // email
}

// posClaims is the internal parsed representation matching the auth-service token format.
type posClaims struct {
	jwt.RegisteredClaims
	TenantID string `json:"tid"`
	StoreID  string `json:"sid"`
	Role     string `json:"role"`
	Email    string `json:"email"`
}

// Verifier validates RS256 JWTs using a public key (no private key required).
type Verifier struct {
	publicKey *rsa.PublicKey
}

// NewVerifier parses a PEM-encoded RSA public key and returns a Verifier.
func NewVerifier(pubKeyPEM []byte) (*Verifier, error) {
	key, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse rsa public key: %w", err)
	}
	return &Verifier{publicKey: key}, nil
}

// Verify parses and validates a JWT token string, returning the extracted claims.
// Returns an error if the token is malformed, expired, or has an unexpected signing method.
func (v *Verifier) Verify(tokenStr string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &posClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return v.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("verify token: %w", err)
	}

	c, ok := parsed.Claims.(*posClaims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if c.ID == "" {
		return nil, fmt.Errorf("token missing jti claim")
	}

	return &Claims{
		JTI:      c.ID,
		UserID:   c.Subject,
		TenantID: c.TenantID,
		StoreID:  c.StoreID,
		Role:     c.Role,
		Email:    c.Email,
	}, nil
}
