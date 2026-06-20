package main

import (
	"context"
	"fmt"
	"log"

	supplierv1 "github.com/pos-stery/pos-stery/gen/go/pos/supplier/v1"
	"github.com/pos-stery/pos-stery/services/_shared/config"
	"github.com/pos-stery/pos-stery/services/_shared/database"
	sharednats "github.com/pos-stery/pos-stery/services/_shared/nats"
	"github.com/pos-stery/pos-stery/services/_shared/observability"
	"github.com/pos-stery/pos-stery/services/_shared/server"
	"github.com/pos-stery/pos-stery/services/supplier-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/supplier-service/internal/infrastructure"
	"github.com/pos-stery/pos-stery/services/supplier-service/internal/infrastructure/grpc_clients"
	"github.com/pos-stery/pos-stery/services/supplier-service/internal/infrastructure/postgres"
	grpcimpl "github.com/pos-stery/pos-stery/services/supplier-service/internal/interfaces/grpc"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("supplier-service: %v", err)
	}
}

func run() error {
	cfg, err := config.Load("supplier-service")
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
	defer func() {
		if err := logger.Sync(); err != nil {
			log.Printf("failed to sync logger: %v", err)
		}
	}()

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
	supplierRepo := postgres.NewSupplierRepository(pool)
	orderRepo := postgres.NewPurchaseOrderRepository(pool)

	// Inventory gRPC client (satisfies application.InventoryPort)
	inventoryClient, err := grpc_clients.NewInventoryClient()
	if err != nil {
		return fmt.Errorf("inventory client: %w", err)
	}

	// NATS event publisher (satisfies application.EventPublisher)
	natsPublisher := infrastructure.NewNATSPublisher(js)

	// Command handlers
	addSupplierHandler := commands.NewAddSupplierHandler(supplierRepo)
	createPOHandler := commands.NewCreatePurchaseOrderHandler(orderRepo, supplierRepo)
	receiveStockHandler := commands.NewReceiveStockHandler(orderRepo, inventoryClient, natsPublisher)

	// gRPC server
	grpcSrv := server.New(server.Config{
		Port:               cfg.GRPC.Port,
		ServiceTokenSecret: cfg.ServiceToken,
	}, logger)

	supplierv1.RegisterSupplierServiceServer(grpcSrv,
		grpcimpl.NewSupplierServiceServer(
			addSupplierHandler,
			createPOHandler,
			receiveStockHandler,
			supplierRepo,
			orderRepo,
		),
	)

	logger.Sugar().Infof("supplier-service listening on :%d", cfg.GRPC.Port)

	go func() {
		if err := server.ListenAndServe(grpcSrv, cfg.GRPC.Port); err != nil {
			logger.Sugar().Fatalf("grpc serve: %v", err)
		}
	}()

	server.GracefulShutdown(grpcSrv, logger, 30e9)
	return nil
}
