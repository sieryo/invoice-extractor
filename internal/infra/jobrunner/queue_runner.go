package jobrunner

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/sieryo/invoice-extractor/internal/domain/job"
)

type JobQueueRunner struct {
	dispatcher *Dispatcher
	repo       job.JobRepository
	queue      chan *job.Job
	wg         sync.WaitGroup
	workers    int
}

type funcProgressReporter struct {
	fn func(int)
}

func (f funcProgressReporter) Report(p int) {
	f.fn(p)
}

func NewJobQueueRunner(repo job.JobRepository, dispatcher *Dispatcher, workers int) *JobQueueRunner {
	return &JobQueueRunner{
		dispatcher: dispatcher,
		repo:       repo,
		queue:      make(chan *job.Job, 10),
		workers:    workers,
	}
}

func (r *JobQueueRunner) Run(ctx context.Context, j *job.Job) error {
	select {
	case r.queue <- j:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *JobQueueRunner) StartPool(ctx context.Context) {
	for i := 0; i < r.workers; i++ {
		r.wg.Add(1)
		go r.worker(ctx, i)
	}
}

func (r *JobQueueRunner) StopPool() {
	close(r.queue)
	r.wg.Wait()
}

func (r *JobQueueRunner) worker(ctx context.Context, id int) {
	defer r.wg.Done()
	for j := range r.queue {
		select {
		case <-ctx.Done():
			fmt.Printf("Worker %d stopping due to context cancel\n", id)
			return
		default:
			r.executeJob(ctx, j)
		}
	}
}

func (r *JobQueueRunner) executeJob(ctx context.Context, j *job.Job) {
	outputManifest, err := r.dispatcher.Dispatch(ctx, j)

	reporter := funcProgressReporter{
		fn: func(p int) {
			_ = r.repo.UpdateProgress(ctx, j.ID, p)
		},
	}

	ctx = WithProgressReporter(ctx, reporter)

	if err != nil {
		errMsg := err.Error()
		j.Status = job.JobFailed
		j.ErrorMessage = &errMsg
		_ = r.repo.Update(ctx, j)

		// Diemin aja progressnya.

		fmt.Printf("Err : %s", errMsg)
	} else {
		j.Status = job.JobSuccess
		j.FinishedAt = ptrTimeNow()
		j.OutputManifest = outputManifest
		j.Progress = 100

		if err := r.repo.Update(ctx, j); err != nil {
			log.Printf("failed to update job %s: %v", j.ID, err)
		}

	}
}

func ptrTimeNow() *time.Time {
	t := time.Now()
	return &t
}
