package invoice

import (
	"context"

	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/app/shared"
)

type InvoiceService struct {
	fileStore shared.FileStore
}

func NewInvoiceService(fileStore shared.FileStore) *InvoiceService {
	return &InvoiceService{fileStore: fileStore}
}

func (s *InvoiceService) LoadInvoice(
	ctx context.Context,
	jobID string,
	file job.JobFile,
) ([]byte, error) {
	return s.fileStore.Read(ctx, jobID, file.Name)
}
