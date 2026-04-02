package fpcoretax

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
	"github.com/sieryo/invoice-extractor/internal/profilepath"
)

type RelationRegistryStatus struct {
	Loaded  bool   `json:"loaded"`
	Count   int    `json:"count"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RelationRegistryService struct {
	rootDir string
	parser  *parser.FPCoretaxRelationExcelParser
}

func NewRelationRegistryService(rootDir string) *RelationRegistryService {
	return &RelationRegistryService{
		rootDir: rootDir,
		parser:  parser.NewFPCoretaxRelationExcelParser(),
	}
}

func (s *RelationRegistryService) Spec(key RelationRegistryKey) parser.FPCoretaxRelationSchemaSpec {
	return s.parser.SchemaSpec(parser.FPCoretaxRelationRegistryKey(key))
}

func (s *RelationRegistryService) Status(profileID string, key RelationRegistryKey) RelationRegistryStatus {
	items, err := s.Load(profileID, key)
	if err != nil {
		if os.IsNotExist(err) {
			return RelationRegistryStatus{
				Loaded:  false,
				Count:   0,
				Code:    registryCode(key, "NOT_READY"),
				Message: registryMessage(key, "belum tersedia. Upload file registry terlebih dahulu."),
			}
		}
		var schemaErr *parser.FPCoretaxRelationSchemaMismatchError
		if errors.As(err, &schemaErr) {
			return RelationRegistryStatus{
				Loaded:  false,
				Count:   0,
				Code:    registryCode(key, "INVALID_SCHEMA"),
				Message: fmt.Sprintf("Schema registry tidak sesuai. Kolom wajib: %s.", strings.Join(requiredRegistryColumns(s.Spec(key)), ", ")),
			}
		}
		return RelationRegistryStatus{
			Loaded:  false,
			Count:   0,
			Code:    registryCode(key, "UNAVAILABLE"),
			Message: registryMessage(key, "tidak dapat dibaca saat ini."),
		}
	}
	if len(items) == 0 {
		return RelationRegistryStatus{
			Loaded:  false,
			Count:   0,
			Code:    registryCode(key, "EMPTY"),
			Message: registryMessage(key, "kosong. Upload file registry yang valid."),
		}
	}
	return RelationRegistryStatus{
		Loaded:  true,
		Count:   len(items),
		Code:    registryCode(key, "READY"),
		Message: registryReadyMessage(key),
	}
}

func (s *RelationRegistryService) List(profileID string, key RelationRegistryKey) ([]RelationRecord, error) {
	items, err := s.Load(profileID, key)
	if err != nil {
		return nil, err
	}
	out := make([]RelationRecord, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

func (s *RelationRegistryService) Load(profileID string, key RelationRegistryKey) (map[string]RelationRecord, error) {
	path := s.dataPath(profileID, key)
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
		return nil, &parser.FPCoretaxRelationSchemaMismatchError{MissingColumns: requiredRegistryColumns(s.Spec(key))}
	}

	headerIndex, missing := resolveRegistryHeaderIndexes(rows[0], []string{"name", "account"})
	if len(missing) > 0 {
		return nil, &parser.FPCoretaxRelationSchemaMismatchError{MissingColumns: missing}
	}

	records := make(map[string]RelationRecord)
	for _, row := range rows[1:] {
		name := registryCell(row, headerIndex["name"])
		account := registryCell(row, headerIndex["account"])
		if name == "" {
			continue
		}
		record := RelationRecord{Name: name, Account: account}
		records[normalizeRegistryLookupKey(name)] = record
	}
	return records, nil
}

func (s *RelationRegistryService) Update(profileID string, key RelationRegistryKey, filePath string) (int, []parser.ValidationIssue, error) {
	items, issues, err := s.parser.Parse(parser.FPCoretaxRelationRegistryKey(key), filePath)
	if err != nil {
		return 0, nil, err
	}
	if err := os.MkdirAll(profilepath.ProfileDir(s.rootDir, profileID), 0o755); err != nil {
		return 0, nil, err
	}

	rows := make([]storage.FPCoretaxRelationCSVRecord, 0, len(items))
	for _, item := range items {
		rows = append(rows, storage.FPCoretaxRelationCSVRecord{
			Name:    item.Name,
			Account: item.Account,
		})
	}
	store := storage.NewFPCoretaxRelationCSVStore(s.dataPath(profileID, key))
	if err := store.Save(rows); err != nil {
		return 0, nil, err
	}
	return len(items), issues, nil
}

func (s *RelationRegistryService) TempFilePath(profileID string, key RelationRegistryKey) string {
	switch key {
	case RelationRegistrySupplier:
		return profilepath.FPCoretaxSupplierUploadTempXLSX(s.rootDir, profileID)
	default:
		return profilepath.FPCoretaxCustomerUploadTempXLSX(s.rootDir, profileID)
	}
}

func (s *RelationRegistryService) IsAcceptedUpload(key RelationRegistryKey, filename string, sizeBytes int64) (bool, string) {
	spec := s.Spec(key)
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

func (s *RelationRegistryService) dataPath(profileID string, key RelationRegistryKey) string {
	switch key {
	case RelationRegistrySupplier:
		return profilepath.FPCoretaxSupplierCSV(s.rootDir, profileID)
	default:
		return profilepath.FPCoretaxCustomerCSV(s.rootDir, profileID)
	}
}

func normalizeRegistryLookupKey(raw string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(raw))), " ")
}

func resolveRegistryHeaderIndexes(headerRow []string, required []string) (map[string]int, []string) {
	normalized := make(map[string]int, len(headerRow))
	for idx, value := range headerRow {
		key := normalizeRegistryLookupKey(value)
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
		key := normalizeRegistryLookupKey(column)
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

func registryCell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func requiredRegistryColumns(spec parser.FPCoretaxRelationSchemaSpec) []string {
	out := make([]string, 0, len(spec.Columns))
	for _, item := range spec.Columns {
		if !item.Required {
			continue
		}
		out = append(out, item.Header)
	}
	sort.Strings(out)
	return out
}

func registryCode(key RelationRegistryKey, suffix string) string {
	prefix := "FP_KELUARAN_CUSTOMER"
	if key == RelationRegistrySupplier {
		prefix = "FP_MASUKAN_SUPPLIER"
	}
	return prefix + "_" + suffix
}

func registryMessage(key RelationRegistryKey, suffix string) string {
	switch key {
	case RelationRegistrySupplier:
		return "Supplier registry FP Masukan " + suffix
	default:
		return "Customer registry FP Keluaran " + suffix
	}
}

func registryReadyMessage(key RelationRegistryKey) string {
	switch key {
	case RelationRegistrySupplier:
		return "Supplier registry FP Masukan siap digunakan."
	default:
		return "Customer registry FP Keluaran siap digunakan."
	}
}
