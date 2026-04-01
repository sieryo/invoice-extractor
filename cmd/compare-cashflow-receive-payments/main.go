package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const maxDiffsToPrint = 200

type receivePaymentsFile struct {
	Header              []string
	RawLineCount        int
	NonEmptyRowCount    int
	BlankSeparatorCount int
	Entries             []receivePaymentsEntry
}

type receivePaymentsEntry struct {
	LastName       string
	FirstName      string
	DepositAccount string
	IDNumber       string
	ReceiptDate    string
	InvoiceNumber  string
	CustomerPO     string
	InvoiceDate    string
	AmountApplied  string
	Memo           string
	CurrencyCode   string
	ExchangeRate   string
	PaymentMethod  string
	PaymentNotes   string
	NameOnCard     string
	CardNumber     string
	ExpiryDate     string
	AuthNumber     string
	BSB            string
	AccountNumber  string
	DrawerName     string
	ChequeNumber   string
}

func main() {
	assetsDir, err := resolveAssetsDir()
	if err != nil {
		fail(err)
	}

	legacy, err := loadReceivePaymentsFile(filepath.Join(assetsDir, "legacy.txt"))
	if err != nil {
		fail(err)
	}
	current, err := loadReceivePaymentsFile(filepath.Join(assetsDir, "new.txt"))
	if err != nil {
		fail(err)
	}

	diffs := diffReceivePaymentsFiles(legacy, current)

	fmt.Println("Compare Cashflow Receive Payments")
	fmt.Println("=================================")
	fmt.Printf("Assets dir         : %s\n", assetsDir)
	fmt.Printf("Header columns     : %d\n", len(current.Header))
	fmt.Printf("Entries            : legacy=%d current=%d\n", len(legacy.Entries), len(current.Entries))
	fmt.Printf("Non-empty rows     : legacy=%d current=%d\n", legacy.NonEmptyRowCount, current.NonEmptyRowCount)
	fmt.Printf("Blank separators   : legacy=%d current=%d\n", legacy.BlankSeparatorCount, current.BlankSeparatorCount)

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
	assetsDir := filepath.Join(backendDir, "assets", "cashflow", "receive_payments")
	info, err := os.Stat(assetsDir)
	if err != nil {
		return "", fmt.Errorf("failed to access assets dir: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("assets path is not a directory: %s", assetsDir)
	}
	return assetsDir, nil
}

func loadReceivePaymentsFile(path string) (receivePaymentsFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return receivePaymentsFile{}, fmt.Errorf("failed to read %s: %w", path, err)
	}

	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return receivePaymentsFile{}, fmt.Errorf("empty file: %s", path)
	}

	result := receivePaymentsFile{
		Header:       splitTabLine(lines[0]),
		RawLineCount: len(lines),
		Entries:      make([]receivePaymentsEntry, 0),
	}

	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			result.BlankSeparatorCount++
			continue
		}
		result.NonEmptyRowCount++
		entry := parseReceivePaymentsEntry(splitTabLine(line))
		if isEmptyEntry(entry) {
			continue
		}
		result.Entries = append(result.Entries, entry)
	}

	return result, nil
}

func splitTabLine(line string) []string {
	return strings.Split(line, "\t")
}

func parseReceivePaymentsEntry(values []string) receivePaymentsEntry {
	return receivePaymentsEntry{
		LastName:       valueAt(values, 0),
		FirstName:      valueAt(values, 1),
		DepositAccount: valueAt(values, 2),
		IDNumber:       valueAt(values, 3),
		ReceiptDate:    valueAt(values, 4),
		InvoiceNumber:  valueAt(values, 5),
		CustomerPO:     valueAt(values, 6),
		InvoiceDate:    valueAt(values, 7),
		AmountApplied:  valueAt(values, 8),
		Memo:           valueAt(values, 9),
		CurrencyCode:   valueAt(values, 10),
		ExchangeRate:   valueAt(values, 11),
		PaymentMethod:  valueAt(values, 12),
		PaymentNotes:   valueAt(values, 13),
		NameOnCard:     valueAt(values, 14),
		CardNumber:     valueAt(values, 15),
		ExpiryDate:     valueAt(values, 16),
		AuthNumber:     valueAt(values, 17),
		BSB:            valueAt(values, 18),
		AccountNumber:  valueAt(values, 19),
		DrawerName:     valueAt(values, 20),
		ChequeNumber:   valueAt(values, 21),
	}
}

