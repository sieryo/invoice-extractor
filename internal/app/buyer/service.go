package buyer

import (
	"os"
	"path/filepath"
	"strings"

	domainbuyer "github.com/sieryo/invoice-extractor/internal/domain/buyer"
	"github.com/sieryo/invoice-extractor/internal/infra/parser"
	"github.com/sieryo/invoice-extractor/internal/infra/storage"
	"github.com/sieryo/invoice-extractor/internal/profilepath"
)

type BuyerRegistryService struct {
	parser  *parser.BuyerExcelParser
	rootDir string
}

type BuyerRegistryStatus struct {
	Loaded  bool   `json:"loaded"`
	Count   int    `json:"count"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewBuyerRegistryService(rootDir string) *BuyerRegistryService {
	return &BuyerRegistryService{
		parser:  parser.NewBuyerExcelParser(),
		rootDir: rootDir,
	}
}

func (s *BuyerRegistryService) List(profileID string) ([]domainbuyer.Buyer, error) {
	return s.load(profileID)
}

func (s *BuyerRegistryService) Lookup(profileID string, name string) (domainbuyer.Buyer, bool, error) {
	buyers, err := s.load(profileID)
	if err != nil {
		return domainbuyer.Buyer{}, false, err
	}

	registry := NewRegistry()
	registry.Load(buyers)
	record, ok := registry.GetByName(name)
	return record, ok, nil
}

func (s *BuyerRegistryService) Spec() parser.BuyerRegistrySchemaSpec {
	return s.parser.SchemaSpec()
}

func (s *BuyerRegistryService) Status(profileID string) BuyerRegistryStatus {
	buyers, err := s.load(profileID)
	if err != nil {
		if os.IsNotExist(err) {
			return BuyerRegistryStatus{
				Loaded:  false,
				Count:   0,
				Code:    "BUYER_REGISTRY_NOT_READY",
				Message: "Buyer registry belum tersedia. Upload file buyer terlebih dahulu.",
			}
		}
		return BuyerRegistryStatus{
			Loaded:  false,
			Count:   0,
			Code:    "BUYER_REGISTRY_UNAVAILABLE",
			Message: "Buyer registry tidak dapat dibaca saat ini.",
		}
	}
	if len(buyers) == 0 {
		return BuyerRegistryStatus{
			Loaded:  false,
			Count:   0,
			Code:    "BUYER_REGISTRY_EMPTY",
			Message: "Buyer registry kosong. Upload file buyer yang valid.",
		}
	}
	return BuyerRegistryStatus{
		Loaded:  true,
		Count:   len(buyers),
		Code:    "BUYER_REGISTRY_READY",
		Message: "Buyer registry siap digunakan.",
	}
}

func (s *BuyerRegistryService) Update(profileID string, filePath string) (int, []parser.ValidationIssue, error) {
	buyers, issues, err := s.parser.Parse(filePath)
	if err != nil {
		return 0, nil, err
	}

	if err := os.MkdirAll(profilepath.ProfileDir(s.rootDir, profileID), 0o755); err != nil {
		return 0, nil, err
	}

	store := storage.NewBuyerCSVStore(profilepath.BuyersCSV(s.rootDir, profileID))
	if err := store.Save(buyers); err != nil {
		return 0, nil, err
	}

	return len(buyers), issues, nil
}

func (s *BuyerRegistryService) DataDir(profileID string) string {
	return profilepath.ProfileDir(s.rootDir, profileID)
}

func (s *BuyerRegistryService) TempFilePath(profileID string) string {
	return profilepath.BuyerUploadTempXLSX(s.rootDir, profileID)
}

func (s *BuyerRegistryService) IsAcceptedUpload(filename string, sizeBytes int64) (bool, string) {
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

func (s *BuyerRegistryService) load(profileID string) ([]domainbuyer.Buyer, error) {
	store := storage.NewBuyerCSVStore(profilepath.BuyersCSV(s.rootDir, profileID))
	return store.Load()
}
