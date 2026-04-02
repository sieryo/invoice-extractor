package configmodule

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	appbukpot "github.com/sieryo/invoice-extractor/internal/app/bukpot"
	"github.com/sieryo/invoice-extractor/internal/app/buyer"
	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
	appcashflowbill "github.com/sieryo/invoice-extractor/internal/app/cashflowbill"
	"github.com/sieryo/invoice-extractor/internal/app/configlayout"
	appfpcoretax "github.com/sieryo/invoice-extractor/internal/app/fpcoretax"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template"
	"github.com/sieryo/invoice-extractor/internal/app/specutil"
	appconfig "github.com/sieryo/invoice-extractor/internal/config"
	"github.com/sieryo/invoice-extractor/internal/infra/parser"
)

const defaultBlockKey = "main"

type ModulePageSpec struct {
	Module ModuleDetailSpec `json:"module"`
	Blocks []any            `json:"blocks"`
}

type StatusBannerSpec struct {
	Tone    string `json:"tone"`
	Message string `json:"message"`
}

type FormValidationSpec struct {
	ReadyMessage    string `json:"readyMessage,omitempty"`
	NotReadyMessage string `json:"notReadyMessage,omitempty"`
	MissingLabel    string `json:"missingLabel,omitempty"`
}

type FormFieldOptionSpec struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type FormFieldValidationSpec struct {
	Min            *float64 `json:"min,omitempty"`
	Max            *float64 `json:"max,omitempty"`
	MinLength      *int     `json:"minLength,omitempty"`
	MaxLength      *int     `json:"maxLength,omitempty"`
	Pattern        string   `json:"pattern,omitempty"`
	PatternMessage string   `json:"patternMessage,omitempty"`
}

type FormFieldPresentationSpec struct {
	Formatter    string `json:"formatter,omitempty"`
	PreviewLabel string `json:"previewLabel,omitempty"`
}

type FormFieldSpec struct {
	Key          string                     `json:"key"`
	Label        string                     `json:"label"`
	Required     bool                       `json:"required"`
	Kind         string                     `json:"kind,omitempty"`
	Description  string                     `json:"description,omitempty"`
	Placeholder  string                     `json:"placeholder,omitempty"`
	HelpText     string                     `json:"helpText,omitempty"`
	Options      []FormFieldOptionSpec      `json:"options,omitempty"`
	Suggestions  []any                      `json:"suggestions,omitempty"`
	Validation   *FormFieldValidationSpec   `json:"validation,omitempty"`
	Presentation *FormFieldPresentationSpec `json:"presentation,omitempty"`
}

type FormVariantSpec struct {
	Key               string            `json:"key"`
	Label             string            `json:"label"`
	Description       string            `json:"description,omitempty"`
	Values            map[string]string `json:"values"`
	RequiredFieldKeys []string          `json:"requiredFieldKeys,omitempty"`
}

type FormSubmitActionSpec struct {
	Method         string `json:"method"`
	URL            string `json:"url"`
	SuccessMessage string `json:"successMessage,omitempty"`
}

type FormBlockSpec struct {
	Type             string                     `json:"type"`
	Key              string                     `json:"key"`
	Title            string                     `json:"title"`
	Description      string                     `json:"description,omitempty"`
	StatusBanner     *StatusBannerSpec          `json:"statusBanner,omitempty"`
	Validation       *FormValidationSpec        `json:"validation,omitempty"`
	Sections         []configlayout.SectionSpec `json:"sections,omitempty"`
	Fields           []FormFieldSpec            `json:"fields"`
	Values           map[string]string          `json:"values,omitempty"`
	Variants         []FormVariantSpec          `json:"variants,omitempty"`
	ActiveVariantKey string                     `json:"activeVariantKey,omitempty"`
	Submit           FormSubmitActionSpec       `json:"submit"`
	SaveLabel        string                     `json:"saveLabel,omitempty"`
	EmptyMessage     string                     `json:"emptyMessage,omitempty"`
}

type RegistryStatusSpec struct {
	Loaded  bool   `json:"loaded"`
	Count   int    `json:"count,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type RegistryUploadActionSpec struct {
	Method         string `json:"method"`
	URL            string `json:"url"`
	FieldName      string `json:"fieldName"`
	SuccessMessage string `json:"successMessage,omitempty"`
}

type RegistryColumnSpec struct {
	Key         string `json:"key"`
	Header      string `json:"header"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

type RegistryUploadSpec struct {
	AcceptedExtensions []string `json:"acceptedExtensions"`
	AcceptedMimeTypes  []string `json:"acceptedMimeTypes,omitempty"`
	MaxFileSizeMB      int      `json:"maxFileSizeMB"`
}

type RegistrySchemaSpec struct {
	SchemaVersion string               `json:"schemaVersion"`
	Columns       []RegistryColumnSpec `json:"columns"`
	Upload        RegistryUploadSpec   `json:"upload"`
	Relative      string               `json:"relative,omitempty"`
	LookupKey     string               `json:"lookupKey,omitempty"`
}

type RegistryMetaRowSpec struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type RegistryMetaGroupSpec struct {
	Title string   `json:"title"`
	Items []string `json:"items"`
}

type RegistryPreviewItemSpec struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
}

type RegistryBlockSpec struct {
	Type             string                    `json:"type"`
	Key              string                    `json:"key"`
	Title            string                    `json:"title"`
	Description      string                    `json:"description,omitempty"`
	Status           RegistryStatusSpec        `json:"status"`
	CountLabel       string                    `json:"countLabel"`
	Schema           RegistrySchemaSpec        `json:"schema"`
	UploadAction     RegistryUploadActionSpec  `json:"uploadAction"`
	UploadTitle      string                    `json:"uploadTitle"`
	PreviewTitle     string                    `json:"previewTitle"`
	PreviewItems     []RegistryPreviewItemSpec `json:"previewItems,omitempty"`
	PreviewEmptyText string                    `json:"previewEmptyText,omitempty"`
	MetadataRows     []RegistryMetaRowSpec     `json:"metadataRows,omitempty"`
	MetadataGroups   []RegistryMetaGroupSpec   `json:"metadataGroups,omitempty"`
}

type CatalogItemSpec struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	Badge    string `json:"badge,omitempty"`
}

