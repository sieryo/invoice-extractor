package extract

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/buyer"
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
	"github.com/sieryo/invoice-extractor/internal/infra/adapter/pdftool"
	"github.com/sieryo/invoice-extractor/internal/infra/jobrunner"
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
	inputFiles []file.ResolvedFile,
	templateID *string,
) (*BatchExtractResult, error) {

	var wg sync.WaitGroup

	errChan := make(chan shared.FileResultError, len(inputFiles))
	invoiceChan := make(chan *invoice.Invoice, len(inputFiles))

	reporter := jobrunner.GetProgressReporter(ctx)

	var done int32
	total := int32(len(inputFiles))

	for _, f := range inputFiles {
		inputFile := f
		p := inputFile.Path

		wg.Add(1)
		go func(refFile file.ResolvedFile) {
			defer wg.Done()

			// context turunan, bukan overwrite
			ctx2 := ctx
			if reporter != nil {
				ctx2 = jobrunner.WithProgressReporter(ctx, reporter)
			}

			defer func() {
				if reporter != nil {
					current := atomic.AddInt32(&done, 1)
					progress := int(float64(current) / float64(total) * 100)
					reporter.Report(progress)
				}
			}()

			if _, err := os.Stat(p); err != nil {
				errChan <- shared.FileResultError{FileID: refFile.ID, FileName: refFile.Name, Err: err}
				return
			}

			text, err := pdftool.ExtractText(ctx2, p, pdftool.DefaultOptions())
			if err != nil {
				errChan <- shared.FileResultError{FileID: refFile.ID, FileName: refFile.Name, Err: err}
				return
			}

			var tpl template.Template
			if templateID != nil {
				t, ok := i.templateRegistry.GetByIdentifier(*templateID)
				if !ok {
					errChan <- shared.FileResultError{
						FileID:   refFile.ID,
						FileName: refFile.Name,
						Err:      fmt.Errorf("unknown template: %s", *templateID),
					}
					return
				}
				tpl = t
			} else {
				t, err := i.templateRegistry.Detect(text)
				if err != nil {
					errChan <- shared.FileResultError{
						FileID:   refFile.ID,
						FileName: refFile.Name,
						Err:      fmt.Errorf("no template matched"),
					}
					return
				}
				tpl = t
			}

			inv, err := tpl.Parse(text)
			if err != nil {
				errChan <- shared.FileResultError{
					FileID:   refFile.ID,
					FileName: refFile.Name,
					Err:      err,
				}
				return
			}

			// Inject metadata
			inv.Metadata = &invoice.InvoiceMetadata{
				SourceFile: invoice.FileRef{
					ID:   refFile.ID,
					Name: refFile.Name,
				},
				TemplateID:  tpl.Identifier(),
				ExtractedAt: time.Now(),
			}

			// buyer enrichment
			if inv.Buyer != nil && inv.Buyer.Name != "" {
				if b, ok := i.buyerRegistry.GetByName(inv.Buyer.Name); ok {
					taxID := b.PrimaryTaxID()
					tku := b.TKU()
					inv.Buyer.TaxID = &taxID
					inv.Buyer.TKU = &tku
				}
			}

			invoiceChan <- inv
		}(inputFile)
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