func diffReceivePaymentsFiles(legacy, current receivePaymentsFile) []string {
	diffs := make([]string, 0)

	if !equalSlices(legacy.Header, current.Header) {
		diffs = append(diffs, fmt.Sprintf("header berbeda: legacy=%v current=%v", legacy.Header, current.Header))
	}
	if len(legacy.Entries) != len(current.Entries) {
		diffs = append(diffs, fmt.Sprintf("jumlah entry berbeda: legacy=%d current=%d", len(legacy.Entries), len(current.Entries)))
	}
	if legacy.NonEmptyRowCount != current.NonEmptyRowCount {
		diffs = append(diffs, fmt.Sprintf("jumlah non-empty row berbeda: legacy=%d current=%d", legacy.NonEmptyRowCount, current.NonEmptyRowCount))
	}
	if legacy.BlankSeparatorCount != current.BlankSeparatorCount {
		diffs = append(diffs, fmt.Sprintf("jumlah blank separator berbeda: legacy=%d current=%d", legacy.BlankSeparatorCount, current.BlankSeparatorCount))
	}

	legacyCount := toEntryCountMap(legacy.Entries)
	currentCount := toEntryCountMap(current.Entries)
	keys := unionKeys(legacyCount, currentCount)

	for _, key := range keys {
		a := legacyCount[key]
		b := currentCount[key]
		if a == b {
			continue
		}
		diffs = append(diffs, fmt.Sprintf("entry count berbeda: legacy=%d current=%d | %s", a, b, key))
	}

	return diffs
}

func toEntryCountMap(entries []receivePaymentsEntry) map[string]int {
	result := make(map[string]int, len(entries))
	for _, entry := range entries {
		result[canonicalEntryKey(entry)]++
	}
	return result
}

func canonicalEntryKey(entry receivePaymentsEntry) string {
	fields := []string{
		entry.LastName,
		entry.FirstName,
		entry.DepositAccount,
		entry.IDNumber,
		entry.ReceiptDate,
		entry.InvoiceNumber,
		entry.CustomerPO,
		entry.InvoiceDate,
		entry.AmountApplied,
		entry.Memo,
		entry.CurrencyCode,
		entry.ExchangeRate,
		entry.PaymentMethod,
		entry.PaymentNotes,
		entry.NameOnCard,
		entry.CardNumber,
		entry.ExpiryDate,
		entry.AuthNumber,
		entry.BSB,
		entry.AccountNumber,
		entry.DrawerName,
		entry.ChequeNumber,
	}
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return strings.Join(fields, " | ")
}

func unionKeys(a, b map[string]int) []string {
	out := make([]string, 0, len(a)+len(b))
	seen := make(map[string]struct{}, len(a)+len(b))
	for key := range a {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	for key := range b {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func isEmptyEntry(entry receivePaymentsEntry) bool {
	return strings.TrimSpace(entry.LastName) == "" &&
		strings.TrimSpace(entry.FirstName) == "" &&
		strings.TrimSpace(entry.DepositAccount) == "" &&
		strings.TrimSpace(entry.IDNumber) == "" &&
		strings.TrimSpace(entry.ReceiptDate) == "" &&
		strings.TrimSpace(entry.InvoiceNumber) == "" &&
		strings.TrimSpace(entry.CustomerPO) == "" &&
		strings.TrimSpace(entry.InvoiceDate) == "" &&
		strings.TrimSpace(entry.AmountApplied) == "" &&
		strings.TrimSpace(entry.Memo) == "" &&
		strings.TrimSpace(entry.CurrencyCode) == "" &&
		strings.TrimSpace(entry.ExchangeRate) == "" &&
		strings.TrimSpace(entry.PaymentMethod) == "" &&
		strings.TrimSpace(entry.PaymentNotes) == "" &&
		strings.TrimSpace(entry.NameOnCard) == "" &&
		strings.TrimSpace(entry.CardNumber) == "" &&
		strings.TrimSpace(entry.ExpiryDate) == "" &&
		strings.TrimSpace(entry.AuthNumber) == "" &&
		strings.TrimSpace(entry.BSB) == "" &&
		strings.TrimSpace(entry.AccountNumber) == "" &&
		strings.TrimSpace(entry.DrawerName) == "" &&
		strings.TrimSpace(entry.ChequeNumber) == ""
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

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
