package fpcoretax

import (
	"fmt"
	"math"
	"strings"
	"time"
)

var miscSalesHeaderRow = []string{
	"Co./Last Name",
	"First Name",
	"Invoice #",
	"Date",
	"Inclusive",
	"Memo",
	"Salesperson Last Name",
	"Salesperson First Name",
	"Referral Source",
	"Description",
	"Account Number",
	"Amount",
	"Inc-Tax Amount",
	"Job",
	"Tax Code",
	"Non-GST Amount",
	"GST Amount",
	"LCT Amount",
	"Sale Status",
	"Currency Code",
	"Exchange Rate",
	"Terms - Payment is Due",
	" - Discount Days",
	" - Balance Due Days",
	" - % Discount",
	" - % Monthly Charge",
	"Amount Paid",
	"Payment Method",
	"Payment Notes",
	"Name on Card",
	"Card Number",
	"Expiry Date",
	"Authorisation Number",
	"BSB",
	"Account Number",
	"Drawer/Account Name",
	"Cheque Number",
	"Category",
}

var miscPurchasesHeaderRow = []string{
	"Co./Last Name",
	"First Name",
	"Purchase #",
	"Date",
	"Inclusive",
	"Memo",
	"Description",
	"Account Number",
	"Amount",
	"Inc-Tax Amount",
	"Job",
	"Tax Code",
	"Non-GST Amount",
	"GST Amount",
	"Import Duty Amount",
	"Purchase Status",
	"Currency Code",
	"Exchange Rate",
	"Terms - Payment is Due",
	" - Discount Days",
	" - Balance Due Days",
	" - % Discount",
	"Amount Paid",
	"Category",
}

type TransactionEntry struct {
	PartyName      string
	AccountNumber  string
	Date           time.Time
	Memo           string
	Description    string
	Amount         float64
	GSTAmount      float64
	IncTaxAmount   float64
	TaxCode        string
	Inclusive      bool
}

func MiscSalesHeader() []string {
	out := make([]string, len(miscSalesHeaderRow))
	copy(out, miscSalesHeaderRow)
	return out
}

func MiscPurchasesHeader() []string {
	out := make([]string, len(miscPurchasesHeaderRow))
	copy(out, miscPurchasesHeaderRow)
	return out
}

func BuildMiscSalesRow(entry TransactionEntry) []string {
	return []string{
		entry.PartyName,
		"",
		"",
		ensureDate(entry.Date).Format("02/01/2006"),
		inclusiveMarker(entry.Inclusive),
		entry.Memo,
		"", "", "",
		entry.Description,
		normalizeAccountCode(entry.AccountNumber),
		formatMYOBMoney(entry.Amount),
		formatMYOBMoney(entry.IncTaxAmount),
		"",
		strings.TrimSpace(entry.TaxCode),
		"0",
		formatMYOBMoney(entry.GSTAmount),
		"0",
		"I",
		"",
		"",
		"0", "0", "0", "0", "0",
		"0",
		"",
		"", "", "", "", "", "", "", "", "", "",
	}
}

func BuildMiscPurchasesRow(entry TransactionEntry) []string {
	return []string{
		entry.PartyName,
		"",
		"",
		ensureDate(entry.Date).Format("02/01/2006"),
		inclusiveMarker(entry.Inclusive),
		entry.Memo,
		entry.Description,
		normalizeAccountCode(entry.AccountNumber),
		formatMYOBMoney(entry.Amount),
		formatMYOBMoney(entry.IncTaxAmount),
		"",
		strings.TrimSpace(entry.TaxCode),
		"0",
		formatMYOBMoney(entry.GSTAmount),
		"0",
		"B",
		"",
		"",
		"0", "0", "0", "0",
		"0",
		"",
	}
}

func EncodeTabDelimitedText(rows [][]string) []byte {
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		if len(row) == 0 {
			lines = append(lines, "")
			continue
		}

		var builder strings.Builder
		for idx, value := range row {
			if idx > 0 {
				builder.WriteByte('\t')
			}
			builder.WriteString(strings.ReplaceAll(value, "\t", " "))
		}
		lines = append(lines, builder.String())
	}
	return []byte(strings.Join(lines, "\r\n"))
}

func formatMYOBMoney(value float64) string {
	negative := value < 0
	absValue := math.Abs(value)
	raw := fmt.Sprintf("%.2f", absValue)
	parts := strings.SplitN(raw, ".", 2)
	whole := parts[0]
	fraction := "00"
	if len(parts) == 2 {
		fraction = parts[1]
	}

	chunks := make([]string, 0, len(whole)/3+1)
	for len(whole) > 3 {
		chunks = append([]string{whole[len(whole)-3:]}, chunks...)
		whole = whole[:len(whole)-3]
	}
	if whole != "" {
		chunks = append([]string{whole}, chunks...)
	}
	formatted := strings.Join(chunks, ".") + "," + fraction
	if negative {
		return "-" + formatted
	}
	return formatted
}

func FormatMYOBTemplateNumber(value float64) string {
	return formatMYOBMoney(value)
}

func normalizeAccountCode(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "-", "")
}

func inclusiveMarker(inclusive bool) string {
	if inclusive {
		return "X"
	}
	return ""
}

func ensureDate(value time.Time) time.Time {
	if !value.IsZero() {
		return value
	}
	return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
}
