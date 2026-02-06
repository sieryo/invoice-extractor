package goodsaletech

import (
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

type GoodSaleTechTemplate struct{}

func NewGoodSaleTechTemplate() *GoodSaleTechTemplate {
	return &GoodSaleTechTemplate{}
}

func (t *GoodSaleTechTemplate) Identifier() string {
	return "GoodSaleTech"
}

func (t *GoodSaleTechTemplate) Name() string {
	return "PT Good Sale Tech"
}

func (t *GoodSaleTechTemplate) Alias() string {
	return "GST"
}

func (t *GoodSaleTechTemplate) Seller() *invoice.Party {
	return &invoice.Party{
		Name:  t.Name(),
		TaxID: helper.Ptr("0902329143011000"),
		TKU:   helper.Ptr("0902329143011000000000"),
	}
}

func (t *GoodSaleTechTemplate) Match(raw string) bool {
	return strings.Contains(raw, strings.ToLower(t.Name())) &&
		strings.Contains(raw, strings.ToLower("INVOICE"))
}

func (t *GoodSaleTechTemplate) FormatInvoiceNumber(inv *invoice.Invoice) string {
	return inv.Number
}
