package collection

import "errors"

var (
	ErrInvalidStatusTransition = errors.New("invalid collection status transition")
	ErrCollectionNotActive     = errors.New("collection not active")
	ErrCollectionNotFound      = errors.New("collection not found")
	ErrCollectionFrozen        = errors.New("collection is frozen")
	ErrCollectionAlreadyFrozen = errors.New("collection is already frozen")
	ErrCollectionBusy          = errors.New("collection is busy")
	ErrInvalidNodeType         = errors.New("invalid collection node type")
	ErrInvalidDocumentType     = errors.New("invalid collection document type")
	ErrInvalidCollectionName   = errors.New("invalid collection name")
	ErrCollectionNameConflict  = errors.New("collection name already exists")
)
