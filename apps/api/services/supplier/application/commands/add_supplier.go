package commands

import (
	"context"
	"fmt"

	"github.com/Briyantama/POS-Stery/services/supplier/domain"
	"uuid"
)

// AddSupplierCommand carries the data needed to register a new supplier.
type AddSupplierCommand struct {
	TenantID uuid.UUID
	Name     string
	Contact  string
	Phone    string
	Email    string
	Address  string
}

// AddSupplierHandler processes AddSupplierCommand.
type AddSupplierHandler struct {
	repo domain.SupplierRepository
}

// NewAddSupplierHandler creates an AddSupplierHandler.
func NewAddSupplierHandler(repo domain.SupplierRepository) *AddSupplierHandler {
	return &AddSupplierHandler{repo: repo}
}

// Handle executes the AddSupplier use case.
func (h *AddSupplierHandler) Handle(ctx context.Context, cmd AddSupplierCommand) (*domain.Supplier, error) {
	if cmd.TenantID == uuid.Nil() {
		return nil, fmt.Errorf("tenantID required: %w", domain.ErrInvalidArgument)
	}
	if cmd.Name == "" {
		return nil, fmt.Errorf("name required: %w", domain.ErrInvalidArgument)
	}

	supplier := &domain.Supplier{
		TenantID: cmd.TenantID,
		Name:     cmd.Name,
		Contact:  cmd.Contact,
		Phone:    cmd.Phone,
		Email:    cmd.Email,
		Address:  cmd.Address,
	}

	if err := h.repo.Add(ctx, supplier); err != nil {
		return nil, fmt.Errorf("add supplier: %w", err)
	}
	return supplier, nil
}
