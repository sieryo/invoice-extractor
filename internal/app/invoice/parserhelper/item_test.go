package parserhelper

import "testing"

func TestParseItem(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantSKU   string
		wantName  string
		wantQty   int
		wantUnit  float64
		wantTotal float64
	}{
		{
			name:      "line with number dot and Rp price",
			line:      "1. SKU-001 Serum Glow 30ml  2 Rp30.351 Rp60.702",
			wantSKU:   "SKU-001",
			wantName:  "Serum Glow 30ml",
			wantQty:   2,
			wantUnit:  30351,
			wantTotal: 60702,
		},
		{
			name:      "line with slash sku and european decimal format",
			line:      "2 AB/12-3 Hydrating Mask  1 34.686,00 34.686,00",
			wantSKU:   "AB/12-3",
			wantName:  "Hydrating Mask",
			wantQty:   1,
			wantUnit:  34686,
			wantTotal: 34686,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			item, err := ParseItem(tc.line)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if item.Sku != tc.wantSKU {
				t.Fatalf("expected sku %q, got %q", tc.wantSKU, item.Sku)
			}
			if item.Name != tc.wantName {
				t.Fatalf("expected name %q, got %q", tc.wantName, item.Name)
			}
			if item.Quantity != tc.wantQty {
				t.Fatalf("expected qty %d, got %d", tc.wantQty, item.Quantity)
			}
			if item.UnitPrice == nil || item.UnitPrice.Amount != tc.wantUnit {
				t.Fatalf("expected unit %.2f, got %+v", tc.wantUnit, item.UnitPrice)
			}
			if item.TotalAmount == nil || item.TotalAmount.Amount != tc.wantTotal {
				t.Fatalf("expected total %.2f, got %+v", tc.wantTotal, item.TotalAmount)
			}
		})
	}
}
