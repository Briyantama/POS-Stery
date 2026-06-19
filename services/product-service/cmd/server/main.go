package main

import (
	"context"
	"fmt"
	"log"

	productv1 "github.com/pos-stery/pos-stery/gen/go/pos/product/v1"
	"github.com/pos-stery/pos-stery/services/_shared/config"
	"github.com/pos-stery/pos-stery/services/_shared/database"
	"github.com/pos-stery/pos-stery/services/_shared/nats"
	"github.com/pos-stery/pos-stery/services/_shared/observability"
	"github.com/pos-stery/pos-stery/services/_shared/server"
	"github.com/pos-stery/pos-stery/services/product-service/internal/application/commands"
	"github.com/pos-stery/pos-stery/services/product-service/internal/application/queries"
	"github.com/pos-stery/pos-stery/services/product-service/internal/infrastructure/postgres"
	grpcimpl "github.com/pos-stery/pos-stery/services/product-service/internal/interfaces/grpc"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("product-service: %v", err)
	}
}

func run() error {
	cfg, err := config.Load("product-service")
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

	// NATS JetStream (product-service may publish low-stock events in future)
	js, err := nats.NewClient(cfg.NATS)
	if err != nil {
		return fmt.Errorf("nats: %w", err)
	}
	if err := nats.EnsureStreams(js); err != nil {
		return fmt.Errorf("nats streams: %w", err)
	}
	_ = js // retained for future event publishing

	// Wire repositories
	productRepo := postgres.NewProductRepository(pool)

	// Wire application handlers
	createHandler := commands.NewCreateProductHandler(productRepo)
	updateHandler := commands.NewUpdateProductHandler(productRepo)
	searchHandler := queries.NewSearchProductsHandler(productRepo)

	// gRPC server
	grpcSrv := server.New(server.Config{
		Port:               cfg.GRPC.Port,
		ServiceTokenSecret: cfg.ServiceToken,
	}, logger)

	productv1.RegisterProductServiceServer(
		grpcSrv,
		grpcimpl.NewProductServiceServer(createHandler, updateHandler, searchHandler),
	)

	logger.Sugar().Infof("product-service listening on :%d", cfg.GRPC.Port)

	go func() {
		if err := server.ListenAndServe(grpcSrv, cfg.GRPC.Port); err != nil {
			logger.Sugar().Fatalf("grpc serve: %v", err)
		}
	}()

	server.GracefulShutdown(grpcSrv, logger, 30e9)
	return nil
}
