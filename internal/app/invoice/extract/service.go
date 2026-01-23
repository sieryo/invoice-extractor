package extract

import "context"

type InvoiceExtractorService struct {
}

func NewInvoiceExtractService() *InvoiceExtractorService {
	return &InvoiceExtractorService{}
}

func (i *InvoiceExtractorService) ExtractBatch(ctx context.Context, pdfPaths []string) error {
	return nil
}
