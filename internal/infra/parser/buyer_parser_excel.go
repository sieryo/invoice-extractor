package parser

import (
	"github.com/sieryo/invoice-extractor/internal/domain/buyer"
	"github.com/xuri/excelize/v2"
)

type BuyerExcelParser struct{}

func NewBuyerExcelParser() *BuyerExcelParser {
	return &BuyerExcelParser{}
}

func (p *BuyerExcelParser) Parse(path string) ([]buyer.Buyer, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}

	var buyers []buyer.Buyer

	for i, row := range rows {
		if i == 0 {
			continue // header
		}

		if len(row) < 3 {
			continue
		}

		b := buyer.Buyer{
			Name:    cell(row, 0),
			NPWP15:  cell(row, 1),
			NPWP16:  cell(row, 2),
			NITKU:   cell(row, 3),
			Email:   cell(row, 4),
			Address: cell(row, 5),
		}

		if b.Name == "" || (b.NPWP15 == "" && b.NPWP16 == "" && b.NITKU == "") {
			continue
		}

		buyers = append(buyers, b)
	}

	return buyers, nil
}

func cell(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return row[idx]
}
