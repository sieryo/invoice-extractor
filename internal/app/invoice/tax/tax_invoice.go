package tax

import (
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
)

type DPPMethod string

const (
	DPPMethodUnknown DPPMethod = "UNKNOWN"
	DPPMethodA       DPPMethod = "A"
	DPPMethodB       DPPMethod = "B"
)

type DPPVerificationStatus string

const (
	DPPVerificationOK      DPPVerificationStatus = "OK"
	DPPVerificationWarning DPPVerificationStatus = "WARNING"
	DPPVerificationError   DPPVerificationStatus = "ERROR"
)

type DPPVerification struct {
	Status     DPPVerificationStatus `json:"status"`
	Threshold  float64               `json:"threshold"`
	BaseAmount float64               `json:"base_amount"`
	ExpectedA  float64               `json:"expected_a"`
	ExpectedB  float64               `json:"expected_b"`
	DiffA      float64               `json:"diff_a"`
	DiffB      float64               `json:"diff_b"`
	Message    string                `json:"message,omitempty"`
}

type TaxInvoice struct {
	SourceFile          string           `json:"source_file"`
	InvoiceNumber       string           `json:"invoice_number"`
	InvoiceDate         time.Time        `json:"invoice_date"`
	SellerName          string           `json:"seller_name"`
	SellerNPWP          string           `json:"seller_npwp"`
	BuyerName           string           `json:"buyer_name"`
	BuyerNPWP           string           `json:"buyer_npwp"`
	References          string           `json:"references"`
	DownPaymentReceived float64          `json:"down_payment_received"`
	IncludeInExport     bool             `json:"include_in_export"`
	ExclusionReason     string           `json:"exclusion_reason,omitempty"`
	DPPMethod           DPPMethod        `json:"dpp_method"`
	DPPVerification     DPPVerification  `json:"dpp_verification"`
	DPP                 float64          `json:"dpp"`
	PPN                 float64          `json:"ppn"`
	Total               float64          `json:"total"`
	Currency            string           `json:"currency"`
	Items               []TaxInvoiceItem `json:"items,omitempty"`
	Anomalies           []string         `json:"anomalies,omitempty"`

	// Legacy compatibility for current rename and warning flow.
	Number string         `json:"-"`
	Buyer  *invoice.Party `json:"-"`
}

type TaxInvoiceItem struct {
	LineNo      int     `json:"line_no"`
	ItemCode    string  `json:"item_code"`
	Description string  `json:"description"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	UnitPrice   float64 `json:"unit_price"`
	LineTotal   float64 `json:"line_total"`
}
