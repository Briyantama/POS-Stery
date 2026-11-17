package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound         = errors.New("not found")
	ErrInvalidArgument  = errors.New("invalid argument")
	ErrUnauthenticated  = errors.New("unauthenticated")
	ErrPermissionDenied = errors.New("permission denied")
)

// ErrStoreRequired is an invalid-argument error with a message the API shows as is.
var ErrStoreRequired = fmt.Errorf("store_id is required for cashier login: %w", ErrInvalidArgument)
