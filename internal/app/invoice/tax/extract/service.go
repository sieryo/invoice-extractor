package extract

import (
	"context"
	"os"

	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax"
	"github.com/sieryo/invoice-extractor/internal/app/job"
	"github.com/sieryo/invoice-extractor/internal/infra/adapter/pdftool"
)

type TaxInvoiceExtractService struct {
}

func NewTaxInvoiceExtractService() *TaxInvoiceExtractService {
	return &TaxInvoiceExtractService{}
}

func (s *TaxInvoiceExtractService) Extract(
	ctx context.Context,
	file job.InputFile,
) (*tax.TaxInvoice, error) {

	if _, err := os.Stat(file.URI); err != nil {
		return nil, err
	}

	text, err := pdftool.ExtractText(ctx, file.URI, pdftool.DefaultOptions())
	if err != nil {
		return nil, err
	}

	info, err := ParseTaxInvoiceText(text)
	if err != nil {
		return nil, err
	}

	return info, nil
}
