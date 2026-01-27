package seamakeup

import (
	"strings"
)

type SeaMakeupTemplate struct{}

func NewSeaMakeupTemplate() *SeaMakeupTemplate {
	return &SeaMakeupTemplate{}
}

func (t *SeaMakeupTemplate) Identifier() string {
	return "SeaMakeup"
}

func (t *SeaMakeupTemplate) Name() string {
	return "PT Sea Makeup Beauty"
}

func (t *SeaMakeupTemplate) TaxID() string {
	return "0632707600016000"
}

func (t *SeaMakeupTemplate) Match(raw string) bool {
	return strings.Contains(raw, "PT Sea Makeup Beauty") &&
		strings.Contains(raw, "INVOICE")
}
