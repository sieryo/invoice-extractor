package storage

import (
	"encoding/csv"
	"os"
)

type FPCoretaxRelationCSVRecord struct {
	Name    string
	Account string
}

type FPCoretaxRelationCSVStore struct {
	path string
}

func NewFPCoretaxRelationCSVStore(path string) *FPCoretaxRelationCSVStore {
	return &FPCoretaxRelationCSVStore{path: path}
}

func (s *FPCoretaxRelationCSVStore) Save(rows []FPCoretaxRelationCSVRecord) error {
	tmpPath := s.path + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)
	_ = w.Write([]string{"name", "account"})
	for _, item := range rows {
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
