package storage

import (
	"encoding/csv"
	"os"

	"github.com/sieryo/invoice-extractor/internal/domain/buyer"
)

type BuyerCSVStore struct {
	path string
}

func NewBuyerCSVStore(path string) *BuyerCSVStore {
	return &BuyerCSVStore{
		path: path,
	}
}

func (s *BuyerCSVStore) Save(buyers []buyer.Buyer) error {
	tmpPath := s.path + ".tmp"

	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)

	_ = w.Write([]string{
		"NAMA", "NPWP 15 DIGIT", "NPWP 16 DIGIT",
		"NITKU", "EMAIL", "ALAMAT",
	})

	for _, b := range buyers {
		_ = w.Write([]string{
			b.Name,
			b.NPWP15,
			b.NPWP16,
			b.NITKU,
			b.Email,
			b.Address,
		})
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

func (s *BuyerCSVStore) Load(path string) ([]buyer.Buyer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var buyers []buyer.Buyer
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) < 3 {
			continue
		}

		buyers = append(buyers, buyer.Buyer{
			Name:    row[0],
			NPWP15:  row[1],
			NPWP16:  row[2],
			NITKU:   row[3],
			Email:   row[4],
			Address: row[5],
		})
	}

	return buyers, nil
}
