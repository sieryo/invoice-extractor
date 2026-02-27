package parserhelper

import (
	"regexp"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

var currencyTokenRe = regexp.MustCompile(`(?i)\b(?:idr|rp)\b\.?`)

func sanitizeMoneyInput(input string) string {
	s := strings.TrimSpace(input)
	s = currencyTokenRe.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func ParseMoney(input string) (*invoice.Money, error) {
	dec, err := helper.ParseDecimal(sanitizeMoneyInput(input))
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

	money, err := ParseMoney(sanitizeMoneyInput(amount))
	if err != nil {
		return nil, false
	}

	return money, true
}
