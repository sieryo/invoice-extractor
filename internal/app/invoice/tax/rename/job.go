package rename

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/job"
)

type TaxInvoiceRenameJob struct {
	fileRepo      file.Repository
	renameService *TaxInvoiceRenameService
	fileStore     file.FileStore
}

func NewTaxInvoiceRenameJob(
	renameService *TaxInvoiceRenameService,
	fileStore file.FileStore,
	fileRepo file.Repository,
) *TaxInvoiceRenameJob {
	return &TaxInvoiceRenameJob{
		fileRepo:      fileRepo,
		renameService: renameService,
		fileStore:     fileStore,
	}
}

func (j *TaxInvoiceRenameJob) Handle(ctx context.Context, jb *job.Job) (*job.OutputManifest, error) {
	var payload Payload
	if err := json.Unmarshal(jb.InputPayload, &payload); err != nil {
		return nil, err
	}

	resolved := make([]file.ResolvedFile, 0, len(payload.InputFiles))

	for _, ref := range payload.InputFiles {
		f, err := j.fileRepo.FindByID(ctx, ref.ID)
		if err != nil {
			return nil, err
		}

		resolved = append(resolved, file.ResolvedFile{
			FileRef: ref,
			Path:    f.Path,
		})
	}

	// proses rename batch
	result, err := j.renameService.RenameBatch(ctx, resolved)
	if err != nil {
		return nil, err
	}

	for _, audit := range result.Audits {
		data, err := json.Marshal(audit)
		if err != nil {
			fmt.Printf("failed to marshal audit for %s: %v\n", audit.SourceFile.Name, err)
			continue
		}

		name := fmt.Sprintf("audit_%s.json", audit.SourceFile.ID)
		if _, err := j.fileStore.SaveAudit(ctx, jb.ID, name, data); err != nil {
			fmt.Printf("failed to save audit for %s: %v\n", audit.SourceFile.Name, err)
		}
	}

	var files []job.OutputFile

	for _, r := range result.Files {

		data, err := os.ReadFile(r.SourceURI)
		if err != nil {
			// append error
			continue
		}

		tempObj, err := j.fileStore.SaveTemp(ctx, jb.ID, r.Name, data)
		if err != nil {
			return nil, err
		}

		finalObj, err := j.fileStore.Commit(ctx, tempObj)
		if err != nil {
			return nil, err
		}

		files = append(files, job.OutputFile{
			ID:             finalObj.ID,
			Name:           r.Name,
			SourceFileID:   &r.SourceID,
			StorageName:    finalObj.ID + filepath.Ext(r.Name),
			SourceFileName: r.SourceName,
			Type:           job.OutputFileTypeTaxInvoice,
			URI:            finalObj.Path,
			Status:         job.OutputFileReady,
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
