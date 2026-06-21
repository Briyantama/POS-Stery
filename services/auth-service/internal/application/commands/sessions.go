package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/domain"
)

type ListSessionsQuery struct {
	AccessToken string
}

type SessionInfo struct {
	ID        uuid.UUID
	FamilyID  uuid.UUID
	IssuedAt  time.Time
	ExpiresAt time.Time
	UserAgent string
	IPAddress string
}

type ListSessionsResult struct {
	Sessions []SessionInfo
}

type ListSessionsHandler struct {
	refreshRepo domain.RefreshTokenRepository
	signer      application.TokenSigner
}

func NewListSessionsHandler(
	refreshRepo domain.RefreshTokenRepository,
	signer application.TokenSigner,
) *ListSessionsHandler {
	return &ListSessionsHandler{refreshRepo: refreshRepo, signer: signer}
}

func (h *ListSessionsHandler) Handle(ctx context.Context, q ListSessionsQuery) (*ListSessionsResult, error) {
	if q.AccessToken == "" {
		return nil, fmt.Errorf("%w: access_token required", sherrors.ErrInvalidArgument)
	}
	claims, err := h.signer.Verify(q.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid token", sherrors.ErrUnauthenticated)
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid user_id in token", sherrors.ErrUnauthenticated)
	}
	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid tenant_id in token", sherrors.ErrUnauthenticated)
	}
	tokens, err := h.refreshRepo.ListActiveSessions(ctx, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	sessions := make([]SessionInfo, len(tokens))
	for i, t := range tokens {
		sessions[i] = SessionInfo{
			ID:        t.ID,
			FamilyID:  t.FamilyID,
			IssuedAt:  t.IssuedAt,
			ExpiresAt: t.ExpiresAt,
			UserAgent: t.UserAgent,
			IPAddress: maskIP(t.IPAddress),
		}
	}
	return &ListSessionsResult{Sessions: sessions}, nil
}

// maskIP obscures the last octet of an IPv4 address or the last segment of
// an IPv6 address so that IP data is not leaked to the session listing caller.
func maskIP(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		return parts[0] + "." + parts[1] + "." + parts[2] + ".xxx"
	}
	if idx := strings.LastIndex(ip, ":"); idx > 0 {
		return ip[:idx] + ":****"
	}
	return ip
}
