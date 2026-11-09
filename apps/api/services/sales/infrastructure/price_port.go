package infrastructure

import (
	"context"
	"errors"
	"fmt"

	proddomain "github.com/Briyantama/POS-Stery/services/product/domain"
	"github.com/Briyantama/POS-Stery/services/sales/domain"
	"uuid"
)

// productFinder is the slice of product's repository the price port needs.
type productFinder interface {
	FindByID(ctx context.Context, tenantID, id uuid.UUID) (*proddomain.Product, error)
}

// PricePort implements application.PriceReader from the product repository.
type PricePort struct{ products productFinder }

func NewPricePort(products productFinder) *PricePort {
	return &PricePort{products: products}
}

// UnitPrice returns the product's sale price when set, otherwise its base price.
func (p *PricePort) UnitPrice(ctx context.Context, tenantID, productID uuid.UUID) (float64, error) {
	prod, err := p.products.FindByID(ctx, tenantID, productID)
	if errors.Is(err, proddomain.ErrNotFound) {
		return 0, fmt.Errorf("%w: product %s", domain.ErrNotFound, productID)
	}
	if err != nil {
		return 0, fmt.Errorf("load product %s: %w", productID, err)
	}
	if !prod.IsActive {
		return 0, fmt.Errorf("%w: product %s is inactive", domain.ErrInvalidArgument, productID)
	}
	if prod.SalePrice != nil {
		return *prod.SalePrice, nil
	}
	return prod.BasePrice, nil
}
