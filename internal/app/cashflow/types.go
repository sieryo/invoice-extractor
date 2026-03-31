package cashflow

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Format string

const (
	StandardFormat   Format = "standard"
	InfluencerFormat Format = "influencer"
)

type Type string

const (
	SpendMoneyType   Type = "spend_money"
	ReceiveMoneyType Type = "receive_money"
)

type ExportMYOBInput struct {
	SheetName string `json:"sheetName"`

	OutputFilename string `json:"outputFilename"`

	HeaderRowNumber      int  `json:"headerRowNumber"`
	StartingChequeNumber *int `json:"startingChequeNumber,omitempty"`

	ChequeAccount  string `json:"chequeAccount"`
	CashflowFormat Format `json:"cashflowFormat"`
	CashflowType   Type   `json:"cashflowType"`

	RemarkDelimiter           string            `json:"remarkDelimiter,omitempty"`
	OtherCostsAccountCode     string            `json:"otherCostsAccountCode,omitempty"`
	DefaultIAccountCode       string            `json:"defaultIAccountCode,omitempty"`
	DefaultBAccountCode       string            `json:"defaultBAccountCode,omitempty"`
	InformationFilterKeywords []string          `json:"informationFilterKeywords,omitempty"`
	FieldMap                  map[string]string `json:"-"`
}

type TaxAccount struct {
	Name    string `json:"name"`
	Account string `json:"account"`
}

type SpendMoneyTransaction struct {
	ChequeNumber *int
	Date         time.Time
	Memo         string
	Amount       float64
	Allocation   string
	Items        []SpendMoneyTransactionItem
}

type SpendMoneyTransactionItem struct {
	AccountCode string
	Amount      float64
	Allocation  string
}

type ReceiveMoneyTransaction struct {
	IDNumber   *int
	Date       time.Time
	Memo       string
	Amount     float64
	Allocation string
	Items      []ReceiveMoneyTransactionItem
}

type ReceiveMoneyTransactionItem struct {
	AccountCode string
	Amount      float64
	Allocation  string
}

func ParseExportMYOBInput(raw json.RawMessage) (ExportMYOBInput, error) {
	input := ExportMYOBInput{
		OutputFilename:  "cashflow-myob",
		HeaderRowNumber: 1,
		CashflowFormat:  StandardFormat,
		CashflowType:    SpendMoneyType,
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
		return input, fmt.Errorf("invalid cashflow input: %w", err)
	}

	if v, ok := payload["sheetName"].(string); ok {
		input.SheetName = strings.TrimSpace(v)
	}
	if v, ok := payload["outputFilename"].(string); ok {
		input.OutputFilename = strings.TrimSpace(v)
	}
	if v, ok := payload["chequeAccount"].(string); ok {
		input.ChequeAccount = strings.TrimSpace(v)
	}
	if v, ok := payload["cashflowFormat"].(string); ok {
		input.CashflowFormat = NormalizeFormat(v)
	}
	if v, ok := payload["cashflowType"].(string); ok {
		input.CashflowType = Type(strings.TrimSpace(v))
	}
	if v, ok := payload["remarkDelimiter"].(string); ok {
		input.RemarkDelimiter = strings.TrimSpace(v)
	}
	if v, ok := payload["otherCostsAccountCode"].(string); ok {
		input.OtherCostsAccountCode = strings.TrimSpace(v)
	}
	if v, ok := payload["defaultIAccountCode"].(string); ok {
		input.DefaultIAccountCode = strings.TrimSpace(v)
	}
	if v, ok := payload["defaultBAccountCode"].(string); ok {
		input.DefaultBAccountCode = strings.TrimSpace(v)
	}
	input.InformationFilterKeywords = parseKeywordList(payload["informationFilterKeywords"])
	input.FieldMap = parseFieldMap(payload)
	if v, exists := payload["headerRowNumber"]; exists {
		n, err := parseInt(v)
		if err != nil {
			return input, fmt.Errorf("invalid headerRowNumber: %w", err)
		}
		input.HeaderRowNumber = n
	}
	if v, exists := payload["startingChequeNumber"]; exists {
		n, err := parseInt(v)
		if err != nil {
			return input, fmt.Errorf("invalid startingChequeNumber: %w", err)
		}
		input.StartingChequeNumber = &n
	}

	input.Normalize()
	return input, nil
}

func (i *ExportMYOBInput) Normalize() {
	i.SheetName = strings.TrimSpace(i.SheetName)
	i.OutputFilename = strings.TrimSpace(i.OutputFilename)
	if i.OutputFilename == "" {
		i.OutputFilename = "cashflow-myob"
	}
	i.ChequeAccount = strings.TrimSpace(i.ChequeAccount)
	i.RemarkDelimiter = strings.TrimSpace(i.RemarkDelimiter)
	i.OtherCostsAccountCode = strings.TrimSpace(i.OtherCostsAccountCode)
	i.DefaultIAccountCode = strings.TrimSpace(i.DefaultIAccountCode)
	i.DefaultBAccountCode = strings.TrimSpace(i.DefaultBAccountCode)
	i.InformationFilterKeywords = normalizeKeywordList(i.InformationFilterKeywords)
	for key, value := range i.FieldMap {
		i.FieldMap[key] = strings.TrimSpace(value)
	}
	if i.HeaderRowNumber <= 0 {
		i.HeaderRowNumber = 1
	}
	if i.StartingChequeNumber != nil && *i.StartingChequeNumber <= 0 {
		i.StartingChequeNumber = nil
	}
	i.CashflowFormat = NormalizeFormat(string(i.CashflowFormat))
	if i.CashflowType == "" {
		i.CashflowType = SpendMoneyType
	}
}

func (i ExportMYOBInput) MappedField(key string) string {
	if len(i.FieldMap) == 0 {
		return ""
	}
	return strings.TrimSpace(i.FieldMap[key])
}

func parseFieldMap(payload map[string]any) map[string]string {
	keys := []string{
		"date",
		"information",
		"coa",
		"otherCost",
		"pp23",
		"pph15",
		"pph21",
		"pph23",
		"pph42",
		"ppn",
		"remark",
		"total",
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

func NormalizeFormat(raw string) Format {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(InfluencerFormat):
		return InfluencerFormat
	case "default", string(StandardFormat):
		return StandardFormat
	default:
		return StandardFormat
	}
}

func parseKeywordList(raw any) []string {
	switch value := raw.(type) {
	case string:
		return splitKeywordList(value)
	case []string:
		return normalizeKeywordList(value)
	case []any:
		items := make([]string, 0, len(value))
		for _, entry := range value {
			text, ok := entry.(string)
			if !ok {
				continue
			}
			items = append(items, text)
		}
		return normalizeKeywordList(items)
	default:
		return nil
	}
}

func splitKeywordList(raw string) []string {
	value := strings.ReplaceAll(raw, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, ",", "\n")
	parts := strings.Split(value, "\n")
	return normalizeKeywordList(parts)
}

func normalizeKeywordList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
