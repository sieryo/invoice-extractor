package storage

import (
	"encoding/csv"
	"os"
)

type TaxAccountCSVStore struct {
	path string
}

type TaxAccountCSVRecord struct {
	Name    string
	Account string
}

func NewTaxAccountCSVStore(path string) *TaxAccountCSVStore {
	return &TaxAccountCSVStore{path: path}
}

func (s *TaxAccountCSVStore) Save(accounts []TaxAccountCSVRecord) error {
	tmpPath := s.path + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)
	_ = w.Write([]string{"name", "account"})
	for _, item := range accounts {
		_ = w.Write([]string{item.Name, item.Account})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.path)
}
