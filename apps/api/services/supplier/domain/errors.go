package domain

import "errors"

// Sentinel errors returned by the supplier service; the HTTP layer maps them to status codes.
var (
	ErrNotFound        = errors.New("not found")
	ErrAlreadyExists   = errors.New("already exists")
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrConflict marks a state conflict, e.g. receiving against an already received order.
	ErrConflict = errors.New("conflict")
)
