package extract

import (
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
)

type BuyerAudit struct {
	ParsedName  string `json:"parsed_name,omitempty"`
	ParsedTaxID string `json:"parsed_tax_id,omitempty"`
	ParsedTKU   string `json:"parsed_tku,omitempty"`

	Enriched      bool   `json:"enriched"`
	RegistryName  string `json:"registry_name,omitempty"`
	RegistryTaxID string `json:"registry_tax_id,omitempty"`
	RegistryTKU   string `json:"registry_tku,omitempty"`

	AppliedTaxID string `json:"applied_tax_id,omitempty"`
	AppliedTKU   string `json:"applied_tku,omitempty"`

	TaxIDKind  string `json:"tax_id_kind,omitempty"`
	TaxIDValid bool   `json:"tax_id_valid"`
	TKUValid   bool   `json:"tku_valid"`
}

type InvoiceAudit struct {
	SourceFile     invoice.FileRef `json:"source_file"`
	TemplateID     string          `json:"template_id"`
	TemplateName   string          `json:"template_name"`
	NormalizedText string          `json:"normalized_text"`
	Warnings       []string        `json:"warnings,omitempty"`
	Buyer          BuyerAudit      `json:"buyer"`
	ExtractedAt    time.Time       `json:"extracted_at"`
}
