package invoice

import (
	"fmt"
	"time"

	"github.com/sieryo/invoice-extractor/pkg/helper"
)

type FileRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Party struct {
	Name    string  `json:"name"`
	TaxID   *string `json:"tax_id"` // NPWP / NIK
	TKU     *string `json:"tku"`    // ID Tempat Kegiatan Usaha (DJP e-Faktur)
	Address *string `json:"address"`
}

type Invoice struct {
	Number      string     `json:"number"`
	Date        *time.Time `json:"date"`
	OrderNumber string     `json:"order_number"`
	PONumber    string     `json:"po_number"`

	Buyer  *Party `json:"buyer"`
	Seller *Party `json:"seller"`

	Items    []Item `json:"items"`
	Subtotal *Money `json:"subtotal"`
	Discount *Money `json:"discount"`
	VAT      *Money `json:"vat"`
	Total    *Money `json:"total"`

	Metadata *InvoiceMetadata `json:"metadata"`
}

type InvoiceMetadata struct {
	SourceFile  FileRef   `json:"source_file"`
	TemplateID  string    `json:"template_id"`
	ExtractedAt time.Time `json:"extracted_at"`
}

type Item struct {
	Sku         string  `json:"sku"`
	Name        string  `json:"name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   *Money  `json:"unit_price"`
	Discount    *Money  `json:"discount"`
	TotalAmount *Money  `json:"total_amount"`
	TaxRate     float64 `json:"tax_rate"` // 0.12
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

func (i *Item) GetExportedName() string {
	return fmt.Sprintf("%s %s", i.Sku, i.Name)
}

func (i *Item) GetNetAmount() float64 {
	return i.GetTotalAmount() - i.GetTotalTax()
}

func FindMostExpensiveItem(items []Item) *Item {
	var maxItem *Item

	for i := range items {
		it := &items[i]

		if it.TotalAmount == nil {
			continue
		}

		if maxItem == nil || it.TotalAmount.Amount > maxItem.TotalAmount.Amount {
			maxItem = it
		}
	}

	return maxItem
}

func ApplyDiscountToMostExpensiveItem(inv *Invoice) {
	if inv.Discount == nil || len(inv.Items) == 0 {
		return
	}

	item := FindMostExpensiveItem(inv.Items)
	if item == nil || item.TotalAmount == nil {
		return
	}

	discount := inv.Discount.Amount

	// Minus gpp untuk sementara
	item.TotalAmount.Amount -= discount

	item.Discount = &Money{
		Amount: discount,
	}
}

// SEMENTARA
type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}
