package document

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
)

const spreadsheetNormalizedReadFallback = "invalid normalized spreadsheet payload"

type SpreadsheetCellValueType string

const (
	SpreadsheetCellValueTypeEmpty  SpreadsheetCellValueType = "empty"
	SpreadsheetCellValueTypeString SpreadsheetCellValueType = "string"
	SpreadsheetCellValueTypeDate   SpreadsheetCellValueType = "date"
	SpreadsheetCellValueTypeFloat  SpreadsheetCellValueType = "float"
	SpreadsheetCellValueTypeBool   SpreadsheetCellValueType = "bool"
)

type SpreadsheetWorkbook struct {
	SourceFile    string             `json:"sourceFile"`
	PrimarySheet  string             `json:"primarySheet,omitempty"`
	SheetCount    int                `json:"sheetCount"`
	TotalRowCount int                `json:"totalRowCount"`
	Sheets        []SpreadsheetSheet `json:"sheets"`
	ExtractedAt   time.Time          `json:"extractedAt"`
}

type SpreadsheetSheet struct {
	Name           string              `json:"name"`
	HeaderRowIndex int                 `json:"headerRowIndex"`
	Headers        []string            `json:"headers,omitempty"`
	HeaderCells    []SpreadsheetCell   `json:"headerCells,omitempty"`
	RawRowNumbers  []int               `json:"rawRowNumbers,omitempty"`
	RawRows        [][]string          `json:"rawRows,omitempty"`
	RawCellRows    [][]SpreadsheetCell `json:"rawCellRows,omitempty"`
	RowNumbers     []int               `json:"rowNumbers,omitempty"`
	Rows           [][]string          `json:"rows,omitempty"`
	CellRows       [][]SpreadsheetCell `json:"cellRows,omitempty"`
	RowCount       int                 `json:"rowCount"`
}

type SpreadsheetCell struct {
	Display     string                   `json:"display,omitempty"`
	Raw         string                   `json:"raw,omitempty"`
	ValueType   SpreadsheetCellValueType `json:"valueType,omitempty"`
	StringValue string                   `json:"stringValue,omitempty"`
	FloatValue  *float64                 `json:"floatValue,omitempty"`
	DateValue   string                   `json:"dateValue,omitempty"`
	BoolValue   *bool                    `json:"boolValue,omitempty"`
}

func ExtractSpreadsheetWorkbook(path string, sourceName string) (SpreadsheetWorkbook, []string, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return SpreadsheetWorkbook{}, nil, err
	}
	defer file.Close()

	return extractSpreadsheetWorkbook(file, sourceName)
}

func ExtractSpreadsheetWorkbookBytes(data []byte, sourceName string) (SpreadsheetWorkbook, []string, error) {
	file, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return SpreadsheetWorkbook{}, nil, err
	}
	defer file.Close()

	return extractSpreadsheetWorkbook(file, sourceName)
}

