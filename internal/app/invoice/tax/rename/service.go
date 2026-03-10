package rename

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax/extract"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
	"github.com/sieryo/invoice-extractor/internal/infra/jobrunner"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

type TaxInvoiceRenameService struct {
	extractor *extract.TaxInvoiceExtractService
}

func NewTaxInvoiceRenameService(extractor *extract.TaxInvoiceExtractService) *TaxInvoiceRenameService {
	return &TaxInvoiceRenameService{
		extractor: extractor,
	}
}

func (s *TaxInvoiceRenameService) RenameBatch(
	ctx context.Context,
	inputFiles []file.ResolvedFile,
) (*BatchRenameResult, error) {

	var wg sync.WaitGroup

	resultChan := make(chan RenamedFile, len(inputFiles))
	errChan := make(chan shared.FileResultError, len(inputFiles))
	auditChan := make(chan TaxInvoiceAudit, len(inputFiles))

	reporter := jobrunner.GetProgressReporter(ctx)

	var done int32
	total := int32(len(inputFiles))

	for _, f := range inputFiles {
		inputFile := f

		wg.Add(1)
		go func(refFile file.ResolvedFile) {
			defer wg.Done()

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

			audit := TaxInvoiceAudit{
				SourceFile: invoice.FileRef{
					ID:   refFile.ID,
					Name: refFile.Name,
				},
				ExtractedAt: time.Now(),
			}

			info, err := s.extractor.Extract(ctx2, refFile)
			if err != nil {
				errChan <- shared.FileResultError{
					FileID:   refFile.ID,
					FileName: refFile.Name,
					Error:    err.Error(),
				}
				audit.Error = err.Error()
				auditChan <- audit
				return
			}

			if info.Invoice != nil {
				audit.Number = info.Invoice.Number
				audit.NormalizedText = info.NormalizedText

				if info.Invoice.Number == "" {
					audit.Warnings = append(audit.Warnings, "missing tax invoice number")
				}

				if info.Invoice.Buyer != nil {
					audit.Buyer.ParsedName = info.Invoice.Buyer.Name
					if info.Invoice.Buyer.Address != nil {
						audit.Buyer.Address = *info.Invoice.Buyer.Address
					}

					if info.Invoice.Buyer.Name == "" {
						audit.Warnings = append(audit.Warnings, "missing buyer name")
					}

					if info.Invoice.Buyer.TaxID != nil {
						rawTaxID := *info.Invoice.Buyer.TaxID
						digits := helper.DigitsOnly(rawTaxID)

						audit.Buyer.ParsedTaxID = rawTaxID
						audit.Buyer.TaxIDKind = helper.TaxIDKind(digits)
						audit.Buyer.TaxIDValid = helper.IsValidTaxID(digits)

						if digits == "" {
							audit.Warnings = append(audit.Warnings, "missing buyer tax id")
						} else if !audit.Buyer.TaxIDValid {
							audit.Warnings = append(audit.Warnings, "invalid buyer tax id")
						}
					} else {
						audit.Warnings = append(audit.Warnings, "missing buyer tax id")
					}
				} else {
					audit.Warnings = append(audit.Warnings, "missing buyer block")
				}
			}

			number := ""
			buyerName := ""
			if info.Invoice != nil {
				number = info.Invoice.Number
				if info.Invoice.Buyer != nil {
					buyerName = info.Invoice.Buyer.Name
				}
			}

			newName := fmt.Sprintf("%s - %s.pdf", number, buyerName)
			audit.RenamedTo = newName

			resultChan <- RenamedFile{
				ID:         refFile.ID,
				Name:       newName,
				SourceID:   refFile.ID,
				SourceName: refFile.Name,
				SourceURI:  refFile.Path,
			}

			auditChan <- audit
		}(inputFile)
	}

	go func() {
		wg.Wait()
		close(errChan)
		close(resultChan)
		close(auditChan)
	}()

	result := &BatchRenameResult{}

	resultDone := false
	errorDone := false
	auditDone := false

	for !(resultDone && errorDone && auditDone) {
		select {
		case err, ok := <-errChan:
			if !ok {
				errorDone = true
				continue
			}
			result.Errors = append(result.Errors, err)

		case file, ok := <-resultChan:
			if !ok {
				resultDone = true
				continue
			}
			result.Files = append(result.Files, file)

		case audit, ok := <-auditChan:
			if !ok {
				auditDone = true
				continue
			}
			result.Audits = append(result.Audits, audit)

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return result, nil
}
