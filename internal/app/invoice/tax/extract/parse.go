package extract

import "github.com/sieryo/invoice-extractor/internal/app/invoice/tax"

func ParseTaxInvoiceText(fileName string, raw string) (*tax.TaxInvoice, string, error) {
	parsed, anomalies, err := parseCoretaxFromText(fileName, raw)
	normalized := normalizeTaxInvoiceText(raw)
	if err != nil {
		return nil, normalized, err
	}
	parsed.Anomalies = append([]string(nil), anomalies...)
	return &parsed, normalized, nil
}
