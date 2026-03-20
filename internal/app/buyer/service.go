package buyer

import (
	"path/filepath"
	"strings"

	domainbuyer "github.com/sieryo/invoice-extractor/internal/domain/buyer"
	"github.com/sieryo/invoice-extractor/internal/infra/parser"
	"github.com/sieryo/invoice-extractor/internal/infra/storage"
)

type BuyerRegistryService struct {
	registry *Registry
	parser   *parser.BuyerExcelParser
	store    *storage.BuyerCSVStore
	dataDir  string
}

type BuyerRegistryStatus struct {
	Loaded  bool   `json:"loaded"`
	Count   int    `json:"count"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewBuyerRegistryService(
	registry *Registry,
	store *storage.BuyerCSVStore,
	dataDir string,
) *BuyerRegistryService {
	return &BuyerRegistryService{
		registry: registry,
		parser:   parser.NewBuyerExcelParser(),
		store:    store,
		dataDir:  dataDir,
	}
}

func (s *BuyerRegistryService) List() []domainbuyer.Buyer {
	s.registry.mu.RLock()
	defer s.registry.mu.RUnlock()

	if !s.registry.loaded {
		return nil
	}

	buyers := make([]domainbuyer.Buyer, 0, len(s.registry.buyers))
	for _, nb := range s.registry.buyers {
		buyers = append(buyers, nb.Raw)
	}

	return buyers
}

func (s *BuyerRegistryService) Spec() parser.BuyerRegistrySchemaSpec {
	return s.parser.SchemaSpec()
}

func (s *BuyerRegistryService) Status() BuyerRegistryStatus {
	loaded := s.registry.IsLoaded()
	count := s.Count()
	if !loaded {
		return BuyerRegistryStatus{
			Loaded:  false,
			Count:   count,
			Code:    "BUYER_REGISTRY_NOT_READY",
			Message: "Buyer registry belum tersedia. Upload file buyer terlebih dahulu.",
		}
	}
	if count <= 0 {
		return BuyerRegistryStatus{
			Loaded:  false,
			Count:   count,
			Code:    "BUYER_REGISTRY_EMPTY",
			Message: "Buyer registry kosong. Upload file buyer yang valid.",
		}
	}
	return BuyerRegistryStatus{
		Loaded:  true,
		Count:   count,
		Code:    "BUYER_REGISTRY_READY",
		Message: "Buyer registry siap digunakan.",
	}
}

func (s *BuyerRegistryService) Update(filePath string) (int, []parser.ValidationIssue, error) {
	buyers, issues, err := s.parser.Parse(filePath)
	if err != nil {
		return 0, nil, err
	}

	if err := s.store.Save(buyers); err != nil {
		return 0, nil, err
	}

	s.registry.Load(buyers)
	s.registry.loaded = true

	return len(buyers), issues, nil
}

func (s *BuyerRegistryService) IsLoaded() bool {
	status := s.Status()
	return status.Loaded
}

func (s *BuyerRegistryService) Count() int {
	s.registry.mu.RLock()
	defer s.registry.mu.RUnlock()
	return len(s.registry.buyers)
}

func (s *BuyerRegistryService) DataDir() string {
	return s.dataDir
}

func (s *BuyerRegistryService) TempFilePath() string {
	return filepath.Join(s.dataDir, "buyer_upload.xlsx")
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
