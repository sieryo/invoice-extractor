package extract

import (
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
)

type BatchExtractResult struct {
	Invoices []*invoice.Invoice
	Errors   []shared.FileResultError
}
