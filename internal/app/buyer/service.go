package buyer

import (
	"path/filepath"

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

func (s *BuyerRegistryService) Update(filePath string) (int, error) {
	buyers, err := s.parser.Parse(filePath)
	if err != nil {
		return 0, err
	}

	if err := s.store.Save(buyers); err != nil {
		return 0, err
	}

	s.registry.Load(buyers)
	s.registry.loaded = true

	return len(buyers), nil
}

func (s *BuyerRegistryService) IsLoaded() bool {
	return s.registry.IsLoaded()
}

func (s *BuyerRegistryService) DataDir() string {
	return s.dataDir
}

func (s *BuyerRegistryService) TempFilePath() string {
	return filepath.Join(s.dataDir, "buyer_upload.xlsx")
}
