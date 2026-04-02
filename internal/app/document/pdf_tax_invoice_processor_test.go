package document

import (
	"archive/zip"
	"bytes"
	"testing"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax"
)

func TestRenderTaxInvoiceFilename_WithKnownTemplate(t *testing.T) {
	inv := &tax.TaxInvoice{
		InvoiceNumber: "04002500396593615",
		InvoiceDate:   time.Date(2025, time.December, 3, 0, 0, 0, 0, time.UTC),
		BuyerName:     "ANEKA KOSMETIK",
		References:    "GST202512033747",
	}

	got, warnings := renderTaxInvoiceFilename("{{references}} - {{buyerName}}", inv, "source.pdf")
	if got != "GST202512033747 - ANEKA KOSMETIK.pdf" {
		t.Fatalf("unexpected filename: %s", got)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got: %#v", warnings)
	}
}

func TestRenderTaxInvoiceFilename_UsesFallbackWhenTemplateUnknown(t *testing.T) {
	inv := &tax.TaxInvoice{
		BuyerName:  "ANEKA KOSMETIK",
		References: "GST202512033747",
	}

	got, warnings := renderTaxInvoiceFilename("{{unknownField}}", inv, "source.pdf")
	if got != "GST202512033747 - ANEKA KOSMETIK.pdf" {
		t.Fatalf("unexpected fallback filename: %s", got)
	}
	if len(warnings) == 0 {
		t.Fatalf("expected warnings when unknown placeholder is used")
	}
}

func TestRememberUniqueArchiveFilename(t *testing.T) {
	used := map[string]struct{}{}

	if !rememberUniqueArchiveFilename("faktur.pdf", used) {
		t.Fatalf("expected first filename to be accepted")
	}
	if rememberUniqueArchiveFilename("faktur.pdf", used) {
		t.Fatalf("expected duplicate filename to be rejected")
	}
	if !rememberUniqueArchiveFilename("faktur-2.pdf", used) {
		t.Fatalf("expected different filename to be accepted")
	}
}

func TestBuildTaxInvoiceZipArchive(t *testing.T) {
	data, err := buildTaxInvoiceZipArchive([]renamedTaxInvoiceFile{
		{Name: "A.pdf", Data: []byte("file-a")},
		{Name: "B.pdf", Data: []byte("file-b")},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("zip data is empty")
	}

	readerAt := bytes.NewReader(data)
	zr, err := zip.NewReader(readerAt, int64(len(data)))
	if err != nil {
		t.Fatalf("invalid zip output: %v", err)
	}
	if len(zr.File) != 2 {
		t.Fatalf("expected 2 files in zip, got %d", len(zr.File))
	}
}
