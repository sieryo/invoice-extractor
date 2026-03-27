package document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
)

const spreadsheetNormalizedReadFallback = "invalid normalized spreadsheet payload"

type SpreadsheetWorkbook struct {
	SourceFile    string             `json:"sourceFile"`
	PrimarySheet  string             `json:"primarySheet,omitempty"`
	SheetCount    int                `json:"sheetCount"`
	TotalRowCount int                `json:"totalRowCount"`
	Sheets        []SpreadsheetSheet `json:"sheets"`
	ExtractedAt   time.Time          `json:"extractedAt"`
}

type SpreadsheetSheet struct {
	Name           string     `json:"name"`
	HeaderRowIndex int        `json:"headerRowIndex"`
	Headers        []string   `json:"headers,omitempty"`
	RawRowNumbers  []int      `json:"rawRowNumbers,omitempty"`
	RawRows        [][]string `json:"rawRows,omitempty"`
	RowNumbers     []int      `json:"rowNumbers,omitempty"`
	Rows           [][]string `json:"rows,omitempty"`
	RowCount       int        `json:"rowCount"`
}

func ExtractSpreadsheetWorkbook(path string, sourceName string) (SpreadsheetWorkbook, []string, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return SpreadsheetWorkbook{}, nil, err
	}
	defer file.Close()

	sheetNames := file.GetSheetList()
	if len(sheetNames) == 0 {
		return SpreadsheetWorkbook{}, nil, errors.New("workbook has no sheets")
	}

	workbook := SpreadsheetWorkbook{
		SourceFile:  sourceName,
		SheetCount:  len(sheetNames),
		Sheets:      make([]SpreadsheetSheet, 0, len(sheetNames)),
		ExtractedAt: time.Now().UTC(),
	}

	warnings := make([]string, 0, 2)
	for _, sheetName := range sheetNames {
		rows, err := file.GetRows(sheetName)
		if err != nil {
			return SpreadsheetWorkbook{}, nil, fmt.Errorf("failed to read sheet %q: %w", sheetName, err)
		}

		sheet, sheetWarnings := buildSpreadsheetSheet(sheetName, rows)
		if workbook.PrimarySheet == "" && len(sheet.RawRows) > 0 {
			workbook.PrimarySheet = sheet.Name
		}
		workbook.TotalRowCount += sheet.RowCount
		workbook.Sheets = append(workbook.Sheets, sheet)
		warnings = append(warnings, sheetWarnings...)
	}

	if workbook.PrimarySheet == "" {
		return SpreadsheetWorkbook{}, nil, errors.New("workbook has no readable rows")
	}

	return workbook, uniqueStrings(warnings), nil
}

func buildSpreadsheetSheet(sheetName string, rows [][]string) (SpreadsheetSheet, []string) {
	sheet := SpreadsheetSheet{Name: sheetName}
	warnings := make([]string, 0, 2)

	headerFound := false
	for rowIndex, row := range rows {
		trimmedRow := trimTrailingEmptyCells(row)
		if len(trimmedRow) == 0 {
			continue
		}

		clonedRow := append([]string(nil), trimmedRow...)
		sheet.RawRowNumbers = append(sheet.RawRowNumbers, rowIndex+1)
		sheet.RawRows = append(sheet.RawRows, clonedRow)

		if !headerFound {
			headerFound = true
			sheet.HeaderRowIndex = rowIndex + 1
			sheet.Headers = append([]string(nil), clonedRow...)
			continue
		}

		sheet.RowNumbers = append(sheet.RowNumbers, rowIndex+1)
		sheet.Rows = append(sheet.Rows, clonedRow)
	}

	sheet.RowCount = len(sheet.Rows)
	if len(sheet.RawRows) == 0 {
		warnings = append(warnings, fmt.Sprintf("sheet %q is empty", sheetName))
		return sheet, warnings
	}
	if sheet.HeaderRowIndex == 0 || len(sheet.Headers) == 0 {
		warnings = append(warnings, fmt.Sprintf("sheet %q has no header row", sheetName))
	}
	if sheet.RowCount == 0 {
		warnings = append(warnings, fmt.Sprintf("sheet %q has no data rows", sheetName))
	}

	return sheet, warnings
}

func trimTrailingEmptyCells(row []string) []string {
	lastNonEmpty := -1
	for idx, cell := range row {
		if strings.TrimSpace(cell) != "" {
			lastNonEmpty = idx
		}
	}
	if lastNonEmpty < 0 {
		return nil
	}

	trimmed := make([]string, 0, lastNonEmpty+1)
	for _, cell := range row[:lastNonEmpty+1] {
		trimmed = append(trimmed, strings.TrimSpace(cell))
	}
	return trimmed
}

func LoadSpreadsheetWorkbook(
	ctx context.Context,
	fileStore dfile.FileStore,
	collectionID string,
	ref string,
) (*SpreadsheetWorkbook, error) {
	if strings.TrimSpace(ref) == "" {
		return nil, errors.New(spreadsheetNormalizedReadFallback)
	}

	b, err := fileStore.Read(ctx, collectionID, ref)
	if err != nil {
		return nil, err
	}

	var wrapped struct {
		Workbook SpreadsheetWorkbook `json:"workbook"`
	}
	if err := json.Unmarshal(b, &wrapped); err == nil && len(wrapped.Workbook.Sheets) > 0 {
		return &wrapped.Workbook, nil
	}

	var workbook SpreadsheetWorkbook
	if err := json.Unmarshal(b, &workbook); err == nil && len(workbook.Sheets) > 0 {
		return &workbook, nil
	}

	return nil, errors.New(spreadsheetNormalizedReadFallback)
}

func FindSpreadsheetSheet(workbook SpreadsheetWorkbook, sheetName string) (SpreadsheetSheet, error) {
	target := strings.TrimSpace(sheetName)
	if target == "" {
		target = strings.TrimSpace(workbook.PrimarySheet)
	}
	for _, sheet := range workbook.Sheets {
		if strings.EqualFold(strings.TrimSpace(sheet.Name), target) {
			return sheet, nil
		}
	}
	return SpreadsheetSheet{}, fmt.Errorf("sheet tidak ditemukan: %s", target)
}

func SpreadsheetRowNumberAt(sheet SpreadsheetSheet, idx int) int {
	if idx >= 0 && idx < len(sheet.RawRowNumbers) && sheet.RawRowNumbers[idx] > 0 {
		return sheet.RawRowNumbers[idx]
	}
	if idx < len(sheet.RawRows) {
		return sheet.HeaderRowIndex + idx
	}
	return idx + 1
}
