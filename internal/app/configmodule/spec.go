package configmodule

import (
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/document"
)

type ModuleKind string

const (
	ModuleKindRegistry       ModuleKind = "registry"
	ModuleKindDefaultProfile ModuleKind = "default_profile"
	ModuleKindSettings       ModuleKind = "settings"
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
			Key:         "app_modules",
			RouteKey:    "app_modules",
			Label:       "Modul Aplikasi",
			Description: "Aktifkan atau nonaktifkan modul fitur aplikasi yang tersedia saat ini.",
			Kind:        string(ModuleKindSettings),
			Group:       "application",
			GroupLabel:  "Aplikasi",
			IconKey:     "sliders-horizontal",
		},
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
			Label:           "Default Profil Request Bukpot GST Deduction MT",
			Description:     "Mapping default untuk action Request Bukpot GST Deduction MT.",
			Kind:            string(ModuleKindDefaultProfile),
			Group:           "request_bukpot",
			GroupLabel:      "Request Bukpot",
			IconKey:         "receipt-text",
			CollectionKinds: []string{string(document.CollectionKindBukpotRequestGSTDeductionMT)},
		},
	}

	if enableCashflowXLSX {
		items = append(items,
			ModuleSpec{
				Key:             "cashflow_spend_money",
				RouteKey:        "cashflow_spend_money",
				Label:           "Default Profil Cashflow Spend Money",
				Description:     "Nilai default untuk action export cashflow ke MYOB Spend Money.",
				Kind:            string(ModuleKindDefaultProfile),
				Group:           "cashflow",
				GroupLabel:      "Cashflow",
				IconKey:         "landmark",
				CollectionKinds: []string{string(document.CollectionKindCashflowImport)},
			},
			ModuleSpec{
				Key:             "cashflow_receive_money",
				RouteKey:        "cashflow_receive_money",
				Label:           "Default Profil Cashflow Receive Money",
				Description:     "Nilai default untuk action export cashflow ke MYOB Receive Money.",
				Kind:            string(ModuleKindDefaultProfile),
				Group:           "cashflow",
				GroupLabel:      "Cashflow",
				IconKey:         "landmark",
				CollectionKinds: []string{string(document.CollectionKindCashflowImport)},
			},
			ModuleSpec{
				Key:             "cashflow_pay_bills",
				RouteKey:        "cashflow_pay_bills",
				Label:           "Default Profil Cashflow Pay Bills",
				Description:     "Nilai default untuk action cashflow ke MYOB Pay Bills.",
				Kind:            string(ModuleKindDefaultProfile),
				Group:           "cashflow",
				GroupLabel:      "Cashflow",
				IconKey:         "landmark",
				CollectionKinds: []string{string(document.CollectionKindCashflowImport)},
			},
			ModuleSpec{
				Key:             "cashflow_receive_payments",
				RouteKey:        "cashflow_receive_payments",
				Label:           "Default Profil Cashflow Receive Payments",
				Description:     "Nilai default untuk action cashflow ke MYOB Receive Payments.",
				Kind:            string(ModuleKindDefaultProfile),
				Group:           "cashflow",
				GroupLabel:      "Cashflow",
				IconKey:         "landmark",
				CollectionKinds: []string{string(document.CollectionKindCashflowImport)},
			},
			ModuleSpec{
				Key:             "cashflow_tax_accounts",
				RouteKey:        "cashflow_tax_accounts",
				Label:           "Tax Accounts",
				Description:     "Lookup account untuk action export cashflow MYOB.",
				Kind:            string(ModuleKindRegistry),
				Group:           "cashflow",
				GroupLabel:      "Cashflow",
				IconKey:         "landmark",
				CollectionKinds: []string{string(document.CollectionKindCashflowImport)},
			},
			ModuleSpec{
				Key:             "cashflow_category_accounts",
				RouteKey:        "cashflow_category_accounts",
				Label:           "Category Accounts",
				Description:     "Lookup category untuk action cashflow bills MYOB.",
				Kind:            string(ModuleKindRegistry),
				Group:           "cashflow",
				GroupLabel:      "Cashflow",
				IconKey:         "landmark",
				CollectionKinds: []string{string(document.CollectionKindCashflowImport)},
			},
		)
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
