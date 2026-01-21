package invoice

import "time"

type Invoice struct {
	InvoiceNo   *string
	InvoiceDate *time.Time
	PONumber    *string
	Items       []Item
	Total       *Money
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