func extractSpreadsheetWorkbook(file *excelize.File, sourceName string) (SpreadsheetWorkbook, []string, error) {

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
		displayRows, err := file.GetRows(sheetName)
		if err != nil {
			return SpreadsheetWorkbook{}, nil, fmt.Errorf("failed to read sheet %q: %w", sheetName, err)
		}
		rawRows, err := file.GetRows(sheetName, excelize.Options{RawCellValue: true})
		if err != nil {
			return SpreadsheetWorkbook{}, nil, fmt.Errorf("failed to read raw sheet %q: %w", sheetName, err)
		}

		sheet, sheetWarnings := buildSpreadsheetSheet(file, sheetName, displayRows, rawRows)
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

func CompactSpreadsheetWorkbook(workbook SpreadsheetWorkbook) SpreadsheetWorkbook {
	compact := SpreadsheetWorkbook{
		SourceFile:    workbook.SourceFile,
		PrimarySheet:  workbook.PrimarySheet,
		SheetCount:    workbook.SheetCount,
		TotalRowCount: workbook.TotalRowCount,
		Sheets:        make([]SpreadsheetSheet, 0, len(workbook.Sheets)),
		ExtractedAt:   workbook.ExtractedAt,
	}

	for _, sheet := range workbook.Sheets {
		compact.Sheets = append(compact.Sheets, SpreadsheetSheet{
			Name:           sheet.Name,
			HeaderRowIndex: sheet.HeaderRowIndex,
			Headers:        append([]string(nil), sheet.Headers...),
			RowCount:       sheet.RowCount,
		})
	}

	return compact
}

func buildSpreadsheetSheet(file *excelize.File, sheetName string, displayRows [][]string, rawRows [][]string) (SpreadsheetSheet, []string) {
	sheet := SpreadsheetSheet{Name: sheetName}
	warnings := make([]string, 0, 2)

	headerFound := false
	rowCount := len(displayRows)
	if len(rawRows) > rowCount {
		rowCount = len(rawRows)
	}

	for rowIndex := 0; rowIndex < rowCount; rowIndex++ {
		displayRow := rowAt(displayRows, rowIndex)
		rawRow := rowAt(rawRows, rowIndex)
		cells := buildSpreadsheetCells(file, sheetName, rowIndex+1, displayRow, rawRow)
		trimmedDisplay, trimmedCells := trimTrailingSpreadsheetCells(cells)
		if len(trimmedDisplay) == 0 {
			continue
		}

		clonedRow := append([]string(nil), trimmedDisplay...)
		clonedCells := append([]SpreadsheetCell(nil), trimmedCells...)
		sheet.RawRowNumbers = append(sheet.RawRowNumbers, rowIndex+1)
		sheet.RawRows = append(sheet.RawRows, clonedRow)
		sheet.RawCellRows = append(sheet.RawCellRows, clonedCells)

		if !headerFound {
			headerFound = true
			sheet.HeaderRowIndex = rowIndex + 1
			sheet.Headers = append([]string(nil), clonedRow...)
			sheet.HeaderCells = append([]SpreadsheetCell(nil), clonedCells...)
			continue
		}

		sheet.RowNumbers = append(sheet.RowNumbers, rowIndex+1)
		sheet.Rows = append(sheet.Rows, clonedRow)
		sheet.CellRows = append(sheet.CellRows, clonedCells)
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

func rowAt(rows [][]string, idx int) []string {
	if idx < 0 || idx >= len(rows) {
		return nil
	}
	return rows[idx]
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

func buildSpreadsheetCells(
	file *excelize.File,
	sheetName string,
	rowNumber int,
	displayRow []string,
	rawRow []string,
) []SpreadsheetCell {
	cellCount := len(displayRow)
	if len(rawRow) > cellCount {
		cellCount = len(rawRow)
	}
	if cellCount == 0 {
		return nil
	}

	cells := make([]SpreadsheetCell, 0, cellCount)
	for colIndex := 0; colIndex < cellCount; colIndex++ {
		displayValue := strings.TrimSpace(valueAt(displayRow, colIndex))
		rawValue := strings.TrimSpace(valueAt(rawRow, colIndex))
		cellName, _ := excelize.CoordinatesToCellName(colIndex+1, rowNumber)
		cellType, _ := file.GetCellType(sheetName, cellName)
		cells = append(cells, buildSpreadsheetCell(displayValue, rawValue, cellType))
	}
	return cells
}

func valueAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func buildSpreadsheetCell(displayValue string, rawValue string, cellType excelize.CellType) SpreadsheetCell {
	cell := SpreadsheetCell{
		Display:   displayValue,
		Raw:       rawValue,
		ValueType: SpreadsheetCellValueTypeString,
	}

	switch cellType {
	case excelize.CellTypeBool:
		if b, ok := parseSpreadsheetBool(rawValue, displayValue); ok {
			cell.ValueType = SpreadsheetCellValueTypeBool
			cell.BoolValue = &b
			cell.StringValue = boolDisplayValue(b, displayValue)
			return cell
		}
	case excelize.CellTypeDate:
		if t, ok := parseSpreadsheetDateValue(rawValue, displayValue); ok {
			cell.ValueType = SpreadsheetCellValueTypeDate
			cell.DateValue = t.Format(time.RFC3339)
			cell.StringValue = displayValue
			return cell
		}
	case excelize.CellTypeNumber:
		if f, ok := parseSpreadsheetFloatValue(rawValue, displayValue); ok {
			cell.ValueType = SpreadsheetCellValueTypeFloat
			cell.FloatValue = &f
			cell.StringValue = displayValue
			return cell
		}
	}

	if displayValue == "" && rawValue == "" {
		cell.ValueType = SpreadsheetCellValueTypeEmpty
	}
	cell.StringValue = firstNonEmpty(displayValue, rawValue)
	return cell
}

func trimTrailingSpreadsheetCells(cells []SpreadsheetCell) ([]string, []SpreadsheetCell) {
	lastNonEmpty := -1
	for idx, cell := range cells {
		if strings.TrimSpace(firstNonEmpty(cell.Display, cell.Raw, cell.StringValue)) != "" || cell.FloatValue != nil || cell.DateValue != "" || cell.BoolValue != nil {
			lastNonEmpty = idx
		}
	}
	if lastNonEmpty < 0 {
		return nil, nil
	}

	trimmedCells := append([]SpreadsheetCell(nil), cells[:lastNonEmpty+1]...)
	trimmedDisplay := make([]string, 0, len(trimmedCells))
	for _, cell := range trimmedCells {
		trimmedDisplay = append(trimmedDisplay, strings.TrimSpace(firstNonEmpty(cell.Display, cell.StringValue, cell.Raw)))
	}
	return trimmedDisplay, trimmedCells
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func parseSpreadsheetFloatValue(rawValue string, displayValue string) (float64, bool) {
	for _, candidate := range []string{rawValue, displayValue} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if normalized, ok := normalizeSpreadsheetNumericString(candidate); ok {
			n, err := strconv.ParseFloat(normalized, 64)
			if err == nil {
				return n, true
			}
		}
		if n, err := strconv.ParseFloat(candidate, 64); err == nil {
			return n, true
		}
		if n, err := strconv.ParseFloat(strings.ReplaceAll(candidate, ",", "."), 64); err == nil {
			return n, true
		}
	}
	return 0, false
}

func normalizeSpreadsheetNumericString(raw string) (string, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false
	}

	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "\u00a0", "")

	sign := ""
	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		sign = value[:1]
		value = value[1:]
	}
	if value == "" {
		return "", false
	}

	for _, r := range value {
		if (r < '0' || r > '9') && r != '.' && r != ',' {
			return "", false
		}
	}

	dotCount := strings.Count(value, ".")
	commaCount := strings.Count(value, ",")

	switch {
	case dotCount > 0 && commaCount > 0:
		lastDot := strings.LastIndex(value, ".")
		lastComma := strings.LastIndex(value, ",")
		if lastComma > lastDot {
			value = strings.ReplaceAll(value, ".", "")
			value = strings.ReplaceAll(value, ",", ".")
		} else {
			value = strings.ReplaceAll(value, ",", "")
		}
	case dotCount > 1 && commaCount == 0:
		value = strings.ReplaceAll(value, ".", "")
	case commaCount > 1 && dotCount == 0:
		value = strings.ReplaceAll(value, ",", "")
	case commaCount == 1 && dotCount == 0:
		commaIdx := strings.Index(value, ",")
		decimals := len(value) - commaIdx - 1
		if decimals == 3 && commaIdx > 0 {
			value = strings.ReplaceAll(value, ",", "")
		} else {
			value = strings.ReplaceAll(value, ",", ".")
		}
	case dotCount == 1 && commaCount == 0:
		dotIdx := strings.Index(value, ".")
		decimals := len(value) - dotIdx - 1
		if decimals == 3 && dotIdx > 0 {
			value = strings.ReplaceAll(value, ".", "")
		}
	}

	if value == "" || value == "." {
		return "", false
	}
	return sign + value, true
}

func parseSpreadsheetDateValue(rawValue string, displayValue string) (time.Time, bool) {
	for _, candidate := range []string{rawValue, displayValue} {
		candidate = normalizeSpreadsheetDateCandidate(candidate)
		if candidate == "" {
			continue
		}
		if n, err := strconv.ParseFloat(strings.ReplaceAll(candidate, ",", "."), 64); err == nil && n > 0 {
			if t, err := spreadsheetExcelSerialToTime(n); err == nil {
				return spreadsheetDateOnly(t), true
			}
		}
		for _, layout := range []string{
			"2006-01-02",
			"2006-01-02 15:04",
			"2006-01-02 15:04:05",
			"02/01/2006",
			"2/1/2006",
			"02/01/2006 15:04",
			"2/1/2006 15:04",
			"02/01/2006 15:04:05",
			"2/1/2006 15:04:05",
			"02-01-2006",
			"2-1-2006",
			"02-01-2006 15:04",
			"2-1-2006 15:04",
			"02-01-2006 15:04:05",
			"2-1-2006 15:04:05",
			"02-Jan-2006",
			"2-Jan-2006",
			"02-Jan-06",
			"2-Jan-06",
			"02 Jan 2006",
			"2 Jan 2006",
			"02 January 2006",
			"2 January 2006",
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02T15:04",
		} {
			if t, err := time.ParseInLocation(layout, candidate, time.Local); err == nil {
				return spreadsheetDateOnly(t), true
			}
		}
	}
	return time.Time{}, false
}

func normalizeSpreadsheetDateCandidate(raw string) string {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return ""
	}

	// Excel exports may serialize date-only values as full timestamps.
	if idx := strings.Index(candidate, "T"); idx > 0 {
		datePart := strings.TrimSpace(candidate[:idx])
		timePart := strings.TrimSpace(candidate[idx+1:])
		if strings.Count(datePart, "-") == 2 {
			if timePart == "" {
				return datePart
			}
			if zoneIdx := strings.IndexAny(timePart, "Zz+-"); zoneIdx >= 0 {
				return strings.TrimSpace(candidate)
			}
			return datePart
		}
	}

	return candidate
}

