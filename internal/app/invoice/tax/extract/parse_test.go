package extract

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sieryo/invoice-extractor/internal/infra/adapter/pdftool"
)

func TestParseTaxInvoiceText_FromExamplePDF(t *testing.T) {
	pdfPath := findExampleTaxInvoicePDF(t)

	text, err := pdftool.ExtractText(context.Background(), pdfPath, pdftool.DefaultOptions())
	if err != nil {
		t.Fatalf("failed to extract text using pdftool: %v", err)
	}
	if strings.TrimSpace(text) == "" {
		t.Fatalf("pdftool returned empty text for %s", pdfPath)
	}

	invoice, normalized, err := ParseTaxInvoiceText(filepath.Base(pdfPath), text)
	if err != nil {
		t.Fatalf("parse tax invoice failed: %v", err)
	}
	if invoice == nil {
		t.Fatalf("parsed invoice is nil")
	}
	if strings.TrimSpace(normalized) == "" {
		t.Fatalf("normalized text is empty")
	}
	if strings.TrimSpace(invoice.InvoiceNumber) == "" {
		t.Fatalf("invoice number is empty")
	}
	if strings.TrimSpace(invoice.Number) == "" {
		t.Fatalf("legacy number is empty")
	}

	if strings.TrimSpace(invoice.References) == "" {
		t.Fatalf("references is empty")
	}
	if invoice.Buyer == nil || strings.TrimSpace(invoice.Buyer.Name) == "" {
		t.Fatalf("buyer is missing")
	}
}

func findExampleTaxInvoicePDF(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if ok {
		backendRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", ".."))
		candidate := filepath.Join(backendRoot, "assets", "pdf", "example_tax_invoice.pdf")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	candidates := []string{
		filepath.Join("assets", "pdf", "example_tax_invoice.pdf"),
		filepath.Join("backend", "assets", "pdf", "example_tax_invoice.pdf"),
	}

	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}

	t.Fatalf("required test file is missing: example_tax_invoice.pdf (expected under backend/assets/pdf)")
	return ""
}
