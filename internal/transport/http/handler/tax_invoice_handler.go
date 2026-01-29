package handler

import (
	"archive/zip"
	"bytes"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/app/shared"
)

type TaxInvoiceHandler struct {
	fileStore  shared.FileStore
	jobService *job.JobService
}

func NewTaxInvoiceHandler(
	fileStore shared.FileStore,
	jobService *job.JobService,
) *TaxInvoiceHandler {
	return &TaxInvoiceHandler{
		fileStore:  fileStore,
		jobService: jobService,
	}
}

func (h *TaxInvoiceHandler) DownloadZip(c *fiber.Ctx) error {
	ctx := c.Context()

	jobID := c.Params("job_id")
	if jobID == "" {
		return SendError(c, fiber.StatusBadRequest, "job_id is required")
	}

	j, err := h.jobService.GetJobByID(ctx, jobID)
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "job not found")
	}

	if j.OutputManifest == nil || len(j.OutputManifest.Files) == 0 {
		return SendError(c, fiber.StatusNotFound, "no files found for this job")
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, f := range j.OutputManifest.Files {
		if f.Status != job.OutputFileReady {
			continue
		}

		fileContent, err := h.fileStore.Read(ctx, j.ID, f.Name)
		if err != nil {
			// Skip file if cannot be read, or log error
			// For now, we continue
			continue
		}

		fWriter, err := zipWriter.Create(f.Name)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "failed to create zip entry")
		}

		_, err = fWriter.Write(fileContent)
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "failed to write to zip entry")
		}
	}

	if err := zipWriter.Close(); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to finalize zip")
	}

	filename := fmt.Sprintf("tax_invoices_%s.zip", time.Now().Format("20060102_150405"))

	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.Send(buf.Bytes())
}
