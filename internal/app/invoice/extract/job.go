package extract

import (
	"context"
	"encoding/json"

	"github.com/sieryo/invoice-extractor/internal/app/job"
)

type InvoiceExtractJob struct {
	extractor *InvoiceExtractorService
}

func NewInvoiceExtractJob(extractor *InvoiceExtractorService) *InvoiceExtractJob {
	return &InvoiceExtractJob{
		extractor: extractor,
	}
}

func (h *InvoiceExtractJob) Handle(ctx context.Context, job *job.Job) error {
	var payload Payload
	if err := json.Unmarshal(job.InputPayload, &payload); err != nil {
		return err
	}

	err := h.extractor.ExtractBatch(ctx, payload.PDFPaths)
	if err != nil {
		return err
	}

	// Stub: set dummy output payload
	dummyOutput := map[string]interface{}{
		"invoice_number": "INV-12345",
		"total_amount":   150000,
		"extracted_at":   "2023-01-01T12:00:00Z",
	}
	outputBytes, _ := json.Marshal(dummyOutput)
	job.OutputPayload = outputBytes

	return nil
}
