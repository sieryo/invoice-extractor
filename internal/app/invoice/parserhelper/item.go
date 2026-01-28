package parserhelper

import (
	"errors"
	"strconv"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
)

func ParseItem(line string) (*invoice.Item, error) {
	m := ItemRegex.FindStringSubmatch(line)
	if m == nil {
		return nil, errors.New("not an item line")
	}

	qtyRaw := matchGroup(ItemRegex, m, "qty")
	qty, err := ParseQty(qtyRaw)
	if err != nil {
		return nil, err
	}

	unitRaw := matchGroup(ItemRegex, m, "unit")
	unit, err := ParseMoney(unitRaw)
	if err != nil {
		return nil, err
	}

	totalRaw := matchGroup(ItemRegex, m, "total")
	total, err := ParseMoney(totalRaw)
	if err != nil {
		return nil, err
	}

	name := CleanString(matchGroup(ItemRegex, m, "name"))

	return &invoice.Item{
		Name:        name,
		Quantity:    qty,
		UnitPrice:   unit,
		TotalAmount: total,
		TaxRate:     0.12, // 12%
	}, nil
}

func ParseQty(raw string) (int, error) {
	clean := strings.ReplaceAll(raw, ".", "")
	return strconv.Atoi(clean)
}
