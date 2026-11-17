package customer

import (
	"net/http"

	"github.com/Briyantama/POS-Stery/services/customer/application/commands"
	"github.com/Briyantama/POS-Stery/services/customer/infrastructure"
	"github.com/Briyantama/POS-Stery/services/customer/infrastructure/postgres"
	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/events"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Deps are the collaborators the customer service needs.
type Deps struct {
	Pool     *pgxpool.Pool
	Events   *events.Publisher
	Verifier authn.Verifier
	// RequireStore validates the store of the caller (auth.RequireStore).
	RequireStore func(http.Handler) http.Handler
}

// Build wires repositories and handlers into the HTTP handler.
func Build(d Deps) (*HTTPHandler, error) {
	customers := postgres.NewCustomerRepository(d.Pool)
	loyalty := postgres.NewLoyaltyRepository(d.Pool)
	return NewHTTPHandler(
		commands.NewCreateCustomerHandler(customers, infrastructure.NewEventPublisher(d.Events)),
		commands.NewAddLoyaltyPointsHandler(customers, loyalty),
		customers,
		loyalty,
		d.Verifier,
		d.RequireStore,
	), nil
}
