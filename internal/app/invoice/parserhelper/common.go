package parserhelper

import (
	"regexp"
)

var (
	ItemRegex = regexp.MustCompile(
		`^\d+\s+[\w-]+\s+(?P<name>.+?)\s{2,}(?P<qty>[\d.]+)\s+(?P<unit>[\d.,]+)\s+(?P<total>[\d.,]+)$`,
	)

	SubtotalRegex = regexp.MustCompile(`Subtotal\s+(?P<amount>[\d.,]+)`)
	VATRegex      = regexp.MustCompile(`^VAT\s+(?P<amount>[\d.,]+)`)
	TotalRegex    = regexp.MustCompile(`^TOTAL\s+(?P<amount>[\d.,]+)`)
)
