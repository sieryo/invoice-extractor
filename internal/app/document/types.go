package document

import (
	"context"
	"encoding/json"
	"time"

	dcollection "github.com/sieryo/invoice-extractor/internal/domain/collection"
)

type DocumentType = dcollection.DocumentType

const (
	DocumentTypePDFInvoice    = dcollection.DocumentTypePDFInvoice
	DocumentTypePDFTaxInvoice = dcollection.DocumentTypePDFTaxInvoice
	DocumentTypeXLSXCashflow  = dcollection.DocumentTypeXLSXCashflow
)

type Artifact struct {
	Kind     string `json:"kind"`
	ObjectID string `json:"object_id"`
	MimeType string `json:"mime_type,omitempty"`
	Checksum string `json:"checksum,omitempty"`
	Size     int64  `json:"size,omitempty"`
}

type IngestPolicy struct {
	KeepRaw            bool `json:"keep_raw"`
	DeleteTempAfterRun bool `json:"delete_temp_after_run"`
}

type IngestSource struct {
	SourceID     string    `json:"source_id"`
	OriginalName string    `json:"original_name"`
	MimeType     string    `json:"mime_type,omitempty"`
	SizeBytes    int64     `json:"size_bytes,omitempty"`
	SHA256       string    `json:"sha256,omitempty"`
	SourceOrder  int       `json:"source_order"`
	TempPath     string    `json:"temp_path"`
	UploadedAt   time.Time `json:"uploaded_at"`
}

type IngestRequest struct {
	RequestID    string       `json:"request_id"`
	UserID       string       `json:"user_id"`
	CollectionID string       `json:"collection_id"`
	DocumentType DocumentType `json:"document_type"`
	Sources      []IngestSource
	Policy       IngestPolicy    `json:"policy"`
	Options      json.RawMessage `json:"options,omitempty"`
	RequestedAt  time.Time       `json:"requested_at"`
}

type IngestItemStatus string

const (
	IngestStatusReady     IngestItemStatus = "ready"
	IngestStatusWarning   IngestItemStatus = "warning"
	IngestStatusFailed    IngestItemStatus = "failed"
	IngestStatusDuplicate IngestItemStatus = "duplicate"
)

type IngestItemResult struct {
	SourceID      string           `json:"source_id"`
	OriginalName  string           `json:"original_name"`
	SHA256        string           `json:"sha256,omitempty"`
	Status        IngestItemStatus `json:"status"`
	Message       string           `json:"message,omitempty"`
	DocumentID    *string          `json:"document_id,omitempty"`
	DuplicateOfID *string          `json:"duplicate_of_id,omitempty"`
	Warnings      []string         `json:"warnings,omitempty"`
	Errors        []string         `json:"errors,omitempty"`
	Artifacts     []Artifact       `json:"artifacts,omitempty"`
}

type IngestResult struct {
	BatchID      string `json:"batch_id"`
	CollectionID string `json:"collection_id"`
	DocumentType string `json:"document_type"`
	Total        int    `json:"total"`
	Success      int    `json:"success"`
	Warning      int    `json:"warning"`
	Failed       int    `json:"failed"`
	Duplicate    int    `json:"duplicate"`
	Items        []IngestItemResult
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
}

type ActionSnapshotDocument struct {
	DocumentID    string `json:"document_id"`
	SourceName    string `json:"source_name"`
	SourceOrder   int    `json:"source_order"`
	Status        string `json:"status"`
	SourceSHA256  string `json:"source_sha256,omitempty"`
	NormalizedRef string `json:"normalized_ref"`
	AuditRef      string `json:"audit_ref,omitempty"`
	RawRef        string `json:"raw_ref,omitempty"`
}

type ActionRequest struct {
	ActionID      string       `json:"action_id"`
	UserID        string       `json:"user_id"`
	CollectionID  string       `json:"collection_id"`
	DocumentType  DocumentType `json:"document_type"`
	ActionType    string       `json:"action_type"`
	SnapshotDocID []string     `json:"snapshot_doc_ids"`
	SnapshotDocs  []ActionSnapshotDocument
	Params        json.RawMessage `json:"params,omitempty"`
	RequestedAt   time.Time       `json:"requested_at"`
}

type ActionOutput struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	ObjectRef string `json:"object_ref"`
	MimeType  string `json:"mime_type,omitempty"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
	Checksum  string `json:"checksum,omitempty"`
}

type ActionItemResult struct {
	DocumentID string   `json:"document_id"`
	Status     string   `json:"status"`
	Message    string   `json:"message,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	Error      string   `json:"error,omitempty"`
}

type ActionResult struct {
	ActionID    string `json:"action_id"`
	ActionType  string `json:"action_type"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
	Total       int    `json:"total"`
	Success     int    `json:"success"`
	Warning     int    `json:"warning"`
	Failed      int    `json:"failed"`
	Skipped     int    `json:"skipped"`
	Outputs     []ActionOutput
	ItemResults []ActionItemResult
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
}

type DocumentProcessor interface {
	Type() DocumentType
	Ingest(ctx context.Context, req IngestRequest) (IngestResult, error)
	RunAction(ctx context.Context, req ActionRequest) (ActionResult, error)
}
