package main

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	legacyXLSX  = "BPPU - GST Deduction MT LEGACY.xlsx"
	currentXLSX = "BPPU - GST Deduction MT.xlsx"
	legacyXML   = "BPPU - GST Deduction MT LEGACY.xml"
	currentXML  = "BPPU - GST Deduction MT.xml"
)

var ignoredWorkbookHeaders = []string{
	"ID TKU Pemotong",
}

var comparedXMLFields = []string{
	"TaxPeriodMonth",
	"TaxPeriodYear",
	"CounterpartTIN",
	"IDPlaceOfBusinessActivityOfIncomeRecipient",
	"TaxCertificate",
	"TaxObjectCode",
	"TaxBase",
	"Rate",
	"Document",
	"DocumentNumber",
	"DocumentDate",
	"GovTreasurerOpt",
	"SP2DNumber",
	"WithholdingDate",
}

type workbookData struct {
	SheetName       string
	HeaderRowNumber int
	Headers         []string
	ComparedHeaders []string
	IgnoredHeaders  []string
	ProfileNPWP     string
	Rows            []map[string]string
}

type xmlData struct {
	ComparedFields []string
	Rows           []map[string]string
}

type bpuBulk struct {
	TIN       string `xml:"TIN"`
	ListOfBpu []bpu  `xml:"ListOfBpu>Bpu"`
}

type bpu struct {
	TaxPeriodMonth                       string `xml:"TaxPeriodMonth"`
	TaxPeriodYear                        string `xml:"TaxPeriodYear"`
	CounterpartTIN                       string `xml:"CounterpartTin"`
	IDPlaceOfBusinessActivityOfRecipient string `xml:"IDPlaceOfBusinessActivityOfIncomeRecipient"`
	TaxCertificate                       string `xml:"TaxCertificate"`
	TaxObjectCode                        string `xml:"TaxObjectCode"`
	TaxBase                              string `xml:"TaxBase"`
	Rate                                 string `xml:"Rate"`
	Document                             string `xml:"Document"`
	DocumentNumber                       string `xml:"DocumentNumber"`
	DocumentDate                         string `xml:"DocumentDate"`
	IDPlaceOfBusinessActivity            string `xml:"IDPlaceOfBusinessActivity"`
	GovTreasurerOpt                      string `xml:"GovTreasurerOpt"`
	SP2DNumber                           string `xml:"SP2DNumber"`
	WithholdingDate                      string `xml:"WithholdingDate"`
}

func main() {
	assetsDir, err := resolveAssetsDir()
	if err != nil {
		fail(err)
	}

	legacyWorkbook, err := loadWorkbookData(filepath.Join(assetsDir, legacyXLSX))
	if err != nil {
		fail(err)
	}
	currentWorkbook, err := loadWorkbookData(filepath.Join(assetsDir, currentXLSX))
	if err != nil {
		fail(err)
	}
	legacyXMLData, err := loadXMLData(filepath.Join(assetsDir, legacyXML))
	if err != nil {
		fail(err)
	}
	currentXMLData, err := loadXMLData(filepath.Join(assetsDir, currentXML))
	if err != nil {
		fail(err)
	}

	workbookDiffs := diffWorkbookData(legacyWorkbook, currentWorkbook)
	xmlDiffs := diffXMLData(legacyXMLData, currentXMLData)

	fmt.Println("Compare Bukpot Deduction Output")
	fmt.Println("===============================")
	fmt.Printf("Assets dir             : %s\n", assetsDir)
	fmt.Printf("Sheet dibandingkan     : %s\n", currentWorkbook.SheetName)
	fmt.Printf("Header row             : %d\n", currentWorkbook.HeaderRowNumber)
	fmt.Printf("Header terdeteksi      : %s\n", strings.Join(filterNonEmptyStrings(currentWorkbook.Headers), " | "))
	fmt.Printf("Header dibandingkan    : %s\n", strings.Join(currentWorkbook.ComparedHeaders, " | "))
	fmt.Printf("Header diabaikan       : NPWP Pemotong (cell C1), %s\n", strings.Join(currentWorkbook.IgnoredHeaders, ", "))
	fmt.Printf("XML field dibandingkan : %s\n", strings.Join(currentXMLData.ComparedFields, " | "))
	fmt.Printf("XML field diabaikan    : TIN, IDPlaceOfBusinessActivity\n")
	fmt.Println()

	printDiffSection("Workbook", workbookDiffs)
	fmt.Println()
	printDiffSection("XML", xmlDiffs)

	if len(workbookDiffs) > 0 || len(xmlDiffs) > 0 {
		os.Exit(1)
	}
}

