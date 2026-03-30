package configmodule

import (
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/document"
)

type ModuleKind string

const (
	ModuleKindRegistry       ModuleKind = "registry"
	ModuleKindDefaultProfile ModuleKind = "default_profile"
)

type ModuleSpec struct {
	Key             string   `json:"key"`
	RouteKey        string   `json:"routeKey"`
	Label           string   `json:"label"`
	Description     string   `json:"description,omitempty"`
	Kind            string   `json:"kind"`
	Group           string   `json:"group"`
	GroupLabel      string   `json:"groupLabel"`
	IconKey         string   `json:"iconKey"`
	CollectionKinds []string `json:"collectionKinds,omitempty"`
}

type ModuleDetailSpec struct {
	ModuleSpec
	LayoutMode string `json:"layoutMode,omitempty"`
}

func ListModules(enableCashflowXLSX bool) []ModuleSpec {
	items := []ModuleSpec{
		{
			Key:             "buyer_registry",
			RouteKey:        "buyer_registry",
			Label:           "Buyer Registry",
			Description:     "Sinkronisasi data buyer untuk enrich data invoice.",
			Kind:            string(ModuleKindRegistry),
			Group:           "invoice",
			GroupLabel:      "Invoice",
			IconKey:         "users",
			CollectionKinds: []string{string(document.CollectionKindInvoiceCompany)},
		},
		{
			Key:             "template_registry",
			RouteKey:        "template_registry",
			Label:           "Template Registry",
			Description:     "Template parser invoice yang tersedia saat ini.",
			Kind:            string(ModuleKindRegistry),
			Group:           "invoice",
			GroupLabel:      "Invoice",
			IconKey:         "file-code",
			CollectionKinds: []string{string(document.CollectionKindInvoiceCompany)},
		},
		{
			Key:             "bukpot_request_gst_deduction_mt",
			RouteKey:        "bukpot_request_gst_deduction_mt",
			Label:           "Default Profil GST Deduction MT",
			Description:     "Mapping default untuk action request bukpot GST Deduction MT.",
			Kind:            string(ModuleKindDefaultProfile),
			Group:           "bukpot",
			GroupLabel:      "Bukpot",
			IconKey:         "receipt-text",
			CollectionKinds: []string{string(document.CollectionKindBukpotRequestGSTDeductionMT)},
		},
	}

	if enableCashflowXLSX {
		items = append(items, ModuleSpec{
			Key:             "cashflow_tax_accounts",
			RouteKey:        "cashflow_tax_accounts",
			Label:           "Tax Accounts",
			Description:     "Lookup account untuk action export cashflow MYOB.",
			Kind:            string(ModuleKindRegistry),
			Group:           "cashflow",
			GroupLabel:      "Cashflow",
			IconKey:         "landmark",
			CollectionKinds: []string{string(document.CollectionKindCashflowImport)},
		})
	}

	return items
}

func FindModule(key string, enableCashflowXLSX bool) (ModuleDetailSpec, bool) {
	target := strings.TrimSpace(key)
	if target == "" {
		return ModuleDetailSpec{}, false
	}

	for _, item := range ListModules(enableCashflowXLSX) {
		if strings.EqualFold(item.Key, target) || strings.EqualFold(item.RouteKey, target) {
			return ModuleDetailSpec{
				ModuleSpec: item,
				LayoutMode: "page",
			}, true
		}
	}

	return ModuleDetailSpec{}, false
}
