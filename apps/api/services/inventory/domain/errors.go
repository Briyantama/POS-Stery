package domain

import "errors"

// Sentinel errors returned by the inventory service; the HTTP layer maps them to status codes.
var (
	ErrNotFound             = errors.New("not found")
	ErrAlreadyExists        = errors.New("already exists")
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrStockWouldGoNegative = errors.New("stock would go negative")
)
