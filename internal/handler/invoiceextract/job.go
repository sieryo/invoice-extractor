package invoiceextract

import (
	"context"
	"encoding/json"
)

type InvoiceExtractJob struct {
	extractor *InvoiceExtractorService
	jobRepo   JobRepository
}

func (h *InvoiceExtractJob) Handle(ctx context.Context, job *Job) error {
	var payload InvoiceExtractPayload
	if err := json.Unmarshal(job.InputPayload, &payload); err != nil {
		return err
	}

	_, err := h.extractor.ExtractBatch(ctx, payload.PDFPaths)
	return err
}
