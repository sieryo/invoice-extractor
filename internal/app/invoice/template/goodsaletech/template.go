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

func (t *GoodSaleTechTemplate) Seller() *invoice.Party {
	return &invoice.Party{
		Name:  t.Name(),
		TaxID: helper.Ptr("0632707600016000"),
		TKU:   helper.Ptr("0632707600016000000000"),
	}
}

func (t *GoodSaleTechTemplate) Match(raw string) bool {
	return strings.Contains(raw, "PT Good Sale Tech") &&
		strings.Contains(raw, "INVOICE")
}
