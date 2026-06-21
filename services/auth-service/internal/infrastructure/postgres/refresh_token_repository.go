package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/infrastructure/postgres/db"
)

// RefreshTokenRepository does NOT use WithTenantContext because token lookup
// happens by cryptographic hash before the tenant is identified. Tenant
// isolation is enforced at the application layer after the hash lookup.
type RefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewRefreshTokenRepository(pool *pgxpool.Pool) *RefreshTokenRepository {
	return &RefreshTokenRepository{pool: pool}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, rt *domain.RefreshToken) error {
	_, err := db.New(r.pool).CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:        rt.ID,
		UserID:    rt.UserID,
		TenantID:  rt.TenantID,
		FamilyID:  rt.FamilyID,
		TokenHash: rt.TokenHash,
		ExpiresAt: rt.ExpiresAt,
		UserAgent: rt.UserAgent,
		IpAddress: rt.IPAddress,
	})
	if err != nil {
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*domain.RefreshToken, error) {
	row, err := db.New(r.pool).FindRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sherrors.ErrNotFound
		}
		return nil, fmt.Errorf("find refresh token by hash: %w", err)
	}
	return toRefreshToken(row), nil
}

func (r *RefreshTokenRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.RefreshToken, error) {
	row, err := db.New(r.pool).FindRefreshTokenByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, sherrors.ErrNotFound
		}
		return nil, fmt.Errorf("find refresh token by id: %w", err)
	}
	return toRefreshToken(row), nil
}

func (r *RefreshTokenRepository) Revoke(ctx context.Context, id uuid.UUID, replacedBy *uuid.UUID) error {
	if err := db.New(r.pool).RevokeRefreshToken(ctx, db.RevokeRefreshTokenParams{
		ID:         id,
		ReplacedBy: replacedBy,
	}); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	if err := db.New(r.pool).RevokeFamilyTokens(ctx, familyID); err != nil {
		return fmt.Errorf("revoke token family: %w", err)
	}
	return nil
}

func (r *RefreshTokenRepository) ListActiveSessions(ctx context.Context, userID, tenantID uuid.UUID) ([]*domain.RefreshToken, error) {
	rows, err := db.New(r.pool).ListActiveSessionsByUser(ctx, db.ListActiveSessionsByUserParams{
		UserID:   userID,
		TenantID: tenantID,
	})
	if err != nil {
		return nil, fmt.Errorf("list active sessions: %w", err)
	}
	tokens := make([]*domain.RefreshToken, len(rows))
	for i, row := range rows {
		tokens[i] = toRefreshToken(row)
	}
	return tokens, nil
}

func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID, tenantID uuid.UUID) error {
	if err := db.New(r.pool).RevokeAllUserTokens(ctx, db.RevokeAllUserTokensParams{
		UserID:   userID,
		TenantID: tenantID,
	}); err != nil {
		return fmt.Errorf("revoke all user tokens: %w", err)
	}
	return nil
}

func toRefreshToken(row db.AuthRefreshToken) *domain.RefreshToken {
	return &domain.RefreshToken{
		ID:         row.ID,
		UserID:     row.UserID,
		TenantID:   row.TenantID,
		FamilyID:   row.FamilyID,
		TokenHash:  row.TokenHash,
		IssuedAt:   row.IssuedAt,
		ExpiresAt:  row.ExpiresAt,
		RevokedAt:  row.RevokedAt,
		ReplacedBy: row.ReplacedBy,
		UserAgent:  row.UserAgent,
		IPAddress:  row.IpAddress,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}
}