func resolveAssetsDir() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("failed to resolve current file location")
	}

	backendDir := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	assetsDir := filepath.Join(backendDir, "assets", "bupot-deduction")
	info, err := os.Stat(assetsDir)
	if err != nil {
		return "", fmt.Errorf("failed to access assets dir: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("assets path is not a directory: %s", assetsDir)
	}
	return assetsDir, nil
}

func loadWorkbookData(path string) (workbookData, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return workbookData{}, fmt.Errorf("failed to open workbook %s: %w", path, err)
	}
	defer file.Close()

	sheetName := "DATA"
	if sheetIndex, _ := file.GetSheetIndex(sheetName); sheetIndex == -1 {
		sheets := file.GetSheetList()
		if len(sheets) == 0 {
			return workbookData{}, fmt.Errorf("workbook has no sheets: %s", path)
		}
		sheetName = sheets[0]
	}

	rows, err := file.GetRows(sheetName)
	if err != nil {
		return workbookData{}, fmt.Errorf("failed to read sheet %q from %s: %w", sheetName, path, err)
	}

	headerRowIndex, headers := detectHeaderRow(rows)
	if headerRowIndex < 0 {
		return workbookData{}, fmt.Errorf("failed to detect header row in %s", path)
	}

	comparedHeaders := make([]string, 0, len(headers))
	ignoredHeaders := make([]string, 0)
	keptIndexes := make([]int, 0, len(headers))
	for idx, header := range headers {
		if strings.TrimSpace(header) == "" {
			continue
		}
		if slices.Contains(ignoredWorkbookHeaders, header) {
			ignoredHeaders = append(ignoredHeaders, header)
			continue
		}
		comparedHeaders = append(comparedHeaders, header)
		keptIndexes = append(keptIndexes, idx)
	}

	rowMaps := make([]map[string]string, 0)
	for rowIndex := headerRowIndex + 1; rowIndex < len(rows); rowIndex++ {
		row := rows[rowIndex]
		record := make(map[string]string, len(comparedHeaders))
		hasValue := false
		for pos, cellIndex := range keptIndexes {
			header := comparedHeaders[pos]
			value := strings.TrimSpace(valueAt(row, cellIndex))
			record[header] = value
			if value != "" {
				hasValue = true
			}
		}
		if !hasValue {
			continue
		}
		rowMaps = append(rowMaps, record)
	}

	profileNPWP, _ := file.GetCellValue(sheetName, "C1")

	return workbookData{
		SheetName:       sheetName,
		HeaderRowNumber: headerRowIndex + 1,
		Headers:         headers,
		ComparedHeaders: comparedHeaders,
		IgnoredHeaders:  ignoredHeaders,
		ProfileNPWP:     strings.TrimSpace(profileNPWP),
		Rows:            rowMaps,
	}, nil
}

func detectHeaderRow(rows [][]string) (int, []string) {
	for rowIndex, row := range rows {
		normalized := make([]string, 0, len(row))
		for _, cell := range row {
			normalized = append(normalized, strings.TrimSpace(cell))
		}

		if contains(normalized, "Masa Pajak") && contains(normalized, "NPWP") && contains(normalized, "Tanggal Pemotongan") {
			headers := trimTrailingEmptyStrings(normalized)
			return rowIndex, headers
		}
	}
	return -1, nil
}

func loadXMLData(path string) (xmlData, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return xmlData{}, fmt.Errorf("failed to read xml %s: %w", path, err)
	}

	var payload bpuBulk
	if err := xml.Unmarshal(b, &payload); err != nil {
		return xmlData{}, fmt.Errorf("failed to decode xml %s: %w", path, err)
	}

	rows := make([]map[string]string, 0, len(payload.ListOfBpu))
	for _, item := range payload.ListOfBpu {
		row := map[string]string{
			"TaxPeriodMonth": strings.TrimSpace(item.TaxPeriodMonth),
			"TaxPeriodYear":  strings.TrimSpace(item.TaxPeriodYear),
			"CounterpartTIN": strings.TrimSpace(item.CounterpartTIN),
			"IDPlaceOfBusinessActivityOfIncomeRecipient": strings.TrimSpace(item.IDPlaceOfBusinessActivityOfRecipient),
			"TaxCertificate":  strings.TrimSpace(item.TaxCertificate),
			"TaxObjectCode":   strings.TrimSpace(item.TaxObjectCode),
			"TaxBase":         strings.TrimSpace(item.TaxBase),
			"Rate":            strings.TrimSpace(item.Rate),
			"Document":        strings.TrimSpace(item.Document),
			"DocumentNumber":  strings.TrimSpace(item.DocumentNumber),
			"DocumentDate":    strings.TrimSpace(item.DocumentDate),
			"GovTreasurerOpt": strings.TrimSpace(item.GovTreasurerOpt),
			"SP2DNumber":      strings.TrimSpace(item.SP2DNumber),
			"WithholdingDate": strings.TrimSpace(item.WithholdingDate),
		}
		rows = append(rows, row)
	}

	return xmlData{
		ComparedFields: append([]string(nil), comparedXMLFields...),
		Rows:           rows,
	}, nil
}

