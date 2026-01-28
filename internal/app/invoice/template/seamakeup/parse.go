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

		// ===== HEADER =====
		if !inTable {
			switch {
			case strings.HasPrefix(line, "Customer Name"):
				val := parserhelper.ExtractValue(line)
				inv.Buyer.Name = parserhelper.CleanString(val)

			case strings.HasPrefix(line, "Invoice No"):
				val, addr := parserhelper.ExtractValueAndAddress(line)
				invoiceNo := parserhelper.CleanString(val)
				inv.Number = invoiceNo
				if addr != "" {
					addressParts = append(addressParts, parserhelper.CleanString(addr))
				}

			case strings.HasPrefix(line, "Invoice Date"):
				val, addr := parserhelper.ExtractValueAndAddress(line)
				inv.Date = parserhelper.ParseDateValue(val)
				if addr != "" {
					addressParts = append(addressParts, addr)
				}

			case strings.HasPrefix(line, "Po. Number"):
				val, addr := parserhelper.ExtractValueAndAddress(line)
				inv.PONumber = parserhelper.CleanString(val)
				if addr != "" {
					addressParts = append(addressParts, addr)
				}

			case strings.HasPrefix(line, "No. Order"):
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

	// ===== ADDRESS =====
	if len(addressParts) > 0 {
		addr := strings.Join(addressParts, ", ")
		addr = parserhelper.CleanAddress(addr)
		addr = parserhelper.CleanString(addr)
		inv.Buyer.Address = &addr
	}

	return inv, nil
}
