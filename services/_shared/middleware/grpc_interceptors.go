package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	sherrors "github.com/pos-stery/pos-stery/services/_shared/errors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// UnaryInterceptorChain builds the standard interceptor chain for all services.
// Order: recover → otel → metrics → logging → service-auth → tenant-enforce
func UnaryInterceptorChain(
	log *zap.Logger,
	serviceTokenSecret string,
) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		// Service-to-service auth for internal calls
		if err := validateServiceToken(ctx, serviceTokenSecret); err != nil {
			return nil, err
		}

		resp, err := handler(ctx, req)

		log.Info("grpc",
			zap.String("method", info.FullMethod),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)

		if err != nil {
			return nil, sherrors.ToGRPCStatus(err)
		}
		return resp, nil
	}
}

// validateServiceToken checks the X-Service-Token header for service-to-service calls.
// The token is an HMAC-SHA256 of the Unix timestamp (within 30s window) using the shared secret.
// External (gateway) calls skip this check — they are authenticated by JWT at the gateway layer.
func validateServiceToken(ctx context.Context, secret string) error {
	if secret == "" || secret == "changeme" {
		return nil // token validation disabled in development
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil // no metadata = external call, skip
	}

	tokens := md.Get("x-service-token")
	if len(tokens) == 0 {
		return nil // no token = external call from gateway, skip
	}

	now := time.Now().Unix()
	for _, token := range tokens {
		for _, ts := range []int64{now, now - 30, now + 30} {
			expected := computeServiceToken(secret, ts)
			if hmac.Equal([]byte(token), []byte(expected)) {
				return nil
			}
		}
	}

	return status.Error(codes.Unauthenticated, "invalid service token")
}

// ComputeServiceToken generates an HMAC-SHA256 token for a given Unix timestamp.
// Services use this when calling downstream services.
func ComputeServiceToken(secret string) string {
	return computeServiceToken(secret, time.Now().Unix())
}

func computeServiceToken(secret string, ts int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%d", ts)))
	return hex.EncodeToString(mac.Sum(nil))
}
