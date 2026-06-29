package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	customerv1 "github.com/pos-stery/pos-stery/gen/go/pos/customer/v1"
	"github.com/pos-stery/pos-stery/services/api-gateway/internal/middleware"
)

// CustomerServicer defines the customer-service operations used by CustomerHandlers.
type CustomerServicer interface {
	CreateCustomer(ctx context.Context, req *customerv1.CreateCustomerRequest) (*customerv1.CreateCustomerResponse, error)
	ListCustomers(ctx context.Context, req *customerv1.ListCustomersRequest) (*customerv1.ListCustomersResponse, error)
}

// CustomerHandlers holds the customer-related HTTP handlers.
type CustomerHandlers struct {
	customer CustomerServicer
}

// NewCustomerHandlers constructs CustomerHandlers.
func NewCustomerHandlers(customer CustomerServicer) *CustomerHandlers {
	return &CustomerHandlers{customer: customer}
}

// List handles GET /api/customers.
func (h *CustomerHandlers) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())

	resp, err := h.customer.ListCustomers(r.Context(), &customerv1.ListCustomersRequest{
		TenantId: tenantID,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusOK, resp)
}

type createCustomerBody struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

// Create handles POST /api/customers.
func (h *CustomerHandlers) Create(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantIDFromCtx(r.Context())

	var body createCustomerBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		validationError(w, "body", "invalid JSON")
		return
	}
	if body.Name == "" {
		validationError(w, "name", "name is required")
		return
	}

	resp, err := h.customer.CreateCustomer(r.Context(), &customerv1.CreateCustomerRequest{
		TenantId: tenantID,
		Name:     body.Name,
		Phone:    body.Phone,
		Email:    body.Email,
	})
	if err != nil {
		renderError(w, err)
		return
	}

	renderJSON(w, http.StatusCreated, resp)
}
