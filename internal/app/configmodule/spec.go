package configmodule

import (
	"strings"

	appbukpot "github.com/sieryo/invoice-extractor/internal/app/bukpot"
	"github.com/sieryo/invoice-extractor/internal/app/document"
	appfpcoretax "github.com/sieryo/invoice-extractor/internal/app/fpcoretax"
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
			Key:             string(appfpcoretax.ProfileConfigFPKeluaranMiscSales),
			RouteKey:        string(appfpcoretax.ProfileConfigFPKeluaranMiscSales),
			Label:           "FP Keluaran Misc Sales",
			Description:     "Parameter action Misc Sales.",
			Kind:            string(ModuleKindDefaultProfile),
			Group:           "tax_invoice_keluaran",
			GroupLabel:      "Faktur Pajak Keluaran",
			IconKey:         "receipt-text",
			CollectionKinds: []string{string(document.CollectionKindFPKeluaranCoretax)},
		},
		{
			Key:             string(appfpcoretax.ProfileConfigFPKeluaranReturMiscSales),
			RouteKey:        string(appfpcoretax.ProfileConfigFPKeluaranReturMiscSales),
			Label:           "FP Keluaran Retur Misc Sales",
			Description:     "Parameter action Misc Sales Retur.",
			Kind:            string(ModuleKindDefaultProfile),
			Group:           "tax_invoice_keluaran",
			GroupLabel:      "Faktur Pajak Keluaran",
			IconKey:         "receipt-text",
			CollectionKinds: []string{string(document.CollectionKindFPKeluaranReturCoretax)},
		},
		{
			Key:             "fp_keluaran_customer_registry",
			RouteKey:        "fp_keluaran_customer_registry",
			Label:           "Customer Registry",
			Description:     "Lookup customer MYOB.",
			Kind:            string(ModuleKindRegistry),
			Group:           "tax_invoice_keluaran",
			GroupLabel:      "Faktur Pajak Keluaran",
			IconKey:         "users",
			CollectionKinds: []string{string(document.CollectionKindFPKeluaranCoretax), string(document.CollectionKindFPKeluaranReturCoretax)},
		},
		{
			Key:             string(appfpcoretax.ProfileConfigFPMasukanMiscPurchases),
			RouteKey:        string(appfpcoretax.ProfileConfigFPMasukanMiscPurchases),
			Label:           "FP Masukan Misc Purchases",
			Description:     "Parameter action Misc Purchases.",
			Kind:            string(ModuleKindDefaultProfile),
			Group:           "tax_invoice_masukan",
			GroupLabel:      "Faktur Pajak Masukan",
			IconKey:         "receipt-text",
			CollectionKinds: []string{string(document.CollectionKindFPMasukanCoretax)},
		},
		{
			Key:             "fp_masukan_supplier_registry",
			RouteKey:        "fp_masukan_supplier_registry",
			Label:           "Supplier Registry",
			Description:     "Lookup supplier MYOB.",
			Kind:            string(ModuleKindRegistry),
			Group:           "tax_invoice_masukan",
			GroupLabel:      "Faktur Pajak Masukan",
			IconKey:         "users",
			CollectionKinds: []string{string(document.CollectionKindFPMasukanCoretax)},
		},
		{
			Key:             "bukpot_request_gst_deduction_mt",
			RouteKey:        "bukpot_request_gst_deduction_mt",
			Label:           "Request Bukpot GST Deduction MT",
			Description:     "Mapping default untuk action Request Bukpot GST Deduction MT.",
			Kind:            string(ModuleKindDefaultProfile),
			Group:           "request_bukpot",
			GroupLabel:      "Request Bukpot",
			IconKey:         "receipt-text",
			CollectionKinds: []string{string(document.CollectionKindBukpotRequestGSTDeductionMT)},
		},
		{
			Key:             string(appbukpot.ActionProfileBPPURenameBukpot),
			RouteKey:        string(appbukpot.ActionProfileBPPURenameBukpot),
			Label:           "BPPU Rename Bukpot",
			Description:     "Nilai default parameter untuk action Rename Bukpot pada collection BPPU.",
			Kind:            string(ModuleKindDefaultProfile),
			Group:           "bukpot",
			GroupLabel:      "Bukpot",
			IconKey:         "receipt-text",
			CollectionKinds: []string{string(document.CollectionKindBukpotBPPU)},
		},
		{
			Key:             string(appbukpot.ActionProfileBP21RenameBukpot),
			RouteKey:        string(appbukpot.ActionProfileBP21RenameBukpot),
			Label:           "BP21 Rename Bukpot",
			Description:     "Nilai default parameter untuk action Rename Bukpot pada collection BP21.",
			Kind:            string(ModuleKindDefaultProfile),
			Group:           "bukpot",
			GroupLabel:      "Bukpot",
			IconKey:         "receipt-text",
			CollectionKinds: []string{string(document.CollectionKindBukpotBP21)},
		},
		{
			Key:             string(appbukpot.ActionProfileBPA1RenameBukpot),
			RouteKey:        string(appbukpot.ActionProfileBPA1RenameBukpot),
			Label:           "BPA1 Rename Bukpot",
			Description:     "Nilai default parameter untuk action Rename Bukpot pada collection BPA1.",
			Kind:            string(ModuleKindDefaultProfile),
			Group:           "bukpot",
			GroupLabel:      "Bukpot",
			IconKey:         "receipt-text",
			CollectionKinds: []string{string(document.CollectionKindBukpotBPA1)},
		},
		{
			Key:             string(appbukpot.ActionProfileBPPURenameByCategory),
			RouteKey:        string(appbukpot.ActionProfileBPPURenameByCategory),
			Label:           "BPPU Rename by Category",
			Description:     "Nilai default parameter untuk action Rename by Category pada collection BPPU.",
			Kind:            string(ModuleKindDefaultProfile),
			Group:           "bukpot",
			GroupLabel:      "Bukpot",
			IconKey:         "receipt-text",
			CollectionKinds: []string{string(document.CollectionKindBukpotBPPU)},
		},
		{
			Key:             string(appbukpot.ActionProfileBP21RenameByCategory),
			RouteKey:        string(appbukpot.ActionProfileBP21RenameByCategory),
			Label:           "BP21 Rename by Category",
			Description:     "Nilai default parameter untuk action Rename by Category pada collection BP21.",
			Kind:            string(ModuleKindDefaultProfile),
			Group:           "bukpot",
			GroupLabel:      "Bukpot",
			IconKey:         "receipt-text",
			CollectionKinds: []string{string(document.CollectionKindBukpotBP21)},
		},
	}

	if enableCashflowXLSX {
		items = append(items,
			ModuleSpec{
				Key:             "cashflow_spend_money",
				RouteKey:        "cashflow_spend_money",
				Label:           "Cashflow Spend Money",
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
				Label:           "Cashflow Receive Money",
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
				Label:           "Cashflow Pay Bills",
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
				Label:           "Cashflow Receive Payments",
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
