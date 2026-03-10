package helper

import "unicode"

func DigitsOnly(input string) string {
	if input == "" {
		return ""
	}

	out := make([]rune, 0, len(input))
	for _, r := range input {
		if unicode.IsDigit(r) {
			out = append(out, r)
		}
	}
	return string(out)
}

func IsNPWP15(input string) bool {
	return len(input) == 15
}

func IsNPWP16(input string) bool {
	return len(input) == 16
}

func IsNITKU(input string) bool {
	return len(input) == 22
}

func TaxIDKind(input string) string {
	switch {
	case IsNPWP16(input):
		return "npwp16"
	case IsNPWP15(input):
		return "npwp15"
	case IsNITKU(input):
		return "nitku"
	default:
		return "unknown"
	}
}

func IsValidTaxID(input string) bool {
	return TaxIDKind(input) != "unknown"
}
