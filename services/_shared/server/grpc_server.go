package server

import (
	"fmt"
	"net"

	"github.com/pos-stery/pos-stery/services/_shared/middleware"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Config struct {
	Port               int
	ServiceTokenSecret string
}

// New creates a gRPC server with the standard interceptor chain.
func New(cfg Config, log *zap.Logger) *grpc.Server {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(
			middleware.UnaryInterceptorChain(log, cfg.ServiceTokenSecret),
		),
	)
	reflection.Register(srv) // enables grpcurl in development
	return srv
}

// ListenAndServe starts the gRPC server on the configured port.
func ListenAndServe(srv *grpc.Server, port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("listen on :%d: %w", port, err)
	}
	return srv.Serve(lis)
}
