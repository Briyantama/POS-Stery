package commands

import (
	"context"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/auth/domain"
)

// ListStoresHandler lists the stores of one tenant.
type ListStoresHandler struct {
	stores domain.StoreRepository
}

func NewListStoresHandler(stores domain.StoreRepository) *ListStoresHandler {
	return &ListStoresHandler{stores: stores}
}

func (h *ListStoresHandler) Handle(ctx context.Context, tenantID domain.TenantID) ([]*domain.Store, error) {
	stores, err := h.stores.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list stores: %w", err)
	}
	return stores, nil
}
