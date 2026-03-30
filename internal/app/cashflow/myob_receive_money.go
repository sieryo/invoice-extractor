package cashflow

import "fmt"

var receiveMoneyHeaderRow = []string{
	"Deposit Account",
	"ID #",
	"Date",
	"Co./Last Name",
	"First Name",
	"Memo",
	"Inclusive",
	"Allocation Account #",
	"Ex-Tax Amount",
	"Inc-Tax Amount",
	"Job #",
	"Tax Code",
	"Non GST/LCT Amount",
	"Tax Amount",
	"LCT Amount",
	"Currency Code",
	"Exchange Rate",
	"Payment Method",
	"Drawer/Account Name",
	"BSB",
	"Account Number",
	"Cheque Number",
	"Card Number",
	"Name on Card",
	"Expiry Date",
	"Authorisation Number",
	"Notes",
	"Allocation Memo",
	"Category",
}

func ReceiveMoneyHeader() []string {
	out := make([]string, len(receiveMoneyHeaderRow))
	copy(out, receiveMoneyHeaderRow)
	return out
}

func BuildReceiveMoneyRows(tx ReceiveMoneyTransaction, depositAccount string) [][]string {
	rows := make([][]string, 0, len(tx.Items)+2)
	idNumber := ""
	if tx.IDNumber != nil && *tx.IDNumber > 0 {
		idNumber = fmt.Sprintf("%d", *tx.IDNumber)
	}
	date := tx.Date.Format("02/01/2006")
	extraMemo := tx.Allocation
	headerAmount := formatMYOBMoney(tx.Amount)

	rows = append(rows, []string{
		normalizeAccountCode(depositAccount),
		idNumber,
		date,
		"",
		"",
		tx.Memo,
		"X",
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
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		extraMemo,
		"",
	})

	for _, item := range tx.Items {
		amount := formatMYOBMoney(item.Amount)
		rows = append(rows, []string{
			normalizeAccountCode(depositAccount),
			idNumber,
			date,
			"",
			"",
			tx.Memo,
			"X",
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
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			item.Allocation,
			"",
		})
	}

	rows = append(rows, []string{})
	return rows
}
