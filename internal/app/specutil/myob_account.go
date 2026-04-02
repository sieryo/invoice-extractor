package specutil

import "strings"

const (
	MyobAccountNumberFormatter      = "myob_account_number"
	MyobAccountNumberPattern        = "^[0-9-]+$"
	MyobAccountNumberPatternMessage = "Hanya angka dan '-' yang diperbolehkan"
)

func IsMyobAccountFieldKey(key string) bool {
	switch strings.TrimSpace(key) {
	case "accountNumber", "chequeAccount", "otherCostsAccountCode", "defaultIAccountCode", "defaultBAccountCode":
		return true
	default:
		return false
	}
}