type CatalogBlockSpec struct {
	Type        string            `json:"type"`
	Key         string            `json:"key"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	CountLabel  string            `json:"countLabel,omitempty"`
	Items       []CatalogItemSpec `json:"items,omitempty"`
	EmptyText   string            `json:"emptyText,omitempty"`
}

type FormBlockInput struct {
	Values           map[string]string            `json:"values,omitempty"`
	Variants         map[string]map[string]string `json:"variants,omitempty"`
	ActiveVariantKey string                       `json:"activeVariantKey,omitempty"`
}

type PageService struct {
	settings             *appconfig.SettingsService
	buyerRegistry        *buyer.BuyerRegistryService
	templateRegistry     *template.TemplateRegistryService
	bukpotRequest        *appbukpot.RequestConfigService
	bukpotActionProfiles *appbukpot.ActionProfileService
	cashflowProfiles     *appcashflow.ProfileConfigService
	cashflowBillProfiles *appcashflowbill.ProfileConfigService
	taxAccounts          *appcashflow.TaxAccountService
	categoryAccounts     *appcashflowbill.CategoryAccountService
	fpCoretaxProfiles    *appfpcoretax.ProfileConfigService
	fpCoretaxRelations   *appfpcoretax.RelationRegistryService
}

func myobAccountFieldPresentation(key string) (*FormFieldValidationSpec, *FormFieldPresentationSpec) {
	if !specutil.IsMyobAccountFieldKey(key) {
		return nil, nil
	}

	return &FormFieldValidationSpec{
			Pattern:        specutil.MyobAccountNumberPattern,
			PatternMessage: specutil.MyobAccountNumberPatternMessage,
		}, &FormFieldPresentationSpec{
			Formatter:    specutil.MyobAccountNumberFormatter,
			PreviewLabel: "Preview MYOB",
		}
}

func NewPageService(
	settings *appconfig.SettingsService,
	buyerRegistry *buyer.BuyerRegistryService,
	templateRegistry *template.TemplateRegistryService,
	bukpotRequest *appbukpot.RequestConfigService,
	bukpotActionProfiles *appbukpot.ActionProfileService,
	cashflowProfiles *appcashflow.ProfileConfigService,
	cashflowBillProfiles *appcashflowbill.ProfileConfigService,
	taxAccounts *appcashflow.TaxAccountService,
	categoryAccounts *appcashflowbill.CategoryAccountService,
	fpCoretaxProfiles *appfpcoretax.ProfileConfigService,
	fpCoretaxRelations *appfpcoretax.RelationRegistryService,
) *PageService {
	return &PageService{
		settings:             settings,
		buyerRegistry:        buyerRegistry,
		templateRegistry:     templateRegistry,
		bukpotRequest:        bukpotRequest,
		bukpotActionProfiles: bukpotActionProfiles,
		cashflowProfiles:     cashflowProfiles,
		cashflowBillProfiles: cashflowBillProfiles,
		taxAccounts:          taxAccounts,
		categoryAccounts:     categoryAccounts,
		fpCoretaxProfiles:    fpCoretaxProfiles,
		fpCoretaxRelations:   fpCoretaxRelations,
	}
}

func (s *PageService) Page(profileID string, moduleKey string, enableCashflowXLSX bool) (ModulePageSpec, error) {
	module, ok := FindModule(moduleKey, enableCashflowXLSX)
	if !ok {
		return ModulePageSpec{}, os.ErrNotExist
	}

	block, err := s.buildBlock(profileID, module.Key)
	if err != nil {
		return ModulePageSpec{}, err
	}

	return ModulePageSpec{
		Module: module,
		Blocks: []any{block},
	}, nil
}

func (s *PageService) UpdateFormBlock(profileID string, moduleKey string, blockKey string, input FormBlockInput) error {
	if strings.TrimSpace(blockKey) != defaultBlockKey {
		return os.ErrNotExist
	}

	switch strings.TrimSpace(moduleKey) {
	case "app_modules":
		return s.updateAppSettings(input)
	case "bukpot_request_gst_deduction_mt":
		return s.updateBukpotRequest(profileID, input)
	case string(appbukpot.ActionProfileBPPURenameBukpot),
		string(appbukpot.ActionProfileBP21RenameBukpot),
		string(appbukpot.ActionProfileBPA1RenameBukpot),
		string(appbukpot.ActionProfileBPPURenameByCategory),
		string(appbukpot.ActionProfileBP21RenameByCategory):
		return s.updateBukpotActionProfile(profileID, appbukpot.ActionProfileKey(moduleKey), input)
	case "cashflow_spend_money":
		return s.updateCashflowProfile(profileID, appcashflow.ProfileConfigSpendMoney, input)
	case "cashflow_receive_money":
		return s.updateCashflowProfile(profileID, appcashflow.ProfileConfigReceiveMoney, input)
	case "cashflow_pay_bills":
		return s.updateCashflowBillProfile(profileID, appcashflowbill.ProfileConfigPayBills, input)
	case "cashflow_receive_payments":
		return s.updateCashflowBillProfile(profileID, appcashflowbill.ProfileConfigReceivePayments, input)
	case string(appfpcoretax.ProfileConfigFPKeluaranMiscSales):
		return s.updateFPCoretaxProfile(profileID, appfpcoretax.ProfileConfigFPKeluaranMiscSales, input)
	case string(appfpcoretax.ProfileConfigFPMasukanMiscPurchases):
		return s.updateFPCoretaxProfile(profileID, appfpcoretax.ProfileConfigFPMasukanMiscPurchases, input)
	default:
		return os.ErrNotExist
	}
}

func (s *PageService) UploadRegistryBlock(profileID string, moduleKey string, blockKey string, filename string, fileSize int64, tempPath string) (int, []parser.ValidationIssue, error) {
	if strings.TrimSpace(blockKey) != defaultBlockKey {
		return 0, nil, os.ErrNotExist
	}

	switch strings.TrimSpace(moduleKey) {
	case "buyer_registry":
		return s.uploadBuyerRegistry(profileID, filename, fileSize, tempPath)
	case "cashflow_tax_accounts":
		return s.uploadTaxAccounts(profileID, filename, fileSize, tempPath)
	case "cashflow_category_accounts":
		return s.uploadCategoryAccounts(profileID, filename, fileSize, tempPath)
	case "fp_keluaran_customer_registry":
		return s.uploadFPCoretaxRegistry(profileID, appfpcoretax.RelationRegistryCustomer, filename, fileSize, tempPath)
	case "fp_masukan_supplier_registry":
		return s.uploadFPCoretaxRegistry(profileID, appfpcoretax.RelationRegistrySupplier, filename, fileSize, tempPath)
	default:
		return 0, nil, os.ErrNotExist
	}
}

func (s *PageService) buildBlock(profileID string, moduleKey string) (any, error) {
	switch strings.TrimSpace(moduleKey) {
	case "app_modules":
		return s.buildAppSettingsBlock()
	case "buyer_registry":
		return s.buildBuyerRegistryBlock(profileID)
	case "template_registry":
		return s.buildTemplateCatalogBlock(), nil
	case "bukpot_request_gst_deduction_mt":
		return s.buildBukpotRequestBlock(profileID)
	case string(appbukpot.ActionProfileBPPURenameBukpot),
		string(appbukpot.ActionProfileBP21RenameBukpot),
		string(appbukpot.ActionProfileBPA1RenameBukpot),
		string(appbukpot.ActionProfileBPPURenameByCategory),
		string(appbukpot.ActionProfileBP21RenameByCategory):
		return s.buildBukpotActionProfileBlock(profileID, appbukpot.ActionProfileKey(moduleKey))
	case "cashflow_spend_money":
		return s.buildCashflowProfileBlock(profileID, appcashflow.ProfileConfigSpendMoney)
	case "cashflow_receive_money":
		return s.buildCashflowProfileBlock(profileID, appcashflow.ProfileConfigReceiveMoney)
	case "cashflow_pay_bills":
		return s.buildCashflowBillProfileBlock(profileID, appcashflowbill.ProfileConfigPayBills)
	case "cashflow_receive_payments":
		return s.buildCashflowBillProfileBlock(profileID, appcashflowbill.ProfileConfigReceivePayments)
	case "cashflow_tax_accounts":
		return s.buildTaxAccountBlock(profileID)
	case "cashflow_category_accounts":
		return s.buildCategoryAccountBlock(profileID)
	case string(appfpcoretax.ProfileConfigFPKeluaranMiscSales):
		return s.buildFPCoretaxProfileBlock(profileID, appfpcoretax.ProfileConfigFPKeluaranMiscSales)
	case "fp_keluaran_customer_registry":
		return s.buildFPCoretaxRegistryBlock(profileID, appfpcoretax.RelationRegistryCustomer)
	case string(appfpcoretax.ProfileConfigFPMasukanMiscPurchases):
		return s.buildFPCoretaxProfileBlock(profileID, appfpcoretax.ProfileConfigFPMasukanMiscPurchases)
	case "fp_masukan_supplier_registry":
		return s.buildFPCoretaxRegistryBlock(profileID, appfpcoretax.RelationRegistrySupplier)
	default:
		return nil, os.ErrNotExist
	}
}

func (s *PageService) buildAppSettingsBlock() (FormBlockSpec, error) {
	settings, err := s.settings.Load()
	if err != nil {
		return FormBlockSpec{}, err
	}
	status := s.settings.Status()

	return FormBlockSpec{
		Type:        "form",
		Key:         defaultBlockKey,
		Title:       "Aktivasi Modul",
		Description: "Aktifkan atau nonaktifkan modul aplikasi yang tersedia saat ini.",
		StatusBanner: &StatusBannerSpec{
			Tone:    map[bool]string{true: "not-ready", false: "ready"}[status.RestartRequired],
			Message: status.Message,
		},
		Sections: []configlayout.SectionSpec{
			{
				Key:         "features",
				Title:       "Fitur",
				Description: "Perubahan disimpan ke pengaturan aplikasi dan diterapkan ke runtime backend.",
				Columns:     1,
				FieldKeys:   []string{"enableCashflowXLSX"},
			},
		},
		Fields: []FormFieldSpec{
			{
				Key:         "enableCashflowXLSX",
				Label:       "Cashflow XLSX",
				Required:    false,
				Kind:        "checkbox",
				Description: "Menampilkan collection cashflow, default profil cashflow, dan registry MYOB terkait.",
			},
		},
		Values: map[string]string{
			"enableCashflowXLSX": boolString(settings.Features.EnableCashflowXLSX),
		},
		Submit: FormSubmitActionSpec{
			Method:         "PUT",
			URL:            "/config/modules/app_modules/blocks/main",
			SuccessMessage: "Pengaturan modul aplikasi diperbarui",
		},
		SaveLabel: "Simpan Pengaturan Modul",
	}, nil
}

func (s *PageService) buildBukpotRequestBlock(profileID string) (FormBlockSpec, error) {
	spec := s.bukpotRequest.Spec()
	cfg, err := s.bukpotRequest.Load(profileID)
	if err != nil {
		return FormBlockSpec{}, err
	}

	fields := make([]FormFieldSpec, 0, len(spec.DefaultFields)+len(spec.Fields))
	for _, item := range spec.DefaultFields {
		validation, presentation := myobAccountFieldPresentation(item.Key)
		fields = append(fields, FormFieldSpec{
			Key:          item.Key,
			Label:        item.Label,
			Required:     item.Required,
			Kind:         item.Kind,
			Description:  item.Description,
			Options:      requestOptionsToForm(item.Options),
			Validation:   validation,
			Presentation: presentation,
		})
	}
	for _, item := range spec.Fields {
		validation, presentation := myobAccountFieldPresentation(item.Key)
		fields = append(fields, FormFieldSpec{
			Key:          item.Key,
			Label:        item.Label,
			Required:     item.Required,
			Kind:         item.Kind,
			Description:  item.Description,
			Options:      requestOptionsToForm(item.Options),
			Validation:   validation,
			Presentation: presentation,
		})
	}

	values := map[string]string{
		"sheetName":       strings.TrimSpace(cfg.Defaults.SheetName),
		"headerRowNumber": strconv.Itoa(cfg.Defaults.HeaderRowNumber),
	}
	for _, item := range cfg.Fields {
		values[item.Key] = strings.TrimSpace(item.Value)
	}

	return FormBlockSpec{
		Type:        "form",
		Key:         defaultBlockKey,
		Title:       specutil.ParameterDefaultActionCardTitle,
		Description: "Parameter ini akan menjadi nilai awal saat action dibuka di Actions Tab. User tetap bisa mengubahnya untuk eksekusi tertentu.",
		Validation: &FormValidationSpec{
			ReadyMessage:    "Default profil request bukpot siap digunakan.",
			NotReadyMessage: "Default profil request bukpot belum lengkap.",
			MissingLabel:    "Field wajib yang belum lengkap",
		},
		Sections: spec.Sections,
		Fields:   fields,
		Values:   values,
		Submit: FormSubmitActionSpec{
			Method:         "PUT",
			URL:            "/config/modules/bukpot_request_gst_deduction_mt/blocks/main",
			SuccessMessage: "Default profil request bukpot diperbarui",
		},
		SaveLabel:    "Simpan Default Profil",
		EmptyMessage: "Belum ada field konfigurasi request bukpot.",
	}, nil
}

func (s *PageService) buildBukpotActionProfileBlock(profileID string, key appbukpot.ActionProfileKey) (FormBlockSpec, error) {
	spec := s.bukpotActionProfiles.Spec(key)
	cfg, err := s.bukpotActionProfiles.Load(profileID, key)
	if err != nil {
		return FormBlockSpec{}, err
	}

	fields := make([]FormFieldSpec, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		validation, presentation := myobAccountFieldPresentation(item.Key)
		fields = append(fields, FormFieldSpec{
			Key:          item.Key,
			Label:        item.Label,
			Required:     item.Required,
			Kind:         item.Kind,
			Description:  item.Description,
			Placeholder:  item.Placeholder,
			HelpText:     item.HelpText,
			Suggestions:  actionSuggestionsToAny(item.Suggestions),
			Validation:   validation,
			Presentation: presentation,
		})
	}

	values := make(map[string]string, len(cfg.Fields))
	for _, item := range cfg.Fields {
		values[item.Key] = strings.TrimSpace(item.Value)
	}

	return FormBlockSpec{
		Type:        "form",
		Key:         defaultBlockKey,
		Title:       specutil.ParameterDefaultActionCardTitle,
		Description: "Parameter ini akan menjadi nilai awal saat action dibuka di Actions Tab. User tetap bisa mengubahnya untuk eksekusi tertentu.",
		Validation: &FormValidationSpec{
			ReadyMessage:    "Default profil bukpot siap digunakan.",
			NotReadyMessage: "Default profil bukpot belum lengkap.",
			MissingLabel:    "Parameter wajib yang belum lengkap",
		},
		Sections: spec.Sections,
		Fields:   fields,
		Values:   values,
		Submit: FormSubmitActionSpec{
			Method:         "PUT",
			URL:            fmt.Sprintf("/config/modules/%s/blocks/main", string(key)),
			SuccessMessage: "Default profil bukpot diperbarui",
		},
		SaveLabel:    "Simpan Default Profil",
		EmptyMessage: "Action ini belum punya parameter yang perlu disimpan di default profile.",
	}, nil
}

func (s *PageService) buildCashflowProfileBlock(profileID string, key appcashflow.ProfileConfigKey) (FormBlockSpec, error) {
	spec := s.cashflowProfiles.Spec(key)
	cfg, err := s.cashflowProfiles.Load(profileID, key)
	if err != nil {
		return FormBlockSpec{}, err
	}

	fields := make([]FormFieldSpec, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		validation, presentation := myobAccountFieldPresentation(item.Key)
		fields = append(fields, FormFieldSpec{
			Key:          item.Key,
			Label:        item.Label,
			Required:     item.Required,
			Kind:         item.Kind,
			Description:  item.Description,
			Options:      cashflowOptionsToForm(item.Options),
			Validation:   validation,
			Presentation: presentation,
		})
	}

	variants := make([]FormVariantSpec, 0, len(cfg.Variants))
	for _, item := range cfg.Variants {
		variants = append(variants, FormVariantSpec{
			Key:               item.Key,
			Label:             item.Label,
			Description:       item.Description,
			Values:            cloneStringMap(item.Values),
			RequiredFieldKeys: cashflowRequiredFieldKeys(item.Key),
		})
	}

	activeVariantKey := strings.TrimSpace(cfg.Defaults.CashflowFormat)
	if activeVariantKey == "" && len(spec.Variants) > 0 {
		activeVariantKey = spec.Variants[0].Key
	}

	moduleKey := "cashflow_spend_money"
	if key == appcashflow.ProfileConfigReceiveMoney {
		moduleKey = "cashflow_receive_money"
	}

	return FormBlockSpec{
		Type:        "form",
		Key:         defaultBlockKey,
		Title:       specutil.ParameterDefaultActionCardTitle,
		Description: "Parameter ini akan menjadi nilai awal saat action dibuka di Actions Tab. User tetap bisa mengubahnya untuk eksekusi tertentu.",
		Validation: &FormValidationSpec{
			ReadyMessage:    "Default profil cashflow siap digunakan untuk format aktif.",
			NotReadyMessage: "Default profil cashflow belum lengkap untuk format aktif.",
			MissingLabel:    "Field wajib yang belum lengkap",
		},
		Sections:         spec.Sections,
		Fields:           fields,
		Variants:         variants,
		ActiveVariantKey: activeVariantKey,
		Submit: FormSubmitActionSpec{
			Method:         "PUT",
			URL:            fmt.Sprintf("/config/modules/%s/blocks/main", moduleKey),
			SuccessMessage: fmt.Sprintf("Default profil %s diperbarui", strings.ToLower(spec.Label)),
		},
		SaveLabel:    "Simpan Default Profil",
		EmptyMessage: "Belum ada field konfigurasi cashflow.",
	}, nil
}

func (s *PageService) buildCashflowBillProfileBlock(profileID string, key appcashflowbill.ProfileConfigKey) (FormBlockSpec, error) {
	spec := s.cashflowBillProfiles.Spec(key)
	cfg, err := s.cashflowBillProfiles.Load(profileID, key)
	if err != nil {
		return FormBlockSpec{}, err
	}

	fields := make([]FormFieldSpec, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		validation, presentation := myobAccountFieldPresentation(item.Key)
		fields = append(fields, FormFieldSpec{
			Key:          item.Key,
			Label:        item.Label,
			Required:     item.Required,
			Kind:         item.Kind,
			Description:  item.Description,
			Options:      cashflowBillOptionsToForm(item.Options),
			Validation:   validation,
			Presentation: presentation,
		})
	}

	variants := make([]FormVariantSpec, 0, len(cfg.Variants))
	for _, item := range cfg.Variants {
		variants = append(variants, FormVariantSpec{
			Key:               item.Key,
			Label:             item.Label,
			Description:       item.Description,
			Values:            cloneStringMap(item.Values),
			RequiredFieldKeys: []string{"headerRowNumber", "outputFilename", "chequeAccount", "date", "category", "information", "partyName", "total"},
		})
	}

	activeVariantKey := strings.TrimSpace(cfg.Defaults.CashflowFormat)
	if activeVariantKey == "" && len(spec.Variants) > 0 {
		activeVariantKey = spec.Variants[0].Key
	}

	moduleKey := "cashflow_pay_bills"
	if key == appcashflowbill.ProfileConfigReceivePayments {
		moduleKey = "cashflow_receive_payments"
	}

	return FormBlockSpec{
		Type:        "form",
		Key:         defaultBlockKey,
		Title:       specutil.ParameterDefaultActionCardTitle,
		Description: "Parameter ini akan menjadi nilai awal saat action dibuka di Actions Tab. User tetap bisa mengubahnya untuk eksekusi tertentu.",
		Validation: &FormValidationSpec{
			ReadyMessage:    "Default profil cashflow bills siap digunakan.",
			NotReadyMessage: "Default profil cashflow bills belum lengkap.",
			MissingLabel:    "Field wajib yang belum lengkap",
		},
		Sections:         spec.Sections,
		Fields:           fields,
		Variants:         variants,
		ActiveVariantKey: activeVariantKey,
		Submit: FormSubmitActionSpec{
			Method:         "PUT",
			URL:            fmt.Sprintf("/config/modules/%s/blocks/main", moduleKey),
			SuccessMessage: fmt.Sprintf("Default profil %s diperbarui", strings.ToLower(spec.Label)),
		},
		SaveLabel:    "Simpan Default Profil",
		EmptyMessage: "Belum ada field konfigurasi cashflow bills.",
	}, nil
}

func (s *PageService) buildFPCoretaxProfileBlock(profileID string, key appfpcoretax.ProfileConfigKey) (FormBlockSpec, error) {
	spec := s.fpCoretaxProfiles.Spec(key)
	cfg, err := s.fpCoretaxProfiles.Load(profileID, key)
	if err != nil {
		return FormBlockSpec{}, err
	}

	fields := make([]FormFieldSpec, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		validation, presentation := myobAccountFieldPresentation(item.Key)
		fields = append(fields, FormFieldSpec{
			Key:          item.Key,
			Label:        item.Label,
			Required:     item.Required,
			Kind:         item.Kind,
			Description:  item.Description,
			Placeholder:  item.Placeholder,
			HelpText:     item.HelpText,
			Options:      fpCoretaxOptionsToForm(item.Options),
			Suggestions:  fpCoretaxSuggestionsToAny(item.Suggestions),
			Validation:   validation,
			Presentation: presentation,
		})
	}

	values := make(map[string]string, len(cfg.Fields))
	for _, item := range cfg.Fields {
		values[item.Key] = strings.TrimSpace(item.Value)
	}

	return FormBlockSpec{
		Type:        "form",
		Key:         defaultBlockKey,
		Title:       specutil.ParameterDefaultActionCardTitle,
		Description: "Nilai awal saat action dibuka.",
		Validation: &FormValidationSpec{
			ReadyMessage:    "Default profil FP Coretax siap digunakan.",
			NotReadyMessage: "Default profil FP Coretax belum lengkap.",
			MissingLabel:    "Parameter wajib yang belum lengkap",
		},
		Sections: spec.Sections,
		Fields:   fields,
		Values:   values,
		Submit: FormSubmitActionSpec{
			Method:         "PUT",
			URL:            fmt.Sprintf("/config/modules/%s/blocks/main", string(key)),
			SuccessMessage: fmt.Sprintf("Default profil %s diperbarui", strings.ToLower(spec.Label)),
		},
		SaveLabel:    "Simpan Default Profil",
		EmptyMessage: "Belum ada field konfigurasi FP Coretax.",
	}, nil
}

func (s *PageService) buildFPCoretaxRegistryBlock(profileID string, key appfpcoretax.RelationRegistryKey) (RegistryBlockSpec, error) {
	spec := s.fpCoretaxRelations.Spec(key)
	status := s.fpCoretaxRelations.Status(profileID, key)
	items, err := s.fpCoretaxRelations.List(profileID, key)
	if err != nil {
		var schemaErr *parser.FPCoretaxRelationSchemaMismatchError
		if !errors.As(err, &schemaErr) && !os.IsNotExist(err) {
			return RegistryBlockSpec{}, err
		}
		items = nil
	}

	previewItems := make([]RegistryPreviewItemSpec, 0, len(items))
	for index, item := range items {
		previewItems = append(previewItems, RegistryPreviewItemSpec{
			Key:      fmt.Sprintf("%s-%d", strings.TrimSpace(item.Name), index),
			Title:    strings.TrimSpace(item.Name),
			Subtitle: strings.TrimSpace(item.Account),
		})
	}

	moduleKey := "fp_keluaran_customer_registry"
	title := "Customer Registry"
	description := "Lookup customer MYOB."
	countLabel := "customer"
	if key == appfpcoretax.RelationRegistrySupplier {
		moduleKey = "fp_masukan_supplier_registry"
		title = "Supplier Registry"
		description = "Lookup supplier MYOB."
		countLabel = "supplier"
	}

	return RegistryBlockSpec{
		Type:        "registry",
		Key:         defaultBlockKey,
		Title:       title,
		Description: description,
		Status: RegistryStatusSpec{
			Loaded:  status.Loaded,
			Count:   len(items),
			Code:    status.Code,
			Message: status.Message,
		},
		CountLabel: countLabel,
		Schema: RegistrySchemaSpec{
			SchemaVersion: spec.SchemaVersion,
			Columns:       fpCoretaxRegistryColumnsToRegistry(spec.Columns),
			Upload: RegistryUploadSpec{
				AcceptedExtensions: spec.Upload.AcceptedExtensions,
				AcceptedMimeTypes:  spec.Upload.AcceptedMIMETypes,
				MaxFileSizeMB:      int(spec.Upload.MaxFileSizeMB),
			},
			Relative:  spec.Relative,
			LookupKey: spec.LookupKey,
		},
		UploadAction: RegistryUploadActionSpec{
			Method:         "POST",
			URL:            fmt.Sprintf("/config/modules/%s/blocks/main/upload", moduleKey),
			FieldName:      "file",
			SuccessMessage: fmt.Sprintf("%s diperbarui", strings.ToLower(title)),
		},
		UploadTitle:      "Upload Registry",
		PreviewTitle:     "Preview Registry",
		PreviewItems:     previewItems,
		PreviewEmptyText: "Belum ada data registry.",
		MetadataRows: []RegistryMetaRowSpec{
			{Label: "Ekstensi", Value: strings.Join(spec.Upload.AcceptedExtensions, ", ")},
			{Label: "Maks. Ukuran", Value: fmt.Sprintf("%d MB", spec.Upload.MaxFileSizeMB)},
			{Label: "Lookup Key", Value: spec.LookupKey},
			{Label: "Output CSV", Value: spec.Relative},
		},
	}, nil
}

func (s *PageService) buildBuyerRegistryBlock(profileID string) (RegistryBlockSpec, error) {
	spec := s.buyerRegistry.Spec()
	status := s.buyerRegistry.Status(profileID)
	items, err := s.buyerRegistry.List(profileID)
	if err != nil {
		items = nil
	}

	previewItems := make([]RegistryPreviewItemSpec, 0, len(items))
	for index, item := range items {
		subtitle := strings.TrimSpace(item.NPWP16)
		if subtitle == "" {
			subtitle = strings.TrimSpace(item.NPWP15)
		}
		if subtitle == "" {
			subtitle = strings.TrimSpace(item.NITKU)
		}
		previewItems = append(previewItems, RegistryPreviewItemSpec{
			Key:      fmt.Sprintf("%s-%d", strings.TrimSpace(item.Name), index),
			Title:    strings.TrimSpace(item.Name),
			Subtitle: subtitle,
		})
	}

	return RegistryBlockSpec{
		Type:        "registry",
		Key:         defaultBlockKey,
		Title:       "Buyer Registry",
		Description: "Sinkronisasi data buyer untuk enrich data invoice.",
		Status: RegistryStatusSpec{
			Loaded:  status.Loaded,
			Count:   len(items),
			Code:    status.Code,
			Message: status.Message,
		},
		CountLabel: "buyer",
		Schema: RegistrySchemaSpec{
			SchemaVersion: spec.SchemaVersion,
			Columns:       buyerColumnsToRegistry(spec.Columns),
			Upload: RegistryUploadSpec{
				AcceptedExtensions: spec.Upload.AcceptedExtensions,
				AcceptedMimeTypes:  spec.Upload.AcceptedMIMETypes,
				MaxFileSizeMB:      int(spec.Upload.MaxFileSizeMB),
			},
		},
		UploadAction: RegistryUploadActionSpec{
			Method:         "POST",
			URL:            "/config/modules/buyer_registry/blocks/main/upload",
			FieldName:      "file",
			SuccessMessage: "Buyer registry diperbarui",
		},
		UploadTitle:      "Upload Buyer Registry",
		PreviewTitle:     "Preview Buyer",
		PreviewItems:     previewItems,
		PreviewEmptyText: "Belum ada data buyer.",
		MetadataRows: []RegistryMetaRowSpec{
			{Label: "Ekstensi", Value: strings.Join(spec.Upload.AcceptedExtensions, ", ")},
			{Label: "Maks. Ukuran", Value: fmt.Sprintf("%d MB", spec.Upload.MaxFileSizeMB)},
			{Label: "MIME Type", Value: fallbackValue(strings.Join(spec.Upload.AcceptedMIMETypes, ", "), "-")},
		},
	}, nil
}

func (s *PageService) buildTemplateCatalogBlock() CatalogBlockSpec {
	items := s.templateRegistry.List()
	out := make([]CatalogItemSpec, 0, len(items))
	for _, item := range items {
		out = append(out, CatalogItemSpec{
			Key:      item.Identifier,
			Title:    item.Name,
			Subtitle: item.Identifier,
			Badge:    item.Alias,
		})
	}

	return CatalogBlockSpec{
		Type:        "catalog",
		Key:         defaultBlockKey,
		Title:       "Template Registry",
		Description: "Template parser invoice yang tersedia saat ini.",
		CountLabel:  "template",
		Items:       out,
		EmptyText:   "Template registry belum tersedia.",
	}
}

func (s *PageService) buildTaxAccountBlock(profileID string) (RegistryBlockSpec, error) {
	spec := s.taxAccounts.Spec()
	status := s.taxAccounts.Status(profileID)
	items, err := s.taxAccounts.List(profileID)
	if err != nil {
		var pathErr *parser.TaxAccountSchemaMismatchError
		if !errors.As(err, &pathErr) && !os.IsNotExist(err) {
			return RegistryBlockSpec{}, err
		}
		items = nil
	}

	previewItems := make([]RegistryPreviewItemSpec, 0, len(items))
	for index, item := range items {
		previewItems = append(previewItems, RegistryPreviewItemSpec{
			Key:      fmt.Sprintf("%s-%d", strings.TrimSpace(item.Name), index),
			Title:    strings.TrimSpace(item.Name),
			Subtitle: strings.TrimSpace(item.Account),
		})
	}

	block := RegistryBlockSpec{
		Type:        "registry",
		Key:         defaultBlockKey,
		Title:       "Tax Accounts",
		Description: "Master data lookup account untuk export cashflow MYOB.",
		Status: RegistryStatusSpec{
			Loaded:  status.Loaded,
			Count:   len(items),
			Code:    status.Code,
			Message: status.Message,
		},
		CountLabel: "account",
		Schema: RegistrySchemaSpec{
			SchemaVersion: spec.SchemaVersion,
			Columns:       taxColumnsToRegistry(spec.Columns),
			Upload: RegistryUploadSpec{
				AcceptedExtensions: spec.Upload.AcceptedExtensions,
				AcceptedMimeTypes:  spec.Upload.AcceptedMIMETypes,
				MaxFileSizeMB:      int(spec.Upload.MaxFileSizeMB),
			},
			Relative:  spec.Relative,
			LookupKey: spec.LookupKey,
		},
		UploadAction: RegistryUploadActionSpec{
			Method:         "POST",
			URL:            "/config/modules/cashflow_tax_accounts/blocks/main/upload",
			FieldName:      "file",
			SuccessMessage: "Tax accounts diperbarui",
		},
		UploadTitle:      "Upload Tax Accounts",
		PreviewTitle:     "Preview Tax Accounts",
		PreviewItems:     previewItems,
		PreviewEmptyText: "Belum ada data tax accounts.",
		MetadataRows: []RegistryMetaRowSpec{
			{Label: "Ekstensi", Value: strings.Join(spec.Upload.AcceptedExtensions, ", ")},
			{Label: "Maks. Ukuran", Value: fmt.Sprintf("%d MB", spec.Upload.MaxFileSizeMB)},
			{Label: "Lookup Key", Value: spec.LookupKey},
			{Label: "Output CSV", Value: spec.Relative},
		},
	}
	if len(spec.AllowedNames) > 0 {
		block.MetadataGroups = []RegistryMetaGroupSpec{
			{Title: "Nama Tax yang Didukung", Items: spec.AllowedNames},
		}
	}
	return block, nil
}

func (s *PageService) buildCategoryAccountBlock(profileID string) (RegistryBlockSpec, error) {
	spec := s.categoryAccounts.Spec()
	status := s.categoryAccounts.Status(profileID)
	items, err := s.categoryAccounts.List(profileID)
	if err != nil {
		var pathErr *parser.TaxAccountSchemaMismatchError
		if !errors.As(err, &pathErr) && !os.IsNotExist(err) {
			return RegistryBlockSpec{}, err
		}
		items = nil
	}

	previewItems := make([]RegistryPreviewItemSpec, 0, len(items))
	for index, item := range items {
		previewItems = append(previewItems, RegistryPreviewItemSpec{
			Key:      fmt.Sprintf("%s-%d", strings.TrimSpace(item.Name), index),
			Title:    strings.TrimSpace(item.Name),
			Subtitle: strings.TrimSpace(item.Account),
		})
	}

	return RegistryBlockSpec{
		Type:        "registry",
		Key:         defaultBlockKey,
		Title:       "Category Accounts",
		Description: "Master data category untuk mencocokkan row cashflow bills MYOB.",
		Status: RegistryStatusSpec{
			Loaded:  status.Loaded,
			Count:   len(items),
			Code:    status.Code,
			Message: status.Message,
		},
		CountLabel: "account",
		Schema: RegistrySchemaSpec{
			SchemaVersion: spec.SchemaVersion,
			Columns:       taxColumnsToRegistry(spec.Columns),
			Upload: RegistryUploadSpec{
				AcceptedExtensions: spec.Upload.AcceptedExtensions,
				AcceptedMimeTypes:  spec.Upload.AcceptedMIMETypes,
				MaxFileSizeMB:      int(spec.Upload.MaxFileSizeMB),
			},
			Relative:  spec.Relative,
			LookupKey: spec.LookupKey,
		},
		UploadAction: RegistryUploadActionSpec{
			Method:         "POST",
			URL:            "/config/modules/cashflow_category_accounts/blocks/main/upload",
			FieldName:      "file",
			SuccessMessage: "Category accounts diperbarui",
		},
		UploadTitle:      "Upload Category Accounts",
		PreviewTitle:     "Preview Category Accounts",
		PreviewItems:     previewItems,
		PreviewEmptyText: "Belum ada data category accounts.",
		MetadataRows: []RegistryMetaRowSpec{
			{Label: "Ekstensi", Value: strings.Join(spec.Upload.AcceptedExtensions, ", ")},
			{Label: "Maks. Ukuran", Value: fmt.Sprintf("%d MB", spec.Upload.MaxFileSizeMB)},
			{Label: "Lookup Key", Value: spec.LookupKey},
			{Label: "Output CSV", Value: spec.Relative},
		},
	}, nil
}

func (s *PageService) updateAppSettings(input FormBlockInput) error {
	_, err := s.settings.Update(appconfig.AppSettings{
		Features: appconfig.FeatureFlags{
			EnableCashflowXLSX: normalizeBool(input.Values["enableCashflowXLSX"]),
		},
	})
	return err
}

func (s *PageService) updateBukpotRequest(profileID string, input FormBlockInput) error {
	spec := s.bukpotRequest.Spec()
	fields := make([]appbukpot.RequestConfigField, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		fields = append(fields, appbukpot.RequestConfigField{
			Key:         item.Key,
			Label:       item.Label,
			Required:    item.Required,
			Value:       strings.TrimSpace(input.Values[item.Key]),
			Description: item.Description,
			Kind:        item.Kind,
			Group:       item.Group,
		})
	}

	_, err := s.bukpotRequest.Update(profileID, appbukpot.RequestConfig{
		SchemaVersion: spec.SchemaVersion,
		Defaults: appbukpot.RequestConfigDefaults{
			SheetName:       strings.TrimSpace(input.Values["sheetName"]),
			HeaderRowNumber: parsePositiveInt(input.Values["headerRowNumber"], spec.Defaults.HeaderRowNumber),
		},
		Fields: fields,
	})
	return err
}

func (s *PageService) updateBukpotActionProfile(profileID string, key appbukpot.ActionProfileKey, input FormBlockInput) error {
	spec := s.bukpotActionProfiles.Spec(key)
	fields := make([]appbukpot.ActionProfileField, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		fields = append(fields, appbukpot.ActionProfileField{
			Key:         item.Key,
			Label:       item.Label,
			Required:    item.Required,
			Value:       strings.TrimSpace(input.Values[item.Key]),
			Description: item.Description,
			Kind:        item.Kind,
		})
	}
	_, err := s.bukpotActionProfiles.Update(profileID, key, appbukpot.ActionProfile{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Fields:        fields,
	})
	return err
}

func (s *PageService) updateCashflowProfile(profileID string, key appcashflow.ProfileConfigKey, input FormBlockInput) error {
	spec := s.cashflowProfiles.Spec(key)
	variants := make([]appcashflow.ProfileConfigVariant, 0, len(spec.Variants))
	for _, item := range spec.Variants {
		values := cloneStringMap(item.Values)
		if current, ok := input.Variants[item.Key]; ok {
			for fieldKey, fieldValue := range current {
				values[strings.TrimSpace(fieldKey)] = strings.TrimSpace(fieldValue)
			}
		}
		variants = append(variants, appcashflow.ProfileConfigVariant{
			Key:         item.Key,
			Label:       item.Label,
			Description: item.Description,
			Values:      values,
		})
	}
	activeVariantKey := strings.TrimSpace(input.ActiveVariantKey)
	if activeVariantKey == "" && len(spec.Variants) > 0 {
		activeVariantKey = spec.Variants[0].Key
	}
	_, err := s.cashflowProfiles.Update(profileID, key, appcashflow.ProfileConfig{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Defaults:      appcashflow.ProfileConfigDefaults{CashflowFormat: activeVariantKey},
		Variants:      variants,
	})
	return err
}

func (s *PageService) updateCashflowBillProfile(profileID string, key appcashflowbill.ProfileConfigKey, input FormBlockInput) error {
	spec := s.cashflowBillProfiles.Spec(key)
	variants := make([]appcashflowbill.ProfileConfigVariant, 0, len(spec.Variants))
	for _, item := range spec.Variants {
		values := cloneStringMap(item.Values)
		if current, ok := input.Variants[item.Key]; ok {
			for fieldKey, fieldValue := range current {
				values[strings.TrimSpace(fieldKey)] = strings.TrimSpace(fieldValue)
			}
		}
		variants = append(variants, appcashflowbill.ProfileConfigVariant{
			Key:         item.Key,
			Label:       item.Label,
			Description: item.Description,
			Values:      values,
		})
	}
	activeVariantKey := strings.TrimSpace(input.ActiveVariantKey)
	if activeVariantKey == "" && len(spec.Variants) > 0 {
		activeVariantKey = spec.Variants[0].Key
	}
	_, err := s.cashflowBillProfiles.Update(profileID, key, appcashflowbill.ProfileConfig{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Defaults:      appcashflowbill.ProfileConfigDefaults{CashflowFormat: activeVariantKey},
		Variants:      variants,
	})
	return err
}

func (s *PageService) updateFPCoretaxProfile(profileID string, key appfpcoretax.ProfileConfigKey, input FormBlockInput) error {
	spec := s.fpCoretaxProfiles.Spec(key)
	fields := make([]appfpcoretax.ProfileConfigField, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		fields = append(fields, appfpcoretax.ProfileConfigField{
			Key:         item.Key,
			Label:       item.Label,
			Required:    item.Required,
			Value:       strings.TrimSpace(input.Values[item.Key]),
			Description: item.Description,
			Kind:        item.Kind,
			Group:       item.Group,
		})
	}

	_, err := s.fpCoretaxProfiles.Update(profileID, key, appfpcoretax.ProfileConfig{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Fields:        fields,
	})
	return err
}

func (s *PageService) uploadBuyerRegistry(profileID string, filename string, fileSize int64, tempPath string) (int, []parser.ValidationIssue, error) {
	if ok, reason := s.buyerRegistry.IsAcceptedUpload(filename, fileSize); !ok {
		spec := s.buyerRegistry.Spec()
		return 0, nil, fmt.Errorf("%s (format: %s, maksimal: %dMB)", fallbackValue(strings.TrimSpace(reason), "file tidak memenuhi format upload buyer registry"), strings.Join(spec.Upload.AcceptedExtensions, ", "), spec.Upload.MaxFileSizeMB)
	}
	count, issues, err := s.buyerRegistry.Update(profileID, tempPath)
	if err != nil {
		var schemaErr *parser.BuyerSchemaMismatchError
		if errors.As(err, &schemaErr) {
			missing := append([]string(nil), schemaErr.MissingColumns...)
			sort.Strings(missing)
			required := requiredBuyerColumns(s.buyerRegistry.Spec().Columns)
			return 0, nil, fmt.Errorf("schema buyer registry tidak sesuai. Kolom wajib: %s. Kolom hilang: %s", strings.Join(required, ", "), strings.Join(missing, ", "))
		}
		return 0, nil, err
	}
	return count, sanitizeValidationIssues(issues), nil
}

func (s *PageService) uploadTaxAccounts(profileID string, filename string, fileSize int64, tempPath string) (int, []parser.ValidationIssue, error) {
	if ok, reason := s.taxAccounts.IsAcceptedUpload(filename, fileSize); !ok {
		spec := s.taxAccounts.Spec()
		return 0, nil, fmt.Errorf("%s (format: %s, maksimal: %dMB)", fallbackValue(strings.TrimSpace(reason), "file tidak memenuhi format upload tax accounts"), strings.Join(spec.Upload.AcceptedExtensions, ", "), spec.Upload.MaxFileSizeMB)
	}
	count, issues, err := s.taxAccounts.Update(profileID, tempPath)
	if err != nil {
		var schemaErr *parser.TaxAccountSchemaMismatchError
		if errors.As(err, &schemaErr) {
			missing := append([]string(nil), schemaErr.MissingColumns...)
			sort.Strings(missing)
			required := requiredTaxAccountColumns(s.taxAccounts.Spec().Columns)
			return 0, nil, fmt.Errorf("schema tax accounts tidak sesuai. Kolom wajib: %s. Kolom hilang: %s", strings.Join(required, ", "), strings.Join(missing, ", "))
		}
		return 0, nil, err
	}
	return count, sanitizeValidationIssues(issues), nil
}

func (s *PageService) uploadCategoryAccounts(profileID string, filename string, fileSize int64, tempPath string) (int, []parser.ValidationIssue, error) {
	if ok, reason := s.categoryAccounts.IsAcceptedUpload(filename, fileSize); !ok {
		spec := s.categoryAccounts.Spec()
		return 0, nil, fmt.Errorf("%s (format: %s, maksimal: %dMB)", fallbackValue(strings.TrimSpace(reason), "file tidak memenuhi format upload category accounts"), strings.Join(spec.Upload.AcceptedExtensions, ", "), spec.Upload.MaxFileSizeMB)
	}
	count, issues, err := s.categoryAccounts.Update(profileID, tempPath)
	if err != nil {
		var schemaErr *parser.TaxAccountSchemaMismatchError
		if errors.As(err, &schemaErr) {
			missing := append([]string(nil), schemaErr.MissingColumns...)
			sort.Strings(missing)
			required := requiredTaxAccountColumns(s.categoryAccounts.Spec().Columns)
			return 0, nil, fmt.Errorf("schema category accounts tidak sesuai. Kolom wajib: %s. Kolom hilang: %s", strings.Join(required, ", "), strings.Join(missing, ", "))
		}
		return 0, nil, err
	}
	return count, sanitizeValidationIssues(issues), nil
}

func (s *PageService) uploadFPCoretaxRegistry(profileID string, key appfpcoretax.RelationRegistryKey, filename string, fileSize int64, tempPath string) (int, []parser.ValidationIssue, error) {
	if ok, reason := s.fpCoretaxRelations.IsAcceptedUpload(key, filename, fileSize); !ok {
		spec := s.fpCoretaxRelations.Spec(key)
		return 0, nil, fmt.Errorf("%s (format: %s, maksimal: %dMB)", fallbackValue(strings.TrimSpace(reason), "file tidak memenuhi format upload registry"), strings.Join(spec.Upload.AcceptedExtensions, ", "), spec.Upload.MaxFileSizeMB)
	}
	count, issues, err := s.fpCoretaxRelations.Update(profileID, key, tempPath)
	if err != nil {
		var schemaErr *parser.FPCoretaxRelationSchemaMismatchError
		if errors.As(err, &schemaErr) {
			missing := append([]string(nil), schemaErr.MissingColumns...)
			sort.Strings(missing)
			required := requiredFPCoretaxRegistryColumns(s.fpCoretaxRelations.Spec(key).Columns)
			return 0, nil, fmt.Errorf("schema registry tidak sesuai. Kolom wajib: %s. Kolom hilang: %s", strings.Join(required, ", "), strings.Join(missing, ", "))
		}
		return 0, nil, err
	}
	return count, sanitizeValidationIssues(issues), nil
}

func requestOptionsToForm(items []appbukpot.RequestConfigFieldOption) []FormFieldOptionSpec {
	if len(items) == 0 {
		return nil
	}
	out := make([]FormFieldOptionSpec, 0, len(items))
	for _, item := range items {
		out = append(out, FormFieldOptionSpec{Label: item.Label, Value: item.Value})
	}
	return out
}

func cashflowOptionsToForm(items []appcashflow.ProfileConfigFieldOption) []FormFieldOptionSpec {
	if len(items) == 0 {
		return nil
	}
	out := make([]FormFieldOptionSpec, 0, len(items))
	for _, item := range items {
		out = append(out, FormFieldOptionSpec{Label: item.Label, Value: item.Value})
	}
	return out
}

func cashflowBillOptionsToForm(items []appcashflowbill.ProfileConfigFieldOption) []FormFieldOptionSpec {
	if len(items) == 0 {
		return nil
	}
	out := make([]FormFieldOptionSpec, 0, len(items))
	for _, item := range items {
		out = append(out, FormFieldOptionSpec{Label: item.Label, Value: item.Value})
	}
	return out
}

func actionSuggestionsToAny(items []appbukpot.ActionProfileSuggestionSpec) []any {
	if len(items) == 0 {
		return nil
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

func fpCoretaxSuggestionsToAny(items []appfpcoretax.ProfileConfigSuggestion) []any {
	if len(items) == 0 {
		return nil
	}
	out := make([]any, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out
}

func fpCoretaxOptionsToForm(items []appfpcoretax.ProfileConfigFieldOption) []FormFieldOptionSpec {
	if len(items) == 0 {
		return nil
	}
	out := make([]FormFieldOptionSpec, 0, len(items))
	for _, item := range items {
		out = append(out, FormFieldOptionSpec{Label: item.Label, Value: item.Value})
	}
	return out
}

func buyerColumnsToRegistry(items []parser.BuyerRegistryColumnSpec) []RegistryColumnSpec {
	out := make([]RegistryColumnSpec, 0, len(items))
	for _, item := range items {
		out = append(out, RegistryColumnSpec{
			Key:         item.Key,
			Header:      item.Header,
			Required:    item.Required,
			Description: item.Description,
		})
	}
	return out
}

func taxColumnsToRegistry(items []parser.TaxAccountColumnSpec) []RegistryColumnSpec {
	out := make([]RegistryColumnSpec, 0, len(items))
	for _, item := range items {
		out = append(out, RegistryColumnSpec{
			Key:         item.Key,
			Header:      item.Header,
			Required:    item.Required,
			Description: item.Description,
		})
	}
	return out
}

func fpCoretaxRegistryColumnsToRegistry(items []parser.FPCoretaxRelationColumnSpec) []RegistryColumnSpec {
	out := make([]RegistryColumnSpec, 0, len(items))
	for _, item := range items {
		out = append(out, RegistryColumnSpec{
			Key:         item.Key,
			Header:      item.Header,
			Required:    item.Required,
			Description: item.Description,
		})
	}
	return out
}

func sanitizeValidationIssues(issues []parser.ValidationIssue) []parser.ValidationIssue {
	if len(issues) == 0 {
		return nil
	}
	safe := make([]parser.ValidationIssue, 0, len(issues))
	for _, issue := range issues {
		safe = append(safe, parser.ValidationIssue{
			Row:     issue.Row,
			Field:   issue.Field,
			Message: issue.Message,
		})
	}
	return safe
}

func requiredBuyerColumns(columns []parser.BuyerRegistryColumnSpec) []string {
	out := make([]string, 0, len(columns))
	for _, item := range columns {
		if !item.Required {
			continue
		}
		label := strings.TrimSpace(item.Header)
		if label == "" {
			continue
		}
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}

func requiredTaxAccountColumns(columns []parser.TaxAccountColumnSpec) []string {
	out := make([]string, 0, len(columns))
	for _, item := range columns {
		if !item.Required {
			continue
		}
		label := strings.TrimSpace(item.Header)
		if label == "" {
			continue
		}
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}

func requiredFPCoretaxRegistryColumns(columns []parser.FPCoretaxRelationColumnSpec) []string {
	out := make([]string, 0, len(columns))
	for _, item := range columns {
		if !item.Required {
			continue
		}
		label := strings.TrimSpace(item.Header)
		if label == "" {
			continue
		}
		out = append(out, label)
	}
	sort.Strings(out)
	return out
}

func cashflowRequiredFieldKeys(variantKey string) []string {
	switch strings.TrimSpace(strings.ToLower(variantKey)) {
	case "influencer":
		return []string{"outputFilename", "chequeAccount", "date", "information", "total", "defaultIAccountCode", "defaultBAccountCode"}
	default:
		return []string{"outputFilename", "chequeAccount", "date", "information", "total", "coa", "remarkDelimiter", "otherCostsAccountCode"}
	}
}

func parsePositiveInt(raw string, fallback int) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func normalizeBool(raw string) bool {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func cloneStringMap(source map[string]string) map[string]string {
	out := make(map[string]string, len(source))
	for key, value := range source {
		normalized := strings.TrimSpace(key)
		if normalized == "" {
			continue
		}
		out[normalized] = strings.TrimSpace(value)
	}
	return out
}

func fallbackValue(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}
