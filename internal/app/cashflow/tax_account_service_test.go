package cashflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTaxAccountService_LoadAndLookup(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "tax_accounts.csv")
	err := os.WriteFile(path, []byte("name,account\nPPH 23,22003\nPPN,10107\nAdmin Bank,66108\n"), 0644)
	require.NoError(t, err)

	service := NewTaxAccountService(path)
	status := service.Status()
	require.True(t, status.Loaded)
	require.Equal(t, 3, status.Count)

	record, ok, err := service.Lookup("pph 23")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "22003", record.Account)
}
