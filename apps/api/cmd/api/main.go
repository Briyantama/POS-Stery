// Command api runs the POS API: every service on one net/http server. Configuration comes from
// environment variables (see internal/config); run `go run ./cmd/migrate up` first to apply the
// database migrations.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Briyantama/stery-kit/go/httpx"
	"github.com/Briyantama/stery-kit/go/ratelimit"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/Briyantama/POS-Stery/internal/config"
)

const (
	eventStream       = "pos:events"
	globalLimit       = 300 // requests per minute per client IP
	connectTimeout    = 10 * time.Second
	shutdownTimeout   = 30 * time.Second
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 60 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Getenv); err != nil {
		slog.Error("api exited with error", "error", err)
		stop()
		os.Exit(1)
	}
}

// run serves HTTP until ctx is cancelled, then shuts down gracefully.
func run(ctx context.Context, getenv func(string) string) error {
	cfg, err := config.Load(getenv)
	if err != nil {
		return err
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return fmt.Errorf("LOG_LEVEL: %w", err)
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})))

	pool, err := connectPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	rdb, err := connectRedis(ctx, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisTLS)
	if err != nil {
		return err
	}
	defer rdb.Close()

	handlers, err := build(cfg, pool, rdb)
	if err != nil {
		return err
	}
	global := ratelimit.Middleware(rdb, "pos:ratelimit:global:", globalLimit, time.Minute, ratelimit.ClientIP)
	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           httpx.TrustedProxies(cfg.TrustedProxies)(httpx.RequestID(global(newMux(handlers...)))),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
	errc := make(chan error, 1)
	go func() {
		slog.Info("api listening", "addr", srv.Addr)
		errc <- srv.ListenAndServe()
	}()

	select {
	case err := <-errc:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}
	sctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(sctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

func connectPostgres(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	return pool, nil
}

// connectRedis opens the Dragonfly/Redis client; useTLS keeps certificate verification on.
func connectRedis(ctx context.Context, addr, password string, useTLS bool) (*redis.Client, error) {
	opts := &redis.Options{Addr: addr, Password: password}
	if useTLS {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	rdb := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("connect to redis: %w", err)
	}
	return rdb, nil
}
