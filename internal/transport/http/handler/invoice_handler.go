package handler

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

	if j.Status != jobdomain.JobSuccess && j.Status != jobdomain.JobWarning {
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

func (h *InvoiceHandler) DownloadTaxInvoices(c *fiber.Ctx) error {
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
		if f.Status != jobdomain.OutputFileReady && f.Status != jobdomain.OutputFileWarning {
			continue
		}

		fileContent, err := h.fileStore.Read(ctx, j.ID, f.StorageName)
		if err != nil {
			fmt.Println(err.Error())
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

func (h *InvoiceHandler) GetInvoiceAudit(c *fiber.Ctx) error {
	ctx := c.Context()

	jobID := c.Params("job_id")
	if jobID == "" {
		return SendError(c, fiber.StatusBadRequest, "job_id is required")
	}

	if _, err := h.jobService.GetJobByID(ctx, jobID); err != nil {
		return SendError(c, fiber.StatusNotFound, "job not found")
	}

	fileID := c.Query("file_id")
	if fileID != "" {
		name := fmt.Sprintf("audit_%s.json", fileID)
		data, err := h.fileStore.ReadAudit(ctx, jobID, name)
		if err != nil {
			return SendError(c, fiber.StatusNotFound, "audit not found")
		}

		var payload any
		if err := json.Unmarshal(data, &payload); err != nil {
			return SendSuccess(c, fiber.StatusOK, fiber.Map{
				"file":       name,
				"raw":        string(data),
				"parseError": err.Error(),
			}, "audit retrieved successfully")
		}

		return SendSuccess(c, fiber.StatusOK, payload, "audit retrieved successfully")
	}

	names, err := h.fileStore.ListAudit(ctx, jobID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to list audit files")
	}

	if len(names) == 0 {
		return SendSuccess(c, fiber.StatusOK, fiber.Map{
			"audits": []any{},
			"count":  0,
		}, "audit files retrieved successfully")
	}

	audits := make([]any, 0, len(names))
	errors := make([]string, 0)

	for _, name := range names {
		data, err := h.fileStore.ReadAudit(ctx, jobID, name)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}

		var payload any
		if err := json.Unmarshal(data, &payload); err != nil {
			audits = append(audits, fiber.Map{
				"file":       name,
				"raw":        string(data),
				"parseError": err.Error(),
			})
			continue
		}

		if m, ok := payload.(map[string]any); ok {
			m["audit_file"] = name
			payload = m
		}

		audits = append(audits, payload)
	}

	if len(errors) > 0 {
		return SendSuccess(c, fiber.StatusOK, fiber.Map{
			"audits": audits,
			"count":  len(audits),
			"errors": strings.Join(errors, "; "),
		}, "audit files retrieved successfully")
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"audits": audits,
		"count":  len(audits),
	}, "audit files retrieved successfully")
}
