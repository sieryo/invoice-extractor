package helper

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseDecimal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "IDR format with decimal comma",
			input: "34.686,00",
			want:  "34686",
		},
		{
			name:  "IDR format thousand separator",
			input: "30.351",
			want:  "30351",
		},
		{
			name:  "currency prefix Rp",
			input: "Rp30.351",
			want:  "30351",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDecimal(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			wantDec := decimal.RequireFromString(tc.want)
			if !got.Equal(wantDec) {
				t.Fatalf("expected %s, got %s", wantDec.String(), got.String())
			}
		})
	}
}
