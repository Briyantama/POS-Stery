package sales

import (
	"net/http"

	invcommands "github.com/Briyantama/POS-Stery/services/inventory/application/commands"
	"github.com/Briyantama/POS-Stery/services/sales/application/commands"
	"github.com/Briyantama/POS-Stery/services/sales/infrastructure"
	"github.com/Briyantama/POS-Stery/services/sales/infrastructure/postgres"
	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/events"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps are the collaborators the sales service needs from cmd/api.
type Deps struct {
	Pool         *pgxpool.Pool
	Events       *events.Publisher
	Verifier     authn.Verifier
	RequireStore func(http.Handler) http.Handler
	// Stock is inventory's UpdateStock use case (built by the inventory package).
	Stock *invcommands.UpdateStockHandler
}

// Build wires repository, saga handler and HTTP handler.
func Build(d Deps) (*HTTPHandler, error) {
	repo := postgres.NewSaleRepository(d.Pool)
	create := commands.NewCreateSaleHandler(
		repo,
		infrastructure.NewInventoryPort(d.Stock),
		infrastructure.NewEventPublisher(d.Events),
	)
	return NewHTTPHandler(create, repo, d.Verifier, d.RequireStore), nil
}
