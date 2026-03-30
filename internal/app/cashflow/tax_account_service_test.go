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
	profileID := "profile-1"
	path := filepath.Join(tempDir, "profiles", profileID, "tax_accounts.csv")
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	require.NoError(t, err)
	err = os.WriteFile(path, []byte("name,account\nPPH 23,22003\nPPN,10107\n"), 0644)
	require.NoError(t, err)

	service := NewTaxAccountService(tempDir)
	status := service.Status(profileID)
	require.True(t, status.Loaded)
	require.Equal(t, 2, status.Count)

	record, ok, err := service.Lookup(profileID, "pph 23")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "22003", record.Account)
}
