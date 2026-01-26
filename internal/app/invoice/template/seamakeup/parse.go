package seamakeup

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

var itemRegex = regexp.MustCompile(
	`^\d+\s+[\w-]+\s+(?P<name>.+?)\s{2,}(?P<qty>[\d.]+)\s+(?P<unit>[\d.,]+)\s+(?P<total>[\d.,]+)$`,
)

var (
	subtotalRegex = regexp.MustCompile(`Subtotal\s+(?P<amount>[\d.,]+)`)
	vatRegex      = regexp.MustCompile(`^VAT\s+(?P<amount>[\d.,]+)`)
	totalRegex    = regexp.MustCompile(`^TOTAL\s+(?P<amount>[\d.,]+)`)
)

/*
========================
MAIN PARSER
========================
*/

func (t *SeaMakeupTemplate) Parse(raw string) (*invoice.Invoice, error) {
	norm := t.Normalize(raw)

	lines := strings.Split(norm, "\n")

	inv := &invoice.Invoice{}
	var addressParts []string

	inTable := false

	for _, line := range lines {

		// detect table header
		if isTableHeader(line) {
			inTable = true
			continue
		}

		// ===== HEADER =====
		if !inTable {
			switch {
			case strings.HasPrefix(line, "Invoice No"):
				val, addr := extractValueAndAddress(line)
				inv.InvoiceNo = val
				if addr != "" {
					addressParts = append(addressParts, addr)
				}

			case strings.HasPrefix(line, "Invoice Date"):
				val, addr := extractValueAndAddress(line)
				inv.InvoiceDate = parseDateValue(val)
				if addr != "" {
					addressParts = append(addressParts, addr)
				}

			case strings.HasPrefix(line, "Po. Number"):
				val, addr := extractValueAndAddress(line)
				inv.PONumber = val
				if addr != "" {
					addressParts = append(addressParts, addr)
				}

			case strings.HasPrefix(line, "No. Order"):
				_, addr := extractValueAndAddress(line)
				if addr != "" {
					addressParts = append(addressParts, addr)
				}
			}

			continue
		}

		// ===== TABLE & SUMMARY =====

		// items
		if item, err := parseItem(line); err == nil {
			inv.Items = append(inv.Items, *item)
			continue
		}

		// subtotal
		if inv.Subtotal == nil {
			if m, ok := parseSummaryMoney(subtotalRegex, line); ok {
				inv.Subtotal = m
				continue
			}
		}

		// VAT
		if inv.VAT == nil {
			if m, ok := parseSummaryMoney(vatRegex, line); ok {
				inv.VAT = m
				continue
			}
		}

		// TOTAL
		if inv.Total == nil {
			if m, ok := parseSummaryMoney(totalRegex, line); ok {
				inv.Total = m
				continue
			}
		}
	}

	// ===== ADDRESS =====
	if len(addressParts) > 0 {
		addr := strings.Join(addressParts, ", ")
		addr = cleanAddress(addr)
		inv.Address = &addr
	}

	return inv, nil
}

/*
========================
HELPERS
========================
*/

// Split value and address after colon
// Example:
// "Invoice No  :  SMB2025  Kota Jakarta"
// => "SMB2025", "Kota Jakarta"
func extractValueAndAddress(line string) (*string, string) {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return nil, ""
	}

	rest := strings.TrimSpace(line[idx+1:])
	if rest == "" {
		return nil, ""
	}

	parts := regexp.MustCompile(`\s{2,}`).Split(rest, 2)

	val := strings.TrimSpace(parts[0])
	if val == "" {
		return nil, ""
	}

	if len(parts) == 2 {
		return &val, strings.TrimSpace(parts[1])
	}

	return &val, ""
}

func matchGroup(
	re *regexp.Regexp,
	match []string,
	name string,
) string {

	if re == nil || match == nil {
		return ""
	}

	idx := re.SubexpIndex(name)
	if idx < 0 || idx >= len(match) {
		return ""
	}

	return strings.TrimSpace(match[idx])
}

func parseSummaryMoney(
	re *regexp.Regexp,
	line string,
) (*invoice.Money, bool) {

	m := re.FindStringSubmatch(line)
	if m == nil {
		return nil, false
	}

	amount := matchGroup(re, m, "amount")
	if amount == "" {
		return nil, false
	}

	money, err := parseMoney(amount)
	if err != nil {
		return nil, false
	}

	return money, true
}

func cleanAddress(addr string) string {
	// buang label "Address :" di awal atau tengah
	addr = regexp.MustCompile(`(?i)\baddress\s*:\s*`).ReplaceAllString(addr, "")

	// rapihin spasi dan koma
	addr = strings.TrimSpace(addr)
	addr = strings.Trim(addr, ",")

	// collapse spasi berlebih
	addr = regexp.MustCompile(`\s{2,}`).ReplaceAllString(addr, " ")

	return addr
}

func parseDateValue(val *string) *time.Time {
	if val == nil {
		return nil
	}

	t, err := helper.ParseDateValue(*val)
	if err != nil {
		return nil
	}

	return t
}

func isTableHeader(line string) bool {
	return strings.HasPrefix(line, "No") &&
		strings.Contains(line, "SKU") &&
		strings.Contains(line, "QTY")
}

/*
========================
ITEM & MONEY
========================
*/

func parseMoney(input string) (*invoice.Money, error) {
	dec, err := helper.ParseDecimal(input)
	if err != nil {
		return nil, err
	}

	return &invoice.Money{
		Amount:   dec.InexactFloat64(),
		Currency: "IDR",
	}, nil
}

func parseItem(line string) (*invoice.Item, error) {
	m := itemRegex.FindStringSubmatch(line)
	if m == nil {
		return nil, errors.New("not an item line")
	}

	qtyRaw := matchGroup(itemRegex, m, "qty")
	qty, err := parseQty(qtyRaw)
	if err != nil {
		return nil, err
	}

	unitRaw := matchGroup(itemRegex, m, "unit")
	unit, err := parseMoney(unitRaw)
	if err != nil {
		return nil, err
	}

	totalRaw := matchGroup(itemRegex, m, "total")
	total, err := parseMoney(totalRaw)
	if err != nil {
		return nil, err
	}

	name := matchGroup(itemRegex, m, "name")

	return &invoice.Item{
		Name:        name,
		Quantity:    qty,
		UnitPrice:   unit,
		TotalAmount: total,
	}, nil
}

func parseQty(raw string) (int, error) {
	clean := strings.ReplaceAll(raw, ".", "")
	return strconv.Atoi(clean)
}
