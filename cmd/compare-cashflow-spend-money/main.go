package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	legacyTXT       = "spend_money_legacy.txt"
	currentTXT      = "spend_money_new.txt"
	maxDiffsToPrint = 200
)

type cashflowFile struct {
	Header              []string
	RawLineCount        int
	NonEmptyRowCount    int
	BlankSeparatorCount int
	TransactionCount    int
	ItemRowCount        int
	Transactions        []cashflowTransaction
}

type cashflowTransaction struct {
	Header cashflowRow
	Items  []cashflowRow
}

type cashflowRow struct {
	ChequeAccount     string
	ChequeNumber      string
	Date              string
	Inclusive         string
	Memo              string
	AllocationAccount string
	ExTaxAmount       string
	IncTaxAmount      string
	TaxCode           string
	AllocationMemo    string
}

func main() {
	assetsDir, err := resolveAssetsDir()
	if err != nil {
		fail(err)
	}

	legacy, err := loadCashflowFile(filepath.Join(assetsDir, legacyTXT))
	if err != nil {
		fail(err)
	}
	current, err := loadCashflowFile(filepath.Join(assetsDir, currentTXT))
	if err != nil {
		fail(err)
	}

	diffs := diffCashflowFiles(legacy, current)

	fmt.Println("Compare Cashflow Spend Money")
	fmt.Println("============================")
	fmt.Printf("Assets dir            : %s\n", assetsDir)
	fmt.Printf("Header columns        : %d\n", len(current.Header))
	fmt.Printf("Transactions          : legacy=%d current=%d\n", legacy.TransactionCount, current.TransactionCount)
	fmt.Printf("Item rows             : legacy=%d current=%d\n", legacy.ItemRowCount, current.ItemRowCount)
	fmt.Printf("Non-empty rows        : legacy=%d current=%d\n", legacy.NonEmptyRowCount, current.NonEmptyRowCount)
	fmt.Printf("Blank separators      : legacy=%d current=%d\n", legacy.BlankSeparatorCount, current.BlankSeparatorCount)
	fmt.Println()

	if len(diffs) == 0 {
		fmt.Println("RESULT: SAME")
		return
	}

	fmt.Printf("RESULT: DIFFERENT (%d)\n", len(diffs))
	limit := len(diffs)
	if limit > maxDiffsToPrint {
		limit = maxDiffsToPrint
	}
	for i := 0; i < limit; i++ {
		fmt.Printf("- %s\n", diffs[i])
	}
	if len(diffs) > limit {
		fmt.Printf("- ... +%d diff lainnya\n", len(diffs)-limit)
	}
	os.Exit(1)
}

func resolveAssetsDir() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("failed to resolve current file location")
	}

	backendDir := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	assetsDir := filepath.Join(backendDir, "assets", "cashflow", "spend_money")
	info, err := os.Stat(assetsDir)
	if err != nil {
		return "", fmt.Errorf("failed to access assets dir: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("assets path is not a directory: %s", assetsDir)
	}
	return assetsDir, nil
}

func loadCashflowFile(path string) (cashflowFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return cashflowFile{}, fmt.Errorf("failed to read %s: %w", path, err)
	}

	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return cashflowFile{}, fmt.Errorf("empty file: %s", path)
	}

	header := splitTabLine(lines[0])
	result := cashflowFile{
		Header:       header,
		RawLineCount: len(lines),
		Transactions: make([]cashflowTransaction, 0),
	}

	var current *cashflowTransaction
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			result.BlankSeparatorCount++
			current = nil
			continue
		}

		result.NonEmptyRowCount++
		row := parseCashflowRow(splitTabLine(line))
		if row.Date == "" && row.Memo == "" && row.AllocationAccount == "" && row.AllocationMemo == "" {
			continue
		}

		if row.AllocationAccount == "" {
			result.TransactionCount++
			result.Transactions = append(result.Transactions, cashflowTransaction{Header: row})
			current = &result.Transactions[len(result.Transactions)-1]
			continue
		}

		result.ItemRowCount++
		if current == nil {
			result.Transactions = append(result.Transactions, cashflowTransaction{})
			current = &result.Transactions[len(result.Transactions)-1]
		}
		current.Items = append(current.Items, row)
	}

	return result, nil
}

func splitTabLine(line string) []string {
	return strings.Split(line, "\t")
}

func parseCashflowRow(values []string) cashflowRow {
	return cashflowRow{
		ChequeAccount:     valueAt(values, 0),
		ChequeNumber:      valueAt(values, 1),
		Date:              valueAt(values, 2),
		Inclusive:         valueAt(values, 3),
		Memo:              valueAt(values, 10),
		AllocationAccount: valueAt(values, 11),
		ExTaxAmount:       valueAt(values, 12),
		IncTaxAmount:      valueAt(values, 13),
		TaxCode:           valueAt(values, 15),
		AllocationMemo:    valueAt(values, 23),
	}
}

