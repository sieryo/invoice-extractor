package extract

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/sieryo/invoice-extractor/internal/infra/adapter/pdftool"
)

type InvoiceExtractorService struct {
}

func NewInvoiceExtractService() *InvoiceExtractorService {
	return &InvoiceExtractorService{}
}

func (i *InvoiceExtractorService) ExtractBatch(ctx context.Context, pdfPaths []string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(pdfPaths))

	for _, path := range pdfPaths {
		path := path
		wg.Add(1)
		go func() {
			defer wg.Done()

			if _, err := os.Stat(path); os.IsNotExist(err) {
				errChan <- fmt.Errorf("file not found: %s", path)
				return
			}

			opts := pdftool.ExtractOptions{
				Layout: true,
			}
			text, err := pdftool.ExtractText(ctx, path, opts)
			if err != nil {
				errChan <- fmt.Errorf("failed to extract %s: %w", path, err)
				return
			}

			_ = text
		}()
	}

	wg.Wait()
	close(errChan)

	// Return first error if any
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}
