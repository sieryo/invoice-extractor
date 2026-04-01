package document

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/sieryo/invoice-extractor/internal/infra/filestore"
)

func TestXLSXCashflowProcessorIngest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempDir := t.TempDir()
	fileStore := filestore.NewLocalFileStore(filepath.Join(tempDir, "storage"))
	processor := NewXLSXCashflowProcessor(fileStore, nil, nil)

	workbookPath := filepath.Join(tempDir, "example_cashflow.xlsx")
	createCashflowWorkbook(t, workbookPath)

	req := IngestRequest{
		RequestID:      "req-1",
		UserID:         "user-1",
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindCashflowImport,
		SourceFormat:   SourceFormatXLSX,
		Sources: []IngestSource{
			{
				SourceID:     "source-1",
				OriginalName: "example_cashflow.xlsx",
				MimeType:     "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				SizeBytes:    0,
				SHA256:       "sha-1",
				SourceOrder:  0,
				TempPath:     workbookPath,
				UploadedAt:   time.Now(),
			},
		},
		Policy: IngestPolicy{
			KeepRaw:            true,
			DeleteTempAfterRun: false,
		},
		RequestedAt: time.Now(),
	}

	result, err := processor.Ingest(ctx, req)
	require.NoError(t, err)
	require.Len(t, result.Items, 1)

	item := result.Items[0]
	require.Equal(t, IngestStatusReady, item.Status)
	require.Equal(t, "Arus Kas", item.DocumentTag)
	require.Len(t, item.Artifacts, 2)

	var normalizedRef string
	for _, artifact := range item.Artifacts {
		if artifact.Kind == "normalized" {
			normalizedRef = artifact.ObjectID
		}
	}
	require.NotEmpty(t, normalizedRef)

	normalizedBytes, err := fileStore.Read(ctx, req.CollectionID, normalizedRef)
	require.NoError(t, err)

	var payload cashflowNormalizedPayload
	require.NoError(t, json.Unmarshal(normalizedBytes, &payload))
	require.Equal(t, "example_cashflow.xlsx", payload.SourceName)
	require.Equal(t, "Arus Kas", payload.Workbook.PrimarySheet)
	require.Len(t, payload.Workbook.Sheets, 2)
	require.Equal(t, []string{"Tanggal", "Deskripsi", "Nominal"}, payload.Workbook.Sheets[0].Headers)
	require.Equal(t, 2, payload.Workbook.Sheets[0].RowCount)
	require.Empty(t, payload.Workbook.Sheets[0].RawRows)
	require.Empty(t, payload.Workbook.Sheets[0].RawCellRows)
	require.Empty(t, payload.Workbook.Sheets[0].Rows)
	require.Empty(t, payload.Workbook.Sheets[0].CellRows)
}

func createCashflowWorkbook(t *testing.T, path string) {
	t.Helper()

	file := excelize.NewFile()
	file.SetSheetName("Sheet1", "Arus Kas")

	require.NoError(t, file.SetSheetRow("Arus Kas", "A1", &[]string{"Tanggal", "Deskripsi", "Nominal"}))
	require.NoError(t, file.SetSheetRow("Arus Kas", "A2", &[]string{"2026-03-01", "Saldo Awal", "100000"}))
	require.NoError(t, file.SetSheetRow("Arus Kas", "A3", &[]string{"2026-03-02", "Penjualan", "250000"}))

	secondSheet := "Referensi"
	_, err := file.NewSheet(secondSheet)
	require.NoError(t, err)
	require.NoError(t, file.SetSheetRow(secondSheet, "A1", &[]string{"Kode", "Nama"}))
	require.NoError(t, file.SetSheetRow(secondSheet, "A2", &[]string{"100", "Kas"}))

	require.NoError(t, file.SaveAs(path))
	require.NoError(t, file.Close())

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.NotZero(t, info.Size())
}
