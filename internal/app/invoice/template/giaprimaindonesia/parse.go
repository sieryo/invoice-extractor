package giaprima

import (
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/parserhelper"
)

func (t *GiaPrimaTemplate) Parse(raw string) (*invoice.Invoice, error) {
	return parserhelper.ParseTemplateA(raw, t.Normalize, t.Seller())
}
