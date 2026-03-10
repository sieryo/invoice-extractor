package rename

import (
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
)

type TaxInvoiceBuyerAudit struct {
	ParsedName  string `json:"parsed_name,omitempty"`
	ParsedTaxID string `json:"parsed_tax_id,omitempty"`
	Address     string `json:"address,omitempty"`

	TaxIDKind  string `json:"tax_id_kind,omitempty"`
	TaxIDValid bool   `json:"tax_id_valid"`
}

type TaxInvoiceAudit struct {
	SourceFile     invoice.FileRef      `json:"source_file"`
	Number         string               `json:"number,omitempty"`
	RenamedTo      string               `json:"renamed_to,omitempty"`
	NormalizedText string               `json:"normalized_text,omitempty"`
	Warnings       []string             `json:"warnings,omitempty"`
	Error          string               `json:"error,omitempty"`
	Buyer          TaxInvoiceBuyerAudit `json:"buyer"`
	ExtractedAt    time.Time            `json:"extracted_at"`
}
