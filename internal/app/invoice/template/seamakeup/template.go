package seamakeup

import (
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

type SeaMakeupTemplate struct{}

func NewSeaMakeupTemplate() *SeaMakeupTemplate {
	return &SeaMakeupTemplate{}
}

func (t *SeaMakeupTemplate) Identifier() string {
	return "SeaMakeup"
}

func (t *SeaMakeupTemplate) Alias() string {
	return "SMB"
}

func (t *SeaMakeupTemplate) Name() string {
	return "PT Sea Makeup Beauty"
}

func (t *SeaMakeupTemplate) Seller() *invoice.Party {
	return &invoice.Party{
		Name:  t.Name(),
		TaxID: helper.Ptr("0632707600016000"),
		TKU:   helper.Ptr("0632707600016000000000"),
	}
}

func (t *SeaMakeupTemplate) Match(raw string) bool {
	return strings.Contains(raw, strings.ToLower(t.Name())) &&
		strings.Contains(raw, strings.ToLower("INVOICE"))
}

func (t *SeaMakeupTemplate) FormatInvoiceNumber(inv *invoice.Invoice) string {
	if inv.Number == "" {
		return ""
	}
	return "Invoice No : " + inv.Number
}
