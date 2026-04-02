package parser

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/xuri/excelize/v2"
)

type FPCoretaxRelationRegistryKey string

const (
	FPCoretaxRelationRegistryCustomer FPCoretaxRelationRegistryKey = "customer"
	FPCoretaxRelationRegistrySupplier FPCoretaxRelationRegistryKey = "supplier"
)

type FPCoretaxRelationColumnSpec struct {
	Key         string `json:"key"`
	Header      string `json:"header"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

type FPCoretaxRelationUploadSpec struct {
	AcceptedExtensions []string `json:"acceptedExtensions"`
	AcceptedMIMETypes  []string `json:"acceptedMimeTypes"`
	MaxFileSizeMB      int64    `json:"maxFileSizeMB"`
}

type FPCoretaxRelationSchemaSpec struct {
	SchemaVersion string                         `json:"schemaVersion"`
	RegistryKey   string                         `json:"registryKey"`
	Label         string                         `json:"label"`
	Columns       []FPCoretaxRelationColumnSpec  `json:"columns"`
	Upload        FPCoretaxRelationUploadSpec    `json:"upload"`
	Relative      string                         `json:"relative"`
	LookupKey     string                         `json:"lookupKey"`
}

type FPCoretaxRelationSchemaMismatchError struct {
	MissingColumns []string
}

type FPCoretaxRelationExcelParser struct{}

type FPCoretaxRelationRecord struct {
	Name    string
	Account string
}

var ErrFPCoretaxRelationSchemaMismatch = errors.New("fp coretax relation schema mismatch")
var fpCoretaxNonAlphaNumHeaderRegex = regexp.MustCompile(`[^a-z0-9]+`)

const fpCoretaxRelationSchemaVersion = "1.0.0"

func NewFPCoretaxRelationExcelParser() *FPCoretaxRelationExcelParser {
	return &FPCoretaxRelationExcelParser{}
}

func (p *FPCoretaxRelationExcelParser) SchemaSpec(key FPCoretaxRelationRegistryKey) FPCoretaxRelationSchemaSpec {
	spec := FPCoretaxRelationSchemaSpec{
		SchemaVersion: fpCoretaxRelationSchemaVersion,
		RegistryKey:   string(key),
		Upload: FPCoretaxRelationUploadSpec{
			AcceptedExtensions: []string{".xlsx", ".xls"},
			AcceptedMIMETypes: []string{
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"application/vnd.ms-excel",
			},
			MaxFileSizeMB: 10,
		},
		LookupKey: "Co./Last Name",
	}

	switch key {
	case FPCoretaxRelationRegistrySupplier:
		spec.Label = "Supplier Registry FP Masukan"
		spec.Relative = "fp_masukan_suppliers.csv"
		spec.Columns = []FPCoretaxRelationColumnSpec{
			{Key: "name", Header: "Co./Last Name", Required: true, Description: "Nama supplier di MYOB."},
			{Key: "account", Header: "supplier account", Required: false, Description: "Akun supplier MYOB. Jika kosong, action akan memakai fallback account number."},
		}
	default:
		spec.Label = "Customer Registry FP Keluaran"
		spec.Relative = "fp_keluaran_customers.csv"
		spec.Columns = []FPCoretaxRelationColumnSpec{
			{Key: "name", Header: "Co./Last Name", Required: true, Description: "Nama customer di MYOB."},
		}
	}

	return spec
}

func (p *FPCoretaxRelationExcelParser) Parse(
	key FPCoretaxRelationRegistryKey,
	path string,
) ([]FPCoretaxRelationRecord, []ValidationIssue, error) {
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
	spec := p.SchemaSpec(key)
	if len(rows) == 0 {
		return nil, nil, &FPCoretaxRelationSchemaMismatchError{MissingColumns: requiredFPCoretaxHeaders(spec)}
	}

	columnIdxByHeader, missingHeaders := resolveFPCoretaxRelationColumnIndexes(rows[0], spec)
	if len(missingHeaders) > 0 {
		return nil, nil, &FPCoretaxRelationSchemaMismatchError{MissingColumns: missingHeaders}
	}

	records := make([]FPCoretaxRelationRecord, 0, len(rows))
	issues := make([]ValidationIssue, 0)
	for i, row := range rows {
		if i == 0 {
			continue
		}
		rowNum := i + 1
		name := fpCoretaxRelationCellValue(row, columnIdxByHeader["name"])
		account := fpCoretaxRelationCellValue(row, columnIdxByHeader["account"])
		if name == "" && account == "" {
			continue
		}
		if name == "" {
			issues = append(issues, ValidationIssue{
				Row:     rowNum,
				Field:   "Co./Last Name",
				Message: "Co./Last Name kosong",
			})
			continue
		}
		records = append(records, FPCoretaxRelationRecord{
			Name:    name,
			Account: account,
		})
	}
	return records, issues, nil
}

func (e *FPCoretaxRelationSchemaMismatchError) Error() string {
	if e == nil || len(e.MissingColumns) == 0 {
		return ErrFPCoretaxRelationSchemaMismatch.Error()
	}
	return fmt.Sprintf("%s: missing required columns: %s", ErrFPCoretaxRelationSchemaMismatch.Error(), strings.Join(e.MissingColumns, ", "))
}

func (e *FPCoretaxRelationSchemaMismatchError) Unwrap() error {
	return ErrFPCoretaxRelationSchemaMismatch
}

func resolveFPCoretaxRelationColumnIndexes(headerRow []string, spec FPCoretaxRelationSchemaSpec) (map[string]int, []string) {
	normalizedToIndex := make(map[string]int, len(headerRow))
	for idx, value := range headerRow {
		normalized := normalizeFPCoretaxRelationHeader(value)
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
		idx, ok := normalizedToIndex[normalizeFPCoretaxRelationHeader(target)]
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

func requiredFPCoretaxHeaders(spec FPCoretaxRelationSchemaSpec) []string {
	headers := make([]string, 0, len(spec.Columns))
	for _, column := range spec.Columns {
		if column.Required {
			headers = append(headers, column.Header)
		}
	}
	sort.Strings(headers)
	return headers
}

func normalizeFPCoretaxRelationHeader(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}
	return fpCoretaxNonAlphaNumHeaderRegex.ReplaceAllString(trimmed, "")
}

func fpCoretaxRelationCellValue(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}
