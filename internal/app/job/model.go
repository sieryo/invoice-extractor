package job

import (
	"time"
)

type JobStatus string

const (
	JobPending  JobStatus = "pending"
	JobRunning  JobStatus = "running"
	JobSuccess  JobStatus = "success"
	JobFailed   JobStatus = "failed"
	JobCanceled JobStatus = "canceled"
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
