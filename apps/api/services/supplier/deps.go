package supplier

import (
	"errors"
	"net/http"

	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/events"
	"github.com/jackc/pgx/v5/pgxpool"

	invcommands "github.com/Briyantama/POS-Stery/services/inventory/application/commands"
	"github.com/Briyantama/POS-Stery/services/supplier/application/commands"
	"github.com/Briyantama/POS-Stery/services/supplier/infrastructure"
	"github.com/Briyantama/POS-Stery/services/supplier/infrastructure/postgres"
)

// Deps are the collaborators the supplier service needs from cmd/api.
type Deps struct {
	Pool         *pgxpool.Pool
	Events       *events.Publisher
	Verifier     authn.Verifier
	RequireStore func(http.Handler) http.Handler
	// Stock is inventory's UpdateStockHandler, called in-process to book goods receipts.
	Stock *invcommands.UpdateStockHandler
}

// Build wires repositories, command handlers and the HTTP handler.
func Build(d Deps) (*HTTPHandler, error) {
	if d.Pool == nil || d.Events == nil || d.Verifier == nil || d.RequireStore == nil || d.Stock == nil {
		return nil, errors.New("supplier: Pool, Events, Verifier, RequireStore and Stock are required")
	}
	suppliers := postgres.NewSupplierRepository(d.Pool)
	orders := postgres.NewPurchaseOrderRepository(d.Pool)
	return NewHTTPHandler(
		commands.NewAddSupplierHandler(suppliers),
		commands.NewCreatePurchaseOrderHandler(orders, suppliers),
		commands.NewReceiveStockHandler(orders, infrastructure.NewInventoryPort(d.Stock), infrastructure.NewEventPublisher(d.Events)),
		suppliers,
		orders,
		d.Verifier,
		d.RequireStore,
	), nil
}
