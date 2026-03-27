package profilepath

import (
	"path/filepath"
	"strings"
)

const (
	profilesDirName = "profiles"
)

func ProfilesRoot(rootDir string) string {
	return filepath.Join(rootDir, profilesDirName)
}

func ProfileDir(rootDir string, profileID string) string {
	normalized := strings.TrimSpace(profileID)
	if normalized == "" {
		normalized = "default"
	}
	return filepath.Join(ProfilesRoot(rootDir), normalized)
}

func ProfileMetadataJSON(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "profile.json")
}

func BuyersCSV(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "buyers.csv")
}

func TaxAccountsCSV(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "tax_accounts.csv")
}

func BuyerUploadTempXLSX(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "buyer_upload.xlsx")
}

func TaxAccountsUploadTempXLSX(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "tax_accounts_upload.xlsx")
}
