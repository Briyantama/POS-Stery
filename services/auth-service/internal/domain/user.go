package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserID = uuid.UUID
type TenantID = uuid.UUID
type StoreID = uuid.UUID
type RoleID = uuid.UUID

type User struct {
	ID           UserID
	TenantID     TenantID
	Email        string
	PasswordHash string
	Name         string
	IsActive     bool
	Roles        []UserRole
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserRole struct {
	RoleID  RoleID
	Role    string // "admin" | "cashier" | "stock_manager"
	StoreID *StoreID
}

type Tenant struct {
	ID        TenantID
	Name      string
	Slug      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Store struct {
	ID        StoreID
	TenantID  TenantID
	Name      string
	Address   string
	Phone     string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// PrimaryRole returns the first role assigned to the user.
// For cashiers this will be store-scoped; for admins store_id will be nil.
func (u *User) PrimaryRole() (role string, storeID *StoreID) {
	if len(u.Roles) == 0 {
		return "", nil
	}
	r := u.Roles[0]
	return r.Role, r.StoreID
}
