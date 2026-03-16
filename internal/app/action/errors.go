package action

import "errors"

var (
	ErrActionNotFound        = errors.New("collection action not found")
	ErrInvalidActionType     = errors.New("invalid action type")
	ErrActionNotSupported    = errors.New("action is not supported for this document type")
	ErrActionDisabled        = errors.New("action is disabled")
	ErrInvalidActionParams   = errors.New("invalid action params")
	ErrInvalidActionSpec     = errors.New("invalid action spec")
	ErrEmptySnapshot         = errors.New("no documents available for action snapshot")
	ErrMinDocumentsRequired  = errors.New("minimum documents requirement is not met")
	ErrInvalidDocumentStatus = errors.New("invalid document status filter")
	ErrInvalidDocumentIDs    = errors.New("invalid document ids")
	ErrSnapshotDocNotFound   = errors.New("some selected documents are not available for snapshot")
	ErrSnapshotDocStatus     = errors.New("selected documents include unsupported status")
	ErrSpecNotFound          = errors.New("action spec not found")
)
