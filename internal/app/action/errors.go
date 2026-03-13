package action

import "errors"

var (
	ErrActionNotFound        = errors.New("collection action not found")
	ErrInvalidActionType     = errors.New("invalid action type")
	ErrEmptySnapshot         = errors.New("no documents available for action snapshot")
	ErrInvalidDocumentStatus = errors.New("invalid document status filter")
)
