package job

import (
	"time"
)

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
	Type          string
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
