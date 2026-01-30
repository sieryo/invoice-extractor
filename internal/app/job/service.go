package job

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/domain/job"
)

type JobService struct {
	repo   job.Repository
	runner job.JobRunner
}

func NewJobService(repo job.Repository, runner job.JobRunner) *JobService {
	return &JobService{repo: repo, runner: runner}
}

func (s *JobService) CreateJob(
	ctx context.Context,
	userID string,
	jobType job.JobType,
	inputPayload []byte,
) (*job.Job, error) {

	j := job.NewJob(
		uuid.NewString(),
		&userID,
		jobType,
		inputPayload,
	)

	j.Status = job.JobPending
	j.CreatedAt = time.Now()

	if err := s.repo.Create(ctx, j); err != nil {
		return nil, err
	}

	return j, nil
}

func (s *JobService) StartJob(ctx context.Context, id string) error {
	j, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if j.Status != job.JobPending && j.Status != job.JobFailed {
		return job.ErrJobNotStartable
	}

	now := time.Now()
	j.Status = job.JobRunning
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

// TODO HARUS PAKE USER ID
func (s *JobService) ListJobs(ctx context.Context) ([]*job.Job, error) {
	return s.repo.List(ctx)
}
