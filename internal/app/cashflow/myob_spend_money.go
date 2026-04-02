package cashflow

import (
	"fmt"
	"math"
	"strings"
	"time"
)

var spendMoneyHeaderRow = []string{
	"Cheque Account",
	"Cheque #",
	"Date",
	"Inclusive",
	"Co./Last Name",
	"First Name",
	"Addr 1 - Line 1",
	"      - Line 2",
	"     - Line 3",
	"     - Line 4",
	"Memo",
	"Allocation Account #",
	"Ex-Tax Amount",
	"Inc-Tax Amount",
	"Job #",
	"Tax Code",
	"Non GST/LCT Amount",
	"Tax Amount",
	"Import Duty Amount",
	"Printed",
	"Currency Code",
	"Exchange Rate",
	"Statement Text",
	"Allocation Memo",
	"Category",
}

func SpendMoneyHeader() []string {
	out := make([]string, len(spendMoneyHeaderRow))
	copy(out, spendMoneyHeaderRow)
	return out
}

func BuildSpendMoneyRows(tx SpendMoneyTransaction, chequeAccount string) [][]string {
	rows := make([][]string, 0, len(tx.Items)+2)
	chequeNo := ""
	if tx.ChequeNumber != nil && *tx.ChequeNumber > 0 {
		chequeNo = fmt.Sprintf("%d", *tx.ChequeNumber)
	}
	date := tx.Date.Format("02/01/2006")
	memo := strings.TrimSpace(tx.Memo)
	allocation := strings.TrimSpace(tx.Allocation)
	headerAmount := formatMYOBMoney(tx.Amount)

	rows = append(rows, []string{
		normalizeAccountCode(chequeAccount),
		chequeNo,
		date,
		"X",
		"", "", "", "", "", "",
		memo,
		"",
		headerAmount,
		headerAmount,
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		allocation,
		"",
	})

	for _, item := range tx.Items {
		amount := formatMYOBMoney(item.Amount)
		itemAllocation := strings.TrimSpace(item.Allocation)
		rows = append(rows, []string{
			"",
			"",
			date,
			"X",
			"", "", "", "", "", "",
			memo,
			normalizeAccountCode(item.AccountCode),
			amount,
			amount,
			"",
			"N-T",
			"0,00",
			"0,00",
			"0,00",
			"",
			"",
			"",
			"",
			itemAllocation,
			"",
		})
	}

	rows = append(rows, []string{})
	return rows
}

func EncodeTabDelimitedText(rows [][]string) ([]byte, error) {
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
	return []byte(strings.Join(lines, "\n")), nil
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

	formattedWhole := whole
	if absValue < 1_000_000_000 {
		chunks := make([]string, 0, len(whole)/3+1)
		for len(whole) > 3 {
			chunks = append([]string{whole[len(whole)-3:]}, chunks...)
			whole = whole[:len(whole)-3]
		}
		if whole != "" {
			chunks = append([]string{whole}, chunks...)
		}
		formattedWhole = strings.Join(chunks, ".")
	}

	formatted := formattedWhole + "," + fraction
	if negative {
		return "-" + formatted
	}
	return formatted
}

func normalizeAccountCode(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), "-", "")
}

func EnsureNonZeroTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return t
}
