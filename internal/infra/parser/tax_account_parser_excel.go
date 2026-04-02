package parser

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/taxcatalog"
	"github.com/xuri/excelize/v2"
)

type TaxAccountColumnSpec struct {
	Key         string `json:"key"`
	Header      string `json:"header"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

type TaxAccountUploadSpec struct {
	AcceptedExtensions []string `json:"acceptedExtensions"`
	AcceptedMIMETypes  []string `json:"acceptedMimeTypes"`
	MaxFileSizeMB      int64    `json:"maxFileSizeMB"`
}

type TaxAccountSchemaSpec struct {
	SchemaVersion string                 `json:"schemaVersion"`
	Columns       []TaxAccountColumnSpec `json:"columns"`
	AllowedNames  []string               `json:"allowedNames,omitempty"`
	Upload        TaxAccountUploadSpec   `json:"upload"`
	Relative      string                 `json:"relative"`
	LookupKey     string                 `json:"lookupKey"`
}

type TaxAccountSchemaMismatchError struct {
	MissingColumns []string
}

type TaxAccountExcelParser struct{}

type TaxAccountRecord struct {
	Name    string
	Account string
}

var ErrTaxAccountSchemaMismatch = errors.New("tax account schema mismatch")
var nonAlphaNumTaxHeaderRegex = regexp.MustCompile(`[^a-z0-9]+`)

const taxAccountSchemaVersion = "1.0.0"

func NewTaxAccountExcelParser() *TaxAccountExcelParser {
	return &TaxAccountExcelParser{}
}

func (p *TaxAccountExcelParser) SchemaSpec() TaxAccountSchemaSpec {
	return TaxAccountSchemaSpec{
		SchemaVersion: taxAccountSchemaVersion,
		Columns: []TaxAccountColumnSpec{
			{Key: "name", Header: "name", Required: true, Description: "Nama akun lookup"},
			{Key: "account", Header: "account", Required: true, Description: "Kode akun MYOB"},
		},
		AllowedNames: taxcatalog.CanonicalTaxNames(),
		Upload: TaxAccountUploadSpec{
			AcceptedExtensions: []string{".xlsx", ".xls"},
			AcceptedMIMETypes: []string{
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"application/vnd.ms-excel",
			},
			MaxFileSizeMB: 10,
		},
		Relative:  "tax_accounts.csv",
		LookupKey: "name",
	}
}

func (p *TaxAccountExcelParser) Parse(path string) ([]TaxAccountRecord, []ValidationIssue, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, err
	}
	if len(rows) == 0 {
		return nil, nil, &TaxAccountSchemaMismatchError{MissingColumns: requiredTaxAccountHeaders(p.SchemaSpec())}
	}

	columnIdxByHeader, missingHeaders := resolveTaxAccountColumnIndexes(rows[0], p.SchemaSpec())
	if len(missingHeaders) > 0 {
		return nil, nil, &TaxAccountSchemaMismatchError{MissingColumns: missingHeaders}
	}

	var accounts []TaxAccountRecord
	var issues []ValidationIssue
	for i, row := range rows {
		if i == 0 {
			continue
		}
		rowNum := i + 1
		name := taxAccountCellValue(row, columnIdxByHeader["name"])
		account := taxAccountCellValue(row, columnIdxByHeader["account"])
		if name == "" && account == "" {
			continue
		}
		if name == "" {
			issues = append(issues, ValidationIssue{
				Row: rowNum, Field: "name", Message: "name kosong",
			})
			continue
		}
		if account == "" {
			issues = append(issues, ValidationIssue{
				Row: rowNum, Field: "account", Message: "account kosong",
			})
			continue
		}
		if !taxcatalog.IsCanonicalTaxName(name) {
			issues = append(issues, ValidationIssue{
				Row:     rowNum,
				Field:   "name",
				Value:   name,
				Message: "name tidak termasuk daftar nama tax yang didukung dan akan dilewati",
			})
			continue
		}
		accounts = append(accounts, TaxAccountRecord{
			Name:    name,
			Account: account,
		})
	}
	return accounts, issues, nil
}

func (e *TaxAccountSchemaMismatchError) Error() string {
	if e == nil || len(e.MissingColumns) == 0 {
		return ErrTaxAccountSchemaMismatch.Error()
	}
	return fmt.Sprintf("%s: missing required columns: %s", ErrTaxAccountSchemaMismatch.Error(), strings.Join(e.MissingColumns, ", "))
}

func (e *TaxAccountSchemaMismatchError) Unwrap() error {
	return ErrTaxAccountSchemaMismatch
}

func resolveTaxAccountColumnIndexes(headerRow []string, spec TaxAccountSchemaSpec) (map[string]int, []string) {
	normalizedToIndex := make(map[string]int, len(headerRow))
	for idx, value := range headerRow {
		normalized := normalizeTaxAccountHeader(value)
		if normalized == "" {
			continue
		}
		if _, exists := normalizedToIndex[normalized]; exists {
			continue
		}
		normalizedToIndex[normalized] = idx
	}

	resolved := make(map[string]int, len(spec.Columns))
	missing := make([]string, 0, len(spec.Columns))
	for _, column := range spec.Columns {
		target := strings.TrimSpace(column.Header)
		if target == "" {
			continue
		}
		idx, ok := normalizedToIndex[normalizeTaxAccountHeader(target)]
		if !ok {
			if column.Required {
				missing = append(missing, target)
			}
			continue
		}
		resolved[column.Key] = idx
	}

	sort.Strings(missing)
	return resolved, missing
}

func requiredTaxAccountHeaders(spec TaxAccountSchemaSpec) []string {
	headers := make([]string, 0, len(spec.Columns))
	for _, column := range spec.Columns {
		if column.Required {
			headers = append(headers, column.Header)
		}
	}
	sort.Strings(headers)
	return headers
}

func normalizeTaxAccountHeader(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}
	return nonAlphaNumTaxHeaderRegex.ReplaceAllString(trimmed, "")
}

func taxAccountCellValue(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}
