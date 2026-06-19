package main

import (
	"context"
	"fmt"
	"log"
	"os"

	authv1 "github.com/pos-stery/pos-stery/gen/go/pos/auth/v1"
	"github.com/pos-stery/pos-stery/services/_shared/config"
	"github.com/pos-stery/pos-stery/services/_shared/nats"
	"github.com/pos-stery/pos-stery/services/_shared/observability"
	"github.com/pos-stery/pos-stery/services/_shared/redis"
	"github.com/pos-stery/pos-stery/services/_shared/server"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/application/commands"
	grpcimpl "github.com/pos-stery/pos-stery/services/auth-service/internal/interfaces/grpc"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/infrastructure/jwt"
	"github.com/pos-stery/pos-stery/services/auth-service/internal/infrastructure/redisstore"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("auth-service: %v", err)
	}
}

func run() error {
	cfg, err := config.Load("auth-service")
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

	// Redis (token blacklist)
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		return fmt.Errorf("redis: %w", err)
	}

	// NATS (event publishing)
	js, err := nats.NewClient(cfg.NATS)
	if err != nil {
		return fmt.Errorf("nats: %w", err)
	}
	if err := nats.EnsureStreams(js); err != nil {
		return fmt.Errorf("nats streams: %w", err)
	}

	// JWT signer
	privateKeyPath := os.Getenv("JWT_RS256_PRIVATE_KEY_PATH")
	publicKeyPath := os.Getenv("JWT_RS256_PUBLIC_KEY_PATH")
	if privateKeyPath == "" || publicKeyPath == "" {
		return fmt.Errorf("JWT_RS256_PRIVATE_KEY_PATH and JWT_RS256_PUBLIC_KEY_PATH must be set")
	}
	signer, err := jwt.NewRSASigner(privateKeyPath, publicKeyPath)
	if err != nil {
		return fmt.Errorf("jwt signer: %w", err)
	}

	blacklist := redisstore.NewTokenBlacklist(redisClient)
	signer.SetBlacklistStore(blacklist)

	// Database pool — auth service needs it for user/tenant lookups
	// TODO: wire postgres user/tenant repositories when implemented
	_ = context.Background() // placeholder until repos are wired

	// Command handlers
	// TODO: replace nil with real repo implementations
	loginHandler := commands.NewLoginHandler(nil, nil, signer)
	validateHandler := commands.NewValidateHandler(signer)

	// gRPC server
	grpcSrv := server.New(server.Config{
		Port:               cfg.GRPC.Port,
		ServiceTokenSecret: cfg.ServiceToken,
	}, logger)

	authv1.RegisterAuthServiceServer(grpcSrv, grpcimpl.NewAuthServiceServer(loginHandler, validateHandler))

	logger.Sugar().Infof("auth-service listening on :%d", cfg.GRPC.Port)

	go func() {
		if err := server.ListenAndServe(grpcSrv, cfg.GRPC.Port); err != nil {
			logger.Sugar().Fatalf("grpc serve: %v", err)
		}
	}()

	server.GracefulShutdown(grpcSrv, logger, 30e9)
	return nil
}
