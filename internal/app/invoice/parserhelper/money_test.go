package parserhelper

import (
	"regexp"
	"testing"
)

func TestParseMoney(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{
			name:  "Rp prefix with thousand separator",
			input: "Rp30.351",
			want:  30351,
		},
		{
			name:  "IDR prefix with decimal comma",
			input: "IDR 34.686,00",
			want:  34686,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseMoney(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Amount != tc.want {
				t.Fatalf("expected amount %.2f, got %.2f", tc.want, got.Amount)
			}
			if got.Currency != "IDR" {
				t.Fatalf("expected currency IDR, got %s", got.Currency)
			}
		})
	}
}

func TestParseSummaryMoney(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		regex *regexp.Regexp
		want  float64
	}{
		{
			name:  "subtotal with Rp",
			line:  "Subtotal : Rp30.351",
			regex: SubtotalRegex,
			want:  30351,
		},
		{
			name:  "discount with IDR",
			line:  "Discount IDR 4.000",
			regex: DiscountRegex,
			want:  4000,
		},
		{
			name:  "vat with european decimal",
			line:  "VAT : 34.686,00",
			regex: VATRegex,
			want:  34686,
		},
		{
			name:  "total with IDR and separator",
			line:  "Total = IDR 100.000",
			regex: TotalRegex,
			want:  100000,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := ParseSummaryMoney(tc.regex, tc.line)
			if !ok {
				t.Fatalf("expected match for line: %q", tc.line)
			}

			if got.Amount != tc.want {
				t.Fatalf("expected amount %.2f, got %.2f", tc.want, got.Amount)
			}
			if got.Currency != "IDR" {
				t.Fatalf("expected currency IDR, got %s", got.Currency)
			}
		})
	}
}

func TestParseSummaryMoney_TotalDoesNotMatchTotalAfterDiscount(t *testing.T) {
	line := "Total After Discount : Rp10.000"
	if got, ok := ParseSummaryMoney(TotalRegex, line); ok || got != nil {
		t.Fatalf("expected no match for total-after-discount line, got %+v", got)
	}

	validLine := "Total : Rp12.000"
	got, ok := ParseSummaryMoney(TotalRegex, validLine)
	if !ok {
		t.Fatalf("expected total line to match: %q", validLine)
	}
	if got.Amount != 12000 {
		t.Fatalf("expected amount 12000, got %.2f", got.Amount)
	}
}
