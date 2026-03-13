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
	ID               string
	UserID           string
	CollectionID     string
	DocumentType     document.DocumentType
	Status           SessionStatus
	TotalChunks      int
	UploadedChunks   int
	ProcessedChunks  int
	FailedChunks     int
	DuplicateChunks  int
	LastHeartbeatAt  *time.Time
	StartedAt        time.Time
	FinishedAt       *time.Time
	ExpiresAt        *time.Time
	ClientSessionKey *string
	MetadataJSON     json.RawMessage
}

type UploadChunk struct {
	ID              string
	SessionID       string
	ChunkIndex      int
	Status          ChunkStatus
	IdempotencyKey  string
	RequestChecksum *string
	FileCount       int
	SizeBytes       int64
	JobID           *string
	ErrorMessage    *string
	PayloadJSON     json.RawMessage
	CreatedAt       time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
}

type DocumentRecord struct {
	ID              string
	UserID          string
	CollectionID    string
	DocumentType    document.DocumentType
	SourceName      string
	SourceSizeBytes int64
	SourceMIME      string
	SourceSHA256    string
	SourceOrder     int
	Status          string
	Message         string
	NormalizedRef   string
	AuditRef        *string
	RawRef          *string
}

type CollectionHistory struct {
	ID             string
	UserID         string
	CollectionID   string
	ActionType     string
	SessionID      *string
	TriggeredBy    string
	Status         string
	StartedAt      time.Time
	FinishedAt     *time.Time
	TotalCount     int
	ReadyCount     int
	WarningCount   int
	FailedCount    int
	DuplicateCount int
	MetadataJSON   json.RawMessage
}

type CollectionHistoryItem struct {
	ID              string
	HistoryID       string
	UserID          string
	CollectionID    string
	DocumentType    document.DocumentType
	SourceName      string
	SourceSizeBytes int64
	SourceMIME      string
	SourceSHA256    string
	SourceOrder     int
	ItemStatus      string
	Message         string
	DocumentID      *string
	DuplicateOfID   *string
	DuplicateKey    *string
	WarningsJSON    json.RawMessage
	ErrorsJSON      json.RawMessage
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
	Session *UploadSession
	Chunks  []*UploadChunk
}
