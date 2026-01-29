package extract

import (
	"context"
	"os"

	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/infra/adapter/pdftool"
)

type TaxInvoiceExtractService struct {
}

func NewTaxInvoiceExtractService() *TaxInvoiceExtractService {
	return &TaxInvoiceExtractService{}
}

func (s *TaxInvoiceExtractService) Extract(
	ctx context.Context,
	file file.ResolvedFile,
) (*tax.TaxInvoice, error) {

	if _, err := os.Stat(file.Path); err != nil {
		return nil, err
	}

	text, err := pdftool.ExtractText(ctx, file.Path, pdftool.DefaultOptions())
	if err != nil {
		return nil, err
	}

	info, err := ParseTaxInvoiceText(text)
	if err != nil {
		return nil, err
	}

	return info, nil
}
