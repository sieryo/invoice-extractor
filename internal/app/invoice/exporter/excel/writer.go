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

	if len(invoices) == 0 {
		return fmt.Errorf("Empty Invoices")
	}

	invoiceSheet := SheetInvoice
	detailSheet := SheetInvoiceDetail

	invoiceRow := InvoiceStartRow
	detailRow := DetailStartRow
	baris := 1

	// Anggap semua invoice itu sellernya sama
	sellerNPWP := invoices[0].Seller.TaxID

	f.SetCellValue(invoiceSheet, cell(ColSpecialNPWP, 1), *sellerNPWP)

	for _, inv := range invoices {

		f.SetCellValue(invoiceSheet, cell(ColInvRow, invoiceRow), baris)

		if inv.Date != nil {
			f.SetCellValue(
				invoiceSheet,
				cell(ColInvDate, invoiceRow),
				inv.Date.Format("2006-01-02"),
			)
		}

		f.SetCellValue(invoiceSheet, cell(ColInvReference, invoiceRow), inv.Number)

		if inv.Buyer != nil {
			f.SetCellValue(invoiceSheet, cell(ColInvBuyerName, invoiceRow), inv.Buyer.Name)

			if inv.Buyer.Address != nil {
				f.SetCellValue(invoiceSheet, cell(ColInvBuyerAddress, invoiceRow), *inv.Buyer.Address)
			}

			if inv.Buyer.TaxID != nil {
				f.SetCellValue(invoiceSheet, cell(ColInvBuyerID, invoiceRow), *inv.Buyer.TaxID)
			}
		}

		for _, item := range inv.Items {

			f.SetCellValue(detailSheet, cell(ColDetRow, detailRow), baris)
			f.SetCellValue(detailSheet, cell(ColDetType, detailRow), "Barang")
			f.SetCellValue(detailSheet, cell(ColDetName, detailRow), item.Name)

			if item.UnitPrice != nil {
				f.SetCellValue(detailSheet, cell(ColDetUnitPrice, detailRow), item.UnitPrice.Amount)
			}

			f.SetCellValue(detailSheet, cell(ColDetQty, detailRow), item.Quantity)

			if item.TotalAmount != nil {
				f.SetCellValue(detailSheet, cell(ColDetDPP, detailRow), item.TotalAmount.Amount)
			}

			f.SetCellValue(detailSheet, cell(ColDetTaxRate, detailRow), 11)

			detailRow++
		}

		invoiceRow++
		baris++
	}

	return nil
}

func cell(col string, row int) string {
	return fmt.Sprintf("%s%d", col, row)
}
