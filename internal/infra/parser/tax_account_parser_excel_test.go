package parser

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestTaxAccountExcelParser_Parse_AllowsOnlyCanonicalNames(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "tax_accounts.xlsx")

	file := excelize.NewFile()
	require.NoError(t, file.SetSheetRow("Sheet1", "A1", &[]string{"name", "account"}))
	require.NoError(t, file.SetSheetRow("Sheet1", "A2", &[]string{"PPN", "2-2100"}))
	require.NoError(t, file.SetSheetRow("Sheet1", "A3", &[]string{"Admin Bank", "6-6108"}))
	require.NoError(t, file.SaveAs(path))
	require.NoError(t, file.Close())

	parser := NewTaxAccountExcelParser()
	records, issues, err := parser.Parse(path)
	require.NoError(t, err)

	require.Len(t, records, 1)
	require.Equal(t, "PPN", records[0].Name)
	require.Equal(t, "2-2100", records[0].Account)

	require.Len(t, issues, 1)
	require.Equal(t, 3, issues[0].Row)
	require.Equal(t, "name", issues[0].Field)
	require.Equal(t, "Admin Bank", issues[0].Value)
	require.Contains(t, issues[0].Message, "nama tax yang didukung")
}
