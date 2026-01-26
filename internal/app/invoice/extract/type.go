package extract

import "github.com/sieryo/invoice-extractor/internal/app/invoice"

type BatchExtractError struct {
	File string
	Err  error
}

type BatchExtractResult struct {
	Invoices []*invoice.Invoice
	Errors   []BatchExtractError
}
