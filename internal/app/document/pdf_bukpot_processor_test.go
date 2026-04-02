package document

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"testing"

	bukpotdomain "github.com/sieryo/invoice-extractor/internal/domain/bukpot"
)

func TestExtractBPPUCategoryAndNumber(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantCat    string
		wantNumber string
	}{
		{
			name:       "with category and extra dash",
			raw:        "INV-2026-001 - Sales - Freight cost",
			wantCat:    "Sales - Freight cost",
			wantNumber: "INV-2026-001",
		},
		{
			name:       "without dash",
			raw:        "INV-2026-001",
			wantCat:    bukpotUnknownCategory,
			wantNumber: "INV-2026-001",
		},
		{
			name:       "empty",
			raw:        "",
			wantCat:    bukpotUnknownCategory,
			wantNumber: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCat, gotNumber := extractCategoryAndDocumentNumberFromReference(tt.raw)
			if gotCat != tt.wantCat {
				t.Fatalf("unexpected category: got %q, want %q", gotCat, tt.wantCat)
			}
			if gotNumber != tt.wantNumber {
				t.Fatalf("unexpected number: got %q, want %q", gotNumber, tt.wantNumber)
			}
		})
	}
}

func TestRenderBukpotFallbackFilename(t *testing.T) {
	parsed := &bukpotdomain.ParsedDocument{
		Kind: bukpotdomain.KindBPPU,
		BPPU: &bukpotdomain.BPPUDocument{
			NomorBuktiPotong: "2507AAPI0",
			NamaPenerima:     "HIBURAN JALAN KELUAR",
		},
	}
	got := renderBukpotFallbackFilename(parsed, "", "")
	if got != "2507AAPI0 - HIBURAN JALAN KELUAR" {
		t.Fatalf("unexpected fallback filename: %q", got)
	}
}

func TestRenderBukpotDocumentNumberFilename(t *testing.T) {
	parsed := &bukpotdomain.ParsedDocument{
		Kind: bukpotdomain.KindBPPU,
		BPPU: &bukpotdomain.BPPUDocument{
			NamaPenerima: "HIBURAN JALAN KELUAR",
		},
	}
	got := renderBukpotDocumentNumberFilename("INV-2026-001", parsed, "", "")
	if got != "INV-2026-001 - HIBURAN JALAN KELUAR" {
		t.Fatalf("unexpected filename: %q", got)
	}
}

func TestRenderBukpotCategoryFilename_IncludesMasaPajak(t *testing.T) {
	parsed := &bukpotdomain.ParsedDocument{
		Kind: bukpotdomain.KindBPPU,
		BPPU: &bukpotdomain.BPPUDocument{
			NamaPenerima: "HIBURAN JALAN KELUAR",
			MasaPajak:    "11-2025",
		},
	}

	got := renderBukpotCategoryFilename("KOL", parsed, "", "")
	if got != "KOL - HIBURAN JALAN KELUAR - 11-2025" {
		t.Fatalf("unexpected filename: %q", got)
	}
}

func TestBuildBPPURenameZipArchive(t *testing.T) {
	data, err := buildBukpotZipArchive([]bukpotZipEntry{
		{
			Category: "Sales",
			Name:     "INV-001.pdf",
			Data:     []byte("a"),
		},
		{
			Category: "UNKNOWN",
			Name:     "INV-002.pdf",
			Data:     []byte("b"),
		},
	}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("invalid zip: %v", err)
	}
	if len(zr.File) != 2 {
		t.Fatalf("unexpected zip entry count: %d", len(zr.File))
	}

	got := map[string]struct{}{}
	for _, f := range zr.File {
		got[f.Name] = struct{}{}
	}

	if _, ok := got["Sales/INV-001.pdf"]; !ok {
		t.Fatalf("missing Sales/INV-001.pdf in zip")
	}
	if _, ok := got["UNKNOWN/INV-002.pdf"]; !ok {
		t.Fatalf("missing UNKNOWN/INV-002.pdf in zip")
	}
}

