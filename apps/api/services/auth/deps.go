package auth

import (
	"net/http"

	"github.com/Briyantama/POS-Stery/internal/config"
	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Deps is everything the auth service needs from the process wiring.
type Deps struct {
	Config   config.Config
	Pool     *pgxpool.Pool
	Redis    *redis.Client
	Verifier authn.Verifier
	// LoginLimit wraps the public login route (rate limit); nil means no limit.
	LoginLimit func(http.Handler) http.Handler
}
