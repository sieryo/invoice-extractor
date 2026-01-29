package collection

import "errors"

var (
	ErrInvalidStatusTransition = errors.New("invalid collection status transition")
	ErrCollectionNotActive     = errors.New("collection not active")
)
