package ingest

import (
	"encoding/json"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/document"
)

type SessionStatus string

const (
	SessionStatusCreated     SessionStatus = "created"
	SessionStatusReceiving   SessionStatus = "receiving"
	SessionStatusProcessing  SessionStatus = "processing"
	SessionStatusFinalized   SessionStatus = "finalized"
	SessionStatusCompleted   SessionStatus = "completed"
	SessionStatusFailed      SessionStatus = "failed"
	SessionStatusInterrupted SessionStatus = "interrupted"
	SessionStatusExpired     SessionStatus = "expired"
)

type ChunkStatus string

const (
	ChunkStatusUploaded   ChunkStatus = "uploaded"
	ChunkStatusQueued     ChunkStatus = "queued"
	ChunkStatusProcessing ChunkStatus = "processing"
	ChunkStatusDone       ChunkStatus = "done"
	ChunkStatusFailed     ChunkStatus = "failed"
	ChunkStatusDuplicate  ChunkStatus = "duplicate"
)

type UploadSession struct {
	ID               string                `json:"id"`
	UserID           string                `json:"userId"`
	CollectionID     string                `json:"collectionId"`
	DocumentType     document.DocumentType `json:"documentType"`
	Status           SessionStatus         `json:"status"`
	TotalChunks      int                   `json:"totalChunks"`
	UploadedChunks   int                   `json:"uploadedChunks"`
	ProcessedChunks  int                   `json:"processedChunks"`
	FailedChunks     int                   `json:"failedChunks"`
	DuplicateChunks  int                   `json:"duplicateChunks"`
	LastHeartbeatAt  *time.Time            `json:"lastHeartbeatAt,omitempty"`
	StartedAt        time.Time             `json:"startedAt"`
	FinishedAt       *time.Time            `json:"finishedAt,omitempty"`
	ExpiresAt        *time.Time            `json:"expiresAt,omitempty"`
	ClientSessionKey *string               `json:"clientSessionKey,omitempty"`
	MetadataJSON     json.RawMessage       `json:"metadata,omitempty"`
}

type UploadChunk struct {
	ID              string          `json:"id"`
	SessionID       string          `json:"sessionId"`
	ChunkIndex      int             `json:"chunkIndex"`
	Status          ChunkStatus     `json:"status"`
	IdempotencyKey  string          `json:"idempotencyKey"`
	RequestChecksum *string         `json:"requestChecksum,omitempty"`
	FileCount       int             `json:"fileCount"`
	SizeBytes       int64           `json:"sizeBytes"`
	JobID           *string         `json:"jobId,omitempty"`
	ErrorMessage    *string         `json:"errorMessage,omitempty"`
	PayloadJSON     json.RawMessage `json:"payload,omitempty"`
	CreatedAt       time.Time       `json:"createdAt"`
	StartedAt       *time.Time      `json:"startedAt,omitempty"`
	FinishedAt      *time.Time      `json:"finishedAt,omitempty"`
}

type DocumentRecord struct {
	ID              string                `json:"id"`
	UserID          string                `json:"userId"`
	CollectionID    string                `json:"collectionId"`
	DocumentType    document.DocumentType `json:"documentType"`
	SourceName      string                `json:"sourceName"`
	SourceSizeBytes int64                 `json:"sourceSizeBytes"`
	SourceMIME      string                `json:"sourceMime"`
	SourceSHA256    string                `json:"sourceSha256"`
	SourceOrder     int                   `json:"sourceOrder"`
	Status          string                `json:"status"`
	Message         string                `json:"message"`
	NormalizedRef   string                `json:"normalizedRef"`
	AuditRef        *string               `json:"auditRef,omitempty"`
	RawRef          *string               `json:"rawRef,omitempty"`
}

type CollectionHistory struct {
	ID             string          `json:"id"`
	UserID         string          `json:"userId"`
	CollectionID   string          `json:"collectionId"`
	ActionType     string          `json:"actionType"`
	SessionID      *string         `json:"sessionId,omitempty"`
	TriggeredBy    string          `json:"triggeredBy"`
	Status         string          `json:"status"`
	StartedAt      time.Time       `json:"startedAt"`
	FinishedAt     *time.Time      `json:"finishedAt,omitempty"`
	TotalCount     int             `json:"totalCount"`
	ReadyCount     int             `json:"readyCount"`
	WarningCount   int             `json:"warningCount"`
	FailedCount    int             `json:"failedCount"`
	DuplicateCount int             `json:"duplicateCount"`
	MetadataJSON   json.RawMessage `json:"metadata,omitempty"`
}

type CollectionHistoryItem struct {
	ID              string                `json:"id"`
	HistoryID       string                `json:"historyId"`
	UserID          string                `json:"userId"`
	CollectionID    string                `json:"collectionId"`
	DocumentType    document.DocumentType `json:"documentType"`
	SourceName      string                `json:"sourceName"`
	SourceSizeBytes int64                 `json:"sourceSizeBytes"`
	SourceMIME      string                `json:"sourceMime"`
	SourceSHA256    string                `json:"sourceSha256"`
	SourceOrder     int                   `json:"sourceOrder"`
	ItemStatus      string                `json:"itemStatus"`
	Message         string                `json:"message"`
	DocumentID      *string               `json:"documentId,omitempty"`
	DuplicateOfID   *string               `json:"duplicateOfId,omitempty"`
	DuplicateKey    *string               `json:"duplicateKey,omitempty"`
	WarningsJSON    json.RawMessage       `json:"warnings,omitempty"`
	ErrorsJSON      json.RawMessage       `json:"errors,omitempty"`
}

type ChunkUploadInput struct {
	ChunkIndex      int
	IdempotencyKey  string
	RequestChecksum *string
	Sources         []document.IngestSource
}

type SourceUploadFile struct {
	Name string
	Data []byte
}

type SessionDetail struct {
	Session *UploadSession `json:"session"`
	Chunks  []*UploadChunk `json:"chunks"`
}
