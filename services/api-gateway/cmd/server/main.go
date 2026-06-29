package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	goredis "github.com/redis/go-redis/v9"
	"github.com/pos-stery/pos-stery/services/api-gateway/internal/clients"
	"github.com/pos-stery/pos-stery/services/api-gateway/internal/config"
	"github.com/pos-stery/pos-stery/services/api-gateway/internal/handlers"
	gatewayjwt "github.com/pos-stery/pos-stery/services/api-gateway/internal/jwt"
	"github.com/pos-stery/pos-stery/services/api-gateway/internal/middleware"
	sharedobs "github.com/pos-stery/pos-stery/services/_shared/observability"
	sharedredis "github.com/pos-stery/pos-stery/services/_shared/redis"
	"go.uber.org/zap"
)

// redisPinger adapts *redis.Client to handlers.RedisPinger.
type redisPinger struct{ c *goredis.Client }

func (p redisPinger) Ping(ctx context.Context) error { return p.c.Ping(ctx).Err() }

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// 1. Config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// 2. Logger
	log, err := sharedobs.NewLogger(
		"api-gateway",
		cfg.Shared.Observability.LogLevel,
		cfg.Shared.Observability.Environment,
	)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer log.Sync() //nolint:errcheck

	// 3. Redis
	redisClient, err := sharedredis.NewClient(cfg.Shared.Redis)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}

	// 4. RSA public key
	pubKeyPEM, err := os.ReadFile(cfg.JWTPublicKeyPath)
	if err != nil {
		return fmt.Errorf("read jwt public key %s: %w", cfg.JWTPublicKeyPath, err)
	}

	// 5. JWT verifier + blacklist
	verifier, err := gatewayjwt.NewVerifier(pubKeyPEM)
	if err != nil {
		return fmt.Errorf("init jwt verifier: %w", err)
	}
	blacklist := gatewayjwt.NewBlacklist(redisClient)

	// 6. Dial all backend services
	authConn, err := clients.Dial(cfg.AuthServiceURL)
	if err != nil {
		return fmt.Errorf("dial auth-service: %w", err)
	}
	defer authConn.Close()

	productConn, err := clients.Dial(cfg.ProductServiceURL)
	if err != nil {
		return fmt.Errorf("dial product-service: %w", err)
	}
	defer productConn.Close()

	inventoryConn, err := clients.Dial(cfg.InventoryServiceURL)
	if err != nil {
		return fmt.Errorf("dial inventory-service: %w", err)
	}
	defer inventoryConn.Close()

	salesConn, err := clients.Dial(cfg.SalesServiceURL)
	if err != nil {
		return fmt.Errorf("dial sales-service: %w", err)
	}
	defer salesConn.Close()

	supplierConn, err := clients.Dial(cfg.SupplierServiceURL)
	if err != nil {
		return fmt.Errorf("dial supplier-service: %w", err)
	}
	defer supplierConn.Close()

	customerConn, err := clients.Dial(cfg.CustomerServiceURL)
	if err != nil {
		return fmt.Errorf("dial customer-service: %w", err)
	}
	defer customerConn.Close()

	token := cfg.Shared.ServiceToken

	// 7. Domain clients
	authClient := clients.NewAuthClient(authConn, token)
	productClient := clients.NewProductClient(productConn, token)
	inventoryClient := clients.NewInventoryClient(inventoryConn, token)
	salesClient := clients.NewSalesClient(salesConn, token)
	supplierClient := clients.NewSupplierClient(supplierConn, token)
	customerClient := clients.NewCustomerClient(customerConn, token)

	// 8. Handlers
	authH := handlers.NewAuthHandlers(authClient)
	productH := handlers.NewProductHandlers(productClient)
	inventoryH := handlers.NewInventoryHandlers(inventoryClient)
	salesH := handlers.NewSalesHandlers(salesClient)
	supplierH := handlers.NewSupplierHandlers(supplierClient)
	customerH := handlers.NewCustomerHandlers(customerClient)

	// 9. Router
	r := chi.NewRouter()
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RequestID)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, 4<<20) // 4 MB
			next.ServeHTTP(w, r)
		})
	})
	r.Use(httprate.Limit(300, time.Minute))

	// Public endpoints (login has a tighter per-IP rate limit)
	r.Get("/healthz", handlers.Health)
	r.Get("/readyz", handlers.Ready(redisPinger{redisClient}))
	r.With(httprate.LimitByIP(20, time.Minute)).Post("/api/login", authH.Login)

	// JWT middleware deps
	jwtDeps := middleware.JWTDeps{
		Verifier:  verifier,
		Blacklist: blacklist,
	}

	// Authenticated routes: JWT → Tenant → Store
	r.Group(func(r chi.Router) {
		r.Use(middleware.JWT(jwtDeps))
		r.Use(middleware.Tenant())
		r.Use(middleware.Store(middleware.StoreDeps{Validator: authClient}))

		r.With(middleware.WithRole("admin", "cashier")).Post("/api/logout", authH.Logout)

		r.With(middleware.WithRole("admin", "cashier")).Get("/api/products", productH.List)
		r.With(middleware.WithRole("admin")).Post("/api/products", productH.Create)
		r.With(middleware.WithRole("admin")).Put("/api/products/{id}", productH.Update)

		r.With(middleware.WithRole("admin", "stock_manager")).Get("/api/inventory", inventoryH.List)
		r.With(middleware.WithRole("admin", "stock_manager")).Post("/api/inventory", inventoryH.Add)

		r.With(middleware.WithRole("cashier")).Post("/api/sales", salesH.Create)
		r.With(middleware.WithRole("admin")).Get("/api/sales/report", salesH.Report)

		r.With(middleware.WithRole("admin", "stock_manager")).Get("/api/suppliers", supplierH.List)
		r.With(middleware.WithRole("admin")).Post("/api/suppliers", supplierH.Add)
		r.With(middleware.WithRole("admin", "stock_manager")).Post("/api/purchase-orders", supplierH.CreatePurchaseOrder)

		r.With(middleware.WithRole("admin", "cashier")).Get("/api/customers", customerH.List)
		r.With(middleware.WithRole("admin", "cashier")).Post("/api/customers", customerH.Create)
	})

	// 10. HTTP server with graceful shutdown
	addr := ":" + cfg.HTTPPort
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Info("api-gateway listening", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("listen", zap.Error(err))
		}
	}()

	<-quit
	log.Info("shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return srv.Shutdown(ctx)
}
