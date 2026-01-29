package job

import "context"

type JobHandler interface {
	Handle(ctx context.Context, job *Job) (*OutputManifest, error)
}

type JobRunner interface {
	Run(ctx context.Context, job *Job) error
}
