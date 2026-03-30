package taxcatalog

import "strings"

type CanonicalTax struct {
	Key   string
	Name  string
	Group string
}

var canonicalTaxCatalog = []CanonicalTax{
	{Key: "ppn", Name: "PPN", Group: "TAX"},
	{Key: "pph15", Name: "PPH 15%", Group: "TAX"},
	{Key: "pph23", Name: "PPH 23", Group: "TAX"},
	{Key: "pph21", Name: "PPH 21", Group: "TAX"},
	{Key: "pph25", Name: "PPH 25", Group: "TAX"},
	{Key: "pph42", Name: "PPH 4 (2)", Group: "TAX"},
	{Key: "pp23", Name: "PP 23", Group: "TAX"},
}

func CanonicalTaxes() []CanonicalTax {
	out := make([]CanonicalTax, len(canonicalTaxCatalog))
	copy(out, canonicalTaxCatalog)
	return out
}

func CanonicalTaxNames() []string {
	out := make([]string, 0, len(canonicalTaxCatalog))
	for _, item := range canonicalTaxCatalog {
		out = append(out, item.Name)
	}
	return out
}

func ResolveCanonicalTaxName(key string) string {
	target := strings.TrimSpace(strings.ToLower(key))
	for _, item := range canonicalTaxCatalog {
		if strings.EqualFold(item.Key, target) {
			return item.Name
		}
	}
	return ""
}

func IsCanonicalTaxName(name string) bool {
	normalized := normalizeTaxLookupKey(name)
	for _, item := range canonicalTaxCatalog {
		if normalizeTaxLookupKey(item.Name) == normalized {
			return true
		}
	}
	return false
}

func normalizeTaxLookupKey(raw string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(raw))), " ")
}
