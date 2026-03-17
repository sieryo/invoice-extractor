package extract

import (
	"context"
	"os"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
	"github.com/sieryo/invoice-extractor/internal/infra/adapter/pdftool"
	"github.com/sieryo/invoice-extractor/internal/infra/jobrunner"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

type TaxInvoiceExtractService struct {
}

type TaxInvoiceExtractResult struct {
	Invoice        *tax.TaxInvoice
	NormalizedText string
}

func NewTaxInvoiceExtractService() *TaxInvoiceExtractService {
	return &TaxInvoiceExtractService{}
}

func (s *TaxInvoiceExtractService) Extract(
	ctx context.Context,
	file file.ResolvedFile,
) (*TaxInvoiceExtractResult, error) {
	item, err := s.extractOne(ctx, file)
	if err != nil {
		return nil, err
	}

	return &TaxInvoiceExtractResult{
		Invoice:        item.Invoice,
		NormalizedText: item.NormalizedText,
	}, nil
}

func (s *TaxInvoiceExtractService) ExtractBatch(
	ctx context.Context,
	inputFiles []file.ResolvedFile,
) (*BatchExtractResult, error) {
	result := &BatchExtractResult{
		Items:  make([]BatchExtractItem, 0, len(inputFiles)),
		Errors: make([]shared.FileResultError, 0),
	}

	if len(inputFiles) == 0 {
		return result, nil
	}

	indexByID := make(map[string]int, len(inputFiles))
	for i, f := range inputFiles {
		indexByID[f.ID] = i
	}

	reporter := jobrunner.GetProgressReporter(ctx)
	var processed int32
	total := int32(len(inputFiles))

	itemChan := make(chan BatchExtractItem, len(inputFiles))
	errChan := make(chan shared.FileResultError, len(inputFiles))
	var wg sync.WaitGroup

	for _, f := range inputFiles {
		inputFile := f
		wg.Add(1)

		go func(refFile file.ResolvedFile) {
			defer wg.Done()
			defer func() {
				current := atomic.AddInt32(&processed, 1)
				if reporter != nil {
					progress := int(float64(current) / float64(total) * 100)
					reporter.Report(progress)
				}
			}()

			ctx2 := ctx
			if reporter != nil {
				ctx2 = jobrunner.WithProgressReporter(ctx, reporter)
			}

			item, err := s.extractOne(ctx2, refFile)
			if err != nil {
				errChan <- shared.FileResultError{
					FileID:   refFile.ID,
					FileName: refFile.Name,
					Error:    err.Error(),
				}
				return
			}

			itemChan <- item
		}(inputFile)
	}

	go func() {
		wg.Wait()
		close(itemChan)
		close(errChan)
	}()

	itemsDone := false
	errDone := false
	for !(itemsDone && errDone) {
		select {
		case item, ok := <-itemChan:
			if !ok {
				itemsDone = true
				continue
			}
			result.Items = append(result.Items, item)
		case e, ok := <-errChan:
			if !ok {
				errDone = true
				continue
			}
			result.Errors = append(result.Errors, e)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	sort.Slice(result.Items, func(i, j int) bool {
		return indexByID[result.Items[i].SourceFile.ID] < indexByID[result.Items[j].SourceFile.ID]
	})

	sort.Slice(result.Errors, func(i, j int) bool {
		return indexByID[result.Errors[i].FileID] < indexByID[result.Errors[j].FileID]
	})

	return result, nil
}

func (s *TaxInvoiceExtractService) extractOne(
	ctx context.Context,
	file file.ResolvedFile,
) (BatchExtractItem, error) {
	if _, err := os.Stat(file.Path); err != nil {
		return BatchExtractItem{}, err
	}

	text, err := pdftool.ExtractText(ctx, file.Path, pdftool.DefaultOptions())
	if err != nil {
		return BatchExtractItem{}, err
	}

	info, normalized, err := ParseTaxInvoiceText(file.Name, text)
	if err != nil {
		return BatchExtractItem{}, err
	}

	return BatchExtractItem{
		SourceFile: invoice.FileRef{
			ID:   file.ID,
			Name: file.Name,
		},
		Invoice:        info,
		NormalizedText: normalized,
		Warnings:       buildWarnings(info),
	}, nil
}

func buildWarnings(info *tax.TaxInvoice) []string {
	if info == nil {
		return []string{"missing tax invoice payload"}
	}

	warnings := make([]string, 0, 8)
	warnings = append(warnings, info.Anomalies...)
	if info.Number == "" {
		warnings = append(warnings, "missing tax invoice number")
	}

	if info.Buyer == nil {
		warnings = append(warnings, "missing buyer block")
		return warnings
	}

	if info.Buyer.Name == "" {
		warnings = append(warnings, "missing buyer name")
	}

	if info.Buyer.TaxID == nil || *info.Buyer.TaxID == "" {
		warnings = append(warnings, "missing buyer tax id")
		return warnings
	}

	digits := helper.DigitsOnly(*info.Buyer.TaxID)
	if digits == "" {
		warnings = append(warnings, "missing buyer tax id")
		return warnings
	}

	if !helper.IsValidTaxID(digits) {
		warnings = append(warnings, "invalid buyer tax id")
	}

	return warnings
}
