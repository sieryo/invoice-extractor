package rename

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax/extract"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
	"github.com/sieryo/invoice-extractor/internal/infra/jobrunner"
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

			info, err := s.extractor.Extract(ctx2, refFile)
			if err != nil {
				errChan <- shared.FileResultError{
					FileID:   refFile.ID,
					FileName: refFile.Name,
					Error:    err.Error(),
				}
				return
			}

			newName := fmt.Sprintf("%s - %s.pdf", info.Number, info.Buyer.Name)

			resultChan <- RenamedFile{
				ID:         refFile.ID,
				Name:       newName,
				SourceID:   refFile.ID,
				SourceName: refFile.Name,
				SourceURI:  refFile.Path,
			}
		}(inputFile)
	}

	go func() {
		wg.Wait()
		close(errChan)
		close(resultChan)
	}()

	result := &BatchRenameResult{}

	for {
		select {
		case err, ok := <-errChan:
			if ok {
				result.Errors = append(result.Errors, err)
			}
		case file, ok := <-resultChan:
			if ok {
				result.Files = append(result.Files, file)
			} else {
				return result, nil
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}
