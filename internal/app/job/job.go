package job

import (
	"time"
)

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

type JobStatus string

const (
	JobPending  JobStatus = "PENDING"
	JobRunning  JobStatus = "RUNNING"
	JobSuccess  JobStatus = "SUCCESS"
	JobFailed   JobStatus = "FAILED"
	JobCanceled JobStatus = "CANCELED"
)

type Job struct {
	ID            string
	UserID        *string
	Type          JobType
	Status        JobStatus
	Progress      int
	InputPayload  []byte // JSON
	OutputPayload []byte // JSON
	ErrorMessage  *string

	CreatedAt  time.Time
	StartedAt  *time.Time
	FinishedAt *time.Time
	ExpiredAt  *time.Time
}
