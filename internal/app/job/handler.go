package job

import "context"

type JobHandler interface {
	Handle(ctx context.Context, job *Job) error
}
