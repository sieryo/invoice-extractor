package excel

const (
	SheetInvoice = "Faktur"

	ColInvSpecialNPWP = "C" // NPWP PENJUAL

	InvoiceHeaderRow = 3
	InvoiceStartRow  = 4

	ColInvRow              = "A"
	ColInvDate             = "B"
	ColInvType             = "C"
	ColInvTransactionCode  = "D"
	ColInvAdditionalInfo   = "E"
	ColInvSupportDoc       = "F"
	ColInvSupportDocPeriod = "G"
	ColInvReference        = "H"
	ColInvFacilityCap      = "I"
	ColInvSellerTKU        = "J"
	ColInvBuyerID          = "K"
	ColInvBuyerIDType      = "L"
	ColInvBuyerCountry     = "M"
	ColInvBuyerDocNumber   = "N"
	ColInvBuyerName        = "O"
	ColInvBuyerAddress     = "P"
	ColInvBuyerEmail       = "Q"
	ColInvBuyerTKU         = "R"

	ColInvSpecialFlagEnd = "A"
)

const (
	SheetInvoiceDetail = "DetailFaktur"

	DetailHeaderRow = 1
	DetailStartRow  = 2

	ColDetRow          = "A"
	ColDetType         = "B"
	ColDetCode         = "C"
	ColDetName         = "D"
	ColDetUnit         = "E"
	ColDetUnitPrice    = "F"
	ColDetQty          = "G"
	ColDetDiscount     = "H"
	ColDetDPP          = "I"
	ColDetTaxBase      = "J"
	ColDetTaxRate      = "K"
	ColDetTaxAmount    = "L"
	ColDetLuxuryRate   = "M"
	ColDetLuxuryAmount = "N"

	ColDetSpecialFlagEnd = "A"
)
