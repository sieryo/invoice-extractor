package collection

import "errors"

var (
	ErrInvalidStatusTransition = errors.New("invalid collection status transition")
)
