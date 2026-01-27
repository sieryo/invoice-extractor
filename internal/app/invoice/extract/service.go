package extract

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/buyer"
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/infra/adapter/pdftool"
)

type InvoiceExtractorService struct {
	buyerRegistry    *buyer.Registry
	templateRegistry *template.Registry
}

func NewInvoiceExtractService(t *template.Registry, b *buyer.Registry) *InvoiceExtractorService {
	return &InvoiceExtractorService{
		buyerRegistry:    b,
		templateRegistry: t,
	}
}

func (i *InvoiceExtractorService) ExtractBatch(
	ctx context.Context,
	jobFiles []job.JobFile,
	templateID *string,
) (*BatchExtractResult, error) {

	var wg sync.WaitGroup

	errChan := make(chan BatchExtractError, len(jobFiles))
	invoiceChan := make(chan *invoice.Invoice, len(jobFiles))

	for _, file := range jobFiles {
		p := file.URI
		wg.Add(1)

		go func() {
			defer wg.Done()

			if _, err := os.Stat(p); err != nil {
				errChan <- BatchExtractError{FileID: file.ID, FileName: file.Name, Err: err}
				return
			}

			text, err := pdftool.ExtractText(ctx, p, pdftool.DefaultOptions())
			if err != nil {
				errChan <- BatchExtractError{FileID: file.ID, FileName: file.Name, Err: err}
				return
			}

			var tpl template.Template

			if templateID != nil {
				t, ok := i.templateRegistry.GetByIdentifier(*templateID)
				if !ok {
					errChan <- BatchExtractError{
						FileID:   file.ID,
						FileName: file.Name,
						Err:      fmt.Errorf("unknown template: %s", *templateID),
					}
					return
				}
				tpl = t
			} else {
				t, err := i.templateRegistry.Detect(text)
				if err != nil {
					errChan <- BatchExtractError{
						FileID:   file.ID,
						FileName: file.Name,
						Err:      fmt.Errorf("no template matched"),
					}
					return
				}
				tpl = t
			}

			inv, err := tpl.Parse(text)
			if err != nil {
				errChan <- BatchExtractError{FileID: file.ID,
					FileName: file.Name, Err: err}
				return
			}

			// Inject metadata
			inv.Metadata = &invoice.InvoiceMetadata{
				SourceFile: invoice.FileRef{
					ID:    file.ID,
					Name:  file.Name,
					Store: "Local", // Untuk sementara local dulu
				},
				TemplateID:  tpl.Identifier(),
				ExtractedAt: time.Now(),
			}

			if inv.Buyer != nil && inv.Buyer.Name != "" {
				if b, ok := i.buyerRegistry.GetByName(inv.Buyer.Name); ok {
					taxID := b.PrimaryTaxID()
					inv.Buyer.TaxID = &taxID
				}
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
