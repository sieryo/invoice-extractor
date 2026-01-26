package job

import "time"

type JobType string

const (
	JobTypeInvoiceExtract JobType = "INVOICE_EXTRACT"
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
	Version int       `json:"version"`
	JobType JobType   `json:"job_type"`
	Summary Summary   `json:"summary"`
	Files   []JobFile `json:"files"`
}

type Summary struct {
	TotalFiles int `json:"total_files"`
	Ready      int `json:"ready"`
	Failed     int `json:"failed"`
}

type JobStatus string

const (
	JobPending  JobStatus = "PENDING"
	JobRunning  JobStatus = "RUNNING"
	JobSuccess  JobStatus = "SUCCESS"
	JobFailed   JobStatus = "FAILED"
	JobCanceled JobStatus = "CANCELED"
)

type Job struct {
	ID             string          `json:"id"`
	UserID         *string         `json:"user_id,omitempty"`
	Type           JobType         `json:"type"`
	Status         JobStatus       `json:"status"`
	Progress       int             `json:"progress"`
	InputPayload   []byte          `json:"input_payload,omitempty"`
	OutputManifest *OutputManifest `json:"output_manifest,omitempty"`
	ErrorMessage   *string         `json:"error_message,omitempty"`

	CreatedAt  time.Time  `json:"created_at"`
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	ExpiredAt  *time.Time `json:"expired_at,omitempty"`
}

type JobFileStatus string

const (
	JobFilePending JobFileStatus = "pending"
	JobFileReady   JobFileStatus = "ready"
	JobFileFailed  JobFileStatus = "failed"
)

type JobFile struct {
	ID     string        `json:"id"`
	Name   string        `json:"name"`
	URI    string        `json:"uri"`
	Status JobFileStatus `json:"status"`
}
