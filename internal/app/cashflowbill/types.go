package cashflowbill

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type Type string

const (
	PayBillsType        Type = "pay_bills"
	ReceivePaymentsType Type = "receive_payments"
)

type ExportInput struct {
	SheetName string `json:"sheetName"`

	OutputFilename string `json:"outputFilename"`
	HeaderRowNumber int   `json:"headerRowNumber"`

	ChequeAccount string `json:"chequeAccount"`
	CashflowType  Type   `json:"cashflowType"`

	LedgerSnapshotRef string            `json:"ledgerSnapshotRef,omitempty"`
	FieldMap          map[string]string `json:"-"`
}

func ParseExportInput(raw json.RawMessage) (ExportInput, error) {
	input := ExportInput{
		OutputFilename:  "cashflow-bills",
		HeaderRowNumber: 1,
		CashflowType:    PayBillsType,
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
		return input, fmt.Errorf("invalid cashflow bills input: %w", err)
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
	if v, ok := payload["cashflowType"].(string); ok {
		input.CashflowType = NormalizeType(v)
	}
	if v, ok := payload["ledgerSnapshotRef"].(string); ok {
		input.LedgerSnapshotRef = strings.TrimSpace(v)
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

func (i *ExportInput) Normalize() {
	i.SheetName = strings.TrimSpace(i.SheetName)
	i.OutputFilename = strings.TrimSpace(i.OutputFilename)
	if i.OutputFilename == "" {
		i.OutputFilename = "cashflow-bills"
	}
	i.ChequeAccount = strings.TrimSpace(i.ChequeAccount)
	i.LedgerSnapshotRef = strings.TrimSpace(i.LedgerSnapshotRef)
	if i.HeaderRowNumber <= 0 {
		i.HeaderRowNumber = 1
	}
	i.CashflowType = NormalizeType(string(i.CashflowType))
	for key, value := range i.FieldMap {
		i.FieldMap[key] = strings.TrimSpace(value)
	}
}

func (i ExportInput) MappedField(key string) string {
	if len(i.FieldMap) == 0 {
		return ""
	}
	return strings.TrimSpace(i.FieldMap[key])
}

func NormalizeType(raw string) Type {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(ReceivePaymentsType):
		return ReceivePaymentsType
	default:
		return PayBillsType
	}
}

type Transaction struct {
	Party      string
	RowNumbers []int
	Items      []TransactionItem
	Total      float64
}

type TransactionItem struct {
	RowSource  int
	RefID      string
	Memo       string
	RefDate    time.Time
	ActionDate time.Time
	Amount     float64
	IsPartial  bool
}

func parseFieldMap(payload map[string]any) map[string]string {
	keys := []string{
		"category",
		"information",
		"date",
		"partyName",
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
