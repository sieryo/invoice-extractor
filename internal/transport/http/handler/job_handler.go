package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/job"
)

type JobHandler struct {
	jobService *job.JobService
}

func NewJobHandler(jobService *job.JobService) *JobHandler {
	return &JobHandler{
		jobService: jobService,
	}
}

func (h *JobHandler) ListJobs(c *fiber.Ctx) error {
	ctx := c.Context()

	jobs, err := h.jobService.ListJobs(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve jobs",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"jobs": jobs,
	})
}

func (h *JobHandler) GetJobByID(c *fiber.Ctx) error {
	ctx := c.Context()
	jobID := c.Params("id")

	if jobID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "job ID is required",
		})
	}

	job, err := h.jobService.GetJobByID(ctx, jobID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "job not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(job)
}
