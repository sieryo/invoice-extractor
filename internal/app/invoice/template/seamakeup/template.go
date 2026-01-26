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

func (t *SeaMakeupTemplate) Match(raw string) bool {
	return strings.Contains(raw, "PT Sea Makeup Beauty") &&
		strings.Contains(raw, "INVOICE")
}
