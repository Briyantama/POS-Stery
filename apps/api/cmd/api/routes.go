package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Briyantama/stery-kit/go/authn"
	"github.com/Briyantama/stery-kit/go/events"
	"github.com/Briyantama/stery-kit/go/health"
	"github.com/Briyantama/stery-kit/go/ratelimit"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Briyantama/POS-Stery/internal/config"
	"github.com/Briyantama/POS-Stery/services/auth"
	"github.com/Briyantama/POS-Stery/services/customer"
	"github.com/Briyantama/POS-Stery/services/inventory"
	"github.com/Briyantama/POS-Stery/services/product"
	"github.com/Briyantama/POS-Stery/services/sales"
	"github.com/Briyantama/POS-Stery/services/supplier"
)

const loginLimit = 20 // login attempts per minute per client IP

type registrar = interface{ Register(*http.ServeMux) }

// build constructs every service's HTTP handler. Sales and supplier book stock through inventory's
// use case in-process.
func build(cfg config.Config, pool *pgxpool.Pool, rdb *redis.Client) ([]registrar, error) {
	pemBytes, err := os.ReadFile(cfg.JWTPublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read jwt public key: %w", err)
	}
	pub, err := jwt.ParseRSAPublicKeyFromPEM(pemBytes)
	if err != nil {
		return nil, fmt.Errorf("parse jwt public key: %w", err)
	}
	verifier := authn.NewRSAVerifier(pub, authn.RedisRevocation(rdb, "pos:token:blacklist:"))
	publisher := events.NewPublisher(rdb, eventStream)

	authDeps := auth.Deps{
		Config: cfg, Pool: pool, Redis: rdb, Verifier: verifier,
		LoginLimit: ratelimit.Middleware(rdb, "pos:ratelimit:login:", loginLimit, time.Minute, ratelimit.ClientIP),
	}
	requireStore := auth.RequireStore(authDeps)
	invDeps := inventory.Deps{Pool: pool, Events: publisher, Verifier: verifier, RequireStore: requireStore}
	stock := inventory.NewUpdateStockHandler(invDeps)

	var handlers []registrar
	for _, b := range []func() (registrar, error){
		func() (registrar, error) { return auth.Build(authDeps) },
		func() (registrar, error) { return product.Build(product.Deps{Pool: pool, Verifier: verifier}) },
		func() (registrar, error) { return inventory.Build(invDeps) },
		func() (registrar, error) {
			return sales.Build(sales.Deps{Pool: pool, Events: publisher, Verifier: verifier, RequireStore: requireStore, Stock: stock})
		},
		func() (registrar, error) {
			return supplier.Build(supplier.Deps{Pool: pool, Events: publisher, Verifier: verifier, RequireStore: requireStore})
		},
		func() (registrar, error) {
			return customer.Build(customer.Deps{Pool: pool, Events: publisher, Verifier: verifier, RequireStore: requireStore})
		},
	} {
		h, err := b()
		if err != nil {
			return nil, err
		}
		handlers = append(handlers, h)
	}
	return handlers, nil
}

// newMux registers liveness and every service's routes.
func newMux(handlers ...registrar) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", health.Handler())
	for _, h := range handlers {
		h.Register(mux)
	}
	return mux
}
