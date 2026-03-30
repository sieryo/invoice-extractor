package document

import "testing"

func TestParseSpreadsheetFloatValue(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		raw     string
		display string
		want    float64
	}{
		{
			name:    "plain integer",
			raw:     "12500",
			display: "12500",
			want:    12500,
		},
		{
			name:    "indonesian thousand separator",
			raw:     "12500",
			display: "12.500",
			want:    12500,
		},
		{
			name:    "english thousand separator",
			raw:     "",
			display: "12,500",
			want:    12500,
		},
		{
			name:    "indonesian decimal format",
			raw:     "",
			display: "12.500,75",
			want:    12500.75,
		},
		{
			name:    "english decimal format",
			raw:     "",
			display: "12,500.75",
			want:    12500.75,
		},
		{
			name:    "negative integer with grouping",
			raw:     "",
			display: "-1.250",
			want:    -1250,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := parseSpreadsheetFloatValue(tc.raw, tc.display)
			if !ok {
				t.Fatalf("parseSpreadsheetFloatValue(%q, %q) returned ok=false", tc.raw, tc.display)
			}
			if got != tc.want {
				t.Fatalf("parseSpreadsheetFloatValue(%q, %q) = %v, want %v", tc.raw, tc.display, got, tc.want)
			}
		})
	}
}

func TestSpreadsheetCellPercent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cell SpreadsheetCell
		want float64
	}{
		{
			name: "text percent integer",
			cell: SpreadsheetCell{
				Display:     "2%",
				StringValue: "2%",
				ValueType:   SpreadsheetCellValueTypeString,
			},
			want: 0.02,
		},
		{
			name: "text percent decimal comma",
			cell: SpreadsheetCell{
				Display:     "2,5%",
				StringValue: "2,5%",
				ValueType:   SpreadsheetCellValueTypeString,
			},
			want: 0.025,
		},
		{
			name: "raw excel fraction",
			cell: SpreadsheetCell{
				Display:    "2%",
				Raw:        "0.02",
				ValueType:  SpreadsheetCellValueTypeFloat,
				FloatValue: floatPtr(0.02),
			},
			want: 0.02,
		},
		{
			name: "raw whole number with percent sign",
			cell: SpreadsheetCell{
				Display:    "2%",
				Raw:        "2",
				ValueType:  SpreadsheetCellValueTypeFloat,
				FloatValue: floatPtr(2),
			},
			want: 0.02,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, ok := SpreadsheetCellPercent(tc.cell)
			if !ok {
				t.Fatalf("SpreadsheetCellPercent(%+v) returned ok=false", tc.cell)
			}
			if got != tc.want {
				t.Fatalf("SpreadsheetCellPercent(%+v) = %v, want %v", tc.cell, got, tc.want)
			}
		})
	}
}

func TestSpreadsheetCellMoney_PrefersRawValueLikeLegacyReader(t *testing.T) {
	t.Parallel()

	cell := SpreadsheetCell{
		Display:    "1.030.820,39",
		Raw:        "1030820.394",
		ValueType:  SpreadsheetCellValueTypeFloat,
		FloatValue: floatPtr(1030820.394),
	}

	got, ok := SpreadsheetCellMoney(cell)
	if !ok {
		t.Fatal("SpreadsheetCellMoney() returned ok=false")
	}
	if got != 1030820.394 {
		t.Fatalf("SpreadsheetCellMoney() = %v, want %v", got, 1030820.394)
	}
}

func TestSpreadsheetCellMoney_FallsBackToDisplayWhenRawEmpty(t *testing.T) {
	t.Parallel()

	cell := SpreadsheetCell{
		Display:     "1.030.820,39",
		Raw:         "",
		ValueType:   SpreadsheetCellValueTypeString,
		StringValue: "1.030.820,39",
	}

	got, ok := SpreadsheetCellMoney(cell)
	if !ok {
		t.Fatal("SpreadsheetCellMoney() returned ok=false")
	}
	if got != 1030820.39 {
		t.Fatalf("SpreadsheetCellMoney() = %v, want %v", got, 1030820.39)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
