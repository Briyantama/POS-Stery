package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	sharedmiddleware "github.com/pos-stery/pos-stery/services/_shared/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const defaultCallTimeout = 10 * time.Second

// callCtx wraps ctx with a per-call deadline so stalled backends don't
// hold HTTP connections open for the full server WriteTimeout.
func callCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, defaultCallTimeout)
}

// Dial opens a plaintext gRPC connection to addr (bare host:port).
//
// Plaintext is safe here: all backend services are reachable only within the
// pos-net Docker bridge / Kubernetes overlay network. No gRPC endpoint is
// accessible from outside those boundaries. Service identity on every call is
// verified by X-Service-Token HMAC-SHA256 (see withServiceToken).
// See docs/security/grpc-transport-security.md.
func Dial(addr string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return conn, nil
}

// withServiceToken attaches an HMAC-SHA256 service token to outgoing gRPC metadata.
// If secret is empty, the context is returned unchanged (skip in dev only via ENVIRONMENT check upstream).
func withServiceToken(ctx context.Context, secret string) context.Context {
	if secret == "" {
		return ctx
	}
	token := sharedmiddleware.ComputeServiceToken(secret)
	return metadata.AppendToOutgoingContext(ctx, "x-service-token", token)
}

// GrpcError wraps a gRPC status error with an HTTP status code for handler use.
type GrpcError struct {
	HTTPStatus int
	Message    string
}

func (e *GrpcError) Error() string {
	return fmt.Sprintf("grpc error %d: %s", e.HTTPStatus, e.Message)
}

// wrapGrpcError converts a gRPC status error into a *GrpcError with the
// corresponding HTTP status code. Non-gRPC errors map to 500.
func wrapGrpcError(err error) *GrpcError {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return &GrpcError{HTTPStatus: http.StatusInternalServerError, Message: err.Error()}
	}
	return &GrpcError{
		HTTPStatus: grpcToHTTP(st.Code()),
		Message:    st.Message(),
	}
}

// grpcToHTTP maps a gRPC status code to its closest HTTP equivalent.
func grpcToHTTP(code codes.Code) int {
	switch code {
	case codes.NotFound:
		return http.StatusNotFound
	case codes.InvalidArgument, codes.OutOfRange:
		return http.StatusBadRequest
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.FailedPrecondition:
		return http.StatusUnprocessableEntity
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
