package extract

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template"
	"github.com/sieryo/invoice-extractor/internal/infra/adapter/pdftool"
)

type InvoiceExtractorService struct {
	templateRegistry *template.Registry
}

func NewInvoiceExtractService(r *template.Registry) *InvoiceExtractorService {
	return &InvoiceExtractorService{
		templateRegistry: r,
	}
}

func (i *InvoiceExtractorService) ExtractBatch(
	ctx context.Context,
	pdfPaths []string,
	templateID *string,
) (*BatchExtractResult, error) {

	var wg sync.WaitGroup

	errChan := make(chan BatchExtractError, len(pdfPaths))
	invoiceChan := make(chan *invoice.Invoice, len(pdfPaths))

	for _, path := range pdfPaths {
		p := path
		wg.Add(1)

		go func() {
			defer wg.Done()

			if _, err := os.Stat(p); err != nil {
				errChan <- BatchExtractError{File: p, Err: err}
				return
			}

			text, err := pdftool.ExtractText(ctx, p, pdftool.DefaultOptions())
			if err != nil {
				errChan <- BatchExtractError{File: p, Err: err}
				return
			}

			var tpl template.Template

			if templateID != nil {
				t, ok := i.templateRegistry.GetByIdentifier(*templateID)
				if !ok {
					errChan <- BatchExtractError{
						File: p,
						Err:  fmt.Errorf("unknown template: %s", *templateID),
					}
					return
				}
				tpl = t
			} else {
				t, err := i.templateRegistry.Detect(text)
				if err != nil {
					errChan <- BatchExtractError{
						File: p,
						Err:  fmt.Errorf("no template matched"),
					}
					return
				}
				tpl = t
			}

			inv, err := tpl.Parse(text)
			if err != nil {
				errChan <- BatchExtractError{File: p, Err: err}
				return
			}

			// metadata injection
			inv.Metadata = &invoice.InvoiceMetadata{
				SourceFile:  invoice.FileRef{},
				TemplateID:  tpl.Identifier(),
				ExtractedAt: time.Now(),
			}

			invoiceChan <- inv
		}()
	}

	go func() {
		wg.Wait()
		close(errChan)
		close(invoiceChan)
	}()

	result := &BatchExtractResult{}

	for {
		select {
		case err, ok := <-errChan:
			if ok {
				result.Errors = append(result.Errors, err)
			}
		case inv, ok := <-invoiceChan:
			if ok {
				result.Invoices = append(result.Invoices, inv)
			} else {
				return result, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
