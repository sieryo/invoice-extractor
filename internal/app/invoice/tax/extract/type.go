package extract

import (
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
)

type BatchExtractItem struct {
	SourceFile     invoice.FileRef
	Invoice        *tax.TaxInvoice
	NormalizedText string
	Warnings       []string
}

type BatchExtractResult struct {
	Items  []BatchExtractItem
	Errors []shared.FileResultError
}
