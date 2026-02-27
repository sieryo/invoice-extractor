package parserhelper

import (
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
)

func ParseTemplateA(
	raw string,
	normalize func(string) string,
	seller *invoice.Party,
) (*invoice.Invoice, error) {
	norm := normalize(raw)

	lines := strings.Split(norm, "\n")
	inv := &invoice.Invoice{
		Buyer:  &invoice.Party{},
		Seller: seller,
	}
	var addressParts []string

	inTable := false

	for _, line := range lines {
		if IsTableHeader(line) {
			inTable = true
			continue
		}

		lower := strings.ToLower(line)

		if !inTable {
			switch {
			case strings.Contains(lower, "customer name"):
				if name := ExtractCustomerName(line); name != "" {
					inv.Buyer.Name = name
				}

			case strings.HasPrefix(lower, "invoice no"):
				val, addr := ExtractValueAndAddress(line)
				invoiceNo := CleanString(val)
				inv.Number = invoiceNo
				if addr != "" {
					addressParts = append(addressParts, CleanString(addr))
				}

			case strings.HasPrefix(lower, "invoice date"):
				val, addr := ExtractValueAndAddress(line)
				inv.Date = ParseDateValue(val)
				if addr != "" {
					addressParts = append(addressParts, addr)
				}

			case strings.HasPrefix(lower, "po. number"):
				val, addr := ExtractValueAndAddress(line)
				inv.PONumber = CleanString(val)
				if addr != "" {
					addressParts = append(addressParts, addr)
				}

			case strings.HasPrefix(lower, "no. order"):
				val, addr := ExtractValueAndAddress(line)
				inv.OrderNumber = CleanString(val)
				if addr != "" {
					addressParts = append(addressParts, addr)
				}
			}

			continue
		}

		if item, err := ParseItem(line); err == nil {
			inv.Items = append(inv.Items, *item)
			continue
		}

		if inv.Subtotal == nil {
			if m, ok := ParseSummaryMoney(SubtotalRegex, line); ok {
				inv.Subtotal = m
				continue
			}
		}

		if inv.Discount == nil {
			if m, ok := ParseSummaryMoney(DiscountRegex, line); ok {
				inv.Discount = m
				continue
			}
		}

		if inv.VAT == nil {
			if m, ok := ParseSummaryMoney(VATRegex, line); ok {
				inv.VAT = m
				continue
			}
		}

		if inv.Total == nil {
			if m, ok := ParseSummaryMoney(TotalRegex, line); ok {
				inv.Total = m
				continue
			}
		}
	}

	invoice.ApplyDiscountToMostExpensiveItem(inv)

	if len(addressParts) > 0 {
		addr := strings.Join(addressParts, ", ")
		addr = CleanAddress(addr)
		addr = CleanString(addr)
		inv.Buyer.Address = &addr
	}

	return inv, nil
}
