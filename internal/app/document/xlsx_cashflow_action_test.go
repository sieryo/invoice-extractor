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

	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
	"github.com/sieryo/invoice-extractor/internal/infra/filestore"
)

func TestXLSXCashflowProcessor_RunActionSpendMoney(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempDir := t.TempDir()
	fileStore := filestore.NewLocalFileStore(filepath.Join(tempDir, "storage"))
	profileID := "user-1"
	taxAccountsPath := filepath.Join(tempDir, "profiles", profileID, "tax_accounts.csv")
	require.NoError(t, os.MkdirAll(filepath.Dir(taxAccountsPath), 0o755))
	require.NoError(t, os.WriteFile(taxAccountsPath, []byte(strings.Join([]string{
		"name,account",
		"PPH 15%,22007",
		"PPH 21,22001",
		"PPH 23,22003",
		"PPH 25,1307",
		"PPH 4 (2),22006",
		"PP 23,22005",
		"PPN,10107",
	}, "\n")), 0644))

	taxService := appcashflow.NewTaxAccountService(tempDir)
	processor := NewXLSXCashflowProcessor(fileStore, taxService)

	workbookPath := filepath.Join(tempDir, "cashflow.xlsx")
	createSpendMoneyWorkbook(t, workbookPath)

	ingestReq := IngestRequest{
		RequestID:      "req-1",
		UserID:         profileID,
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindCashflowImport,
		SourceFormat:   SourceFormatXLSX,
		Sources: []IngestSource{
			{
				SourceID:     "source-1",
				OriginalName: "cashflow.xlsx",
				TempPath:     workbookPath,
				UploadedAt:   time.Now(),
			},
		},
		Policy: IngestPolicy{
			KeepRaw: true,
		},
		RequestedAt: time.Now(),
	}

	ingestResult, err := processor.Ingest(ctx, ingestReq)
	require.NoError(t, err)
	require.Len(t, ingestResult.Items, 1)
	require.Equal(t, IngestStatusReady, ingestResult.Items[0].Status)

	var normalizedRef string
	for _, artifact := range ingestResult.Items[0].Artifacts {
		if artifact.Kind == "normalized" {
			normalizedRef = artifact.ObjectID
		}
	}
	require.NotEmpty(t, normalizedRef)

	input := map[string]any{
		"sheetName":             "Cashflow",
		"outputFilename":        "cashflow-spend-money",
		"headerRowNumber":       1,
		"startingChequeNumber":  17500,
		"chequeAccount":         "11102",
		"cashflowFormat":        "default",
		"remarkDelimiter":       "*",
		"otherCostsAccountCode": "62099",
	}
	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := processor.RunAction(ctx, ActionRequest{
		ActionID:       "action-1",
		UserID:         profileID,
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindCashflowImport,
		SourceFormat:   SourceFormatXLSX,
		ActionType:     cashflowSpendActionType,
		SnapshotDocs: []ActionSnapshotDocument{
			{
				DocumentID:    "doc-1",
				SourceName:    "cashflow.xlsx",
				Status:        "ready",
				NormalizedRef: normalizedRef,
			},
		},
		Input:       inputJSON,
		RequestedAt: time.Now(),
	})
	require.NoError(t, err)
	require.Equal(t, "success", result.Status)
	require.Len(t, result.ItemResults, 1)
	require.Len(t, result.Outputs, 1)
	require.Equal(t, "text/plain; charset=utf-8", result.Outputs[0].MimeType)
	require.True(t, strings.HasSuffix(result.Outputs[0].Name, ".txt"))

	body, readErr := os.ReadFile(result.Outputs[0].ObjectRef)
	require.NoError(t, readErr)
	content := string(body)

	require.Contains(t, content, "Cheque Account\tCheque #\tDate")
	require.Contains(t, content, "11102\t17500\t03/01/2026")
	require.Contains(t, content, "\n\t\t03/01/2026\tX")
	require.Contains(t, content, "13.625.000,00\t13.625.000,00")
	require.Contains(t, content, "66023\t12.500.000,00")
	require.Contains(t, content, "10107\t1.375.000,00")
	require.Contains(t, content, "22003\t-250.000,00")
	require.Contains(t, content, "66108\t26.100,00")
	require.Contains(t, content, "11102\t17501\t02/01/2026")
	require.NotContains(t, content, "\"      - Line 2\"")
	require.NotContains(t, content, "PPh 23 - PPN")
}

