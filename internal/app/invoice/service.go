package invoice

import (
	"context"
	"encoding/json"

	"github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/job"
)

type InvoiceService struct {
	exporter  InvoiceExporter
	fileStore file.FileStore
}

func NewInvoiceService(
	exporter InvoiceExporter,
	fileStore file.FileStore,
) *InvoiceService {
	return &InvoiceService{
		exporter:  exporter,
		fileStore: fileStore,
	}
}

func (s *InvoiceService) LoadInvoice(
	ctx context.Context,
	jobID string,
	fileID string,
) (*Invoice, error) {

	b, err := s.fileStore.Read(ctx, jobID, fileID)

	if err != nil {
		return nil, err
	}

	var invoice Invoice

	if err := json.Unmarshal(b, &invoice); err != nil {
		return nil, err
	}

	return &invoice, nil

}

func (s *InvoiceService) Export(
	ctx context.Context,
	invoices []*Invoice,
) ([]byte, error) {
	return s.exporter.Export(ctx, invoices)
}

func (s *InvoiceService) LoadInvoicesByJob(
	ctx context.Context,
	j *job.Job,
) ([]*Invoice, ExportStat, error) {

	var stat ExportStat
	var invoices []*Invoice

	if j.OutputManifest == nil || len(j.OutputManifest.Files) == 0 {
		return nil, stat, nil
	}

	stat.Total = len(j.OutputManifest.Files)

	for _, f := range j.OutputManifest.Files {
		if f.Status != job.OutputFileReady && f.Status != job.OutputFileWarning {
			stat.Failed++
			continue
		}

		targetName := f.Name
		if f.StorageName != "" {
			targetName = f.StorageName
		}

		inv, err := s.LoadInvoice(ctx, j.ID, targetName)
		if err != nil {
			stat.Failed++
			continue
		}

		invoices = append(invoices, inv)
		stat.Success++
	}

	return invoices, stat, nil
}
