package handler

import "github.com/sieryo/invoice-extractor/internal/app/job"

type JobHandler struct {
	jobService *job.JobService
}

func NewJobHandler(jobService *job.JobService) *JobHandler {
	return &JobHandler{
		jobService: jobService,
	}
}
