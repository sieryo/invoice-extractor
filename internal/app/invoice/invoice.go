package invoice

import "time"

type FileRef struct {
	ID    string
	Name  string
	Store string
}

type Party struct {
	Name    string
	TaxID   *string // NPWP / NIK
	Address *string
}

type Invoice struct {
	Number      string
	Date        *time.Time
	OrderNumber string
	PONumber    string

	Buyer  *Party
	Seller *Party

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