func TestXLSXCashflowProcessor_RunActionReceiveMoney(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempDir := t.TempDir()
	fileStore := filestore.NewLocalFileStore(filepath.Join(tempDir, "storage"))
	profileID := "user-1"
	taxAccountsPath := filepath.Join(tempDir, "profiles", profileID, "tax_accounts.csv")
	require.NoError(t, os.MkdirAll(filepath.Dir(taxAccountsPath), 0o755))
	require.NoError(t, os.WriteFile(taxAccountsPath, []byte(strings.Join([]string{
		"name,account",
		"PPH 15%,22007",
		"PPH 21,22001",
		"PPH 23,22003",
		"PPH 25,1307",
		"PPH 4 (2),22006",
		"PP 23,22005",
		"PPN,10107",
	}, "\n")), 0644))

	taxService := appcashflow.NewTaxAccountService(tempDir)
	processor := NewXLSXCashflowProcessor(fileStore, taxService)

	workbookPath := filepath.Join(tempDir, "cashflow-receive.xlsx")
	createReceiveMoneyWorkbook(t, workbookPath)

	ingestReq := IngestRequest{
		RequestID:      "req-1",
		UserID:         profileID,
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindCashflowImport,
		SourceFormat:   SourceFormatXLSX,
		Sources: []IngestSource{{
			SourceID:     "source-1",
			OriginalName: "cashflow-receive.xlsx",
			TempPath:     workbookPath,
			UploadedAt:   time.Now(),
		}},
		Policy:      IngestPolicy{KeepRaw: true},
		RequestedAt: time.Now(),
	}

	ingestResult, err := processor.Ingest(ctx, ingestReq)
	require.NoError(t, err)
	require.Len(t, ingestResult.Items, 1)
	require.Equal(t, IngestStatusReady, ingestResult.Items[0].Status)

	var normalizedRef string
	for _, artifact := range ingestResult.Items[0].Artifacts {
		if artifact.Kind == "normalized" {
			normalizedRef = artifact.ObjectID
		}
	}
	require.NotEmpty(t, normalizedRef)

	inputJSON, err := json.Marshal(map[string]any{
		"sheetName":       "Cashflow",
		"outputFilename":  "cashflow-receive-money",
		"headerRowNumber": 1,
		"chequeAccount":   "11102",
		"cashflowFormat":  "default",
	})
	require.NoError(t, err)

	result, err := processor.RunAction(ctx, ActionRequest{
		ActionID:       "action-1",
		UserID:         profileID,
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindCashflowImport,
		SourceFormat:   SourceFormatXLSX,
		ActionType:     cashflowRecvActionType,
		SnapshotDocs: []ActionSnapshotDocument{{
			DocumentID:    "doc-1",
			SourceName:    "cashflow-receive.xlsx",
			Status:        "ready",
			NormalizedRef: normalizedRef,
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

	require.Contains(t, content, "Deposit Account\tID #\tDate")
	require.Contains(t, content, "11102\t\t03/01/2026")
	require.Contains(t, content, "66023\t6.540.000,00")
	require.Contains(t, content, "66108\t26.100,00")
}

func TestXLSXCashflowProcessor_RunActionSpendMoney_SkipsMissingChartOfAccounts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tempDir := t.TempDir()
	fileStore := filestore.NewLocalFileStore(filepath.Join(tempDir, "storage"))
	profileID := "user-1"
	taxAccountsPath := filepath.Join(tempDir, "profiles", profileID, "tax_accounts.csv")
	require.NoError(t, os.MkdirAll(filepath.Dir(taxAccountsPath), 0o755))
	require.NoError(t, os.WriteFile(taxAccountsPath, []byte(strings.Join([]string{
		"name,account",
		"PPH 15%,22007",
		"PPH 21,22001",
		"PPH 23,22003",
		"PPH 25,1307",
		"PPH 4 (2),22006",
		"PP 23,22005",
		"PPN,10107",
	}, "\n")), 0644))

	taxService := appcashflow.NewTaxAccountService(tempDir)
	processor := NewXLSXCashflowProcessor(fileStore, taxService)

	workbookPath := filepath.Join(tempDir, "cashflow-missing-coa.xlsx")
	createSpendMoneyWorkbookWithMissingCOA(t, workbookPath)

	ingestResult, err := processor.Ingest(ctx, IngestRequest{
		RequestID:      "req-1",
		UserID:         profileID,
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindCashflowImport,
		SourceFormat:   SourceFormatXLSX,
		Sources: []IngestSource{{
			SourceID:     "source-1",
			OriginalName: "cashflow-missing-coa.xlsx",
			TempPath:     workbookPath,
			UploadedAt:   time.Now(),
		}},
		Policy:      IngestPolicy{KeepRaw: true},
		RequestedAt: time.Now(),
	})
	require.NoError(t, err)

	var normalizedRef string
	for _, artifact := range ingestResult.Items[0].Artifacts {
		if artifact.Kind == "normalized" {
			normalizedRef = artifact.ObjectID
		}
	}
	require.NotEmpty(t, normalizedRef)

	inputJSON, err := json.Marshal(map[string]any{
		"sheetName":             "Cashflow",
		"outputFilename":        "cashflow-spend-money",
		"headerRowNumber":       1,
		"startingChequeNumber":  17500,
		"chequeAccount":         "11102",
		"cashflowFormat":        "default",
		"remarkDelimiter":       "*",
		"otherCostsAccountCode": "62099",
	})
	require.NoError(t, err)

	result, err := processor.RunAction(ctx, ActionRequest{
		ActionID:       "action-1",
		UserID:         profileID,
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindCashflowImport,
		SourceFormat:   SourceFormatXLSX,
		ActionType:     cashflowSpendActionType,
		SnapshotDocs: []ActionSnapshotDocument{{
			DocumentID:    "doc-1",
			SourceName:    "cashflow-missing-coa.xlsx",
			Status:        "ready",
			NormalizedRef: normalizedRef,
		}},
		Input:       inputJSON,
		RequestedAt: time.Now(),
	})
	require.NoError(t, err)
	require.Equal(t, "warning", result.ItemResults[0].Status)
	require.NotEmpty(t, result.ItemResults[0].Warnings)
	require.Contains(t, strings.Join(result.ItemResults[0].Warnings, " | "), `chart of accounts "ADMIN BANK" tidak ditemukan, row dilewati`)

	body, readErr := os.ReadFile(result.Outputs[0].ObjectRef)
	require.NoError(t, readErr)
	content := string(body)
	require.Contains(t, content, "66023\t6.540.000,00")
	require.NotContains(t, content, "ADMIN BANK SALARY BA")
}

func createSpendMoneyWorkbook(t *testing.T, path string) {
	t.Helper()

	file := excelize.NewFile()
	file.SetSheetName("Sheet1", "Cashflow")

	require.NoError(t, file.SetSheetRow("Cashflow", "A1", &[]string{
		"Tanggal", "note", "coa", "PPH 23", "PPN", "idr", "catatan",
	}))
	require.NoError(t, file.SetSheetRow("Cashflow", "A2", &[]any{
		"03/01/2026",
		"PT CITRA RETAILINDO UMUM INVCIREUMULFSMUBIX2025 / LISTING FEE 8 SKU THEO SEA",
		"66023",
		250000,
		1375000,
		-13625000,
		"PT CITRA RETAILINDO UMUM INVCIREUMULFSMUBIX2025 / LISTING FEE 8 SKU THEO SEA",
	}))
	require.NoError(t, file.SetSheetRow("Cashflow", "A3", &[]any{
		"02/01/2026",
		"ADMIN BANK SALARY BA",
		"66108",
		"",
		"",
		-26100,
		"ADMIN BANK SALARY BA",
	}))

	require.NoError(t, file.SaveAs(path))
	require.NoError(t, file.Close())
}

func createReceiveMoneyWorkbook(t *testing.T, path string) {
	t.Helper()

	file := excelize.NewFile()
	file.SetSheetName("Sheet1", "Cashflow")

	require.NoError(t, file.SetSheetRow("Cashflow", "A1", &[]string{
		"Tanggal", "note", "coa", "PPH 23", "PPN", "idr", "catatan",
	}))
	require.NoError(t, file.SetSheetRow("Cashflow", "A2", &[]any{
		"03/01/2026",
		"PEMASUKAN MARKETPLACE",
		"66023",
		"",
		"",
		6540000,
		"PEMASUKAN MARKETPLACE",
	}))
	require.NoError(t, file.SetSheetRow("Cashflow", "A3", &[]any{
		"02/01/2026",
		"ADMIN BANK MASUK",
		"66108",
		"",
		"",
		26100,
		"ADMIN BANK MASUK",
	}))

	require.NoError(t, file.SaveAs(path))
	require.NoError(t, file.Close())
}

func createSpendMoneyWorkbookWithMissingCOA(t *testing.T, path string) {
	t.Helper()

	file := excelize.NewFile()
	file.SetSheetName("Sheet1", "Cashflow")

	require.NoError(t, file.SetSheetRow("Cashflow", "A1", &[]string{
		"Tanggal", "note", "coa", "PPH 23", "PPN", "idr", "catatan",
	}))
	require.NoError(t, file.SetSheetRow("Cashflow", "A2", &[]any{
		"03/01/2026",
		"PT CITRA RETAILINDO UMUM INVCIREUMULFSMUBIX2025 / LISTING FEE 8 SKU THEO SEA",
		"66023",
		"",
		"",
		-6540000,
		"PT CITRA RETAILINDO UMUM INVCIREUMULFSMUBIX2025 / LISTING FEE 8 SKU THEO SEA",
	}))
	require.NoError(t, file.SetSheetRow("Cashflow", "A3", &[]any{
		"02/01/2026",
		"ADMIN BANK SALARY BA",
		"Admin Bank",
		"",
		"",
		-26100,
		"ADMIN BANK SALARY BA",
	}))

	require.NoError(t, file.SaveAs(path))
	require.NoError(t, file.Close())
}

func TestNormalizeCashflowRecord(t *testing.T) {
	t.Parallel()

	record := normalizeCashflowRecord(cashflowRowRecord{
		Information: "Admin Bank",
		COA:         "ab-120",
		Total:       -1000,
		OtherCost:   250,
		PPH21:       25,
		PPN:         110,
	})

	require.Equal(t, "ADMIN BANK", record.Information)
	require.Equal(t, "AB-120", record.COA)
	require.Equal(t, 1000.0, record.Total)
	require.Equal(t, 250.0, record.OtherCost)
	require.Equal(t, -25.0, record.PPH21)
	require.Equal(t, 110.0, record.PPN)
}

func TestResolveCashflowAllocationMemo_UsesTextAfterFirstDelimiter(t *testing.T) {
	t.Parallel()

	got := resolveCashflowAllocationMemo(cashflowRowRecord{
		Information: "KETERANGAN UTAMA",
		Remark:      "PPH*hanya jasa",
	}, appcashflow.ExportMYOBInput{
		RemarkDelimiter: "*",
	})

	require.Equal(t, "hanya jasa", got)
}
