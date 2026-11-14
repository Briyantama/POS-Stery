package queries

import (
	"context"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/product/domain"
	"uuid"
)

// SearchProductsQuery carries the parameters for a product search.
type SearchProductsQuery struct {
	TenantID uuid.UUID
	Query    string
	Category string
	Limit    int
	Offset   int
}

// SearchProductsResult holds the paged search result.
type SearchProductsResult struct {
	Products []*domain.Product
	Total    int
}

// SearchProductsHandler handles the SearchProductsQuery.
type SearchProductsHandler struct {
	repo domain.ProductRepository
}

// NewSearchProductsHandler constructs a handler with the given repository.
func NewSearchProductsHandler(repo domain.ProductRepository) *SearchProductsHandler {
	return &SearchProductsHandler{repo: repo}
}

// Handle executes a product search and returns the paged result.
func (h *SearchProductsHandler) Handle(ctx context.Context, q SearchProductsQuery) (*SearchProductsResult, error) {
	products, total, err := h.repo.Search(ctx, q.TenantID, q.Query, q.Category, q.Limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("search products: %w", err)
	}

	return &SearchProductsResult{
		Products: products,
		Total:    total,
	}, nil
}
