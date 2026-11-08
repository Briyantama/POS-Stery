package commands

import (
	"context"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/product/domain"
	"github.com/google/uuid"
)

// ensureCategory rejects a category_id that does not belong to the tenant.
func ensureCategory(ctx context.Context, repo domain.ProductRepository, tenantID uuid.UUID, categoryID *uuid.UUID) error {
	if categoryID == nil {
		return nil
	}
	ok, err := repo.CategoryExists(ctx, tenantID, *categoryID)
	if err != nil {
		return fmt.Errorf("check category: %w", err)
	}
	if !ok {
		return fmt.Errorf("category_id does not exist: %w", domain.ErrInvalidArgument)
	}
	return nil
}
