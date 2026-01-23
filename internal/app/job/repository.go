package job

import "context"

type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	FindByID(ctx context.Context, id string) (*Job, error)
	Update(ctx context.Context, job *Job) error
	UpdateStatus(ctx context.Context, id string, status JobStatus) error
	UpdateProgress(ctx context.Context, id string, progress int) error
}
