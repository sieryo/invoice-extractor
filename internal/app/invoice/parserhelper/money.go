package parserhelper

import (
	"regexp"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

func ParseMoney(input string) (*invoice.Money, error) {
	dec, err := helper.ParseDecimal(input)
	if err != nil {
		return nil, err
	}

	return &invoice.Money{
		Amount:   dec.InexactFloat64(),
		Currency: "IDR",
	}, nil
}

func ParseSummaryMoney(
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

	money, err := ParseMoney(amount)
	if err != nil {
		return nil, false
	}

	return money, true
}
