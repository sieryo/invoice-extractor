package action

import (
	"errors"
	"strings"
)

var (
	ErrActionNotFound        = errors.New("collection action not found")
	ErrInvalidActionType     = errors.New("invalid action type")
	ErrActionNotSupported    = errors.New("action is not supported for this document type")
	ErrActionDisabled        = errors.New("action is disabled")
	ErrInvalidActionParams   = errors.New("invalid action params")
	ErrInvalidActionSpec     = errors.New("invalid action spec")
	ErrEmptySnapshot         = errors.New("no documents available for action snapshot")
	ErrMinDocumentsRequired  = errors.New("minimum documents requirement is not met")
	ErrMaxDocumentsExceeded  = errors.New("maximum documents requirement is exceeded")
	ErrInvalidDocumentStatus = errors.New("invalid document status filter")
	ErrInvalidDocumentIDs    = errors.New("invalid document ids")
	ErrSnapshotDocNotFound   = errors.New("some selected documents are not available for snapshot")
	ErrSnapshotDocStatus     = errors.New("selected documents include unsupported status")
	ErrSpecNotFound          = errors.New("action spec not found")
	ErrActionRequirement     = errors.New("action requirement is not satisfied")
)

type DisabledActionError struct {
	Reason string
}

func (e *DisabledActionError) Error() string {
	if e == nil {
		return ErrActionDisabled.Error()
	}
	reason := strings.TrimSpace(e.Reason)
	if reason == "" {
		return ErrActionDisabled.Error()
	}
	return reason
}

func (e *DisabledActionError) Unwrap() error {
	return ErrActionDisabled
}

type RequirementError struct {
	Code    string
	Message string
}

func (e *RequirementError) Error() string {
	if e == nil {
		return ErrActionRequirement.Error()
	}
	message := strings.TrimSpace(e.Message)
	if message == "" {
		return ErrActionRequirement.Error()
	}
	return message
}

func (e *RequirementError) Unwrap() error {
	return ErrActionRequirement
}

type MaxDocumentsError struct {
	Limit int
}

func (e *MaxDocumentsError) Error() string {
	if e == nil || e.Limit <= 0 {
		return ErrMaxDocumentsExceeded.Error()
	}
	return "jumlah dokumen melebihi batas maksimum untuk action ini"
}

func (e *MaxDocumentsError) Unwrap() error {
	return ErrMaxDocumentsExceeded
}
