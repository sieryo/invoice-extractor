package jobrunner

import (
	"context"
	"fmt"

	"github.com/sieryo/invoice-extractor/internal/domain/job"
)

type Dispatcher struct {
	handlers map[job.JobType]job.JobHandler
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		handlers: make(map[job.JobType]job.JobHandler),
	}
}

func (d *Dispatcher) Has(jobType job.JobType) bool {
	_, ok := d.handlers[jobType]
	return ok
}

func (d *Dispatcher) MustRegister(jobType job.JobType, handler job.JobHandler) {
	if _, exists := d.handlers[jobType]; exists {
		panic(fmt.Sprintf("job handler already registered for %s", jobType))
	}
	d.handlers[jobType] = handler
}

func (d *Dispatcher) Dispatch(ctx context.Context, j *job.Job) (*job.OutputManifest, error) {
	handler, ok := d.handlers[j.Type]
	if !ok {
		return nil, fmt.Errorf("no handler registered for job type %s", j.Type)
	}
	return handler.Handle(ctx, j)
}
