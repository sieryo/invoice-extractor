package parserhelper

import (
	"testing"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
)

func TestBuildParseWarnings(t *testing.T) {
	inv := &invoice.Invoice{
		Buyer: &invoice.Party{},
		Items: []invoice.Item{},
	}

	warnings := BuildParseWarnings(inv)
	if len(warnings) == 0 {
		t.Fatalf("expected warnings, got none")
	}
}

func TestBuildParseWarnings_TotalMismatch(t *testing.T) {
	inv := &invoice.Invoice{
		Number: "INV-1",
		Buyer:  &invoice.Party{Name: "Buyer"},
		Items: []invoice.Item{
			{Sku: "SKU", Name: "Item", Quantity: 1},
		},
		Subtotal: &invoice.Money{Amount: 100000, Currency: "IDR"},
		VAT:      &invoice.Money{Amount: 11000, Currency: "IDR"},
		Total:    &invoice.Money{Amount: 100000, Currency: "IDR"},
	}

	warnings := BuildParseWarnings(inv)
	found := false
	for _, w := range warnings {
		if len(w) >= 14 && w[:14] == "total mismatch" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected total mismatch warning, got %v", warnings)
	}
}
