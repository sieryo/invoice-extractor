package seamakeup

import (
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/parserhelper"
)

/*
========================
MAIN PARSER
========================
*/

func (t *SeaMakeupTemplate) Parse(raw string) (*invoice.Invoice, error) {
	norm := t.Normalize(raw)

	lines := strings.Split(norm, "\n")
	inv := &invoice.Invoice{
		Buyer:  &invoice.Party{},
		Seller: t.Seller(),
	}
	var addressParts []string

	inTable := false

	for _, line := range lines {

		// detect table header
		if parserhelper.IsTableHeader(line) {
			inTable = true
			continue
		}

		lower := strings.ToLower(line)

		// ===== HEADER =====
		if !inTable {
			switch {
			case strings.Contains(lower, "customer name"):
				if name := parserhelper.ExtractCustomerName(line); name != "" {
					inv.Buyer.Name = name
				}

			case strings.HasPrefix(lower, "invoice no"):
				val, addr := parserhelper.ExtractValueAndAddress(line)
				invoiceNo := parserhelper.CleanString(val)
				inv.Number = invoiceNo
				if addr != "" {
					addressParts = append(addressParts, parserhelper.CleanString(addr))
				}

			case strings.HasPrefix(lower, "invoice date"):
				val, addr := parserhelper.ExtractValueAndAddress(line)
				inv.Date = parserhelper.ParseDateValue(val)
				if addr != "" {
					addressParts = append(addressParts, addr)
				}

			case strings.HasPrefix(lower, "po. number"):
				val, addr := parserhelper.ExtractValueAndAddress(line)
				inv.PONumber = parserhelper.CleanString(val)
				if addr != "" {
					addressParts = append(addressParts, addr)
				}

			case strings.HasPrefix(lower, "no. order"):
				val, addr := parserhelper.ExtractValueAndAddress(line)
				inv.OrderNumber = parserhelper.CleanString(val)
				if addr != "" {
					addressParts = append(addressParts, addr)
				}
			}

			continue
		}

		// ===== TABLE & SUMMARY =====

		// items
		if item, err := parserhelper.ParseItem(line); err == nil {
			inv.Items = append(inv.Items, *item)
			continue
		}

		// subtotal
		if inv.Subtotal == nil {
			if m, ok := parserhelper.ParseSummaryMoney(parserhelper.SubtotalRegex, line); ok {
				inv.Subtotal = m
				continue
			}
		}

		// Discount
		if inv.Discount == nil {
			if m, ok := parserhelper.ParseSummaryMoney(parserhelper.DiscountRegex, line); ok {
				inv.Discount = m
				continue
			}
		}

		// VAT
		if inv.VAT == nil {
			if m, ok := parserhelper.ParseSummaryMoney(parserhelper.VATRegex, line); ok {
				inv.VAT = m
				continue
			}
		}

		// TOTAL
		if inv.Total == nil {
			if m, ok := parserhelper.ParseSummaryMoney(parserhelper.TotalRegex, line); ok {
				inv.Total = m
				continue
			}
		}
	}

	invoice.ApplyDiscountToMostExpensiveItem(inv)

	// ===== ADDRESS =====
	if len(addressParts) > 0 {
		addr := strings.Join(addressParts, ", ")
		addr = parserhelper.CleanAddress(addr)
		addr = parserhelper.CleanString(addr)
		inv.Buyer.Address = &addr
	}

	return inv, nil
}
