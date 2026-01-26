package jobrunner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/job"
)

type JobQueueRunner struct {
	dispatcher *Dispatcher
	repo       job.JobRepository
	queue      chan *job.Job
	wg         sync.WaitGroup
	workers    int
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
	_ = r.repo.UpdateStatus(ctx, j.ID, job.JobRunning)

	err := r.dispatcher.Dispatch(ctx, j)

	if err != nil {
		errMsg := err.Error()
		_ = r.repo.Update(ctx, &job.Job{
			ID:           j.ID,
			Status:       job.JobFailed,
			ErrorMessage: &errMsg,
			FinishedAt:   ptrTimeNow(),
		})

		fmt.Printf("Err : %s", errMsg)
	} else {
		_ = r.repo.Update(ctx, &job.Job{
			ID:         j.ID,
			Status:     job.JobSuccess,
			FinishedAt: ptrTimeNow(),
		})
	}
}

func ptrTimeNow() *time.Time {
	t := time.Now()
	return &t
}
