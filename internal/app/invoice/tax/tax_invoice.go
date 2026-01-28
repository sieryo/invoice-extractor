package tax

import "github.com/sieryo/invoice-extractor/internal/app/invoice"

type TaxInvoice struct {
	Number string
	Buyer  *invoice.Party
}
