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

	dotCount := strings.Count(s, ".")
	commaCount := strings.Count(s, ",")

	// Helper: cek apakah string diakhiri decimal 2 digit
	endsWith2Digits := func(sep string) bool {
		parts := strings.Split(s, sep)
		return len(parts) > 1 && len(parts[len(parts)-1]) == 2
	}

	switch {
	// Format: 40.259,00
	case commaCount == 1 && endsWith2Digits(","):
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")

	// Format: 40,259.00
	case dotCount == 1 && endsWith2Digits("."):
		s = strings.ReplaceAll(s, ",", "")

	// Format: 2.415.540 or 966.216
	case dotCount > 0 && commaCount == 0:
		s = strings.ReplaceAll(s, ".", "")

	// Format: 2,415,540
	case commaCount > 0 && dotCount == 0:
		s = strings.ReplaceAll(s, ",", "")
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
