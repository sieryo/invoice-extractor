package handler

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/app/shared"
)

type InvoiceHandler struct {
	invoiceService *invoice.InvoiceService
	fileStore      shared.FileStore
	jobService     *job.JobService
}

func NewInvoiceHandler(
	invoiceService *invoice.InvoiceService,
	fileStore shared.FileStore,
	jobService *job.JobService,
) *InvoiceHandler {
	return &InvoiceHandler{
		invoiceService: invoiceService,
		fileStore:      fileStore,
		jobService:     jobService,
	}
}

func (h *InvoiceHandler) LoadInvoice(
	ctx context.Context,
	jobID string,
	file job.JobFile,
) (*invoice.Invoice, error) {

	return h.invoiceService.LoadInvoice(ctx, jobID, file.Name)
}

func (h *InvoiceHandler) ExportInvoices(c *fiber.Ctx) error {
	ctx := c.Context()

	jobID := c.FormValue("job_id")
	if jobID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "job_id is required",
		})
	}

	j, err := h.jobService.GetJobByID(ctx, jobID)
	if err != nil {
		return err
	}

	manifest := j.OutputManifest
	if manifest == nil || len(manifest.Files) == 0 {
		return c.Status(404).JSON(fiber.Map{
			"error": "no output files",
		})
	}

	stat := invoice.ExportStat{
		Total: len(manifest.Files),
	}

	invoices := make([]*invoice.Invoice, 0, len(manifest.Files))

	for _, f := range manifest.Files {
		if f.Status != job.JobFileReady {
			stat.Failed++
			continue
		}

		inv, err := h.invoiceService.LoadInvoice(ctx, jobID, f.Name)
		if err != nil {
			stat.Failed++
			continue
		}

		invoices = append(invoices, inv)
		stat.Success++
	}

	if len(invoices) == 0 {
		return c.Status(404).JSON(stat)
	}

	data, err := h.invoiceService.Export(ctx, invoices)
	if err != nil {
		return err
	}

	filename := "invoices.xlsx"

	tempObj, err := h.fileStore.SaveTemp(ctx, j.ID, filename, data)
	if err != nil {
		return err
	}

	finalObj, err := h.fileStore.Commit(ctx, tempObj)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"stat":   stat,
		"file": fiber.Map{
			"name": filename,
			"uri":  finalObj.Path,
		},
	})
}
