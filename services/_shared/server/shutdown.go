package server

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// GracefulShutdown waits for SIGINT/SIGTERM and shuts down the gRPC server
// within the given timeout. Call this as the last line of main().
func GracefulShutdown(srv *grpc.Server, log *zap.Logger, timeout time.Duration) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down", zap.Duration("timeout", timeout))

	stopped := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(stopped)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	select {
	case <-stopped:
		log.Info("shutdown complete")
	case <-ctx.Done():
		log.Warn("shutdown timeout exceeded, forcing stop")
		srv.Stop()
	}
}
