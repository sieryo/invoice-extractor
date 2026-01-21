package seamakeup

import (
	"strings"
)

type SeaMakeupTemplate struct{}

func (t *SeaMakeupTemplate) Name() string {
	return "PT Sea Makeup Beauty"
}

func (t *SeaMakeupTemplate) Match(raw string) bool {
	return strings.Contains(raw, "PT Sea Makeup Beauty") &&
		strings.Contains(raw, "INVOICE")
}

func (t *SeaMakeupTemplate) Normalize(raw string) (string, error) {
	// merge item lines khas template ini
	return normalizeSeaMakeup(raw), nil
}
