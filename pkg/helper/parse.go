package helper

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

func ParseDecimal(input string) (decimal.Decimal, error) {
	s := strings.TrimSpace(input)

	// buang currency symbol & teks
	re := regexp.MustCompile(`[^\d.,-]`)
	s = re.ReplaceAllString(s, "")

	if s == "" {
		return decimal.Zero, errors.New("empty amount")
	}

	lastDot := strings.LastIndex(s, ".")
	lastComma := strings.LastIndex(s, ",")

	var decimalSep string
	var thousandSep string

	if lastDot > lastComma {
		decimalSep = "."
		thousandSep = ","
	} else if lastComma > lastDot {
		decimalSep = ","
		thousandSep = "."
	}

	if thousandSep != "" {
		s = strings.ReplaceAll(s, thousandSep, "")
	}
	if decimalSep != "" && decimalSep != "." {
		s = strings.ReplaceAll(s, decimalSep, ".")
	}

	return decimal.NewFromString(s)
}

func ParseDateValue(input string) (*time.Time, error) {
	layouts := []string{
		"2006-01-02",
		"02-01-2006",
		"02/01/2006",
	}

	input = strings.TrimSpace(input)

	for _, layout := range layouts {
		if t, err := time.Parse(layout, input); err == nil {
			return &t, nil
		}
	}

	return nil, errors.New("unsupported date format")
}
