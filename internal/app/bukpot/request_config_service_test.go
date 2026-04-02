package bukpot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sieryo/invoice-extractor/internal/profilepath"
)

func TestRequestConfigServiceLoadDefaultsWhenMissing(t *testing.T) {
	rootDir := t.TempDir()
	svc := NewRequestConfigService(rootDir)

	cfg, err := svc.Load("profile-1")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SchemaVersion == "" {
		t.Fatalf("expected schema version")
	}
	if len(cfg.Fields) == 0 {
		t.Fatalf("expected default fields")
	}
	if cfg.Defaults.HeaderRowNumber != 1 {
		t.Fatalf("expected default header row 1, got %d", cfg.Defaults.HeaderRowNumber)
	}
}

func TestRequestConfigServiceUpdateAndStatus(t *testing.T) {
	rootDir := t.TempDir()
	svc := NewRequestConfigService(rootDir)
	profileID := "profile-1"

	cfg := RequestConfig{
		Defaults: RequestConfigDefaults{
			SheetName:       "Deduction MT",
			HeaderRowNumber: 3,
		},
		Fields: []RequestConfigField{
			{Key: "entity", Value: "Entity"},
			{Key: "settlementDate", Value: "Settlement Date"},
			{Key: "npwp", Value: "NPWP"},
			{Key: "nitku", Value: "NITKU"},
			{Key: "taxObjectCode", Value: "Kode Objek Pajak"},
			{Key: "taxBase", Value: "Total Invoice"},
			{Key: "withholdingRate", Value: "WHT"},
			{Key: "referenceNumber", Value: "Invoice No"},
			{Key: "referenceDate", Value: "FP DATE"},
		},
	}

	stored, err := svc.Update(profileID, cfg)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if stored.Defaults.SheetName != "Deduction MT" {
		t.Fatalf("expected saved sheet name")
	}

	path := profilepath.BukpotRequestConfigJSON(rootDir, profileID)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file to exist: %v", err)
	}

	status := svc.Status(profileID)
	if !status.Configured {
		t.Fatalf("expected config to be ready, got %+v", status)
	}

	loaded, err := svc.Load(profileID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Defaults.HeaderRowNumber != 3 {
		t.Fatalf("expected header row 3, got %d", loaded.Defaults.HeaderRowNumber)
	}

	var found bool
	for _, field := range loaded.Fields {
		if field.Key == "settlementDate" {
			found = true
			if field.Value != "Settlement Date" {
				t.Fatalf("unexpected settlement date value: %s", field.Value)
			}
		}
	}
	if !found {
		t.Fatalf("expected settlementDate field")
	}

	if filepath.Dir(path) == "" {
		t.Fatalf("expected path dir")
	}
}
