package parserhelper

import (
	"regexp"
)

var (
	summaryAmountPattern = `(?P<amount>(?:(?:idr|rp)\.?\s*)?[\d.,]+)`

	ItemRegex = regexp.MustCompile(
		`^\d+\.?\s+(?P<sku>\S+)\s+(?P<name>.+?)\s{2,}(?P<qty>[\d.,]+)\s+(?P<unit>(?i:(?:idr|rp)\.?\s*)?[\d.,]+)\s+(?P<total>(?i:(?:idr|rp)\.?\s*)?[\d.,]+)$`,
	)

	SubtotalRegex = regexp.MustCompile(`(?i)\bsubtotal\b[^\d]*(?:` + summaryAmountPattern + `)`)
	DiscountRegex = regexp.MustCompile(`(?i)\bdiscount\b[^\d]*(?:` + summaryAmountPattern + `)`)
	VATRegex      = regexp.MustCompile(`(?i)\bvat\b[^\d]*(?:` + summaryAmountPattern + `)`)
	TotalRegex    = regexp.MustCompile(`(?i)\btotal\b[^\d]*(?:` + summaryAmountPattern + `)`)
)
