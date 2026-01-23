package extract

import "context"

type InvoiceExtractorService struct {
}

func (i *InvoiceExtractorService) ExtractBatch(ctx context.Context, pdfPaths []string) error {
	return nil
}
