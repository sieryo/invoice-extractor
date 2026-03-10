package parser

import (
	"fmt"

	"github.com/sieryo/invoice-extractor/internal/domain/buyer"
	"github.com/sieryo/invoice-extractor/pkg/helper"
	"github.com/xuri/excelize/v2"
)

type BuyerExcelParser struct{}

func NewBuyerExcelParser() *BuyerExcelParser {
	return &BuyerExcelParser{}
}

func (p *BuyerExcelParser) Parse(path string) ([]buyer.Buyer, []ValidationIssue, error) {
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

	var buyers []buyer.Buyer
	var issues []ValidationIssue

	for i, row := range rows {
		if i == 0 {
			continue // header
		}

		if len(row) < 3 {
			continue
		}

		rowNum := i + 1

		npwp15Raw := cell(row, 1)
		npwp16Raw := cell(row, 2)
		nitkuRaw := cell(row, 3)

		npwp15 := helper.DigitsOnly(npwp15Raw)
		npwp16 := helper.DigitsOnly(npwp16Raw)
		nitku := helper.DigitsOnly(nitkuRaw)

		if npwp15 != "" && !helper.IsNPWP15(npwp15) {
			issues = append(issues, ValidationIssue{
				Row:     rowNum,
				Field:   "npwp_15",
				Value:   npwp15Raw,
				Message: fmt.Sprintf("invalid length: expected 15 digits, got %d", len(npwp15)),
			})
			npwp15 = ""
		}
		if npwp16 != "" && !helper.IsNPWP16(npwp16) {
			issues = append(issues, ValidationIssue{
				Row:     rowNum,
				Field:   "npwp_16",
				Value:   npwp16Raw,
				Message: fmt.Sprintf("invalid length: expected 16 digits, got %d", len(npwp16)),
			})
			npwp16 = ""
		}
		if nitku != "" && !helper.IsNITKU(nitku) {
			issues = append(issues, ValidationIssue{
				Row:     rowNum,
				Field:   "nitku",
				Value:   nitkuRaw,
				Message: fmt.Sprintf("invalid length: expected 22 digits, got %d", len(nitku)),
			})
			nitku = ""
		}

		b := buyer.Buyer{
			Name:    cell(row, 0),
			NPWP15:  npwp15,
			NPWP16:  npwp16,
			NITKU:   nitku,
			Email:   cell(row, 4),
			Address: cell(row, 5),
		}

		if b.Name == "" || (b.NPWP15 == "" && b.NPWP16 == "" && b.NITKU == "") {
			continue
		}

		buyers = append(buyers, b)
	}

	return buyers, issues, nil
}

func cell(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return row[idx]
}
