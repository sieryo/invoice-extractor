package excel

const (
	SheetInvoice       = "Faktur"
	SheetInvoiceDetail = "DetailFaktur"

	// SPESIAL: NPWP PENJUAL
	CellSpesialNpwpPenjual = "C1"

	// Header
	CellInvoiceNumber = "B2"
	CellInvoiceDate   = "B3"
	CellCustomerName  = "B4"

	// Table
	TableStartRow = 10

	ColItemNo     = "A"
	ColItemName   = "B"
	ColItemQty    = "C"
	ColItemPrice  = "D"
	ColItemAmount = "E"

	// Summary
	CellSubTotal = "E20"
	CellTax      = "E21"
	CellTotal    = "E22"
)
