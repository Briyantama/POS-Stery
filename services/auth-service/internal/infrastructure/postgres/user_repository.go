package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pos-stery/pos-stery/services/_shared/database"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/infrastructure/postgres/db"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) FindByEmail(ctx context.Context, tenantID domain.TenantID, email string) (*domain.User, error) {
	var u *domain.User
	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.GetUserByEmail(ctx, db.GetUserByEmailParams{
			Email:    email,
			TenantID: tenantID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return sherrors.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get user by email: %w", err)
		}
		roles, err := q.GetUserRoles(ctx, db.GetUserRolesParams{
			UserID:   row.ID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("get user roles: %w", err)
		}
		u = toUser(row.ID, row.TenantID, row.Email, row.PasswordHash, row.Name, row.IsActive, row.CreatedAt, row.UpdatedAt, roles)
		return nil
	})
	return u, err
}

func (r *UserRepository) FindByID(ctx context.Context, tenantID domain.TenantID, id domain.UserID) (*domain.User, error) {
	var u *domain.User
	err := database.WithTenantContext(ctx, r.pool, tenantID.String(), "", func(tx pgx.Tx) error {
		q := db.New(tx)
		row, err := q.GetUserByID(ctx, db.GetUserByIDParams{
			ID:       id,
			TenantID: tenantID,
		})
		if errors.Is(err, pgx.ErrNoRows) {
			return sherrors.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get user by id: %w", err)
		}
		roles, err := q.GetUserRoles(ctx, db.GetUserRolesParams{
			UserID:   row.ID,
			TenantID: tenantID,
		})
		if err != nil {
			return fmt.Errorf("get user roles: %w", err)
		}
		u = toUser(row.ID, row.TenantID, row.Email, row.PasswordHash, row.Name, row.IsActive, row.CreatedAt, row.UpdatedAt, roles)
		return nil
	})
	return u, err
}

// ResolveTenantByEmail resolves the tenant for a public login by email, before
// any tenant context exists. It calls the SECURITY DEFINER function
// auth.resolve_login_tenant, which bypasses RLS and returns a tenant only when
// exactly one active user owns the email (NULL otherwise → ErrNotFound).
func (r *UserRepository) ResolveTenantByEmail(ctx context.Context, email string) (domain.TenantID, error) {
	const q = `SELECT COALESCE(auth.resolve_login_tenant($1), '00000000-0000-0000-0000-000000000000'::uuid)`
	var tenantID uuid.UUID
	if err := r.pool.QueryRow(ctx, q, email).Scan(&tenantID); err != nil {
		return uuid.Nil, fmt.Errorf("resolve login tenant: %w", err)
	}
	if tenantID == uuid.Nil {
		return uuid.Nil, sherrors.ErrNotFound
	}
	return tenantID, nil
}

func toUser(id, tenantID uuid.UUID, email, passwordHash, name string, isActive bool, createdAt, updatedAt time.Time, roleRows []db.GetUserRolesRow) *domain.User {
	roles := make([]domain.UserRole, len(roleRows))
	for i, r := range roleRows {
		roles[i] = domain.UserRole{
			RoleID:  r.RoleID,
			Role:    r.RoleName,
			StoreID: r.StoreID,
		}
	}
	return &domain.User{
		ID:           id,
		TenantID:     tenantID,
		Email:        email,
		PasswordHash: passwordHash,
		Name:         name,
		IsActive:     isActive,
		Roles:        roles,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}
}
