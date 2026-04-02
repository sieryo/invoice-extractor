package document

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	appfpcoretax "github.com/sieryo/invoice-extractor/internal/app/fpcoretax"
	"github.com/sieryo/invoice-extractor/internal/infra/filestore"
	"github.com/sieryo/invoice-extractor/internal/profilepath"
)

func TestXLSXFPCoretaxProcessor_RunActionMiscSales(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempDir := t.TempDir()
	fileStore := filestore.NewLocalFileStore(filepath.Join(tempDir, "storage"))
	profileID := "user-1"

	customerPath := profilepath.FPCoretaxCustomerCSV(tempDir, profileID)
	require.NoError(t, os.MkdirAll(filepath.Dir(customerPath), 0o755))
	require.NoError(t, os.WriteFile(customerPath, []byte(strings.Join([]string{
		"name,account",
		"BERKAT JAYA ABADI,",
	}, "\n")), 0o644))

	registryService := appfpcoretax.NewRelationRegistryService(tempDir)
	processor := NewXLSXFPCoretaxProcessor(CollectionKindFPKeluaranCoretax, fileStore, registryService)

	workbookPath := filepath.Join(tempDir, "fp-keluaran.xlsx")
	createFPCoretaxWorkbook(t, workbookPath, "Nama Pembeli")

	ingestResult, err := processor.Ingest(ctx, IngestRequest{
		RequestID:      "req-1",
		UserID:         profileID,
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindFPKeluaranCoretax,
		SourceFormat:   SourceFormatXLSX,
		Sources: []IngestSource{{
			SourceID:     "source-1",
			OriginalName: "fp-keluaran.xlsx",
			TempPath:     workbookPath,
			UploadedAt:   time.Now(),
		}},
		Policy:      IngestPolicy{KeepRaw: true},
		RequestedAt: time.Now(),
	})
	require.NoError(t, err)
	require.Len(t, ingestResult.Items, 1)
	require.Equal(t, IngestStatusReady, ingestResult.Items[0].Status)

	var normalizedRef string
	var rawRef string
	for _, artifact := range ingestResult.Items[0].Artifacts {
		if artifact.Kind == "normalized" {
			normalizedRef = artifact.ObjectID
		}
		if artifact.Kind == "raw" {
			rawRef = artifact.ObjectID
		}
	}
	require.NotEmpty(t, normalizedRef)
	require.NotEmpty(t, rawRef)

	inputJSON, err := json.Marshal(map[string]any{
		"sheetName":            "Faktur",
		"outputFilename":       "misc-sales-test",
		"headerRowNumber":      1,
		"accountNumber":        "41001",
		"memoTemplate":         "{{nomorFakturPajak}}",
		"descriptionTemplate":  "{{namaPembeli}} - {{nomorFakturPajak}}",
		"taxCode":              "PPN",
		"inclusive":            false,
		"partyName":            "Nama Pembeli",
		"documentNumber":       "Nomor Faktur Pajak",
		"date":                 "Tanggal Faktur Pajak",
		"taxBase":              "Harga Jual/Penggantian/DPP",
		"tax":                  "PPN",
		"reference":            "Referensi",
	})
	require.NoError(t, err)

	result, err := processor.RunAction(ctx, ActionRequest{
		ActionID:       "action-1",
		UserID:         profileID,
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindFPKeluaranCoretax,
		SourceFormat:   SourceFormatXLSX,
		ActionType:     fpKeluaranMiscSalesActionType,
		SnapshotDocs: []ActionSnapshotDocument{{
			DocumentID:    "doc-1",
			SourceName:    "fp-keluaran.xlsx",
			Status:        "ready",
			NormalizedRef: normalizedRef,
			RawRef:        rawRef,
		}},
		Input:       inputJSON,
		RequestedAt: time.Now(),
	})
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)
	require.Len(t, result.ItemResults, 1)
	require.Len(t, result.Outputs, 1)

	body, readErr := os.ReadFile(result.Outputs[0].ObjectRef)
	require.NoError(t, readErr)
	content := string(body)

	require.Contains(t, content, "Co./Last Name\tFirst Name\tInvoice #\tDate")
	require.Contains(t, content, "BERKAT JAYA ABADI\t\t\t14/10/2025")
	require.Contains(t, content, "\t04002500352440493\t")
	require.Contains(t, content, "BERKAT JAYA ABADI - 04002500352440493")
	require.Contains(t, content, "\t41001\t5.331.532,00\t5.918.000,00")
	require.Contains(t, content, "\tPPN\t0\t586.468,00")
}

