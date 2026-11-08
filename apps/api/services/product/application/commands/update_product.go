package commands

import (
	"context"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/product/domain"
	"github.com/google/uuid"
)

// UpdateProductCommand carries the data required to update an existing product.
type UpdateProductCommand struct {
	TenantID    uuid.UUID
	ProductID   uuid.UUID
	CategoryID  *uuid.UUID
	Name        string
	Barcode     string
	BasePrice   float64
	SalePrice   *float64
	Description string
	IsActive    bool
}

// UpdateProductHandler handles the UpdateProductCommand.
type UpdateProductHandler struct {
	repo domain.ProductRepository
}

// NewUpdateProductHandler constructs a handler with the given repository.
func NewUpdateProductHandler(repo domain.ProductRepository) *UpdateProductHandler {
	return &UpdateProductHandler{repo: repo}
}

// Handle fetches the existing product, applies the command fields and persists.
func (h *UpdateProductHandler) Handle(ctx context.Context, cmd UpdateProductCommand) (*domain.Product, error) {
	if cmd.Name == "" {
		return nil, fmt.Errorf("name is required: %w", domain.ErrInvalidArgument)
	}
	if cmd.BasePrice < 0 {
		return nil, fmt.Errorf("base_price must be non-negative: %w", domain.ErrInvalidArgument)
	}

	p, err := h.repo.FindByID(ctx, cmd.TenantID, cmd.ProductID)
	if err != nil {
		return nil, fmt.Errorf("find product: %w", err)
	}

	if err := ensureCategory(ctx, h.repo, cmd.TenantID, cmd.CategoryID); err != nil {
		return nil, err
	}

	p.CategoryID = cmd.CategoryID
	p.Name = cmd.Name
	p.Barcode = cmd.Barcode
	p.BasePrice = cmd.BasePrice
	p.SalePrice = cmd.SalePrice
	p.Description = cmd.Description
	p.IsActive = cmd.IsActive

	if err := h.repo.Update(ctx, cmd.TenantID, p); err != nil {
		return nil, fmt.Errorf("update product: %w", err)
	}

	return p, nil
}
