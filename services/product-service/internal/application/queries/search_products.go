package queries

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pos-stery/pos-stery/services/product-service/internal/domain"
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

// GetByID fetches a single product by its ID for the given tenant.
func (h *SearchProductsHandler) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Product, error) {
	p, err := h.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("get product by id: %w", err)
	}
	return p, nil
}

// Handle executes a product search and returns the paged result.
func (h *SearchProductsHandler) Handle(ctx context.Context, q SearchProductsQuery) (*SearchProductsResult, error) {
	if q.Limit <= 0 {
		q.Limit = 20
	}
	if q.Limit > 100 {
		q.Limit = 100
	}

	products, total, err := h.repo.Search(ctx, q.TenantID, q.Query, q.Category, q.Limit, q.Offset)
	if err != nil {
		return nil, fmt.Errorf("search products: %w", err)
	}

	return &SearchProductsResult{
		Products: products,
		Total:    total,
	}, nil
}