func diffCashflowFiles(legacy, current cashflowFile) []string {
	diffs := make([]string, 0)

	if !equalSlices(legacy.Header, current.Header) {
		diffs = append(diffs, fmt.Sprintf("header berbeda: legacy=%v current=%v", legacy.Header, current.Header))
	}
	if legacy.TransactionCount != current.TransactionCount {
		diffs = append(diffs, fmt.Sprintf("jumlah transaksi berbeda: legacy=%d current=%d", legacy.TransactionCount, current.TransactionCount))
	}
	if legacy.ItemRowCount != current.ItemRowCount {
		diffs = append(diffs, fmt.Sprintf("jumlah item row berbeda: legacy=%d current=%d", legacy.ItemRowCount, current.ItemRowCount))
	}
	if legacy.NonEmptyRowCount != current.NonEmptyRowCount {
		diffs = append(diffs, fmt.Sprintf("jumlah non-empty row berbeda: legacy=%d current=%d", legacy.NonEmptyRowCount, current.NonEmptyRowCount))
	}
	if legacy.BlankSeparatorCount != current.BlankSeparatorCount {
		diffs = append(diffs, fmt.Sprintf("jumlah blank separator berbeda: legacy=%d current=%d", legacy.BlankSeparatorCount, current.BlankSeparatorCount))
	}

	maxTransactions := max(len(legacy.Transactions), len(current.Transactions))
	for i := 0; i < maxTransactions; i++ {
		switch {
		case i >= len(legacy.Transactions):
			diffs = append(diffs, fmt.Sprintf("transaksi %d hanya ada di current", i+1))
			continue
		case i >= len(current.Transactions):
			diffs = append(diffs, fmt.Sprintf("transaksi %d hanya ada di legacy", i+1))
			continue
		}

		legacyTx := legacy.Transactions[i]
		currentTx := current.Transactions[i]
		memoHint := transactionMemoHint(legacyTx, currentTx)

		diffs = append(diffs, diffRows(fmt.Sprintf("transaksi %d [%s] / header", i+1, memoHint), legacyTx.Header, currentTx.Header)...)
		if len(legacyTx.Items) != len(currentTx.Items) {
			diffs = append(diffs, fmt.Sprintf("transaksi %d [%s] / jumlah item berbeda: legacy=%d current=%d", i+1, memoHint, len(legacyTx.Items), len(currentTx.Items)))
		}

		maxItems := max(len(legacyTx.Items), len(currentTx.Items))
		for itemIdx := 0; itemIdx < maxItems; itemIdx++ {
			switch {
			case itemIdx >= len(legacyTx.Items):
				diffs = append(diffs, fmt.Sprintf("transaksi %d [%s] / item %d hanya ada di current", i+1, memoHint, itemIdx+1))
				continue
			case itemIdx >= len(currentTx.Items):
				diffs = append(diffs, fmt.Sprintf("transaksi %d [%s] / item %d hanya ada di legacy", i+1, memoHint, itemIdx+1))
				continue
			}

			diffs = append(diffs, diffRows(fmt.Sprintf("transaksi %d [%s] / item %d", i+1, memoHint, itemIdx+1), legacyTx.Items[itemIdx], currentTx.Items[itemIdx])...)
		}
	}

	return diffs
}

func diffRows(prefix string, legacy, current cashflowRow) []string {
	type field struct {
		label string
		a     string
		b     string
	}
	fields := []field{
		{"Cheque Account", legacy.ChequeAccount, current.ChequeAccount},
		{"Cheque #", legacy.ChequeNumber, current.ChequeNumber},
		{"Date", legacy.Date, current.Date},
		{"Inclusive", legacy.Inclusive, current.Inclusive},
		{"Memo", legacy.Memo, current.Memo},
		{"Allocation Account #", legacy.AllocationAccount, current.AllocationAccount},
		{"Ex-Tax Amount", legacy.ExTaxAmount, current.ExTaxAmount},
		{"Inc-Tax Amount", legacy.IncTaxAmount, current.IncTaxAmount},
		{"Tax Code", legacy.TaxCode, current.TaxCode},
		{"Allocation Memo", legacy.AllocationMemo, current.AllocationMemo},
	}

	diffs := make([]string, 0)
	for _, f := range fields {
		if f.a == f.b {
			continue
		}
		diffs = append(diffs, fmt.Sprintf("%s / %s berbeda: legacy=%q current=%q", prefix, f.label, f.a, f.b))
	}
	return diffs
}

func valueAt(values []string, index int) string {
	if index < 0 || index >= len(values) {
		return ""
	}
	return strings.TrimSpace(values[index])
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func transactionMemoHint(legacy, current cashflowTransaction) string {
	memo := strings.TrimSpace(legacy.Header.Memo)
	if memo == "" {
		memo = strings.TrimSpace(current.Header.Memo)
	}
	if memo == "" {
		return "-"
	}
	if len(memo) > 72 {
		return memo[:69] + "..."
	}
	return memo
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
