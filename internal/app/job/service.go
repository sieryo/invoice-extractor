package job

import (
	"context"
	"time"

	"github.com/sieryo/invoice-extractor/internal/domain/job"
)

type JobService struct {
	repo   job.JobRepository
	runner job.JobRunner
}

func NewJobService(repo job.JobRepository, runner job.JobRunner) *JobService {
	return &JobService{repo: repo, runner: runner}
}

func (s *JobService) CreateJob(ctx context.Context, j *job.Job) error {
	j.Status = job.JobPending
	j.CreatedAt = time.Now()
	return s.repo.Create(ctx, j)
}

func (s *JobService) StartJob(ctx context.Context, j *job.Job) error {
	j.Status = job.JobRunning
	now := time.Now()
	j.StartedAt = &now
	if err := s.repo.UpdateStatus(ctx, j.ID, j.Status); err != nil {
		return err
	}
	return s.runner.Run(ctx, j)
}

func (s *JobService) UpdateProgress(ctx context.Context, id string, progress int) error {
	return s.repo.UpdateProgress(ctx, id, progress)
}

func (s *JobService) FinishJob(ctx context.Context, j *job.Job, success bool, errMsg *string) error {
	now := time.Now()
	j.FinishedAt = &now
	if success {
		j.Status = job.JobSuccess
	} else {
		j.Status = job.JobFailed
		j.ErrorMessage = errMsg
	}
	return s.repo.Update(ctx, j)
}

func (s *JobService) GetJobByID(ctx context.Context, id string) (*job.Job, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *JobService) ListJobs(ctx context.Context) ([]*job.Job, error) {
	return s.repo.List(ctx)
}
