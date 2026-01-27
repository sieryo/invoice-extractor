package handler

import (
	"context"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/app/shared"
)

type InvoiceHandler struct {
	invoiceService *invoice.InvoiceService
	fileStore      shared.FileStore
}

func NewInvoiceHandler(
	invoiceService *invoice.InvoiceService,
	fileStore shared.FileStore,
) *InvoiceHandler {
	return &InvoiceHandler{
		invoiceService: invoiceService,
		fileStore:      fileStore,
	}
}

func (h *InvoiceHandler) LoadInvoice(
	ctx context.Context,
	jobID string,
	file job.JobFile,
) (*invoice.Invoice, error) {

	return h.invoiceService.LoadInvoice(ctx, jobID, file.Name)
}

func (h *InvoiceHandler) ExportInvoices(
	ctx context.Context,
	jobID string,
	files []job.JobFile,
) ([]byte, invoice.ExportStat, error) {

	stat := invoice.ExportStat{
		Total: len(files),
	}

	invoices := make([]*invoice.Invoice, 0, len(files))

	for _, f := range files {
		inv, err := h.invoiceService.LoadInvoice(ctx, jobID, f.Name)
		if err != nil {
			stat.Failed++
			continue
		}

		invoices = append(invoices, inv)
		stat.Success++
	}

	if len(invoices) == 0 {
		return nil, stat, nil
	}

	out, err := h.invoiceService.Export(ctx, invoices)
	if err != nil {
		return nil, stat, err
	}

	return out, stat, nil
}
