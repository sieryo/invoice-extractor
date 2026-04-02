package cashflowbill

import (
	"fmt"
	"math"
	"strings"
	"time"
)

var payBillsHeaderRow = []string{
	"CO/LAST NAME",
	"FIRST NAME",
	"LINE 1",
	"LINE 2",
	"LINE 3",
	"LINE 4",
	"PAYMENT ACCOUNT",
	"CHEQUE NUMBER",
	"PAYMENT DATE",
	"STATEMENT TEXT",
	"PURCHASE",
	"SUPPLIER",
	"BILL DATE",
	"AMOUNT APPLIED",
	"MEMO",
	"ALREADY PRINTED",
	"CURRENCY CODE",
	"EXCHANGE RATE",
}

var receivePaymentsHeaderRow = []string{
	"CO./LAST NAME",
	"FIRST NAME",
	"DEPOSIT ACCOUNT #",
	"ID #",
	"RECEIPT DATE",
	"INVOICE #",
	"CUSTOMER PO #",
	"INVOICE DATE",
	"AMOUNT APPLIED",
	"MEMO",
	"CURRENCY CODE",
	"EXCHANGE RATE",
	"PAYMENT METHOD",
	"PAYMENT NOTES",
	"NAME ON CARD",
	"CARD NUMBER",
	"EXPIRY DATE",
	"AUTHORISATION NUMBER",
	"BSB",
	"ACCOUNT NUMBER",
	"DRAWER/ACCOUNT NAME",
	"CHEQUE NUMBER",
}

func PayBillsHeader() []string {
	out := make([]string, len(payBillsHeaderRow))
	copy(out, payBillsHeaderRow)
	return out
}

func ReceivePaymentsHeader() []string {
	out := make([]string, len(receivePaymentsHeaderRow))
	copy(out, receivePaymentsHeaderRow)
	return out
}

func BuildPayBillsRows(tx Transaction, account string) [][]string {
	rows := make([][]string, 0, len(tx.Items)*2)
	for _, item := range tx.Items {
		rows = append(rows, []string{
			tx.Party,
			"",
			"",
			"",
			"",
			"",
			normalizeAccountCode(account),
			"",
			formatDate(item.ActionDate),
			"",
			item.RefID,
			"",
			formatDate(item.RefDate),
			formatMYOBMoney(item.Amount),
			memoForMYOBOutput(item),
			"",
			"",
			"",
		})
		rows = append(rows, []string{})
	}
	return rows
}

func BuildReceivePaymentsRows(tx Transaction, account string) [][]string {
	rows := make([][]string, 0, len(tx.Items)*2)
	for _, item := range tx.Items {
		rows = append(rows, []string{
			tx.Party,
			"",
			normalizeAccountCode(account),
			"",
			formatDate(item.ActionDate),
			item.RefID,
			"",
			formatDate(item.RefDate),
			formatMYOBMoney(item.Amount),
			memoForMYOBOutput(item),
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
		})
		rows = append(rows, []string{})
	}
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

func memoForMYOBOutput(item TransactionItem) string {
	if strings.TrimSpace(item.RefID) == "" {
		return ""
	}
	return strings.TrimSpace(item.Memo)
}

func formatDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("02/01/2006")
}
