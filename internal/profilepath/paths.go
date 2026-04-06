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

func ProfileModulesJSON(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "modules.json")
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

func BukpotRequestConfigJSON(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "request_bukpot.json")
}

func BukpotActionProfilesDir(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "bukpot_actions")
}

func BukpotActionProfileJSON(rootDir string, profileID string, key string) string {
	return filepath.Join(BukpotActionProfilesDir(rootDir, profileID), strings.TrimSpace(key)+".json")
}

func CashflowConfigDir(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "cashflow")
}

func FPCoretaxConfigDir(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "fp_coretax")
}

func FPKeluaranMiscSalesConfigJSON(rootDir string, profileID string) string {
	return filepath.Join(FPCoretaxConfigDir(rootDir, profileID), "fp_keluaran_misc_sales.json")
}

func FPKeluaranReturMiscSalesConfigJSON(rootDir string, profileID string) string {
	return filepath.Join(FPCoretaxConfigDir(rootDir, profileID), "fp_keluaran_retur_misc_sales.json")
}

func FPMasukanMiscPurchasesConfigJSON(rootDir string, profileID string) string {
	return filepath.Join(FPCoretaxConfigDir(rootDir, profileID), "fp_masukan_misc_purchases.json")
}

func FPCoretaxCustomerCSV(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "fp_keluaran_customers.csv")
}

func FPCoretaxSupplierCSV(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "fp_masukan_suppliers.csv")
}

func FPCoretaxCustomerUploadTempXLSX(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "fp_keluaran_customer_upload.xlsx")
}

func FPCoretaxSupplierUploadTempXLSX(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "fp_masukan_supplier_upload.xlsx")
}

func CashflowSpendMoneyConfigJSON(rootDir string, profileID string) string {
	return filepath.Join(CashflowConfigDir(rootDir, profileID), "spend_money.json")
}

func CashflowReceiveMoneyConfigJSON(rootDir string, profileID string) string {
	return filepath.Join(CashflowConfigDir(rootDir, profileID), "receive_money.json")
}

func CashflowBillsConfigDir(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "cashflow_bills")
}

func CashflowPayBillsConfigJSON(rootDir string, profileID string) string {
	return filepath.Join(CashflowBillsConfigDir(rootDir, profileID), "pay_bills.json")
}

func CashflowReceivePaymentsConfigJSON(rootDir string, profileID string) string {
	return filepath.Join(CashflowBillsConfigDir(rootDir, profileID), "receive_payments.json")
}

func CashflowCategoryAccountsCSV(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "category_accounts.csv")
}

func CashflowCategoryAccountsUploadTempXLSX(rootDir string, profileID string) string {
	return filepath.Join(ProfileDir(rootDir, profileID), "category_accounts_upload.xlsx")
}
