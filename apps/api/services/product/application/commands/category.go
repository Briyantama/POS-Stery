package commands

import (
	"context"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/product/domain"
	"uuid"
)

// validPrice bounds money to NUMERIC(12,2) so overflow is a 422, not a Postgres 500; NaN fails too.
func validPrice(v float64) bool { return v >= 0 && v < 1e10 }

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