func TestXLSXFPCoretaxProcessor_RunActionMiscPurchases_UsesSupplierAccount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempDir := t.TempDir()
	fileStore := filestore.NewLocalFileStore(filepath.Join(tempDir, "storage"))
	profileID := "user-1"

	supplierPath := profilepath.FPCoretaxSupplierCSV(tempDir, profileID)
	require.NoError(t, os.MkdirAll(filepath.Dir(supplierPath), 0o755))
	require.NoError(t, os.WriteFile(supplierPath, []byte(strings.Join([]string{
		"name,account",
		"SAGE KONSTRUKSI INDONESIA,51007",
	}, "\n")), 0o644))

	registryService := appfpcoretax.NewRelationRegistryService(tempDir)
	processor := NewXLSXFPCoretaxProcessor(CollectionKindFPMasukanCoretax, fileStore, registryService)

	workbookPath := filepath.Join(tempDir, "fp-masukan.xlsx")
	createFPCoretaxWorkbook(t, workbookPath, "Nama Penjual")

	ingestResult, err := processor.Ingest(ctx, IngestRequest{
		RequestID:      "req-1",
		UserID:         profileID,
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindFPMasukanCoretax,
		SourceFormat:   SourceFormatXLSX,
		Sources: []IngestSource{{
			SourceID:     "source-1",
			OriginalName: "fp-masukan.xlsx",
			TempPath:     workbookPath,
			UploadedAt:   time.Now(),
		}},
		Policy:      IngestPolicy{KeepRaw: true},
		RequestedAt: time.Now(),
	})
	require.NoError(t, err)

	var normalizedRef string
	var rawRef string
	for _, artifact := range ingestResult.Items[0].Artifacts {
		if artifact.Kind == "normalized" {
			normalizedRef = artifact.ObjectID
		}
		if artifact.Kind == "raw" {
			rawRef = artifact.ObjectID
		}
	}

	inputJSON, err := json.Marshal(map[string]any{
		"sheetName":            "Faktur",
		"outputFilename":       "misc-purchases-test",
		"headerRowNumber":      1,
		"accountNumber":        "41001",
		"memoTemplate":         "{{nomorFakturPajak}}",
		"descriptionTemplate":  "{{namaPenjual}} - {{nomorFakturPajak}}",
		"taxCode":              "PPN",
		"inclusive":            true,
		"partyName":            "Nama Penjual",
		"documentNumber":       "Nomor Faktur Pajak",
		"date":                 "Tanggal Faktur Pajak",
		"taxBase":              "Harga Jual/Penggantian/DPP",
		"tax":                  "PPN",
		"reference":            "Referensi",
	})
	require.NoError(t, err)

	result, err := processor.RunAction(ctx, ActionRequest{
		ActionID:       "action-1",
		UserID:         profileID,
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindFPMasukanCoretax,
		SourceFormat:   SourceFormatXLSX,
		ActionType:     fpMasukanMiscPurchasesActionType,
		SnapshotDocs: []ActionSnapshotDocument{{
			DocumentID:    "doc-1",
			SourceName:    "fp-masukan.xlsx",
			Status:        "ready",
			NormalizedRef: normalizedRef,
			RawRef:        rawRef,
		}},
		Input:       inputJSON,
		RequestedAt: time.Now(),
	})
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)
	require.Len(t, result.Outputs, 1)

	body, readErr := os.ReadFile(result.Outputs[0].ObjectRef)
	require.NoError(t, readErr)
	content := string(body)

	require.Contains(t, content, "Co./Last Name\tFirst Name\tPurchase #\tDate")
	require.Contains(t, content, "SAGE KONSTRUKSI INDONESIA\t\t\t14/10/2025\tX")
	require.Contains(t, content, "\t51007\t5.331.532,00\t5.918.000,00")
	require.Contains(t, content, "SAGE KONSTRUKSI INDONESIA - 04002500352440493")
}

func createFPCoretaxWorkbook(t *testing.T, path string, partyHeader string) {
	t.Helper()

	file := excelize.NewFile()
	file.SetSheetName("Sheet1", "Faktur")

	require.NoError(t, file.SetSheetRow("Faktur", "A1", &[]string{
		partyHeader,
		"Nomor Faktur Pajak",
		"Tanggal Faktur Pajak",
		"Harga Jual/Penggantian/DPP",
		"PPN",
		"Referensi",
	}))
	require.NoError(t, file.SetSheetRow("Faktur", "A2", &[]any{
		map[bool]string{true: "SAGE KONSTRUKSI INDONESIA", false: "BERKAT JAYA ABADI"}[strings.Contains(strings.ToLower(partyHeader), "penjual")],
		"04002500352440493",
		"14/10/2025",
		5331532,
		586468,
		"04002500352440493",
	}))

	require.NoError(t, file.SaveAs(path))
	require.NoError(t, file.Close())
}
