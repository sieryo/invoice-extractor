package parserhelper

import (
	"regexp"
)

var (
	ItemRegex = regexp.MustCompile(
		`^\d+\s+(?P<sku>[\w-]+)\s+(?P<name>.+?)\s{2,}(?P<qty>[\d.]+)\s+(?P<unit>[\d.,]+)\s+(?P<total>[\d.,]+)$`,
	)

	SubtotalRegex = regexp.MustCompile(`(?i)subtotal\s+(?P<amount>[\d.,]+)`)
	DiscountRegex = regexp.MustCompile(`(?i)discount\s+(?P<amount>[\d.,]+)`)
	VATRegex      = regexp.MustCompile(`(?i)vat\s+(?P<amount>[\d.,]+)`)
	TotalRegex    = regexp.MustCompile(`(?i)total\s+(?P<amount>[\d.,]+)`)
)
