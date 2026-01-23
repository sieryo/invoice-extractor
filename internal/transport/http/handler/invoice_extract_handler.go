package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/infra/filestore"
)

type InvoiceExtractHandler struct {
	jobService *job.JobService
	fileStore  filestore.FileStore
}

func NewInvoiceExtractHandler(jobService *job.JobService, fileStore filestore.FileStore) *InvoiceExtractHandler {
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
	jobDir := filepath.Join("jobs", jobID, "input")

	var savedPaths []string

	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to open uploaded file"})
		}
		defer src.Close()

		savedPath, err := h.fileStore.Save(c.Context(), filepath.Join(jobDir, file.Filename), src)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to save file"})
		}
		savedPaths = append(savedPaths, savedPath)
	}

	payload := map[string]interface{}{
		"pdf_paths": savedPaths,
	}
	payloadBytes, _ := json.Marshal(payload)

	newJob := &job.Job{
		ID:           jobID,
		Type:         "INVOICE_EXTRACT",
		InputPayload: payloadBytes,
	}

	if err := h.jobService.CreateJob(c.Context(), newJob); err != nil {
		fmt.Printf("%s", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create job"})
	}

	go func() {

		_ = h.jobService.StartJob(context.Background(), newJob)
	}()

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"job_id":  jobID,
		"message": "job submitted successfully",
		"files":   len(savedPaths),
	})
}
