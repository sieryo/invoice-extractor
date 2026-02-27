package goodsaletech

import (
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/parserhelper"
)

func (t *GoodSaleTechTemplate) Parse(raw string) (*invoice.Invoice, error) {
	return parserhelper.ParseTemplateA(raw, t.Normalize, t.Seller())
}
