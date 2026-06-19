package main

import (
	"context"
	"fmt"
	"log"

	inventoryv1 "github.com/pos-stery/pos-stery/gen/go/pos/inventory/v1"
	"github.com/pos-stery/pos-stery/services/_shared/config"
	"github.com/pos-stery/pos-stery/services/_shared/database"
	sharednats "github.com/pos-stery/pos-stery/services/_shared/nats"
	"github.com/pos-stery/pos-stery/services/_shared/observability"
	"github.com/pos-stery/pos-stery/services/_shared/server"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/application/queries"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/infrastructure"
	"github.com/pos-stery/pos-stery/services/inventory-service/internal/infrastructure/postgres"
	grpcimpl "github.com/pos-stery/pos-stery/services/inventory-service/internal/interfaces/grpc"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("inventory-service: %v", err)
	}
}

func run() error {
	cfg, err := config.Load("inventory-service")
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
	defer logger.Sync()

	// Database pool
	pool, err := database.NewPool(context.Background(), cfg.DB)
	if err != nil {
		return fmt.Errorf("database pool: %w", err)
	}
	defer pool.Close()

	// NATS JetStream
	js, err := sharednats.NewClient(cfg.NATS)
	if err != nil {
		return fmt.Errorf("nats: %w", err)
	}
	if err := sharednats.EnsureStreams(js); err != nil {
		return fmt.Errorf("nats streams: %w", err)
	}

	// Repositories
	stockRepo := postgres.NewStockRepository(pool)
	thresholdRepo := postgres.NewThresholdRepository(pool)

	// Event publisher
	publisher := infrastructure.NewNATSPublisher(js)

	// Command handlers
	addItemHandler := commands.NewAddItemHandler(stockRepo, thresholdRepo)
	updateStockHandler := commands.NewUpdateStockHandler(stockRepo, thresholdRepo, publisher)
	setThresholdHandler := commands.NewSetThresholdHandler(thresholdRepo)

	// Query handlers
	getItemHandler := queries.NewGetItemHandler(stockRepo)
	listItemsHandler := queries.NewListItemsHandler(stockRepo)

	// gRPC server
	grpcSrv := server.New(server.Config{
		Port:               cfg.GRPC.Port,
		ServiceTokenSecret: cfg.ServiceToken,
	}, logger)

	inventoryv1.RegisterInventoryServiceServer(grpcSrv,
		grpcimpl.NewInventoryServiceServer(
			addItemHandler,
			updateStockHandler,
			setThresholdHandler,
			getItemHandler,
			listItemsHandler,
		),
	)

	logger.Sugar().Infof("inventory-service listening on :%d", cfg.GRPC.Port)

	go func() {
		if err := server.ListenAndServe(grpcSrv, cfg.GRPC.Port); err != nil {
			logger.Sugar().Fatalf("grpc serve: %v", err)
		}
	}()

	server.GracefulShutdown(grpcSrv, logger, 30e9)
	return nil
}
