package action

import "errors"

var (
	ErrActionNotFound        = errors.New("collection action not found")
	ErrInvalidActionType     = errors.New("invalid action type")
	ErrEmptySnapshot         = errors.New("no documents available for action snapshot")
	ErrInvalidDocumentStatus = errors.New("invalid document status filter")
	ErrInvalidDocumentIDs    = errors.New("invalid document ids")
	ErrSnapshotDocNotFound   = errors.New("some selected documents are not available for snapshot")
	ErrSnapshotDocStatus     = errors.New("selected documents include unsupported status")
	ErrSpecNotFound          = errors.New("action spec not found")
)
