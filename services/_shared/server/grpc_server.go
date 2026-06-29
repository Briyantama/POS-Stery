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
//
// Transport security: plaintext. All gRPC servers listen on the internal pos-net
// Docker bridge (development) or the Kubernetes pod overlay network (production).
// No gRPC port is exposed outside those trusted network boundaries; the only
// external-facing surface is the HTTP API gateway. Application-level service
// identity is enforced by the X-Service-Token HMAC-SHA256 interceptor below.
// If the deployment model changes to expose gRPC externally, replace this call
// with grpc.Creds(credentials.NewTLS(tlsCfg)) and provision per-service certs.
// See docs/security/grpc-transport-security.md.
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
