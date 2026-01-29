package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"

	"github.com/sieryo/invoice-extractor/internal/app/job"
	jobdomain "github.com/sieryo/invoice-extractor/internal/domain/job"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler/helper"
)

type JobHandler struct {
	jobService *job.JobService
}

// JOB HANDLER AKAN MENAMBAH JOB DAN TIPE DARI JOB BERASAL DARI FE, CARI DULU APAKAH ADA, KALAU ADA MAKA LANGSUNG BUAT AJAH. INPUT JOB DARI FE JUGA DAN NANTI AKAN DI-HANDLE PADA RUNNER.

func NewJobHandler(jobService *job.JobService) *JobHandler {
	return &JobHandler{
		jobService: jobService,
	}
}

type CreateJobRequest struct {
	JobType jobdomain.JobType `json:"job_type"`
	Payload json.RawMessage   `json:"payload"`
}

func (h *JobHandler) CreateJob(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	var req CreateJobRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if !req.JobType.IsValid() {
		return fiber.NewError(fiber.StatusBadRequest, "invalid job_type")
	}

	payloadBytes, err := helper.ValidateJobPayload(req.JobType, req.Payload)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	job, err := h.jobService.CreateJob(ctx, userID, req.JobType, payloadBytes)
	if err != nil {
		return err
	}
	return c.JSON(job)
}

func (h *JobHandler) ListJobs(c *fiber.Ctx) error {
	ctx := c.Context()

	jobs, err := h.jobService.ListJobs(ctx)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to retrieve jobs")
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"jobs": jobs,
	}, "jobs retrieved successfully")
}

func (h *JobHandler) GetJobByID(c *fiber.Ctx) error {
	ctx := c.Context()
	jobID := c.Params("id")

	if jobID == "" {
		return SendError(c, fiber.StatusBadRequest, "job ID is required")
	}

	job, err := h.jobService.GetJobByID(ctx, jobID)
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "job not found")
	}

	return SendSuccess(c, fiber.StatusOK, job, "job retrieved successfully")
}
