package handler

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/extract"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/app/shared"
)

type InvoiceExtractHandler struct {
	jobService *job.JobService
	fileStore  shared.FileStore
}

func NewInvoiceExtractHandler(jobService *job.JobService, fileStore shared.FileStore) *InvoiceExtractHandler {
	return &InvoiceExtractHandler{
		jobService: jobService,
		fileStore:  fileStore,
	}
}

func (h *InvoiceExtractHandler) Handle(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "failed to parse multipart form"})
	}

	files := form.File["files"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "no files uploaded"})
	}

	jobID := uuid.New().String()

	ctx := c.Context()

	var jobFiles []job.JobFile

	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to open file"})
		}
		defer src.Close()

		data, err := io.ReadAll(src)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to read file"})
		}

		tmp, err := h.fileStore.SaveTemp(ctx, jobID, file.Filename, data)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "failed to save file"})
		}

		jobFile := job.JobFile{
			ID:     tmp.ID,
			Name:   tmp.Name,
			URI:    tmp.Path,
			Status: job.JobFilePending,
		}

		jobFiles = append(jobFiles, jobFile)
	}

	payloadStruct := extract.Payload{
		JobFiles: jobFiles,
	}

	payloadBytes, err := json.Marshal(payloadStruct)
	if err != nil {
		return err
	}

	userID := c.Locals("userId").(string)

	newJob := &job.Job{
		ID:           jobID,
		UserID:       &userID,
		Type:         job.JobTypeInvoiceExtract,
		InputPayload: payloadBytes,
	}

	if err := h.jobService.CreateJob(ctx, newJob); err != nil {
		fmt.Printf("%s", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create job"})
	}

	// Ini harusnya async
	_ = h.jobService.StartJob(ctx, newJob)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"job_id":  jobID,
		"message": "job submitted successfully",
	})
}
