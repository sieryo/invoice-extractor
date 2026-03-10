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
	warningCount := 0
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
		outputStatus := job.OutputFileReady
		outputWarnings := []string(nil)
		if inv.Metadata != nil && len(inv.Metadata.Warnings) > 0 {
			outputStatus = job.OutputFileWarning
			outputWarnings = inv.Metadata.Warnings
			warningCount++
		}

		files = append(files, job.OutputFile{
			ID:             finalObj.ID,
			SourceFileID:   &inv.Metadata.SourceFile.ID,
			SourceFileName: inv.Metadata.SourceFile.Name,
			Name:           name,
			StorageName:    finalObj.ID + filepath.Ext(name),
			Type:           job.OutputFileTypeInvoice,
			URI:            finalObj.Path,
			Status:         outputStatus,
			Warnings:       outputWarnings,
		})
	}

	for _, audit := range result.Audits {
		data, err := json.Marshal(audit)
		if err != nil {
			fmt.Printf("failed to marshal audit for %s: %v\n", audit.SourceFile.Name, err)
			continue
		}

		name := fmt.Sprintf("audit_%s.json", audit.SourceFile.ID)
		if _, err := h.fileStore.SaveAudit(ctx, j.ID, name, data); err != nil {
			fmt.Printf("failed to save audit for %s: %v\n", audit.SourceFile.Name, err)
		}
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
			Warnings:   warningCount,
		},
		Files: files,
	}

	return &outputManifest, nil
}
