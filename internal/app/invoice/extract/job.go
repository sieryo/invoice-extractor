package extract

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/app/shared"
)

// INI HANDLER
type InvoiceExtractJob struct {
	extractor *InvoiceExtractorService
	fileStore shared.FileStore
}

func NewInvoiceExtractJob(extractor *InvoiceExtractorService, fileStore shared.FileStore) *InvoiceExtractJob {
	return &InvoiceExtractJob{
		fileStore: fileStore,
		extractor: extractor,
	}
}

func (h *InvoiceExtractJob) Handle(ctx context.Context, j *job.Job) (*job.OutputManifest, error) {
	var payload Payload
	if err := json.Unmarshal(j.InputPayload, &payload); err != nil {
		return nil, err
	}

	result, err := h.extractor.ExtractBatch(ctx, payload.JobFiles, payload.Template)
	if err != nil {
		return nil, err
	}

	var files []job.JobFile

	// SUCCESS invoices
	for i, inv := range result.Invoices {
		name := fmt.Sprintf("invoice_%d.json", i)

		data, _ := json.Marshal(inv)

		tempObj, err := h.fileStore.SaveTemp(ctx, j.ID, name, data)
		if err != nil {
			return nil, err
		}

		finalObj, err := h.fileStore.Commit(ctx, tempObj)
		if err != nil {
			return nil, err
		}

		files = append(files, job.JobFile{
			ID:     finalObj.ID,
			Name:   name,
			URI:    finalObj.Path,
			Status: job.JobFileReady,
		})
	}

	for _, e := range result.Errors {
		files = append(files, job.JobFile{
			ID:     e.FileID,
			Name:   e.FileName,
			URI:    "",
			Status: job.JobFileFailed,
		})
	}

	if err := h.fileStore.CleanupTemp(ctx, j.ID); err != nil {
		return nil, err
	}

	outputManifest := job.OutputManifest{
		Version: 1,
		JobType: job.JobTypeInvoiceExtract,
		Summary: job.Summary{
			TotalFiles: len(result.Invoices) + len(result.Errors),
			Ready:      len(result.Invoices),
			Failed:     len(result.Errors),
		},
		Files: files,
	}

	return &outputManifest, nil
}
