package invoice

import (
	"time"

	"github.com/sieryo/invoice-extractor/pkg/helper"
)

type FileRef struct {
	ID   string
	Name string
}

type Party struct {
	Name    string
	TaxID   *string // NPWP / NIK
	TKU     *string // ID Tempat Kegiatan Usaha (DJP e-Faktur) TKU
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

type Item struct {
	Name        string
	Quantity    int
	UnitPrice   *Money
	TotalAmount *Money
	TaxRate     float64 // 0.12
}

func (i *Item) GetTaxBase() float64 {
	total := i.GetTotalAmount()
	if total == 0 {
		return 0
	}

	return helper.Round2(total * 11 / 12)
}

func (i *Item) GetUnitPrice() float64 {
	if i.UnitPrice == nil {
		return 0
	}
	return i.UnitPrice.Amount
}

func (i *Item) GetTotalAmount() float64 {
	if i.TotalAmount == nil {
		return 0
	}
	return i.TotalAmount.Amount
}

func (i *Item) GetTotalTax() float64 {
	taxBase := i.GetTaxBase()
	if taxBase == 0 || i.TaxRate <= 0 {
		return 0
	}

	return helper.Round2(taxBase * i.TaxRate)
}

func (i *Item) GetNetAmount() float64 {
	return i.GetTotalAmount() - i.GetTotalTax()
}

// SEMENTARA
type Money struct {
	Amount   float64
	Currency string
}
