package job

import (
	"encoding/json"
	"time"
)

type JobType string

const (
	JobTypeInvoiceExtract JobType = "invoice_extract"
)

func (t JobType) IsValid() bool {
	switch t {
	case JobTypeInvoiceExtract:
		return true
	default:
		return false
	}
}

type OutputManifest struct {
	Version int          `json:"version"`
	Summary Summary      `json:"summary"`
	Files   []OutputFile `json:"files"`
}

type Summary struct {
	TotalFiles int `json:"total_files"`
	Ready      int `json:"ready"`
	Failed     int `json:"failed"`
}

type JobStatus string

const (
	JobPending  JobStatus = "pending"
	JobRunning  JobStatus = "running"
	JobSuccess  JobStatus = "success"
	JobFailed   JobStatus = "failed"
	JobCanceled JobStatus = "canceled"
)

type Job struct {
	ID     string  `json:"id"`
	UserID *string `json:"user_id,omitempty"`

	Type     JobType   `json:"type"`
	Status   JobStatus `json:"status"`
	Progress int       `json:"progress"`

	// opaque, immutable
	InputPayload json.RawMessage `json:"input_payload,omitempty"`

	// structured, optional
	OutputManifest *OutputManifest `json:"output_manifest,omitempty"`

	ErrorMessage *string `json:"error_message,omitempty"`

	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	ExpiredAt  *time.Time `json:"expired_at,omitempty"`
}

type OutputFileStatus string

const (
	OutputFileReady  OutputFileStatus = "ready"
	OutputFileFailed OutputFileStatus = "failed"
)

type OutputFileType string

const (
	OutputFileTypeInvoice OutputFileType = "invoice_json"
	OutputFileTypeRaw     OutputFileType = "raw"
)

type OutputFile struct {
	ID     string           `json:"id"`
	Name   string           `json:"name"`
	Type   OutputFileType   `json:"type"`
	URI    string           `json:"uri"`
	Status OutputFileStatus `json:"status"`
}

type InputFile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URI  string `json:"uri"`
}
