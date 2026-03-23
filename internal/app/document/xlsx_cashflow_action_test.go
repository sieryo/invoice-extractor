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
	taxAccountsPath := filepath.Join(tempDir, "tax_accounts.csv")
	require.NoError(t, os.WriteFile(taxAccountsPath, []byte(strings.Join([]string{
		"name,account",
		"PPH 23,22003",
		"PPN,10107",
		"Admin Bank,66108",
	}, "\n")), 0644))

	taxService := appcashflow.NewTaxAccountService(taxAccountsPath)
	processor := NewXLSXCashflowProcessor(fileStore, taxService)

	workbookPath := filepath.Join(tempDir, "cashflow.xlsx")
	createSpendMoneyWorkbook(t, workbookPath)

	ingestReq := IngestRequest{
		RequestID:      "req-1",
		UserID:         "user-1",
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
		"cashflowType":          "spend_money",
		"remarkDelimiter":       "*",
		"otherCostsAccountCode": "62099",
		"skipPositiveTotal":     false,
	}
	inputJSON, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := processor.RunAction(ctx, ActionRequest{
		ActionID:       "action-1",
		UserID:         "user-1",
		CollectionID:   "collection-1",
		CollectionKind: CollectionKindCashflowImport,
		SourceFormat:   SourceFormatXLSX,
		ActionType:     cashflowMYOBActionType,
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
	require.Contains(t, content, "\t17500\t03/01/2026\tX")
	require.Contains(t, content, "66023\t6.000.000,00")
	require.Contains(t, content, "22003\t-120.000,00")
	require.Contains(t, content, "10107\t660.000,00")
	require.Contains(t, content, "66108\t26.100,00")
	require.Contains(t, content, "11102\t17501\t02/01/2026")
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
		-120000,
		660000,
		6540000,
		"PT CITRA RETAILINDO UMUM INVCIREUMULFSMUBIX2025 / LISTING FEE 8 SKU THEO SEA",
	}))
	require.NoError(t, file.SetSheetRow("Cashflow", "A3", &[]any{
		"02/01/2026",
		"ADMIN BANK SALARY BA",
		"Admin Bank",
		"",
		"",
		26100,
		"ADMIN BANK SALARY BA",
	}))

	require.NoError(t, file.SaveAs(path))
	require.NoError(t, file.Close())
}
