package document

import "errors"

var (
	ErrNilProcessor               = errors.New("document processor is nil")
	ErrInvalidDocumentType        = errors.New("invalid document type")
	ErrProcessorAlreadyRegistered = errors.New("processor already registered")
	ErrProcessorNotImplemented    = errors.New("processor not implemented")
)
