package giaprima

import (
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

type GiaPrimaTemplate struct{}

func NewGiaPrimaTemplate() *GiaPrimaTemplate {
	return &GiaPrimaTemplate{}
}

func (t *GiaPrimaTemplate) Identifier() string {
	return "GiaPrima"
}

func (t *GiaPrimaTemplate) Alias() string {
	return "GPI"
}

func (t *GiaPrimaTemplate) Name() string {
	return "PT Gia Prima Indonesia"
}

func (t *GiaPrimaTemplate) Seller() *invoice.Party {
	return &invoice.Party{
		Name:  t.Name(),
		TaxID: helper.Ptr("0538490020034000"),
		TKU:   helper.Ptr("0538490020034000000000"),
	}
}

func (t *GiaPrimaTemplate) Match(raw string) bool {
	return strings.Contains(raw, strings.ToLower(t.Name())) &&
		strings.Contains(raw, strings.ToLower("INVOICE"))
}

func (t *GiaPrimaTemplate) FormatInvoiceNumber(inv *invoice.Invoice) string {
	if inv.Number == "" {
		return ""
	}
	return "Invoice No : " + inv.Number
}
