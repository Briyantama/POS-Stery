package main

import (
	"context"
	"fmt"
	"log"
	"os"

	salesv1 "github.com/pos-stery/pos-stery/gen/go/pos/sales/v1"
	"github.com/pos-stery/pos-stery/services/_shared/config"
	"github.com/pos-stery/pos-stery/services/_shared/database"
	sharednats "github.com/pos-stery/pos-stery/services/_shared/nats"
	"github.com/pos-stery/pos-stery/services/_shared/observability"
	"github.com/pos-stery/pos-stery/services/_shared/server"
	"github.com/pos-stery/pos-stery/services/sales-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/sales-service/internal/infrastructure"
	grpcclient "github.com/pos-stery/pos-stery/services/sales-service/internal/infrastructure/grpc_clients"
	"github.com/pos-stery/pos-stery/services/sales-service/internal/infrastructure/postgres"
	grpcimpl "github.com/pos-stery/pos-stery/services/sales-service/internal/interfaces/grpc"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("sales-service: %v", err)
	}
}

func run() error {
	cfg, err := config.Load("sales-service")
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

	// ── Database ─────────────────────────────────────────────────────────────
	pool, err := database.NewPool(context.Background(), cfg.DB)
	if err != nil {
		return fmt.Errorf("database pool: %w", err)
	}

	// ── NATS ─────────────────────────────────────────────────────────────────
	js, err := sharednats.NewClient(cfg.NATS)
	if err != nil {
		return fmt.Errorf("nats: %w", err)
	}
	if err := sharednats.EnsureStreams(js); err != nil {
		return fmt.Errorf("nats streams: %w", err)
	}

	// ── Inventory gRPC client ────────────────────────────────────────────────
	inventoryURL := os.Getenv("INVENTORY_SERVICE_URL")
	inventoryClient, err := grpcclient.NewInventoryClient(grpcclient.InventoryClientConfig{
		ServiceURL:   inventoryURL,
		ServiceToken: cfg.ServiceToken,
	})
	if err != nil {
		return fmt.Errorf("inventory client: %w", err)
	}

	// ── Infrastructure wiring ────────────────────────────────────────────────
	saleRepo := postgres.NewSaleRepository(pool)
	publisher := infrastructure.NewNATSPublisher(js)

	// ── Command handlers ─────────────────────────────────────────────────────
	createSaleHandler := commands.NewCreateSaleHandler(saleRepo, inventoryClient, publisher, logger)

	// ── gRPC server ──────────────────────────────────────────────────────────
	grpcSrv := server.New(server.Config{
		Port:               cfg.GRPC.Port,
		ServiceTokenSecret: cfg.ServiceToken,
	}, logger)

	salesv1.RegisterSalesServiceServer(grpcSrv, grpcimpl.NewSalesServiceServer(createSaleHandler, saleRepo))

	logger.Info("sales-service starting", zap.Int("port", cfg.GRPC.Port))

	go func() {
		if err := server.ListenAndServe(grpcSrv, cfg.GRPC.Port); err != nil {
			logger.Fatal("grpc serve failed", zap.Error(err))
		}
	}()

	server.GracefulShutdown(grpcSrv, logger, 30e9)
	return nil
}
