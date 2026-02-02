package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
	jobdomain "github.com/sieryo/invoice-extractor/internal/domain/job"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

type InvoiceHandler struct {
	invoiceService *invoice.InvoiceService
	fileStore      file.FileStore
	jobService     *job.JobService
}

func NewInvoiceHandler(
	invoiceService *invoice.InvoiceService,
	fileStore file.FileStore,
	jobService *job.JobService,
) *InvoiceHandler {
	return &InvoiceHandler{
		invoiceService: invoiceService,
		fileStore:      fileStore,
		jobService:     jobService,
	}
}

type LoadInvoiceRequest struct {
	JobID  string `json:"job_id"`
	FileID string `json:"file_id"`
}

func (h *InvoiceHandler) LoadInvoice(c *fiber.Ctx) error {
	ctx := c.Context()

	var req LoadInvoiceRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid form data")
	}

	jobID := req.JobID
	if jobID == "" {
		return SendError(c, fiber.StatusBadRequest, "job_id is required")
	}

	fileID := req.FileID
	if fileID == "" {
		return SendError(c, fiber.StatusBadRequest, "name is required")
	}

	inv, err := h.invoiceService.LoadInvoice(ctx, jobID, fileID)
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "invoice not found")
	}

	return SendSuccess(c, fiber.StatusOK, inv, "invoice loaded successfully")
}

func (h *InvoiceHandler) ExportInvoices(c *fiber.Ctx) error {
	ctx := c.Context()

	var req struct {
		JobID string `json:"job_id"`
	}

	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	jobID := req.JobID
	if jobID == "" {
		return SendError(c, fiber.StatusBadRequest, "job_id is required")
	}

	j, err := h.jobService.GetJobByID(ctx, jobID)
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "job not found")
	}

	if j.Status != jobdomain.JobSuccess {
		return SendError(c, fiber.StatusBadRequest, "job is not completed")
	}

	if j.Type != jobdomain.JobTypeExtractInvoice {
		return SendError(c, fiber.StatusBadRequest, "job is not an invoice extract job")
	}

	invoices, _, err := h.invoiceService.LoadInvoicesByJob(ctx, j)
	if err != nil {
		return err
	}

	if len(invoices) == 0 {
		return SendError(c, fiber.StatusNotFound, "no invoices found")
	}

	data, err := h.invoiceService.Export(ctx, invoices)
	if err != nil {
		return err
	}

	sellerTaxID := "UNKNOWN"
	if invoices[0].Seller != nil && invoices[0].Seller.TaxID != nil {
		sellerTaxID = *invoices[0].Seller.TaxID
	}

	date := helper.GetIndonesiaDateStr()

	filename := fmt.Sprintf("%s - %s.xlsx", sellerTaxID, date)

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	return c.Send(data)
}

func (h *InvoiceHandler) ListInvoices(c *fiber.Ctx) error {
	ctx := c.Context()

	jobID := c.Params("job_id")
	if jobID == "" {
		return SendError(c, fiber.StatusBadRequest, "job_id is required")
	}

	j, err := h.jobService.GetJobByID(ctx, jobID)
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "job not found")
	}

	invoices, stat, err := h.invoiceService.LoadInvoicesByJob(ctx, j)
	if err != nil {
		return err
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"stat":     stat,
		"invoices": invoices,
	}, "invoices loaded successfully")
}
