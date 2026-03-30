package cashflow

import "testing"

func TestFormatMYOBMoney_BelowOneBillionKeepsThousandsSeparator(t *testing.T) {
	t.Parallel()

	got := formatMYOBMoney(999_999_999.99)
	want := "999.999.999,99"
	if got != want {
		t.Fatalf("formatMYOBMoney() = %q, want %q", got, want)
	}
}

func TestFormatMYOBMoney_OneBillionOrMoreRemovesThousandsSeparator(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		value float64
		want  string
	}{
		{name: "exact one billion", value: 1_000_000_000, want: "1000000000,00"},
		{name: "above one billion", value: 1_682_025_634, want: "1682025634,00"},
		{name: "negative above one billion", value: -1_682_025_634, want: "-1682025634,00"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := formatMYOBMoney(tc.value)
			if got != tc.want {
				t.Fatalf("formatMYOBMoney(%v) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}
