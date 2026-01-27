package invoice

import (
	"context"
	"encoding/json"

	"github.com/sieryo/invoice-extractor/internal/app/shared"
)

type InvoiceService struct {
	exporter  InvoiceExporter
	fileStore shared.FileStore
}

func NewInvoiceService(
	exporter InvoiceExporter,
	fileStore shared.FileStore,
) *InvoiceService {
	return &InvoiceService{
		exporter:  exporter,
		fileStore: fileStore,
	}
}

func (s *InvoiceService) LoadInvoice(
	ctx context.Context,
	jobID string,
	name string,
) (*Invoice, error) {

	b, err := s.fileStore.Read(ctx, jobID, name)

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
