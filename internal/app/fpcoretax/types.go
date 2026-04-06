package fpcoretax

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type ProfileConfigKey string

const (
	ProfileConfigFPKeluaranMiscSales      ProfileConfigKey = "fp_keluaran_misc_sales"
	ProfileConfigFPKeluaranReturMiscSales ProfileConfigKey = "fp_keluaran_retur_misc_sales"
	ProfileConfigFPMasukanMiscPurchases   ProfileConfigKey = "fp_masukan_misc_purchases"
)

type RelationRegistryKey string

const (
	RelationRegistryCustomer RelationRegistryKey = "customer"
	RelationRegistrySupplier RelationRegistryKey = "supplier"
)

type ExportMYOBInput struct {
	SheetName string `json:"sheetName"`

	OutputFilename string `json:"outputFilename"`
	AccountNumber  string `json:"accountNumber"`
	IsReturn       bool   `json:"-"`

	HeaderRowNumber int `json:"headerRowNumber"`

	MemoTemplate        string `json:"memoTemplate"`
	DescriptionTemplate string `json:"descriptionTemplate"`
	TaxCode             string `json:"taxCode"`
	Inclusive           bool   `json:"inclusive"`

	FieldMap map[string]string `json:"-"`
}

type RelationRecord struct {
	Name    string `json:"name"`
	Account string `json:"account,omitempty"`
}

func ParseExportMYOBInput(raw json.RawMessage) (ExportMYOBInput, error) {
	input := ExportMYOBInput{
		HeaderRowNumber:     1,
		MemoTemplate:        "{{nomorFakturPajak}}",
		DescriptionTemplate: "{{nomorFakturPajak}}",
		TaxCode:             "PPN",
	}

	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		input.Normalize()
		return input, nil
	}

	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()
	payload := map[string]any{}
	if err := dec.Decode(&payload); err != nil {
		return input, fmt.Errorf("invalid fp coretax input: %w", err)
	}

	if v, ok := payload["sheetName"].(string); ok {
		input.SheetName = strings.TrimSpace(v)
	}
	if v, ok := payload["outputFilename"].(string); ok {
		input.OutputFilename = strings.TrimSpace(v)
	}
	if v, ok := payload["accountNumber"].(string); ok {
		input.AccountNumber = strings.TrimSpace(v)
	}
	if v, ok := payload["memoTemplate"].(string); ok {
		input.MemoTemplate = strings.TrimSpace(v)
	}
	if v, ok := payload["descriptionTemplate"].(string); ok {
		input.DescriptionTemplate = strings.TrimSpace(v)
	}
	if v, ok := payload["taxCode"].(string); ok {
		input.TaxCode = strings.TrimSpace(v)
	}
	if v, ok := payload["inclusive"].(bool); ok {
		input.Inclusive = v
	}
	if v, exists := payload["headerRowNumber"]; exists {
		n, err := parseInt(v)
		if err != nil {
			return input, fmt.Errorf("invalid headerRowNumber: %w", err)
		}
		input.HeaderRowNumber = n
	}
	input.FieldMap = parseFieldMap(payload)
	input.Normalize()
	return input, nil
}

func (i *ExportMYOBInput) Normalize() {
	i.SheetName = strings.TrimSpace(i.SheetName)
	i.OutputFilename = strings.TrimSpace(i.OutputFilename)
	i.AccountNumber = strings.TrimSpace(i.AccountNumber)
	i.MemoTemplate = strings.TrimSpace(i.MemoTemplate)
	if i.MemoTemplate == "" {
		i.MemoTemplate = "{{nomorFakturPajak}}"
	}
	i.DescriptionTemplate = strings.TrimSpace(i.DescriptionTemplate)
	if i.DescriptionTemplate == "" {
		i.DescriptionTemplate = "{{nomorFakturPajak}}"
	}
	i.TaxCode = strings.TrimSpace(i.TaxCode)
	if i.TaxCode == "" {
		i.TaxCode = "PPN"
	}
	if i.HeaderRowNumber <= 0 {
		i.HeaderRowNumber = 1
	}
	for key, value := range i.FieldMap {
		i.FieldMap[key] = strings.TrimSpace(value)
	}
}

func (i ExportMYOBInput) MappedField(key string) string {
	if len(i.FieldMap) == 0 {
		return ""
	}
	return strings.TrimSpace(i.FieldMap[key])
}

func ResolveProfileConfigValues(cfg ProfileConfig) map[string]string {
	values := make(map[string]string, len(cfg.Fields))
	for _, item := range cfg.Fields {
		values[strings.TrimSpace(item.Key)] = strings.TrimSpace(item.Value)
	}
	return values
}

func parseFieldMap(payload map[string]any) map[string]string {
	keys := []string{
		"partyName",
		"documentNumber",
		"returnDocumentNumber",
		"date",
		"returnDate",
		"taxBase",
		"tax",
		"reference",
	}

	fieldMap := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := payload[key].(string); ok {
			fieldMap[key] = strings.TrimSpace(value)
		}
	}
	return fieldMap
}

func parseInt(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case json.Number:
		i, err := v.Int64()
		if err == nil {
			return int(i), nil
		}
		f, ferr := v.Float64()
		if ferr != nil {
			return 0, ferr
		}
		return int(f), nil
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, err
		}
		return i, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}
