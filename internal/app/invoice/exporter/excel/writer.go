package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
)

func writeInvoices(
	f *excelize.File,
	invoices []*invoice.Invoice,
) error {

	for i, inv := range invoices {
		sheet := SheetInvoice

		// kalau mau 1 invoice = 1 sheet
		if i > 0 {
			newSheet := fmt.Sprintf("%s_%d", SheetInvoice, i+1)
			f.NewSheet(newSheet)
			sheetIndex, err := f.GetSheetIndex(sheet)

			if err != nil {
				return err
			}

			newSheetIndex, err := f.GetSheetIndex(newSheet)

			if err != nil {
				return err
			}

			f.CopySheet(sheetIndex, newSheetIndex)
			sheet = newSheet
		}

		if err := writeHeader(f, sheet, inv); err != nil {
			return err
		}

		if err := writeItems(f, sheet, inv); err != nil {
			return err
		}

		if err := writeSummary(f, sheet, inv); err != nil {
			return err
		}
	}

	return nil
}

func writeHeader(
	f *excelize.File,
	sheet string,
	inv *invoice.Invoice,
) error {

	f.SetCellValue(sheet, CellInvoiceNumber, inv.Number)
	f.SetCellValue(sheet, CellInvoiceDate, inv.Date)
	f.SetCellValue(sheet, CellCustomerName, inv.Buyer.Name)

	return nil
}

func writeItems(
	f *excelize.File,
	sheet string,
	inv *invoice.Invoice,
) error {

	row := TableStartRow

	for i, item := range inv.Items {
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", ColItemNo, row), i+1)
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", ColItemName, row), item.Name)
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", ColItemQty, row), item.Quantity)
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", ColItemPrice, row), item.UnitPrice.Amount)
		f.SetCellValue(sheet, fmt.Sprintf("%s%d", ColItemAmount, row), item.TotalAmount.Amount)

		row++
	}

	return nil
}

func writeSummary(
	f *excelize.File,
	sheet string,
	inv *invoice.Invoice,
) error {

	f.SetCellValue(sheet, CellSubTotal, inv.Subtotal.Amount)
	f.SetCellValue(sheet, CellTax, inv.VAT.Amount)
	f.SetCellValue(sheet, CellTotal, inv.Total)

	return nil
}
