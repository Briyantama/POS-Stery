package commands

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"github.com/pos-stery/pos-stery/services/product-service/internal/domain"
)

// CreateProductCommand carries the data required to create a new product.
type CreateProductCommand struct {
	TenantID    uuid.UUID
	CategoryID  *uuid.UUID
	Name        string
	SKU         string
	Barcode     string
	BasePrice   float64
	Description string
	Unit        string
}

// CreateProductHandler handles the CreateProductCommand.
type CreateProductHandler struct {
	repo domain.ProductRepository
}

// NewCreateProductHandler constructs a handler with the given repository.
func NewCreateProductHandler(repo domain.ProductRepository) *CreateProductHandler {
	return &CreateProductHandler{repo: repo}
}

// Handle validates the command, constructs a Product aggregate and persists it.
func (h *CreateProductHandler) Handle(ctx context.Context, cmd CreateProductCommand) (*domain.Product, error) {
	if cmd.Name == "" {
		return nil, fmt.Errorf("name is required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.SKU == "" {
		return nil, fmt.Errorf("sku is required: %w", sherrors.ErrInvalidArgument)
	}
	if cmd.BasePrice < 0 {
		return nil, fmt.Errorf("base_price must be non-negative: %w", sherrors.ErrInvalidArgument)
	}

	p := &domain.Product{
		ID:          uuid.New(),
		TenantID:    cmd.TenantID,
		CategoryID:  cmd.CategoryID,
		Name:        cmd.Name,
		SKU:         cmd.SKU,
		Barcode:     cmd.Barcode,
		BasePrice:   cmd.BasePrice,
		Description: cmd.Description,
		Unit:        cmd.Unit,
		IsActive:    true,
	}

	if err := h.repo.Create(ctx, cmd.TenantID, p); err != nil {
		return nil, fmt.Errorf("create product: %w", err)
	}

	return p, nil
}
