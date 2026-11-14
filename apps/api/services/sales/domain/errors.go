package domain

import "errors"

// Sentinel errors the HTTP layer maps to status codes.
var (
	ErrNotFound             = errors.New("not found")
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrStockWouldGoNegative = errors.New("stock would go negative")
)
