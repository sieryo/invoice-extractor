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
	`^\d+\s+[\w-]+\s+(?P<name>.+?)\s{2,}(?P<qty>\d+)\s+(?P<unit>[\d.,]+)\s+(?P<total>[\d.,]+)$`,
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

		// Stop header parsing when table starts
		if isTableHeader(line) {
			inTable = true
			continue
		}

		if !inTable {

			// HEADER FIELDS WITH POSSIBLE ADDRESS
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

		// TABLE ITEMS
		if item, err := parseItem(line); err == nil {
			inv.Items = append(inv.Items, *item)
		}
	}

	if len(addressParts) > 0 {
		addr := strings.Join(addressParts, ", ")

		// cleanup ringan label yang kebawa
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

	name := strings.TrimSpace(m[itemRegex.SubexpIndex("name")])
	qtyStr := m[itemRegex.SubexpIndex("qty")]
	unitStr := m[itemRegex.SubexpIndex("unit")]
	totalStr := m[itemRegex.SubexpIndex("total")]

	qty, err := strconv.Atoi(qtyStr)
	if err != nil {
		return nil, err
	}

	unit, err := parseMoney(unitStr)
	if err != nil {
		return nil, err
	}

	total, err := parseMoney(totalStr)
	if err != nil {
		return nil, err
	}

	return &invoice.Item{
		Name:        name,
		Quantity:    qty,
		UnitPrice:   unit,
		TotalAmount: total,
	}, nil
}
