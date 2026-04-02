package cashflowbill

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/infra/parser"
	"github.com/sieryo/invoice-extractor/internal/infra/storage"
	"github.com/sieryo/invoice-extractor/internal/profilepath"
	"github.com/xuri/excelize/v2"
)

type CategoryAccount struct {
	Name    string `json:"name"`
	Account string `json:"account"`
}

type CategoryAccountStatus struct {
	Loaded  bool   `json:"loaded"`
	Count   int    `json:"count"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CategoryAccountService struct {
	rootDir string
}

func NewCategoryAccountService(rootDir string) *CategoryAccountService {
	return &CategoryAccountService{
		rootDir: rootDir,
	}
}

func (s *CategoryAccountService) Spec() parser.TaxAccountSchemaSpec {
	return parser.TaxAccountSchemaSpec{
		SchemaVersion: "1.0.0",
		Columns: []parser.TaxAccountColumnSpec{
			{Key: "name", Header: "name", Required: true, Description: "Nama category lookup"},
			{Key: "account", Header: "account", Required: true, Description: "Kode akun MYOB"},
		},
		Upload: parser.TaxAccountUploadSpec{
			AcceptedExtensions: []string{".xlsx", ".xls"},
			AcceptedMIMETypes: []string{
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				"application/vnd.ms-excel",
			},
			MaxFileSizeMB: 10,
		},
		Relative:  "category_accounts.csv",
		LookupKey: "name",
	}
}

func (s *CategoryAccountService) Status(profileID string) CategoryAccountStatus {
	accounts, err := s.Load(profileID)
	if err != nil {
		if os.IsNotExist(err) {
			return CategoryAccountStatus{
				Loaded:  false,
				Code:    "CASHFLOW_CATEGORY_ACCOUNTS_NOT_READY",
				Message: "Master data category accounts belum tersedia. Upload file category accounts terlebih dahulu.",
			}
		}
		var schemaErr *parser.TaxAccountSchemaMismatchError
		if errors.As(err, &schemaErr) {
			return CategoryAccountStatus{
				Loaded:  false,
				Code:    "CASHFLOW_CATEGORY_ACCOUNTS_INVALID_SCHEMA",
				Message: fmt.Sprintf("Schema category accounts tidak sesuai. Kolom wajib: %s.", strings.Join(requiredCategoryColumns(s.Spec()), ", ")),
			}
		}
		return CategoryAccountStatus{
			Loaded:  false,
			Code:    "CASHFLOW_CATEGORY_ACCOUNTS_UNAVAILABLE",
			Message: "Master data category accounts tidak dapat dibaca saat ini.",
		}
	}
	if len(accounts) == 0 {
		return CategoryAccountStatus{
			Loaded:  false,
			Code:    "CASHFLOW_CATEGORY_ACCOUNTS_EMPTY",
			Message: "Master data category accounts kosong. Upload file category accounts yang valid.",
		}
	}
	return CategoryAccountStatus{
		Loaded:  true,
		Count:   len(accounts),
		Code:    "CASHFLOW_CATEGORY_ACCOUNTS_READY",
		Message: "Master data category accounts siap digunakan.",
	}
}

func (s *CategoryAccountService) List(profileID string) ([]CategoryAccount, error) {
	accounts, err := s.Load(profileID)
	if err != nil {
		return nil, err
	}
	out := make([]CategoryAccount, 0, len(accounts))
	for _, item := range accounts {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (s *CategoryAccountService) Load(profileID string) (map[string]CategoryAccount, error) {
	path := profilepath.CashflowCategoryAccountsCSV(s.rootDir, profileID)
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, &parser.TaxAccountSchemaMismatchError{MissingColumns: requiredCategoryColumns(s.Spec())}
	}

	headerIndex, missing := resolveCategoryAccountHeaderIndexes(rows[0], []string{"name", "account"})
	if len(missing) > 0 {
		return nil, &parser.TaxAccountSchemaMismatchError{MissingColumns: missing}
	}

	accounts := make(map[string]CategoryAccount)
	for _, row := range rows[1:] {
		name := categoryAccountCell(row, headerIndex["name"])
		account := categoryAccountCell(row, headerIndex["account"])
		if name == "" || account == "" {
			continue
		}
		record := CategoryAccount{Name: name, Account: account}
		accounts[normalizeCategoryLookupKey(name)] = record
	}
	return accounts, nil
}

func (s *CategoryAccountService) Update(profileID string, filePath string) (int, []parser.ValidationIssue, error) {
	accounts, issues, err := s.parseWorkbook(filePath)
	if err != nil {
		return 0, nil, err
	}
	if err := os.MkdirAll(profilepath.ProfileDir(s.rootDir, profileID), 0o755); err != nil {
		return 0, nil, err
	}
	store := storage.NewTaxAccountCSVStore(profilepath.CashflowCategoryAccountsCSV(s.rootDir, profileID))
	rows := make([]storage.TaxAccountCSVRecord, 0, len(accounts))
	for _, account := range accounts {
		rows = append(rows, storage.TaxAccountCSVRecord{
			Name:    account.Name,
			Account: account.Account,
		})
	}
	if err := store.Save(rows); err != nil {
		return 0, nil, err
	}
	return len(accounts), issues, nil
}

func (s *CategoryAccountService) parseWorkbook(path string) ([]parser.TaxAccountRecord, []parser.ValidationIssue, error) {
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
		return nil, nil, &parser.TaxAccountSchemaMismatchError{MissingColumns: requiredCategoryColumns(s.Spec())}
	}

	headerIndex, missing := resolveCategoryAccountHeaderIndexes(rows[0], []string{"name", "account"})
	if len(missing) > 0 {
		return nil, nil, &parser.TaxAccountSchemaMismatchError{MissingColumns: missing}
	}

	records := make([]parser.TaxAccountRecord, 0, len(rows))
	issues := make([]parser.ValidationIssue, 0)
	for i, row := range rows {
		if i == 0 {
			continue
		}
		rowNum := i + 1
		name := categoryAccountCell(row, headerIndex["name"])
		account := categoryAccountCell(row, headerIndex["account"])
		if name == "" && account == "" {
			continue
		}
		if name == "" {
			issues = append(issues, parser.ValidationIssue{
				Row: rowNum, Field: "name", Message: "name kosong",
			})
			continue
		}
		if account == "" {
			issues = append(issues, parser.ValidationIssue{
				Row: rowNum, Field: "account", Message: "account kosong",
			})
			continue
		}
		records = append(records, parser.TaxAccountRecord{
			Name:    name,
			Account: account,
		})
	}

	return records, issues, nil
}

func (s *CategoryAccountService) TempFilePath(profileID string) string {
	return profilepath.CashflowCategoryAccountsUploadTempXLSX(s.rootDir, profileID)
}

func (s *CategoryAccountService) IsAcceptedUpload(filename string, sizeBytes int64) (bool, string) {
	spec := s.Spec()
	ext := strings.ToLower(strings.TrimSpace(filepathExt(filename)))
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

func normalizeCategoryLookupKey(raw string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(raw))), " ")
}

func resolveCategoryAccountHeaderIndexes(headerRow []string, required []string) (map[string]int, []string) {
	normalized := make(map[string]int, len(headerRow))
	for idx, value := range headerRow {
		key := normalizeCategoryLookupKey(value)
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
		key := normalizeCategoryLookupKey(column)
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

func categoryAccountCell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func requiredCategoryColumns(spec parser.TaxAccountSchemaSpec) []string {
	columns := make([]string, 0, len(spec.Columns))
	for _, column := range spec.Columns {
		if column.Required {
			columns = append(columns, column.Header)
		}
	}
	sort.Strings(columns)
	return columns
}

func filepathExt(filename string) string {
	dot := strings.LastIndex(filename, ".")
	if dot < 0 {
		return ""
	}
	return filename[dot:]
}
