package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

func writeInvoices(
	f *excelize.File,
	invoices []*invoice.Invoice,
	templateRegistry *template.Registry,
) error {

	if len(invoices) == 0 {
		return fmt.Errorf("Empty Invoices")
	}

	boldStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
	})
	if err != nil {
		return err
	}

	invoiceSheet := SheetInvoice
	detailSheet := SheetInvoiceDetail

	invoiceRow := InvoiceStartRow
	detailRow := DetailStartRow
	baris := 1

	// Anggap semua invoice itu sellernya sama
	sellerNPWP := invoices[0].Seller.TaxID

	f.SetCellValue(invoiceSheet, cell(ColInvSpecialNPWP, 1), *sellerNPWP)

	for _, inv := range invoices {

		tmpl, ok := templateRegistry.GetByIdentifier(inv.Metadata.TemplateID)

		var invoiceNumber string
		if !ok {
			// Ini anggep aja pake invoice number langsung.
			invoiceNumber = inv.Number
		} else {
			invoiceNumber = tmpl.FormatInvoiceNumber(inv)
		}

		f.SetCellValue(invoiceSheet, cell(ColInvRow, invoiceRow), baris)

		dateStr := helper.FormatDateDDMMYYYY(inv.Date)
		if dateStr != "" {
			f.SetCellValue(
				invoiceSheet,
				cell(ColInvDate, invoiceRow),
				dateStr,
			)
		}

		f.SetCellValue(invoiceSheet, cell(ColInvType, invoiceRow), "Normal")
		f.SetCellValue(invoiceSheet, cell(ColInvTransactionCode, invoiceRow), "04")

		f.SetCellValue(invoiceSheet, cell(ColInvReference, invoiceRow), invoiceNumber)

		if inv.Seller != nil {

			if inv.Seller.TKU != nil {
				f.SetCellValue(invoiceSheet, cell(ColInvSellerTKU, invoiceRow), *inv.Seller.TKU)
			}
		}

		f.SetCellValue(invoiceSheet, cell(ColInvBuyerIDType, invoiceRow), "TIN")
		f.SetCellValue(invoiceSheet, cell(ColInvBuyerCountry, invoiceRow), "IDN")
		f.SetCellValue(invoiceSheet, cell(ColInvBuyerDocNumber, invoiceRow), "-")

		if inv.Buyer != nil {
			f.SetCellValue(invoiceSheet, cell(ColInvBuyerName, invoiceRow), inv.Buyer.Name)

			if inv.Buyer.Address != nil {
				f.SetCellValue(invoiceSheet, cell(ColInvBuyerAddress, invoiceRow), *inv.Buyer.Address)
			}

			if inv.Buyer.TaxID != nil {
				f.SetCellValue(invoiceSheet, cell(ColInvBuyerID, invoiceRow), *inv.Buyer.TaxID)
			}

			if inv.Buyer.TKU != nil {
				f.SetCellValue(invoiceSheet, cell(ColInvBuyerTKU, invoiceRow), *inv.Buyer.TKU)
			}
		}

		for _, item := range inv.Items {

			f.SetCellValue(detailSheet, cell(ColDetRow, detailRow), baris)
			f.SetCellValue(detailSheet, cell(ColDetType, detailRow), "A")
			f.SetCellValue(detailSheet, cell(ColDetCode, detailRow), "000000")

			f.SetCellValue(detailSheet, cell(ColDetName, detailRow), item.GetExportedName())

			f.SetCellValue(detailSheet, cell(ColDetUnit, detailRow), "UM.0018")

			if item.UnitPrice != nil {
				f.SetCellValue(detailSheet, cell(ColDetUnitPrice, detailRow), item.UnitPrice.Amount)
			}

			f.SetCellValue(detailSheet, cell(ColDetQty, detailRow), item.Quantity)

			if item.Discount != nil {
				f.SetCellValue(detailSheet, cell(ColDetDiscount, detailRow), item.Discount.Amount)
			} else {
				f.SetCellValue(detailSheet, cell(ColDetDiscount, detailRow), 0)
			}

			if item.TotalAmount != nil {
				f.SetCellValue(detailSheet, cell(ColDetDPP, detailRow), item.TotalAmount.Amount)
				f.SetCellValue(detailSheet, cell(ColDetTaxBase, detailRow), item.GetTaxBase())

			}

			if item.TaxRate > 0 {
				f.SetCellValue(detailSheet, cell(ColDetTaxRate, detailRow), item.TaxRate*100)
			}

			f.SetCellValue(detailSheet, cell(ColDetTaxAmount, detailRow), item.GetTotalTax())

			f.SetCellValue(detailSheet, cell(ColDetLuxuryRate, detailRow), 0)
			f.SetCellValue(detailSheet, cell(ColDetLuxuryAmount, detailRow), 0)

			detailRow++
		}

		invoiceRow++
		baris++
	}

	// FLAG END
	f.SetCellValue(
		invoiceSheet,
		cell(ColInvSpecialFlagEnd, invoiceRow),
		"END",
	)
	f.SetCellStyle(
		invoiceSheet,
		cell(ColInvSpecialFlagEnd, invoiceRow),
		cell(ColInvSpecialFlagEnd, invoiceRow),
		boldStyle,
	)

	f.SetCellValue(
		detailSheet,
		cell(ColDetSpecialFlagEnd, detailRow),
		"END",
	)
	f.SetCellStyle(
		detailSheet,
		cell(ColDetSpecialFlagEnd, detailRow),
		cell(ColDetSpecialFlagEnd, detailRow),
		boldStyle,
	)

	return nil
}

func cell(col string, row int) string {
	return fmt.Sprintf("%s%d", col, row)
}
