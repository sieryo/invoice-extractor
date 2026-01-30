package extract

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/job"
)

// INI HANDLER
type InvoiceExtractJob struct {
	extractor *InvoiceExtractorService
	fileStore file.FileStore
	fileRepo  file.Repository
}

func NewInvoiceExtractJob(extractor *InvoiceExtractorService, fileStore file.FileStore, fileRepo file.Repository) *InvoiceExtractJob {
	return &InvoiceExtractJob{
		fileRepo:  fileRepo,
		fileStore: fileStore,
		extractor: extractor,
	}
}

func (h *InvoiceExtractJob) Handle(ctx context.Context, j *job.Job) (*job.OutputManifest, error) {
	var payload Payload
	if err := json.Unmarshal(j.InputPayload, &payload); err != nil {
		return nil, err
	}

	resolved := make([]file.ResolvedFile, 0, len(payload.InputFiles))

	for _, ref := range payload.InputFiles {
		f, err := h.fileRepo.FindByID(ctx, ref.ID)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, file.ResolvedFile{
			FileRef: ref,
			Path:    f.Path,
		})
	}

	result, err := h.extractor.ExtractBatch(ctx, resolved, payload.Template)
	if err != nil {
		return nil, err
	}

	var files []job.OutputFile

	// SUCCESS invoices
	for i, inv := range result.Invoices {
		count := i + 1
		name := fmt.Sprintf("invoice_%d.json", count)

		data, _ := json.Marshal(inv)

		tempObj, err := h.fileStore.SaveTemp(ctx, j.ID, name, data)
		if err != nil {
			return nil, err
		}

		finalObj, err := h.fileStore.Commit(ctx, tempObj)
		if err != nil {
			return nil, err
		}
		files = append(files, job.OutputFile{
			ID:             finalObj.ID,
			SourceFileID:   &inv.Metadata.SourceFile.ID,
			SourceFileName: inv.Metadata.SourceFile.Name,
			Name:           name,
			StorageName:    finalObj.ID + filepath.Ext(name),
			Type:           job.OutputFileTypeInvoice,
			URI:            finalObj.Path,
			Status:         job.OutputFileReady,
		})
	}

	for _, e := range result.Errors {
		files = append(files, job.OutputFile{
			ID:             e.FileID,
			Name:           e.FileName,
			SourceFileName: e.FileName,
			URI:            "",
			Status:         job.OutputFileFailed,
			Type:           job.OutputFileTypeInvoice,
		})

	}

	// if err := h.fileStore.CleanupTemp(ctx, payload.CollectionID); err != nil {
	// 	return nil, err
	// }

	outputManifest := job.OutputManifest{
		Version: 1,
		Summary: job.Summary{
			TotalFiles: len(result.Invoices) + len(result.Errors),
			Ready:      len(result.Invoices),
			Failed:     len(result.Errors),
		},
		Files: files,
	}

	return &outputManifest, nil
}
