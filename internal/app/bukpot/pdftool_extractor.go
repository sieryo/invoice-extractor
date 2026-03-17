package bukpot

import (
	"context"

	"github.com/sieryo/invoice-extractor/internal/infra/adapter/pdftool"
)

type PDFToolExtractor struct{}

func NewPDFToolExtractor() *PDFToolExtractor {
	return &PDFToolExtractor{}
}

func (e *PDFToolExtractor) ExtractText(
	ctx context.Context,
	pdfPath string,
	opts pdftool.ExtractOptions,
) (string, error) {
	return pdftool.ExtractText(ctx, pdfPath, opts)
}