func diffWorkbookData(legacy workbookData, current workbookData) []string {
	diffs := make([]string, 0)

	if !equalStringSlices(legacy.Headers, current.Headers) {
		diffs = append(diffs, fmt.Sprintf("header list berbeda: legacy=%v current=%v", legacy.Headers, current.Headers))
	}
	if !equalStringSlices(legacy.ComparedHeaders, current.ComparedHeaders) {
		diffs = append(diffs, fmt.Sprintf("header compare berbeda: legacy=%v current=%v", legacy.ComparedHeaders, current.ComparedHeaders))
	}
	if len(legacy.Rows) != len(current.Rows) {
		diffs = append(diffs, fmt.Sprintf("jumlah row workbook berbeda: legacy=%d current=%d", len(legacy.Rows), len(current.Rows)))
	}

	maxRows := len(legacy.Rows)
	if len(current.Rows) > maxRows {
		maxRows = len(current.Rows)
	}
	for rowIndex := 0; rowIndex < maxRows; rowIndex++ {
		switch {
		case rowIndex >= len(legacy.Rows):
			diffs = append(diffs, fmt.Sprintf("row %d hanya ada di current workbook", rowIndex+1))
			continue
		case rowIndex >= len(current.Rows):
			diffs = append(diffs, fmt.Sprintf("row %d hanya ada di legacy workbook", rowIndex+1))
			continue
		}

		for _, header := range current.ComparedHeaders {
			legacyValue := legacy.Rows[rowIndex][header]
			currentValue := current.Rows[rowIndex][header]
			if legacyValue == currentValue {
				continue
			}
			diffs = append(diffs, fmt.Sprintf("row %d / %s berbeda: legacy=%q current=%q", rowIndex+1, header, legacyValue, currentValue))
		}
	}

	return diffs
}

func diffXMLData(legacy xmlData, current xmlData) []string {
	diffs := make([]string, 0)
	if len(legacy.Rows) != len(current.Rows) {
		diffs = append(diffs, fmt.Sprintf("jumlah row xml berbeda: legacy=%d current=%d", len(legacy.Rows), len(current.Rows)))
	}

	maxRows := len(legacy.Rows)
	if len(current.Rows) > maxRows {
		maxRows = len(current.Rows)
	}
	for rowIndex := 0; rowIndex < maxRows; rowIndex++ {
		switch {
		case rowIndex >= len(legacy.Rows):
			diffs = append(diffs, fmt.Sprintf("row %d hanya ada di current xml", rowIndex+1))
			continue
		case rowIndex >= len(current.Rows):
			diffs = append(diffs, fmt.Sprintf("row %d hanya ada di legacy xml", rowIndex+1))
			continue
		}

		for _, field := range current.ComparedFields {
			legacyValue := legacy.Rows[rowIndex][field]
			currentValue := current.Rows[rowIndex][field]
			if legacyValue == currentValue {
				continue
			}
			diffs = append(diffs, fmt.Sprintf("row %d / %s berbeda: legacy=%q current=%q", rowIndex+1, field, legacyValue, currentValue))
		}
	}

	return diffs
}

func printDiffSection(label string, diffs []string) {
	if len(diffs) == 0 {
		fmt.Printf("%s: SAME\n", label)
		return
	}

	fmt.Printf("%s: DIFFERENT (%d)\n", label, len(diffs))
	for _, diff := range diffs {
		fmt.Printf("- %s\n", diff)
	}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func trimTrailingEmptyStrings(values []string) []string {
	lastNonEmpty := -1
	for idx, value := range values {
		if strings.TrimSpace(value) != "" {
			lastNonEmpty = idx
		}
	}
	if lastNonEmpty < 0 {
		return nil
	}
	out := make([]string, 0, lastNonEmpty+1)
	for _, value := range values[:lastNonEmpty+1] {
		out = append(out, strings.TrimSpace(value))
	}
	return out
}

func valueAt(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func equalStringSlices(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func filterNonEmptyStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
