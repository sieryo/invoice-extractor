package extract

import (
	"context"
	"fmt"
	"os"
	"sort"
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

	var extracted int32
	total := int32(len(inputFiles))

	for _, f := range inputFiles {
		inputFile := f
		p := inputFile.Path

		wg.Add(1)
		go func(refFile file.ResolvedFile) {
			defer wg.Done()

			defer func() {
				current := atomic.AddInt32(&extracted, 1)

				if reporter != nil {
					p := float64(current) / float64(total)
					progress := int(p * 70)
					reporter.Report(progress)
				}
			}()

			// context turunan, bukan overwrite
			ctx2 := ctx
			if reporter != nil {
				ctx2 = jobrunner.WithProgressReporter(ctx, reporter)
			}

			if _, err := os.Stat(p); err != nil {
				errChan <- shared.FileResultError{FileID: refFile.ID, FileName: refFile.Name, Error: err.Error()}
				return
			}

			text, err := pdftool.ExtractText(ctx2, p, pdftool.DefaultOptions())
			if err != nil {
				errChan <- shared.FileResultError{FileID: refFile.ID, FileName: refFile.Name, Error: err.Error()}
				return
			}

			var tpl template.Template
			if templateID != nil {
				t, ok := i.templateRegistry.GetByIdentifier(*templateID)
				if !ok {
					errChan <- shared.FileResultError{
						FileID:   refFile.ID,
						FileName: refFile.Name,
						Error:    fmt.Sprintf("unknown template: %s", *templateID),
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
						Error:    "no template matched",
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
					Error:    err.Error(),
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

	invoiceDone := false
	errorDone := false

	for !(invoiceDone && errorDone) {
		select {
		case err, ok := <-errChan:
			if !ok {
				errorDone = true
				continue
			}
			result.Errors = append(result.Errors, err)

		case inv, ok := <-invoiceChan:
			if !ok {
				invoiceDone = true
				continue
			}
			result.Invoices = append(result.Invoices, inv)

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	sort.Slice(result.Invoices, func(i, j int) bool {
		di := result.Invoices[i].Date
		dj := result.Invoices[j].Date

		// invoice tanpa tanggal taruh paling belakang
		if di == nil && dj == nil {
			return false
		}
		if di == nil {
			return false
		}
		if dj == nil {
			return true
		}

		// oldest dulu
		return di.Before(*dj)
	})

	if reporter != nil {
		total := len(result.Invoices)
		for idx := range result.Invoices {
			progress := int(float64(idx+1) / float64(total) * 100)
			reporter.Report(progress)
		}
	}

	return result, nil
}
