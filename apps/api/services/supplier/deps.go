package supplier

import (
	"errors"
	"net/http"

	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/events"
	"github.com/jackc/pgx/v5/pgxpool"

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
}

// Build wires repositories, command handlers and the HTTP handler.
func Build(d Deps) (*HTTPHandler, error) {
	if d.Pool == nil || d.Events == nil || d.Verifier == nil || d.RequireStore == nil {
		return nil, errors.New("supplier: Pool, Events, Verifier and RequireStore are required")
	}
	suppliers := postgres.NewSupplierRepository(d.Pool)
	orders := postgres.NewPurchaseOrderRepository(d.Pool)
	return NewHTTPHandler(
		commands.NewAddSupplierHandler(suppliers),
		commands.NewCreatePurchaseOrderHandler(orders, suppliers),
		commands.NewReceiveStockHandler(orders, infrastructure.NewEventPublisher(d.Events)),
		suppliers,
		orders,
		d.Verifier,
		d.RequireStore,
	), nil
}
