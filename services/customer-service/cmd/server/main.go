package main

import (
	"context"
	"fmt"
	"log"

	customerv1 "github.com/pos-stery/pos-stery/gen/go/pos/customer/v1"
	"github.com/pos-stery/pos-stery/services/_shared/config"
	"github.com/pos-stery/pos-stery/services/_shared/database"
	"github.com/pos-stery/pos-stery/services/_shared/nats"
	"github.com/pos-stery/pos-stery/services/_shared/observability"
	"github.com/pos-stery/pos-stery/services/_shared/server"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/application/queries"
	"github.com/pos-stery/pos-stery/services/customer-service/internal/infrastructure/postgres"
	grpcimpl "github.com/pos-stery/pos-stery/services/customer-service/internal/interfaces/grpc"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("customer-service: %v", err)
	}
}

func run() error {
	cfg, err := config.Load("customer-service")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger, err := observability.NewLogger(
		cfg.Observability.ServiceName,
		cfg.Observability.LogLevel,
		cfg.Observability.Environment,
	)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.Sync() //nolint:errcheck

	// Database pool
	pool, err := database.NewPool(context.Background(), cfg.DB)
	if err != nil {
		return fmt.Errorf("database pool: %w", err)
	}
	defer pool.Close()

	// NATS JetStream — used to publish Customer.New events
	js, err := nats.NewClient(cfg.NATS)
	if err != nil {
		return fmt.Errorf("nats: %w", err)
	}
	if err := nats.EnsureStreams(js); err != nil {
		return fmt.Errorf("nats streams: %w", err)
	}

	// Wire repositories
	customerRepo := postgres.NewCustomerRepository(pool)
	loyaltyRepo := postgres.NewLoyaltyRepository(pool)

	// Wire application handlers
	createHandler := commands.NewCreateCustomerHandler(customerRepo, js)
	loyaltyHandler := commands.NewAddLoyaltyPointsHandler(customerRepo, loyaltyRepo)
	queryHandler := queries.NewCustomerQueryHandler(customerRepo, loyaltyRepo)

	// gRPC server
	grpcSrv := server.New(server.Config{
		Port:               cfg.GRPC.Port,
		ServiceTokenSecret: cfg.ServiceToken,
	}, logger)

	customerv1.RegisterCustomerServiceServer(
		grpcSrv,
		grpcimpl.NewCustomerServiceServer(createHandler, loyaltyHandler, queryHandler),
	)

	logger.Sugar().Infof("customer-service listening on :%d", cfg.GRPC.Port)

	go func() {
		if err := server.ListenAndServe(grpcSrv, cfg.GRPC.Port); err != nil {
			logger.Sugar().Fatalf("grpc serve: %v", err)
		}
	}()

	server.GracefulShutdown(grpcSrv, logger, 30e9)
	return nil
}
