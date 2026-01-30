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

type StartJobRequest struct {
	JobID string `json:"job_id"`
}

func (h *JobHandler) CreateJob(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	var req CreateJobRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if !req.JobType.IsValid() {
		return SendError(c, fiber.StatusBadRequest, "invalid job_type")
	}

	payloadBytes, err := helper.ValidateJobPayload(req.JobType, req.Payload)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	job, err := h.jobService.CreateJob(ctx, userID, req.JobType, payloadBytes)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}
	return SendSuccess(c, fiber.StatusOK, job, "job created successfully")
}

func (h *JobHandler) StartJob(c *fiber.Ctx) error {
	ctx := c.Context()

	var req StartJobRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	if req.JobID == "" {
		return SendError(c, fiber.StatusBadRequest, "job_id is required")
	}

	err := h.jobService.StartJob(ctx, req.JobID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}
	return SendSuccess(c, fiber.StatusOK, nil, "job started successfully")
}

func (h *JobHandler) ListJobs(c *fiber.Ctx) error {
	ctx := c.Context()

	jobs, err := h.jobService.ListJobs(ctx)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to retrieve jobs")
	}

	if len(jobs) == 0 {
		return SendSuccess(c, fiber.StatusOK, []jobdomain.Job{}, "jobs retrieved successfully")
	}

	return SendSuccess(c, fiber.StatusOK, jobs, "jobs retrieved successfully")
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
