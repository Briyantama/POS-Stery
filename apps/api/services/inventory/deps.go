// Package inventory is the stock service: per-store quantities, low-stock thresholds and the
// in-process stock adjustment used by sales and supplier.
package inventory

import (
	"net/http"

	"github.com/Briyantama/POS-Stery/services/inventory/application/commands"
	"github.com/Briyantama/POS-Stery/services/inventory/application/queries"
	"github.com/Briyantama/POS-Stery/services/inventory/infrastructure"
	"github.com/Briyantama/POS-Stery/services/inventory/infrastructure/postgres"
	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/events"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps is everything the inventory service needs from the process.
type Deps struct {
	Pool         *pgxpool.Pool
	Events       *events.Publisher
	Verifier     authn.Verifier
	RequireStore func(http.Handler) http.Handler
}

// NewUpdateStockHandler builds the stock adjustment handler. Sales and supplier call it in-process
// (Deps needs Pool and Events only).
func NewUpdateStockHandler(d Deps) *commands.UpdateStockHandler {
	return commands.NewUpdateStockHandler(
		postgres.NewStockRepository(d.Pool),
		postgres.NewThresholdRepository(d.Pool),
		infrastructure.NewEventPublisher(d.Events),
	)
}

// Build wires repositories, handlers and the HTTP handler.
func Build(d Deps) (*HTTPHandler, error) {
	stock := postgres.NewStockRepository(d.Pool)
	threshold := postgres.NewThresholdRepository(d.Pool)
	return NewHTTPHandler(
		commands.NewAddItemHandler(stock, threshold),
		NewUpdateStockHandler(d),
		commands.NewSetThresholdHandler(threshold),
		queries.NewGetItemHandler(stock),
		queries.NewListItemsHandler(stock),
		d.Verifier,
		d.RequireStore,
	), nil
}
