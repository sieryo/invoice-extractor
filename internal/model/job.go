package model

import (
	"context"
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

type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	FindByID(ctx context.Context, id string) (*Job, error)
	Update(ctx context.Context, job *Job) error
	UpdateStatus(ctx context.Context, id string, status JobStatus) error
	UpdateProgress(ctx context.Context, id string, progress int) error
}
