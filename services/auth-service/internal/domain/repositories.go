package domain

import (
	"context"

	"github.com/google/uuid"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, tenantID TenantID, email string) (*User, error)
	FindByID(ctx context.Context, tenantID TenantID, id UserID) (*User, error)
}

type TenantRepository interface {
	FindByID(ctx context.Context, id TenantID) (*Tenant, error)
}

type StoreRepository interface {
	FindByID(ctx context.Context, tenantID TenantID, id StoreID) (*Store, error)
	ExistsInTenant(ctx context.Context, tenantID TenantID, id StoreID) (bool, error)
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, rt *RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*RefreshToken, error)
	FindByID(ctx context.Context, id uuid.UUID) (*RefreshToken, error)
	Revoke(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID) error
	ListActiveSessions(ctx context.Context, userID, tenantID uuid.UUID) ([]*RefreshToken, error)
	RevokeAllForUser(ctx context.Context, userID, tenantID uuid.UUID) error
}
