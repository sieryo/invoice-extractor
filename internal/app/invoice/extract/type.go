package extract

import "github.com/sieryo/invoice-extractor/internal/app/invoice"

type BatchExtractError struct {
	FileID   string
	FileName string
	Err      error
}

type BatchExtractResult struct {
	Invoices []*invoice.Invoice
	Errors   []BatchExtractError
}
