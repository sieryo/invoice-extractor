package extract

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/infra/filestore"
)

// INI HANDLER
type InvoiceExtractJob struct {
	extractor *InvoiceExtractorService
	fileStore filestore.FileStore
}

func NewInvoiceExtractJob(extractor *InvoiceExtractorService, fileStore filestore.FileStore) *InvoiceExtractJob {
	return &InvoiceExtractJob{
		fileStore: fileStore,
		extractor: extractor,
	}
}

func (h *InvoiceExtractJob) Handle(ctx context.Context, j *job.Job) error {
	var payload Payload
	if err := json.Unmarshal(j.InputPayload, &payload); err != nil {
		return err
	}

	result, err := h.extractor.ExtractBatch(ctx, payload.PDFPaths, payload.Template)
	if err != nil {
		// h.fileStore.CleanupTemp(ctx, j.ID)
		return err
	}

	invoices := result.Invoices

	for i, inv := range invoices {
		data, _ := json.Marshal(inv)

		tempObj, err := h.fileStore.SaveTemp(
			ctx,
			j.ID,
			fmt.Sprintf("invoice_%d.json", i),
			data,
		)
		if err != nil {
			h.fileStore.CleanupTemp(ctx, j.ID)
			return err
		}

		if _, err := h.fileStore.Commit(ctx, tempObj); err != nil {
			h.fileStore.CleanupTemp(ctx, j.ID)
			return err
		}
	}

	return nil
}
