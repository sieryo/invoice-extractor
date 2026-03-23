package cashflow

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/infra/parser"
	"github.com/sieryo/invoice-extractor/internal/infra/storage"
)

type TaxAccountStatus struct {
	Loaded  bool   `json:"loaded"`
	Count   int    `json:"count"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TaxAccountService struct {
	path   string
	parser *parser.TaxAccountExcelParser
	store  *storage.TaxAccountCSVStore
}

func NewTaxAccountService(path string) *TaxAccountService {
	return &TaxAccountService{
		path:   path,
		parser: parser.NewTaxAccountExcelParser(),
		store:  storage.NewTaxAccountCSVStore(path),
	}
}

func (s *TaxAccountService) Spec() parser.TaxAccountSchemaSpec {
	return s.parser.SchemaSpec()
}

func (s *TaxAccountService) Status() TaxAccountStatus {
	accounts, err := s.Load()
	if err != nil {
		if os.IsNotExist(err) {
			return TaxAccountStatus{
				Loaded:  false,
				Count:   0,
				Code:    "CASHFLOW_TAX_ACCOUNTS_NOT_READY",
				Message: "Master data tax accounts belum tersedia. Upload file tax accounts terlebih dahulu.",
			}
		}
		var schemaErr *parser.TaxAccountSchemaMismatchError
		if errors.As(err, &schemaErr) {
			return TaxAccountStatus{
				Loaded:  false,
				Count:   0,
				Code:    "CASHFLOW_TAX_ACCOUNTS_INVALID_SCHEMA",
				Message: fmt.Sprintf("Schema tax accounts tidak sesuai. Kolom wajib: %s.", strings.Join(requiredTaxColumns(s.Spec()), ", ")),
			}
		}
		return TaxAccountStatus{
			Loaded:  false,
			Count:   0,
			Code:    "CASHFLOW_TAX_ACCOUNTS_UNAVAILABLE",
			Message: "Master data tax accounts tidak dapat dibaca saat ini.",
		}
	}
	if len(accounts) == 0 {
		return TaxAccountStatus{
			Loaded:  false,
			Count:   0,
			Code:    "CASHFLOW_TAX_ACCOUNTS_EMPTY",
			Message: "Master data tax accounts kosong. Upload file tax accounts yang valid.",
		}
	}
	return TaxAccountStatus{
		Loaded:  true,
		Count:   len(accounts),
		Code:    "CASHFLOW_TAX_ACCOUNTS_READY",
		Message: "Master data tax accounts siap digunakan.",
	}
}

func (s *TaxAccountService) List() ([]TaxAccount, error) {
	accounts, err := s.Load()
	if err != nil {
		return nil, err
	}
	out := make([]TaxAccount, 0, len(accounts))
	for _, item := range accounts {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (s *TaxAccountService) Load() (map[string]TaxAccount, error) {
	f, err := os.Open(s.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, &parser.TaxAccountSchemaMismatchError{MissingColumns: requiredTaxColumns(s.Spec())}
	}

	headerIndex, missing := resolveTaxAccountHeaderIndexes(rows[0], []string{"name", "account"})
	if len(missing) > 0 {
		return nil, &parser.TaxAccountSchemaMismatchError{MissingColumns: missing}
	}

	accounts := make(map[string]TaxAccount)
	for _, row := range rows[1:] {
		name := taxAccountCell(row, headerIndex["name"])
		account := taxAccountCell(row, headerIndex["account"])
		if name == "" || account == "" {
			continue
		}
		record := TaxAccount{Name: name, Account: account}
		accounts[normalizeTaxLookupKey(name)] = record
	}
	return accounts, nil
}

func (s *TaxAccountService) Lookup(name string) (TaxAccount, bool, error) {
	accounts, err := s.Load()
	if err != nil {
		return TaxAccount{}, false, err
	}
	account, ok := accounts[normalizeTaxLookupKey(name)]
	return account, ok, nil
}

func (s *TaxAccountService) Update(filePath string) (int, []parser.ValidationIssue, error) {
	accounts, issues, err := s.parser.Parse(filePath)
	if err != nil {
		return 0, nil, err
	}
	rows := make([]storage.TaxAccountCSVRecord, 0, len(accounts))
	for _, account := range accounts {
		rows = append(rows, storage.TaxAccountCSVRecord{
			Name:    account.Name,
			Account: account.Account,
		})
	}
	if err := s.store.Save(rows); err != nil {
		return 0, nil, err
	}
	return len(accounts), issues, nil
}

func (s *TaxAccountService) TempFilePath() string {
	return filepath.Join(filepath.Dir(s.path), "tax_accounts_upload.xlsx")
}

func (s *TaxAccountService) IsAcceptedUpload(filename string, sizeBytes int64) (bool, string) {
	spec := s.Spec()
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(filename)))
	if ext == "" {
		return false, "format file tidak dikenali"
	}
	for _, allowed := range spec.Upload.AcceptedExtensions {
		if strings.EqualFold(strings.TrimSpace(allowed), ext) {
			if spec.Upload.MaxFileSizeMB > 0 && sizeBytes > spec.Upload.MaxFileSizeMB*1024*1024 {
				return false, "ukuran file melebihi batas maksimal"
			}
			return true, ""
		}
	}
	return false, "format file tidak didukung"
}

func normalizeTaxLookupKey(raw string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(raw))), " ")
}

func resolveTaxAccountHeaderIndexes(headerRow []string, required []string) (map[string]int, []string) {
	normalized := make(map[string]int, len(headerRow))
	for idx, value := range headerRow {
		key := normalizeTaxLookupKey(value)
		if key == "" {
			continue
		}
		if _, exists := normalized[key]; exists {
			continue
		}
		normalized[key] = idx
	}

	out := make(map[string]int, len(required))
	missing := make([]string, 0)
	for _, column := range required {
		key := normalizeTaxLookupKey(column)
		idx, ok := normalized[key]
		if !ok {
			missing = append(missing, column)
			continue
		}
		out[key] = idx
	}
	sort.Strings(missing)
	return out, missing
}

func taxAccountCell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func requiredTaxColumns(spec parser.TaxAccountSchemaSpec) []string {
	columns := make([]string, 0, len(spec.Columns))
	for _, column := range spec.Columns {
		if column.Required {
			columns = append(columns, column.Header)
		}
	}
	sort.Strings(columns)
	return columns
}
