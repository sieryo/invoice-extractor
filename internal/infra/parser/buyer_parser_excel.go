package parser

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/domain/buyer"
	"github.com/sieryo/invoice-extractor/pkg/helper"
	"github.com/xuri/excelize/v2"
)

type BuyerExcelParser struct{}

type BuyerRegistryColumnSpec struct {
	Key         string `json:"key"`
	Header      string `json:"header"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

type BuyerRegistryUploadSpec struct {
	AcceptedExtensions []string `json:"acceptedExtensions"`
	AcceptedMIMETypes  []string `json:"acceptedMimeTypes"`
	MaxFileSizeMB      int64    `json:"maxFileSizeMB"`
}

type BuyerRegistrySchemaSpec struct {
	SchemaVersion string                    `json:"schemaVersion"`
	Columns       []BuyerRegistryColumnSpec `json:"columns"`
	Upload        BuyerRegistryUploadSpec   `json:"upload"`
}

type BuyerSchemaMismatchError struct {
	MissingColumns []string
}

var ErrBuyerSchemaMismatch = errors.New("buyer registry schema mismatch")
var nonAlphaNumHeaderRegex = regexp.MustCompile(`[^a-z0-9]+`)

const (
	buyerSchemaVersion = "1.0.0"
)

func (e *BuyerSchemaMismatchError) Error() string {
	if e == nil {
		return ErrBuyerSchemaMismatch.Error()
	}
	if len(e.MissingColumns) == 0 {
		return ErrBuyerSchemaMismatch.Error()
	}
	return fmt.Sprintf(
		"%s: missing required columns: %s",
		ErrBuyerSchemaMismatch.Error(),
		strings.Join(e.MissingColumns, ", "),
	)
}

func (e *BuyerSchemaMismatchError) Unwrap() error {
	return ErrBuyerSchemaMismatch
}

func NewBuyerExcelParser() *BuyerExcelParser {
	return &BuyerExcelParser{}
}

func (p *BuyerExcelParser) SchemaSpec() BuyerRegistrySchemaSpec {
	return BuyerRegistrySchemaSpec{
		SchemaVersion: buyerSchemaVersion,
		Columns: []BuyerRegistryColumnSpec{
			{Key: "name", Header: "NAMA", Required: true, Description: "Nama buyer"},
			{Key: "npwp15", Header: "NPWP 15 DIGIT", Required: true, Description: "NPWP lama 15 digit"},
			{Key: "npwp16", Header: "NPWP 16 DIGIT", Required: true, Description: "NPWP baru 16 digit"},
			{Key: "nitku", Header: "NITKU", Required: true, Description: "Nomor Identitas Tempat Kegiatan Usaha"},
			{Key: "email", Header: "EMAIL", Required: true, Description: "Email buyer"},
			{Key: "address", Header: "ALAMAT", Required: true, Description: "Alamat buyer"},
		},
		Upload: BuyerRegistryUploadSpec{
			AcceptedExtensions: []string{".xlsx", ".xls"},
			AcceptedMIMETypes: []string{
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"application/vnd.ms-excel",
			},
			MaxFileSizeMB: 10,
		},
	}
}

func (p *BuyerExcelParser) Parse(path string) ([]buyer.Buyer, []ValidationIssue, error) {
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
		return nil, nil, &BuyerSchemaMismatchError{MissingColumns: requiredBuyerHeaders(p.SchemaSpec())}
	}

	columnIdxByHeader, missingHeaders := resolveBuyerColumnIndexes(rows[0], p.SchemaSpec())
	if len(missingHeaders) > 0 {
		return nil, nil, &BuyerSchemaMismatchError{MissingColumns: missingHeaders}
	}

	var buyers []buyer.Buyer
	var issues []ValidationIssue

	for i, row := range rows {
		if i == 0 {
			continue // header
		}

		rowNum := i + 1

		npwp15Raw := cell(row, columnIdxByHeader["NPWP 15 DIGIT"])
		npwp16Raw := cell(row, columnIdxByHeader["NPWP 16 DIGIT"])
		nitkuRaw := cell(row, columnIdxByHeader["NITKU"])

		npwp15 := helper.DigitsOnly(npwp15Raw)
		npwp16 := helper.DigitsOnly(npwp16Raw)
		nitku := helper.DigitsOnly(nitkuRaw)

		if npwp15 != "" && !helper.IsNPWP15(npwp15) {
			issues = append(issues, ValidationIssue{
				Row:     rowNum,
				Field:   "npwp_15",
				Value:   npwp15Raw,
				Message: fmt.Sprintf("invalid length: expected 15 digits, got %d", len(npwp15)),
			})
			npwp15 = ""
		}
		if npwp16 != "" && !helper.IsNPWP16(npwp16) {
			issues = append(issues, ValidationIssue{
				Row:     rowNum,
				Field:   "npwp_16",
				Value:   npwp16Raw,
				Message: fmt.Sprintf("invalid length: expected 16 digits, got %d", len(npwp16)),
			})
			npwp16 = ""
		}
		if nitku != "" && !helper.IsNITKU(nitku) {
			issues = append(issues, ValidationIssue{
				Row:     rowNum,
				Field:   "nitku",
				Value:   nitkuRaw,
				Message: fmt.Sprintf("invalid length: expected 22 digits, got %d", len(nitku)),
			})
			nitku = ""
		}

		b := buyer.Buyer{
			Name:    cell(row, columnIdxByHeader["NAMA"]),
			NPWP15:  npwp15,
			NPWP16:  npwp16,
			NITKU:   nitku,
			Email:   cell(row, columnIdxByHeader["EMAIL"]),
			Address: cell(row, columnIdxByHeader["ALAMAT"]),
		}

		if b.Name == "" || (b.NPWP15 == "" && b.NPWP16 == "" && b.NITKU == "") {
			continue
		}

		buyers = append(buyers, b)
	}

	return buyers, issues, nil
}

func cell(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func resolveBuyerColumnIndexes(
	headerRow []string,
	spec BuyerRegistrySchemaSpec,
) (map[string]int, []string) {
	normalizedToIndex := make(map[string]int, len(headerRow))
	for idx, value := range headerRow {
		normalized := normalizeBuyerHeader(value)
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

		idx, ok := normalizedToIndex[normalizeBuyerHeader(target)]
		if !ok {
			if column.Required {
				missing = append(missing, target)
			}
			continue
		}
		resolved[target] = idx
	}

	sort.Strings(missing)
	return resolved, missing
}

func requiredBuyerHeaders(spec BuyerRegistrySchemaSpec) []string {
	headers := make([]string, 0, len(spec.Columns))
	for _, column := range spec.Columns {
		if !column.Required {
			continue
		}
		headers = append(headers, column.Header)
	}
	sort.Strings(headers)
	return headers
}

func normalizeBuyerHeader(raw string) string {
	trimmed := strings.TrimSpace(strings.ToLower(raw))
	if trimmed == "" {
		return ""
	}
	return nonAlphaNumHeaderRegex.ReplaceAllString(trimmed, "")
}