func TestRenderBukpotFilename(t *testing.T) {
	values := map[string]string{
		"nomorbuktipotong": "2507AAPI0",
		"namapenerima":     "HIBURAN JALAN KELUAR",
	}
	filename, warnings := renderBukpotFilename("{{nomorBuktiPotong}} - {{namaPenerima}}", values)
	if filename != "2507AAPI0 - HIBURAN JALAN KELUAR.pdf" {
		t.Fatalf("unexpected filename: %s", filename)
	}
	if len(warnings) != 0 {
		t.Fatalf("expected no warnings, got: %#v", warnings)
	}
}

func TestBuildBukpotTemplateMapBPA1IncludesPosisi(t *testing.T) {
	parsed := &bukpotdomain.ParsedDocument{
		Kind: bukpotdomain.KindBPA1,
		BPA1: &bukpotdomain.BPA1Document{
			NomorBuktiPotong:   "2507TJSDP",
			NamaPenerima:       "Aprilia Nuraeni",
			PeriodePenghasilan: "01-2025-12-2025",
			Posisi:             "STAFF",
			StatusPTKP:         "TK0",
		},
	}

	values := buildBukpotTemplateMap(CollectionKindBukpotBPA1, parsed, "example_bpa1.pdf", "BPA1")

	if values["posisi"] != "STAFF" {
		t.Fatalf("expected posisi token to be populated, got %q", values["posisi"])
	}
	if values["statusptkp"] != "TK0" {
		t.Fatalf("expected statusPtkp token to be populated, got %q", values["statusptkp"])
	}
}

func TestParseBukpotRenameParams_DefaultsOnlyNormalStatus(t *testing.T) {
	params, err := parseBukpotRenameParams(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !params.OnlyNormalStatus {
		t.Fatalf("expected onlyNormalStatus default to true")
	}
	if params.FilenameTemplate != defaultBukpotNameTemplate {
		t.Fatalf("unexpected default template: %q", params.FilenameTemplate)
	}
}

func TestParseBukpotRenameParams_AllowsDisablingOnlyNormalStatus(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"onlyNormalStatus": false,
	})
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	params, parseErr := parseBukpotRenameParams(raw)
	if parseErr != nil {
		t.Fatalf("unexpected error: %v", parseErr)
	}
	if params.OnlyNormalStatus {
		t.Fatalf("expected onlyNormalStatus to be false")
	}
}

func TestShouldRejectBukpotStatus(t *testing.T) {
	tests := []struct {
		name             string
		statusBukti      string
		onlyNormalStatus bool
		wantRejected     bool
		wantStatus       string
	}{
		{
			name:             "normal is allowed",
			statusBukti:      "NORMAL",
			onlyNormalStatus: true,
			wantRejected:     false,
			wantStatus:       "NORMAL",
		},
		{
			name:             "pembetulan is rejected",
			statusBukti:      "PEMBETULAN",
			onlyNormalStatus: true,
			wantRejected:     true,
			wantStatus:       "PEMBETULAN",
		},
		{
			name:             "dibatalkan is rejected",
			statusBukti:      "DIBATALKAN",
			onlyNormalStatus: true,
			wantRejected:     true,
			wantStatus:       "DIBATALKAN",
		},
		{
			name:             "filter disabled allows all statuses",
			statusBukti:      "PEMBETULAN",
			onlyNormalStatus: false,
			wantRejected:     false,
			wantStatus:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := &bukpotdomain.ParsedDocument{
				Kind: bukpotdomain.KindBPPU,
				BPPU: &bukpotdomain.BPPUDocument{
					StatusBukti: tt.statusBukti,
				},
			}

			rejected, statusLabel := shouldRejectBukpotStatus(parsed, tt.onlyNormalStatus)
			if rejected != tt.wantRejected {
				t.Fatalf("unexpected rejected flag: got %v, want %v", rejected, tt.wantRejected)
			}
			if statusLabel != tt.wantStatus {
				t.Fatalf("unexpected status label: got %q, want %q", statusLabel, tt.wantStatus)
			}
		})
	}
}

func TestRememberUniqueBukpotFilename(t *testing.T) {
	used := map[string]struct{}{}

	if !rememberUniqueBukpotFilename("bukpot.pdf", used) {
		t.Fatalf("expected first filename to be accepted")
	}
	if rememberUniqueBukpotFilename("bukpot.pdf", used) {
		t.Fatalf("expected duplicate filename to be rejected")
	}
	if !rememberUniqueBukpotFilename("bukpot-2.pdf", used) {
		t.Fatalf("expected different filename to be accepted")
	}
}
