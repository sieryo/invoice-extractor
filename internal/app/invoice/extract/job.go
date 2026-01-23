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
	return err
}
