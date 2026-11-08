package product

import (
	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Briyantama/POS-Stery/services/product/application/commands"
	"github.com/Briyantama/POS-Stery/services/product/application/queries"
	"github.com/Briyantama/POS-Stery/services/product/infrastructure/postgres"
)

// Deps are the collaborators the product service needs.
type Deps struct {
	Pool     *pgxpool.Pool
	Verifier authn.Verifier
}

// Build wires repository, handlers and the HTTP layer.
func Build(d Deps) (*HTTPHandler, error) {
	repo := postgres.NewProductRepository(d.Pool)
	search := queries.NewSearchProductsHandler(repo)
	return NewHTTPHandler(commands.NewCreateProductHandler(repo), commands.NewUpdateProductHandler(repo), search, search, d.Verifier), nil
}
