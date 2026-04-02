package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const maxDiffsToPrint = 200

type fpFile struct {
	Header              []string
	RawLineCount        int
	NonEmptyRowCount    int
	BlankSeparatorCount int
	Rows                [][]string
}

func main() {
	assetsDir, err := resolveAssetsDir()
	if err != nil {
		fail(err)
	}

	legacy, err := loadFPFile(filepath.Join(assetsDir, "legacy.txt"))
	if err != nil {
		fail(err)
	}
	current, err := loadFPFile(filepath.Join(assetsDir, "new.txt"))
	if err != nil {
		fail(err)
	}

	diffs := diffFPFiles(legacy, current)

	fmt.Println("Compare FP Keluaran")
	fmt.Println("===================")
	fmt.Printf("Assets dir         : %s\n", assetsDir)
	fmt.Printf("Header columns     : %d\n", len(current.Header))
	fmt.Printf("Rows               : legacy=%d current=%d\n", len(legacy.Rows), len(current.Rows))
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
	assetsDir := filepath.Join(backendDir, "assets", "fp", "keluaran")
	info, err := os.Stat(assetsDir)
	if err != nil {
		return "", fmt.Errorf("failed to access assets dir: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("assets path is not a directory: %s", assetsDir)
	}
	return assetsDir, nil
}

func loadFPFile(path string) (fpFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return fpFile{}, fmt.Errorf("failed to read %s: %w", path, err)
	}

	lines := strings.Split(strings.ReplaceAll(string(content), "\r\n", "\n"), "\n")
	if len(lines) == 0 {
		return fpFile{}, fmt.Errorf("empty file: %s", path)
	}

	result := fpFile{
		Header:       splitTabLine(lines[0]),
		RawLineCount: len(lines),
		Rows:         make([][]string, 0),
	}

	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			result.BlankSeparatorCount++
			continue
		}

		row := splitTabLine(line)
		if isEmptyRow(row) {
			continue
		}

		result.NonEmptyRowCount++
		result.Rows = append(result.Rows, row)
	}

	return result, nil
}

func diffFPFiles(legacy, current fpFile) []string {
	diffs := make([]string, 0)

	if !equalSlices(legacy.Header, current.Header) {
		diffs = append(diffs, fmt.Sprintf("header berbeda: legacy=%v current=%v", legacy.Header, current.Header))
	}
	if len(legacy.Rows) != len(current.Rows) {
		diffs = append(diffs, fmt.Sprintf("jumlah row berbeda: legacy=%d current=%d", len(legacy.Rows), len(current.Rows)))
	}
	if legacy.NonEmptyRowCount != current.NonEmptyRowCount {
		diffs = append(diffs, fmt.Sprintf("jumlah non-empty row berbeda: legacy=%d current=%d", legacy.NonEmptyRowCount, current.NonEmptyRowCount))
	}
	if legacy.BlankSeparatorCount != current.BlankSeparatorCount {
		diffs = append(diffs, fmt.Sprintf("jumlah blank separator berbeda: legacy=%d current=%d", legacy.BlankSeparatorCount, current.BlankSeparatorCount))
	}

	maxRows := max(len(legacy.Rows), len(current.Rows))
	for i := 0; i < maxRows; i++ {
		switch {
		case i >= len(legacy.Rows):
			diffs = append(diffs, fmt.Sprintf("row %d hanya ada di current", i+1))
			continue
		case i >= len(current.Rows):
			diffs = append(diffs, fmt.Sprintf("row %d hanya ada di legacy", i+1))
			continue
		}

		diffs = append(diffs, diffRow(i+1, legacy.Header, legacy.Rows[i], current.Rows[i])...)
	}

	return diffs
}

func diffRow(rowNumber int, header []string, legacyRow []string, currentRow []string) []string {
	diffs := make([]string, 0)
	maxCols := max(len(legacyRow), len(currentRow))
	rowHint := buildRowHint(legacyRow, currentRow)

	for colIdx := 0; colIdx < maxCols; colIdx++ {
		legacyValue := valueAt(legacyRow, colIdx)
		currentValue := valueAt(currentRow, colIdx)
		if legacyValue == currentValue {
			continue
		}

		columnLabel := fmt.Sprintf("Column %d", colIdx+1)
		if colIdx < len(header) && strings.TrimSpace(header[colIdx]) != "" {
			columnLabel = strings.TrimSpace(header[colIdx])
		}

		diffs = append(diffs, fmt.Sprintf("row %d [%s] / %s berbeda: legacy=%q current=%q", rowNumber, rowHint, columnLabel, legacyValue, currentValue))
	}

	return diffs
}

func buildRowHint(legacyRow []string, currentRow []string) string {
	candidates := []string{
		firstNonEmpty(valueAt(legacyRow, 0), valueAt(currentRow, 0)),
		firstNonEmpty(valueAt(legacyRow, 5), valueAt(currentRow, 5)),
		firstNonEmpty(valueAt(legacyRow, 3), valueAt(currentRow, 3)),
	}

	parts := make([]string, 0, len(candidates))
	for _, item := range candidates {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		parts = append(parts, item)
	}

	if len(parts) == 0 {
		return "-"
	}

	hint := strings.Join(parts, " | ")
	if len(hint) > 96 {
		return hint[:93] + "..."
	}
	return hint
}

func splitTabLine(line string) []string {
	return strings.Split(line, "\t")
}

func isEmptyRow(values []string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
