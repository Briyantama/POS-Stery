package errors

import (
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrNotFound              = errors.New("not found")
	ErrAlreadyExists         = errors.New("already exists")
	ErrInvalidArgument       = errors.New("invalid argument")
	ErrUnauthenticated       = errors.New("unauthenticated")
	ErrPermissionDenied      = errors.New("permission denied")
	ErrStockWouldGoNegative  = errors.New("stock would go negative")
	ErrCrossTenantAccess     = errors.New("cross-tenant access denied")
	ErrStoreNotInTenant      = errors.New("store does not belong to tenant")
)

// ToGRPCStatus maps domain errors to gRPC status codes.
// Unmapped errors become Internal.
func ToGRPCStatus(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrUnauthenticated):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, ErrPermissionDenied),
		errors.Is(err, ErrCrossTenantAccess),
		errors.Is(err, ErrStoreNotInTenant):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, ErrStockWouldGoNegative):
		return status.Error(codes.FailedPrecondition, err.Error())
	default:
		return status.Error(codes.Internal, fmt.Sprintf("internal: %v", err))
	}
}

// Wrap adds context to an error while preserving the original for errors.Is.
func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}
