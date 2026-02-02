package rename

import (
	"context"
	"fmt"

	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax/extract"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
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
	inputFiles []file.ResolvedFile,
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
				Error:    err.Error(),
			})
			continue
		}

		newName := fmt.Sprintf("%s - %s.pdf", info.Number, info.Buyer.Name)

		results = append(results, RenamedFile{
			ID:         f.ID,
			Name:       newName,
			SourceID:   f.ID,
			SourceName: f.Name,
			SourceURI:  f.Path,
		})
	}

	return &BatchRenameResult{
		Files:  results,
		Errors: errors,
	}, nil
}
