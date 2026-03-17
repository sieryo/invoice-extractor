package action

import (
	"encoding/json"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/document"
)

type Status string

const (
	StatusQueued   Status = "queued"
	StatusRunning  Status = "running"
	StatusSuccess  Status = "success"
	StatusWarning  Status = "warning"
	StatusPartial  Status = "partial"
	StatusFailed   Status = "failed"
	StatusCanceled Status = "canceled"
)

type ItemStatus string

const (
	ItemStatusSuccess  ItemStatus = "success"
	ItemStatusWarning  ItemStatus = "warning"
	ItemStatusFailed   ItemStatus = "failed"
	ItemStatusSkipped  ItemStatus = "skipped"
	ItemStatusCanceled ItemStatus = "canceled"
)

type OutputKind string

const (
	OutputKindFile    OutputKind = "file"
	OutputKindLink    OutputKind = "link"
	OutputKindPayload OutputKind = "payload"
)

type SnapshotDocument struct {
	DocumentID    string `json:"documentId"`
	SourceName    string `json:"sourceName"`
	SourceOrder   int    `json:"sourceOrder"`
	Status        string `json:"status"`
	DocumentTag   string `json:"documentTag,omitempty"`
	SourceSHA256  string `json:"sourceSha256,omitempty"`
	NormalizedRef string `json:"normalizedRef"`
	AuditRef      string `json:"auditRef,omitempty"`
	RawRef        string `json:"rawRef,omitempty"`
}

type CollectionAction struct {
	ID             string                `json:"id"`
	UserID         string                `json:"userId"`
	CollectionID   string                `json:"collectionId"`
	DocumentType   document.DocumentType `json:"documentType"`
	ActionType     string                `json:"actionType"`
	Status         Status                `json:"status"`
	Message        string                `json:"message,omitempty"`
	ParamsJSON     json.RawMessage       `json:"params,omitempty"`
	SnapshotJSON   json.RawMessage       `json:"-"`
	SnapshotHash   string                `json:"snapshotHash"`
	SnapshotTotal  int                   `json:"snapshotTotal"`
	RerunOfAction  *string               `json:"rerunOfActionId,omitempty"`
	IdempotencyKey *string               `json:"idempotencyKey,omitempty"`
	TotalCount     int                   `json:"totalCount"`
	SuccessCount   int                   `json:"successCount"`
	WarningCount   int                   `json:"warningCount"`
	FailedCount    int                   `json:"failedCount"`
	SkippedCount   int                   `json:"skippedCount"`
	StartedAt      *time.Time            `json:"startedAt,omitempty"`
	FinishedAt     *time.Time            `json:"finishedAt,omitempty"`
	CreatedAt      time.Time             `json:"createdAt"`
	UpdatedAt      time.Time             `json:"updatedAt"`
}

type CollectionActionItem struct {
	ID           string          `json:"id"`
	ActionID     string          `json:"actionId"`
	DocumentID   *string         `json:"documentId,omitempty"`
	SourceName   *string         `json:"sourceName,omitempty"`
	Status       ItemStatus      `json:"status"`
	Message      string          `json:"message,omitempty"`
	WarningsJSON json.RawMessage `json:"warnings,omitempty"`
	Error        string          `json:"error,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
}

type CollectionActionOutput struct {
	ID        string     `json:"id"`
	ActionID  string     `json:"actionId"`
	Kind      OutputKind `json:"kind"`
	Name      string     `json:"name"`
	ObjectRef string     `json:"objectRef"`
	MimeType  string     `json:"mimeType,omitempty"`
	SizeBytes int64      `json:"sizeBytes,omitempty"`
	Checksum  string     `json:"checksum,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type ActionDetail struct {
	Action  *CollectionAction         `json:"action"`
	Items   []*CollectionActionItem   `json:"items"`
	Outputs []*CollectionActionOutput `json:"outputs"`
}

type RunRequest struct {
	UserID           string          `json:"userId"`
	CollectionID     string          `json:"collectionId"`
	ActionType       string          `json:"actionType"`
	Params           json.RawMessage `json:"params,omitempty"`
	DocumentIDs      []string        `json:"documentIds,omitempty"`
	DocumentStatuses []string        `json:"documentStatuses,omitempty"`
	IdempotencyKey   *string         `json:"idempotencyKey,omitempty"`
	RerunOfActionID  *string         `json:"rerunOfActionId,omitempty"`
}

type ListRequest struct {
	UserID       string `json:"userId"`
	CollectionID string `json:"collectionId"`
	Status       string `json:"status,omitempty"`
	Limit        int    `json:"limit"`
	Offset       int    `json:"offset"`
}
