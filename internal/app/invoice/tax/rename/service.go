package rename

import (
	"context"
	"fmt"

	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax/extract"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
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
	inputFiles []job.InputFile,
) (*BatchRenameResult, error) {

	var (
		results []RenamedFile
		errors  []shared.FileResultError
	)

	for _, f := range inputFiles {
		info, err := s.extractor.Extract(ctx, f)
		if err != nil {
			errors = append(errors, shared.FileResultError{
				FileID:   f.ID,
				FileName: f.Name,
				Err:      err,
			})
			continue
		}

		newName := fmt.Sprintf("%s - %s", info.Number, info.Buyer.Name)

		results = append(results, RenamedFile{
			FileID:    f.ID,
			OldName:   f.Name,
			NewName:   newName,
			SourceURI: f.URI,
		})
	}

	return &BatchRenameResult{
		Files:  results,
		Errors: errors,
	}, nil
}
