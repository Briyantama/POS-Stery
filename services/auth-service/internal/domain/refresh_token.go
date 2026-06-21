package domain

import (
	"time"

	"github.com/google/uuid"
)

const RefreshTokenTTL = 30 * 24 * time.Hour

type RefreshToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TenantID   uuid.UUID
	FamilyID   uuid.UUID
	TokenHash  string
	IssuedAt   time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
	ReplacedBy *uuid.UUID
	UserAgent  string
	IPAddress  string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (rt *RefreshToken) IsRevoked() bool { return rt.RevokedAt != nil }
func (rt *RefreshToken) IsExpired() bool { return time.Now().After(rt.ExpiresAt) }
func (rt *RefreshToken) IsValid() bool   { return !rt.IsRevoked() && !rt.IsExpired() }
