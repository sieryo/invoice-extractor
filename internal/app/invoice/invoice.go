package invoice

import "time"

type FileRef struct {
	ID    string
	Name  string
	Store string
}

type Invoice struct {
	InvoiceNo   string
	InvoiceDate *time.Time
	PONumber    string
	Address     string

	Items    []Item
	Subtotal *Money
	VAT      *Money
	Total    *Money

	Metadata *InvoiceMetadata
}

type InvoiceMetadata struct {
	SourceFile  FileRef
	TemplateID  string
	ExtractedAt time.Time
}

// SEMENTARA
type Item struct {
	Name        string
	Quantity    int
	UnitPrice   *Money
	TotalAmount *Money
}

// SEMENTARA
type Money struct {
	Amount   float64
	Currency string
}
