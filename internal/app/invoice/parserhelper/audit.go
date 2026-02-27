package parserhelper

import (
	"fmt"
	"math"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
)

func BuildParseWarnings(inv *invoice.Invoice) []string {
	if inv == nil {
		return []string{"invoice parse returned nil"}
	}

	warnings := make([]string, 0, 8)

	if inv.Number == "" {
		warnings = append(warnings, "missing invoice number")
	}
	if inv.Date == nil {
		warnings = append(warnings, "missing invoice date")
	}
	if inv.Buyer == nil || inv.Buyer.Name == "" {
		warnings = append(warnings, "missing buyer name")
	}
	if len(inv.Items) == 0 {
		warnings = append(warnings, "no items parsed")
	}
	if inv.Subtotal == nil {
		warnings = append(warnings, "missing subtotal")
	}
	if inv.Total == nil {
		warnings = append(warnings, "missing total")
	}

	if inv.Subtotal != nil && inv.Total != nil {
		expected := inv.Subtotal.Amount
		if inv.VAT != nil {
			expected += inv.VAT.Amount
		}
		if inv.Discount != nil {
			expected -= inv.Discount.Amount
		}

		if math.Abs(expected-inv.Total.Amount) > 1 {
			warnings = append(
				warnings,
				fmt.Sprintf("total mismatch: expected %.2f but got %.2f", expected, inv.Total.Amount),
			)
		}
	}

	return warnings
}
