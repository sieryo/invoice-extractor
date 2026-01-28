package rename

import (
	"context"
	"encoding/json"
	"os"

	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/app/shared"
)

type TaxInvoiceRenameJob struct {
	renameService *TaxInvoiceRenameService
	fileStore     shared.FileStore
}

func NewTaxInvoiceRenameJob(
	renameService *TaxInvoiceRenameService,
	fileStore shared.FileStore,
) *TaxInvoiceRenameJob {
	return &TaxInvoiceRenameJob{
		renameService: renameService,
		fileStore:     fileStore,
	}
}

func (j *TaxInvoiceRenameJob) Handle(ctx context.Context, jb *job.Job) (*job.OutputManifest, error) {
	var payload Payload
	if err := json.Unmarshal(jb.InputPayload, &payload); err != nil {
		return nil, err
	}

	// proses rename batch
	result, err := j.renameService.RenameBatch(ctx, payload.InputFiles)
	if err != nil {
		return nil, err
	}

	var files []job.OutputFile

	for _, r := range result.Files {

		data, err := os.ReadFile(r.SourceURI)
		if err != nil {
			// append error
			continue
		}

		tempObj, err := j.fileStore.SaveTemp(ctx, jb.ID, r.NewName, data)
		if err != nil {
			return nil, err
		}

		finalObj, err := j.fileStore.Commit(ctx, tempObj)
		if err != nil {
			return nil, err
		}

		files = append(files, job.OutputFile{
			ID:     finalObj.ID,
			Name:   r.NewName,
			Type:   job.OutputFileTypeTaxInvoice,
			URI:    finalObj.Path,
			Status: job.OutputFileReady,
		})
	}

	for _, e := range result.Errors {
		files = append(files, job.OutputFile{
			ID:     e.FileID,
			Name:   e.FileName,
			URI:    "",
			Status: job.OutputFileFailed,
			Type:   job.OutputFileTypeTaxInvoice,
		})
	}

	if err := j.fileStore.CleanupTemp(ctx, jb.ID); err != nil {
		return nil, err
	}

	outputManifest := job.OutputManifest{
		Version: 1,
		Summary: job.Summary{
			TotalFiles: len(result.Files) + len(result.Errors),
			Ready:      len(result.Files),
			Failed:     len(result.Errors),
		},
		Files: files,
	}

	return &outputManifest, nil
}
