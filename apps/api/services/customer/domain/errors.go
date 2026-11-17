package domain

import "errors"

var (
	ErrNotFound        = errors.New("not found")
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrAlreadyAwarded: loyalty points were already awarded for this sale.
	ErrAlreadyAwarded = errors.New("points already awarded for sale")
)