func spreadsheetDateOnly(t time.Time) time.Time {
	location := t.Location()
	if location == nil {
		location = time.Local
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, location)
}

func parseSpreadsheetBool(rawValue string, displayValue string) (bool, bool) {
	for _, candidate := range []string{rawValue, displayValue} {
		switch strings.ToLower(strings.TrimSpace(candidate)) {
		case "true", "1":
			return true, true
		case "false", "0":
			return false, true
		}
	}
	return false, false
}

func boolDisplayValue(value bool, displayValue string) string {
	if strings.TrimSpace(displayValue) != "" {
		return strings.TrimSpace(displayValue)
	}
	if value {
		return "TRUE"
	}
	return "FALSE"
}

func spreadsheetExcelSerialToTime(serial float64) (time.Time, error) {
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	if serial < 1 {
		return time.Time{}, errors.New("invalid excel serial")
	}
	days := int(serial)
	fraction := serial - float64(days)
	seconds := int(math.Round(fraction * 24 * 60 * 60))
	return base.AddDate(0, 0, days).Add(time.Duration(seconds) * time.Second), nil
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

func LoadSpreadsheetWorkbookForExecution(
	ctx context.Context,
	fileStore dfile.FileStore,
	collectionID string,
	normalizedRef string,
	rawRef string,
) (*SpreadsheetWorkbook, error) {
	if strings.TrimSpace(rawRef) != "" {
		rawBytes, err := fileStore.Read(ctx, collectionID, rawRef)
		if err == nil {
			workbook, _, extractErr := ExtractSpreadsheetWorkbookBytes(rawBytes, rawRef)
			if extractErr == nil {
				return &workbook, nil
			}
		}
	}

	return LoadSpreadsheetWorkbook(ctx, fileStore, collectionID, normalizedRef)
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

func SpreadsheetCellAt(cells []SpreadsheetCell, indexes map[string]int, key string) SpreadsheetCell {
	idx, ok := indexes[key]
	if !ok || idx < 0 || idx >= len(cells) {
		return SpreadsheetCell{}
	}
	return cells[idx]
}

func SpreadsheetCellText(cell SpreadsheetCell) string {
	return strings.TrimSpace(firstNonEmpty(cell.Display, cell.StringValue, cell.Raw))
}

func SpreadsheetCellFloat(cell SpreadsheetCell) (float64, bool) {
	if cell.FloatValue != nil {
		return *cell.FloatValue, true
	}
	return parseSpreadsheetFloatValue(cell.Raw, cell.Display)
}

func SpreadsheetCellMoney(cell SpreadsheetCell) (float64, bool) {
	candidates := []string{}
	for _, candidate := range []string{cell.Raw, cell.StringValue, cell.Display} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if !containsString(candidates, candidate) {
			candidates = append(candidates, candidate)
		}
	}

	for _, candidate := range candidates {
		if value, ok := parseSpreadsheetMoneyValue(candidate); ok {
			return value, true
		}
	}

	if cell.FloatValue != nil {
		return *cell.FloatValue, true
	}
	return 0, false
}

func SpreadsheetCellPercent(cell SpreadsheetCell) (float64, bool) {
	containsPercent := strings.Contains(cell.Display, "%") || strings.Contains(cell.Raw, "%") || strings.Contains(cell.StringValue, "%")
	if cell.FloatValue != nil {
		value := *cell.FloatValue
		if containsPercent {
			if math.Abs(value) <= 1 {
				return value, true
			}
			return value / 100, true
		}
		return value, true
	}

	for _, candidate := range []string{cell.Raw, cell.Display, cell.StringValue} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if strings.Contains(candidate, "%") {
			candidate = strings.ReplaceAll(candidate, "%", "")
			value, ok := parseSpreadsheetFloatValue(candidate, "")
			if !ok {
				continue
			}
			if math.Abs(value) <= 1 {
				return value, true
			}
			return value / 100, true
		}
	}

	return SpreadsheetCellFloat(cell)
}

func parseSpreadsheetMoneyValue(raw string) (float64, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false
	}
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "\u00a0", "")

	sign := ""
	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		sign = value[:1]
		value = value[1:]
	}
	if value == "" {
		return 0, false
	}

	if strings.Contains(value, ",") {
		value = strings.ReplaceAll(value, ".", "")
		value = strings.ReplaceAll(value, ",", ".")
	} else if strings.Count(value, ".") > 1 {
		value = strings.ReplaceAll(value, ".", "")
	}

	n, err := strconv.ParseFloat(sign+value, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func SpreadsheetCellDate(cell SpreadsheetCell) (time.Time, bool) {
	if strings.TrimSpace(cell.DateValue) != "" {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(cell.DateValue)); err == nil {
			return spreadsheetDateOnly(t), true
		}
	}
	return parseSpreadsheetDateValue(cell.Raw, cell.Display)
}
