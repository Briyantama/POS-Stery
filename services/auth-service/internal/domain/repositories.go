package domain

import "context"

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
