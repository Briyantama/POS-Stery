package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	salesv1 "github.com/pos-stery/pos-stery/gen/go/pos/sales/v1"
	"github.com/pos-stery/pos-stery/services/api-gateway/internal/middleware"
)

// SalesServicer defines the sales-service operations used by SalesHandlers.
type SalesServicer interface {
	CreateSale(ctx context.Context, req *salesv1.CreateSaleRequest) (*salesv1.CreateSaleResponse, error)
	GetSalesReport(ctx context.Context, req *salesv1.GetSalesReportRequest) (*salesv1.GetSalesReportResponse, error)
}

// SalesHandlers holds the sales-related HTTP handlers.
type SalesHandlers struct {
	sales SalesServicer
}

// NewSalesHandlers constructs SalesHandlers.
func NewSalesHandlers(sales SalesServicer) *SalesHandlers {
	return &SalesHandlers{sales: sales}
}

type saleItemBody struct {
	ProductID string  `json:"product_id"`
	Quantity  int32   `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

type createSaleBody struct {
	CustomerID     string         `json:"customer_id"`
	Items          []saleItemBody `json:"items"`
	DiscountAmount float64        `json:"discount_amount"`
	Notes          string         `json:"notes"`
}

// Create handles POST /api/sales.
func (h *SalesHandlers) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromCtx(ctx)
	storeID := middleware.StoreIDFromCtx(ctx)
	cashierID := middleware.UserIDFromCtx(ctx)

	var body createSaleBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		validationError(w, "body", "invalid JSON")
		return
	}
	if len(body.Items) == 0 {
		validationError(w, "items", "at least one item is required")
		return
	}

	items := make([]*salesv1.SaleItem, len(body.Items))
	for i, it := range body.Items {
		items[i] = &salesv1.SaleItem{
			ProductId: it.ProductID,
			Quantity:  it.Quantity,
			UnitPrice: it.UnitPrice,
		}
	}

	resp, err := h.sales.CreateSale(ctx, &salesv1.CreateSaleRequest{
		TenantId:       tenantID,
		StoreId:        storeID,
		CashierId:      cashierID,
		CustomerId:     body.CustomerID,
		Items:          items,
		DiscountAmount: body.DiscountAmount,
		Notes:          body.Notes,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusCreated, resp)
}

// Report handles GET /api/sales/report.
func (h *SalesHandlers) Report(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := middleware.TenantIDFromCtx(ctx)
	storeID := middleware.StoreIDFromCtx(ctx)

	resp, err := h.sales.GetSalesReport(ctx, &salesv1.GetSalesReportRequest{
		TenantId: tenantID,
		StoreId:  storeID,
		Date:     r.URL.Query().Get("date"),
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, resp)
}
