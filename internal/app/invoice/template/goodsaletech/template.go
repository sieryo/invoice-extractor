package goodsaletech

import (
	"strings"
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

func (t *GoodSaleTechTemplate) TaxID() string {
	return "0632707600016000"
}

func (t *GoodSaleTechTemplate) Match(raw string) bool {
	return strings.Contains(raw, "PT Good Sale Tech") &&
		strings.Contains(raw, "INVOICE")
}
