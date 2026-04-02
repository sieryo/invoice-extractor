package document

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/specutil"
)

type FeatureFlags struct {
	EnableCashflowXLSX bool
}

var featureFlags = FeatureFlags{}

func SetFeatureFlags(flags FeatureFlags) {
	featureFlags = flags
}

func GetFeatureFlags() FeatureFlags {
	return featureFlags
}

func IsCollectionKindEnabled(collectionKind CollectionKind) bool {
	switch collectionKind {
	case CollectionKindCashflowImport:
		return featureFlags.EnableCashflowXLSX
	default:
		return true
	}
}

const (
	FormFieldKindText     = "text"
	FormFieldKindTextarea = "textarea"
	FormFieldKindNumber   = "number"
	FormFieldKindSelect   = "select"
	FormFieldKindCheckbox = "checkbox"
	FormFieldKindTemplate = "template"
	FormFieldKindMapping  = "mapping"

	FormFieldRuleRequiredIf = "required_if"
)

type UploadRuleSpec struct {
	AcceptExtensions []string `json:"acceptExtensions"`
	AcceptMIMETypes  []string `json:"acceptMimeTypes"`
	MaxChunkMB       int      `json:"maxChunkMB"`
	MaxFilesPerBatch int      `json:"maxFilesPerBatch"`
}

type ArtifactRuleSpec struct {
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
}

type IngestRuleSpec struct {
	KeepRaw            bool               `json:"keepRaw"`
	DeleteTempAfterRun bool               `json:"deleteTempAfterRun"`
	Artifacts          []ArtifactRuleSpec `json:"artifacts"`
}

type ActionStateSpec struct {
	Enabled bool   `json:"enabled"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type ActionPresentationSpec struct {
	Mode  string `json:"mode"`
	Width string `json:"width,omitempty"`
}

type FormSpec struct {
	Title         string                 `json:"title,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Sections      []FormSectionSpec      `json:"sections"`
	VariantGroups []FormVariantGroupSpec `json:"variantGroups,omitempty"`
}

type FormSectionSpec struct {
	Key         string          `json:"key"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Columns     int             `json:"columns,omitempty"`
	Fields      []FormFieldSpec `json:"fields"`
}

type FormVariantGroupSpec struct {
	Key               string            `json:"key"`
	FieldKey          string            `json:"fieldKey"`
	Label             string            `json:"label"`
	Description       string            `json:"description,omitempty"`
	Component         string            `json:"component,omitempty"`
	DefaultVariantKey string            `json:"defaultVariantKey,omitempty"`
	Variants          []FormVariantSpec `json:"variants"`
}

type FormVariantSpec struct {
	Key         string            `json:"key"`
	Label       string            `json:"label"`
	Description string            `json:"description,omitempty"`
	Sections    []FormSectionSpec `json:"sections"`
	Values      map[string]any    `json:"values,omitempty"`
}

type FormFieldSpec struct {
	Key          string                    `json:"key"`
	Kind         string                    `json:"kind"`
	Label        string                    `json:"label"`
	Required     bool                      `json:"required"`
	DefaultValue any                       `json:"defaultValue,omitempty"`
	Options      []FormFieldOption         `json:"options,omitempty"`
	Rules        []FormFieldRuleSpec       `json:"rules,omitempty"`
	Interaction  *FormFieldInteractionSpec `json:"interaction,omitempty"`
	State        FormFieldStateSpec        `json:"state"`
	Suggestions  []FormSuggestionSpec      `json:"suggestions,omitempty"`
	HelpText     string                    `json:"helpText,omitempty"`
	Placeholder  string                    `json:"placeholder,omitempty"`
	Span         int                       `json:"span,omitempty"`
}

type FormFieldOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type FormSuggestionSpec struct {
	Token       string `json:"token"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
}

type FormFieldStateSpec struct {
	Visible  bool   `json:"visible"`
	Disabled bool   `json:"disabled"`
	Message  string `json:"message,omitempty"`
}

type FormFieldRuleSpec struct {
	Type    string `json:"type"`
	Field   string `json:"field"`
	Equals  string `json:"equals,omitempty"`
	Message string `json:"message,omitempty"`
}

type FormFieldInteractionSpec struct {
	Trigger       string   `json:"trigger"`
	Effect        string   `json:"effect"`
	ResetSections []string `json:"resetSections,omitempty"`
	PreserveKeys  []string `json:"preserveKeys,omitempty"`
}

type ActionSelectionSpec struct {
	Mode           string   `json:"mode"`
	AllowCheckAll  bool     `json:"allowCheckAll"`
	AllowedStatus  []string `json:"allowedStatuses"`
	MinDocumentCnt int      `json:"minDocuments"`
	MaxDocumentCnt int      `json:"maxDocuments,omitempty"`
}

type ActionOutputSpec struct {
	Kind       string `json:"kind"`
	MimeType   string `json:"mimeType,omitempty"`
	Ext        string `json:"ext,omitempty"`
	DownloadOK bool   `json:"downloadable"`
}

type ActionRequirementSpec struct {
	Key       string `json:"key"`
	Label     string `json:"label"`
	Required  bool   `json:"required"`
	Satisfied bool   `json:"satisfied"`
	Code      string `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
}

type ActionDetailSpec struct {
	Summary  string   `json:"summary,omitempty"`
	Bullets  []string `json:"bullets,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type ActionMasterDataSpec struct {
	Relative     string   `json:"relative"`
	LookupKey    string   `json:"lookupKey"`
	RequiredCols []string `json:"requiredCols,omitempty"`
}

type ActionArtifactInputSpec struct {
	Key              string   `json:"key"`
	ValueType        string   `json:"valueType,omitempty"`
	Label            string   `json:"label"`
	Required         bool     `json:"required"`
	Description      string   `json:"description,omitempty"`
	AcceptExtensions []string `json:"acceptExtensions,omitempty"`
	AcceptMIMETypes  []string `json:"acceptMimeTypes,omitempty"`
}

type ActionColumnSpec struct {
	Key      string              `json:"key"`
	Label    string              `json:"label"`
	Type     string              `json:"type"`
	Required bool                `json:"required"`
	Aliases  []string            `json:"aliases,omitempty"`
	Group    string              `json:"group,omitempty"`
	Rules    []FormFieldRuleSpec `json:"rules,omitempty"`
}

type ActionSpec struct {
	CollectionKind string                          `json:"collectionKind,omitempty"`
	ActionType     string                          `json:"actionType"`
	Label          string                          `json:"label"`
	Description    string                          `json:"description,omitempty"`
	State          ActionStateSpec                 `json:"state"`
	Presentation   ActionPresentationSpec          `json:"presentation"`
	Selection      ActionSelectionSpec             `json:"selection"`
	Form           *FormSpec                       `json:"form,omitempty"`
	Detail         *ActionDetailSpec               `json:"detail,omitempty"`
	Requirements   []ActionRequirementSpec         `json:"requirements,omitempty"`
	MasterData     map[string]ActionMasterDataSpec `json:"masterData,omitempty"`
	ArtifactInputs []ActionArtifactInputSpec       `json:"artifactInputs,omitempty"`
	Columns        []ActionColumnSpec              `json:"columns,omitempty"`
	Outputs        []ActionOutputSpec              `json:"outputs"`
}

type CollectionSpec struct {
	CollectionKind CollectionKind `json:"collectionKind"`
	SourceFormat   SourceFormat   `json:"sourceFormat"`
	Label          string         `json:"label"`
	Description    string         `json:"description,omitempty"`
	Availability   *CollectionAvailabilitySpec `json:"availability,omitempty"`
	Upload         UploadRuleSpec `json:"upload"`
	Ingest         IngestRuleSpec `json:"ingest"`
	Actions        []ActionSpec   `json:"actions"`
}

type CollectionAvailabilitySpec struct {
	Enabled   bool   `json:"enabled"`
	ModuleKey string `json:"moduleKey,omitempty"`
	Label     string `json:"label,omitempty"`
	Code      string `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
}

type CreateCollectionSourceFormatSpec struct {
	Value       SourceFormat `json:"value"`
	Label       string       `json:"label"`
	Description string       `json:"description,omitempty"`
}

type CreateCollectionKindSpec struct {
	CollectionKind CollectionKind `json:"collectionKind"`
	Label          string         `json:"label"`
	Description    string         `json:"description,omitempty"`
	SourceFormats  []SourceFormat `json:"sourceFormats"`
	PrimaryActions []string       `json:"primaryActions,omitempty"`
}

type CreateCollectionSpec struct {
	DefaultSourceFormat SourceFormat                       `json:"defaultSourceFormat,omitempty"`
	SourceFormats       []CreateCollectionSourceFormatSpec `json:"sourceFormats"`
	CollectionKinds     []CreateCollectionKindSpec         `json:"collectionKinds"`
}

func buildHeaderRowField(label string, helpText string) FormFieldSpec {
	options := make([]FormFieldOption, 0, 10)
	for _, row := range specutil.HeaderRowNumbers(10) {
		value := strconv.Itoa(row)
		options = append(options, FormFieldOption{
			Label: value,
			Value: value,
		})
	}

	return FormFieldSpec{
		Key:          "headerRowNumber",
		Kind:         FormFieldKindSelect,
		Label:        label,
		Required:     true,
		DefaultValue: "1",
		Options:      options,
		HelpText:     helpText,
		State: FormFieldStateSpec{
			Visible: true,
		},
	}
}

func buildCashflowMetaSections(isReceive bool) []FormSectionSpec {
	return []FormSectionSpec{
		{
			Key:     "meta",
			Title:   "Meta",
			Columns: 1,
			Fields: []FormFieldSpec{
				{
					Key:          "cashflowFormat",
					Kind:         FormFieldKindSelect,
					Label:        "Format Cashflow",
					Required:     true,
					DefaultValue: "standard",
					Options: []FormFieldOption{
						{Label: "Standard", Value: "standard"},
						{Label: "Influencer", Value: "influencer"},
					},
					State: FormFieldStateSpec{
						Visible: false,
					},
				},
			},
		},
	}
}

func buildCashflowVariantGroup(isReceive bool) FormVariantGroupSpec {
	return FormVariantGroupSpec{
		Key:               "cashflow_variant",
		FieldKey:          "cashflowFormat",
		Label:             "Format Cashflow",
		Description:       "Pilih format cashflow. Form tetap sama, lalu nilai default akan mengikuti format yang aktif.",
		Component:         "tabs",
		DefaultVariantKey: "standard",
		Variants: []FormVariantSpec{
			{
				Key:         "standard",
				Label:       "Standard",
				Description: "Format cashflow standard.",
				Values:      buildCashflowVariantValues(isReceive, "standard"),
			},
			{
				Key:         "influencer",
				Label:       "Influencer",
				Description: "Format cashflow influencer.",
				Values:      buildCashflowVariantValues(isReceive, "influencer"),
			},
		},
	}
}

func buildCashflowSections(isReceive bool) []FormSectionSpec {
	return []FormSectionSpec{
		{
			Key:     "source",
			Title:   "Sumber Data",
			Columns: 2,
			Fields: []FormFieldSpec{
				{
					Key:          "sheetName",
					Kind:         FormFieldKindSelect,
					Label:        "Sheet",
					Required:     true,
					DefaultValue: "",
					HelpText:     "Pilih dokumen cashflow terlebih dahulu untuk melihat sheet yang tersedia.",
					State: FormFieldStateSpec{
						Visible:  true,
						Disabled: true,
					},
				},
				buildHeaderRowField("Baris Header", "Nomor baris header pada sheet yang dipilih."),
			},
		},
		{
			Key:     "output",
			Title:   "Output",
			Columns: 2,
			Fields: []FormFieldSpec{
				{
					Key:          "outputFilename",
					Kind:         FormFieldKindText,
					Label:        "Nama Output",
					Required:     true,
					DefaultValue: map[bool]string{true: "receive_money", false: "spend_money"}[isReceive],
					Placeholder:  map[bool]string{true: "receive_money", false: "spend_money"}[isReceive],
					HelpText:     "Tanpa ekstensi file.",
					State: FormFieldStateSpec{
						Visible: true,
					},
				},
				{
					Key:          "chequeAccount",
					Kind:         FormFieldKindText,
					Label:        map[bool]string{true: "Deposit Account", false: "Cheque Account"}[isReceive],
					Required:     true,
					DefaultValue: "12021",
					HelpText:     map[bool]string{true: "Akun bank/deposit utama untuk file MYOB.", false: "Akun cheque utama untuk file MYOB."}[isReceive],
					State:        FormFieldStateSpec{Visible: true},
				},
			},
		},
		buildCashflowParameterSection(),
		{
			Key:     "mapping",
			Title:   "Mapping Header",
			Columns: 2,
			Fields:  buildCashflowMappingFields(),
		},
	}
}

func buildCashflowParameterSection() FormSectionSpec {
	return FormSectionSpec{
		Key:     "parameters",
		Title:   specutil.ParameterActionSectionTitle,
		Columns: 2,
		Fields: []FormFieldSpec{
			{
				Key:          "remarkDelimiter",
				Kind:         FormFieldKindText,
				Label:        "Remark Delimiter",
				Required:     false,
				DefaultValue: "*",
				Placeholder:  "*",
				HelpText:     "Dipakai untuk memecah catatan biaya lainnya.",
				State:        FormFieldStateSpec{Visible: true},
			},
			{
				Key:          "otherCostsAccountCode",
				Kind:         FormFieldKindText,
				Label:        "Kode Akun Biaya Lain",
				Required:     false,
				DefaultValue: "62099",
				HelpText:     "Dipakai saat terdapat komponen biaya lainnya.",
				State:        FormFieldStateSpec{Visible: true},
			},
			{
				Key:          "defaultIAccountCode",
				Kind:         FormFieldKindText,
				Label:        "Default Influencer Account Code",
				Required:     false,
				DefaultValue: "",
				HelpText:     "Dipakai saat format influencer memilih account influencer.",
				State:        FormFieldStateSpec{Visible: true},
			},
			{
				Key:          "defaultBAccountCode",
				Kind:         FormFieldKindText,
				Label:        "Default Bank Account Code",
				Required:     false,
				DefaultValue: "",
				HelpText:     "Dipakai saat format influencer mendeteksi transaksi bank.",
				State:        FormFieldStateSpec{Visible: true},
			},
		},
	}
}

func buildCashflowMappingFields() []FormFieldSpec {
	return []FormFieldSpec{
		{Key: "date", Kind: FormFieldKindText, Label: "Tanggal", Required: true, DefaultValue: "Tanggal", State: FormFieldStateSpec{Visible: true}},
		{Key: "information", Kind: FormFieldKindText, Label: "Keterangan", Required: true, DefaultValue: "note", State: FormFieldStateSpec{Visible: true}},
		{
			Key:          "coa",
			Kind:         FormFieldKindText,
			Label:        "Chart of Account",
			Required:     false,
			DefaultValue: "coa",
			HelpText:     "Wajib untuk format standard. Pada format influencer, akun utama tidak diambil dari COA/WHT CoA tetapi dari Default Influencer/Admin Bank Account Code.",
			Rules: []FormFieldRuleSpec{
				{
					Type:    FormFieldRuleRequiredIf,
					Field:   "cashflowFormat",
					Equals:  "standard",
					Message: "Chart of Account wajib diisi untuk format standard",
				},
			},
			State: FormFieldStateSpec{Visible: true},
		},
		{Key: "otherCost", Kind: FormFieldKindText, Label: "Biaya Lainnya", Required: false, DefaultValue: "By Lainnya", State: FormFieldStateSpec{Visible: true}},
		{Key: "pp23", Kind: FormFieldKindText, Label: "PP 23", Required: false, DefaultValue: "PP 23", State: FormFieldStateSpec{Visible: true}},
		{Key: "pph15", Kind: FormFieldKindText, Label: "PPh 15%", Required: false, DefaultValue: "PPh 15%", State: FormFieldStateSpec{Visible: true}},
		{Key: "pph21", Kind: FormFieldKindText, Label: "PPH 21", Required: false, DefaultValue: "PPH 21", State: FormFieldStateSpec{Visible: true}},
		{Key: "pph23", Kind: FormFieldKindText, Label: "PPH 23", Required: false, DefaultValue: "PPH 23", State: FormFieldStateSpec{Visible: true}},
		{Key: "pph42", Kind: FormFieldKindText, Label: "PPH 4 (2)", Required: false, DefaultValue: "PPH 4(2)", State: FormFieldStateSpec{Visible: true}},
		{Key: "ppn", Kind: FormFieldKindText, Label: "PPN", Required: false, DefaultValue: "PPN", State: FormFieldStateSpec{Visible: true}},
		{Key: "remark", Kind: FormFieldKindText, Label: "Catatan", Required: false, DefaultValue: "catatan", State: FormFieldStateSpec{Visible: true}},
		{Key: "total", Kind: FormFieldKindText, Label: "Total", Required: true, DefaultValue: "idr", State: FormFieldStateSpec{Visible: true}},
	}
}

func buildCashflowBillSections(isReceive bool) []FormSectionSpec {
	accountLabel := "Payment Account"
	accountHelpText := "Akun payment utama untuk file MYOB."
	outputDefault := "pay_bills"
	if isReceive {
		accountLabel = "Deposit Account"
		accountHelpText = "Akun deposit utama untuk file MYOB."
		outputDefault = "receive_payments"
	}

	return []FormSectionSpec{
		{
			Key:     "source",
			Title:   "Sumber Data",
			Columns: 2,
			Fields: []FormFieldSpec{
				{
					Key:          "sheetName",
					Kind:         FormFieldKindSelect,
					Label:        "Sheet",
					Required:     true,
					DefaultValue: "",
					HelpText:     "Pilih dokumen cashflow terlebih dahulu untuk melihat sheet yang tersedia.",
					State: FormFieldStateSpec{
						Visible:  true,
						Disabled: true,
					},
				},
				buildHeaderRowField("Baris Header", "Nomor baris header pada sheet yang dipilih."),
			},
		},
		{
			Key:     "output",
			Title:   "Output",
			Columns: 2,
			Fields: []FormFieldSpec{
				{
					Key:          "outputFilename",
					Kind:         FormFieldKindText,
					Label:        "Nama Output",
					Required:     true,
					DefaultValue: outputDefault,
					Placeholder:  outputDefault,
					HelpText:     "Tanpa ekstensi file.",
					State:        FormFieldStateSpec{Visible: true},
				},
				{
					Key:          "chequeAccount",
					Kind:         FormFieldKindText,
					Label:        accountLabel,
					Required:     true,
					DefaultValue: "12021",
					HelpText:     accountHelpText,
					State:        FormFieldStateSpec{Visible: true},
				},
			},
		},
		{
			Key:     "mapping",
			Title:   "Mapping Header",
			Columns: 2,
			Fields: []FormFieldSpec{
				{Key: "date", Kind: FormFieldKindText, Label: "Tanggal", Required: true, DefaultValue: "date", State: FormFieldStateSpec{Visible: true}},
				{Key: "category", Kind: FormFieldKindText, Label: "Category", Required: true, DefaultValue: "Category", State: FormFieldStateSpec{Visible: true}},
				{Key: "information", Kind: FormFieldKindText, Label: "Keterangan", Required: true, DefaultValue: "Note", State: FormFieldStateSpec{Visible: true}},
				{Key: "partyName", Kind: FormFieldKindText, Label: "Nama Customer / Supplier", Required: true, DefaultValue: "nama customer / supplier", State: FormFieldStateSpec{Visible: true}},
				{Key: "total", Kind: FormFieldKindText, Label: "Total", Required: true, DefaultValue: "idr", State: FormFieldStateSpec{Visible: true}},
			},
		},
	}
}

func buildCashflowVariantValues(isReceive bool, variant string) map[string]any {
	values := map[string]any{
		"sheetName":             "",
		"headerRowNumber":       "1",
		"outputFilename":        map[bool]string{true: "receive_money", false: "spend_money"}[isReceive],
		"chequeAccount":         "12021",
		"remarkDelimiter":       "*",
		"otherCostsAccountCode": "62099",
		"defaultIAccountCode":   "",
		"defaultBAccountCode":   "",
		"date":                  "Tanggal",
		"information":           "note",
		"coa":                   "coa",
		"otherCost":             "By Lainnya",
		"pp23":                  "PP 23",
		"pph15":                 "PPh 15%",
		"pph21":                 "PPH 21",
		"pph23":                 "PPH 23",
		"pph42":                 "PPH 4(2)",
		"ppn":                   "PPN",
		"remark":                "catatan",
		"total":                 "idr",
	}

	if variant == "influencer" {
		values["defaultIAccountCode"] = "62004"
		values["defaultBAccountCode"] = "90900"
		values["date"] = "*Posting Date: # date"
		values["information"] = "Notes"
		values["coa"] = "WHT CoA"
		values["pph42"] = "PPh 4 (2)"
		values["remark"] = "WHT"
	}

	return values
}

func (s CollectionSpec) FindActionSpec(actionType string) (ActionSpec, bool) {
	target := strings.ToLower(strings.TrimSpace(actionType))
	if target == "" {
		return ActionSpec{}, false
	}
	for _, action := range s.Actions {
		if strings.EqualFold(action.ActionType, target) {
			return action, true
		}
	}
	return ActionSpec{}, false
}

func BuildCollectionSpec(collectionKind CollectionKind) (CollectionSpec, bool) {
	if !IsCollectionKindEnabled(collectionKind) {
		return CollectionSpec{}, false
	}

	switch collectionKind {
	case CollectionKindInvoiceCompany:
		return CollectionSpec{
			CollectionKind: collectionKind,
			SourceFormat:   SourceFormatPDF,
			Label:          "Invoice",
			Description:    "PDF invoice document for extraction and e-Faktur export.",
			Upload: UploadRuleSpec{
				AcceptExtensions: []string{".pdf"},
				AcceptMIMETypes:  []string{"application/pdf"},
				MaxChunkMB:       15,
				MaxFilesPerBatch: 2000,
			},
			Ingest: IngestRuleSpec{
				KeepRaw:            false,
				DeleteTempAfterRun: true,
				Artifacts: []ArtifactRuleSpec{
					{Kind: "normalized", Required: true},
					{Kind: "audit", Required: false},
				},
			},
			Actions: []ActionSpec{
				{
					CollectionKind: string(collectionKind),
					ActionType:     "export_faktur_keluaran",
					Label:          "Export e-Faktur",
					Description:    "Export invoice terpilih ke format e-Faktur keluaran Coretax.",
					State: ActionStateSpec{
						Enabled: true,
					},
					Presentation: ActionPresentationSpec{
						Mode:  "inline",
						Width: "md",
					},
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  true,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Form: &FormSpec{
						Title: "Pengaturan Export",
						Sections: []FormSectionSpec{
							{
								Key:     "general",
								Title:   "Umum",
								Columns: 1,
								Fields: []FormFieldSpec{
									{
										Key:          "filenamePrefix",
										Kind:         FormFieldKindText,
										Label:        "Filename Prefix",
										Required:     false,
										DefaultValue: "faktur-keluaran",
										HelpText:     "Optional prefix for exported file name.",
										Placeholder:  "faktur-keluaran",
										State: FormFieldStateSpec{
											Visible: true,
										},
									},
								},
							},
						},
					},
					Requirements: []ActionRequirementSpec{
						{
							Key:      "buyerRegistry",
							Label:    "Buyer Registry",
							Required: true,
						},
					},
					Outputs: []ActionOutputSpec{
						{
							Kind:       "file",
							MimeType:   "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
							Ext:        "xlsx",
							DownloadOK: true,
						},
					},
				},
			},
		}, true
	case CollectionKindTaxInvoiceCoretax:
		return CollectionSpec{
			CollectionKind: collectionKind,
			SourceFormat:   SourceFormatPDF,
			Label:          "Faktur Pajak",
			Description:    "Dokumen PDF faktur pajak Coretax untuk ekstraksi dan action lanjutan.",
			Upload: UploadRuleSpec{
				AcceptExtensions: []string{".pdf"},
				AcceptMIMETypes:  []string{"application/pdf"},
				MaxChunkMB:       15,
				MaxFilesPerBatch: 2000,
			},
			Ingest: IngestRuleSpec{
				KeepRaw:            true,
				DeleteTempAfterRun: true,
				Artifacts: []ArtifactRuleSpec{
					{Kind: "raw", Required: true},
					{Kind: "normalized", Required: true},
					{Kind: "audit", Required: false},
				},
			},
			Actions: []ActionSpec{
				{
					CollectionKind: string(collectionKind),
					ActionType:     "rename_tax_invoice",
					Label:          "Rename Faktur Pajak",
					Description:    "Ganti nama file faktur pajak berdasarkan template placeholder dan hasilkan ZIP.",
					State: ActionStateSpec{
						Enabled: true,
					},
					Presentation: ActionPresentationSpec{
						Mode:  "inline",
						Width: "md",
					},
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  true,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Form: &FormSpec{
						Title: "Pengaturan Rename",
						Sections: []FormSectionSpec{
							{
								Key:     "main",
								Title:   specutil.ParameterActionSectionTitle,
								Columns: 1,
								Fields: []FormFieldSpec{
									{
										Key:          "filenameTemplate",
										Kind:         FormFieldKindTemplate,
										Label:        "Template Nama File",
										Required:     true,
										DefaultValue: "{{references}} - {{buyerName}}",
										Suggestions: []FormSuggestionSpec{
											{Token: "references", Label: "Referensi", Description: "Nilai referensi dari faktur pajak", Example: "{{references}}"},
											{Token: "invoiceNumber", Label: "Nomor Faktur", Description: "Nomor seri faktur pajak", Example: "{{invoiceNumber}}"},
											{Token: "buyerName", Label: "Nama Pembeli", Description: "Nama pembeli dari faktur pajak", Example: "{{buyerName}}"},
											{Token: "buyerNPWP", Label: "NPWP Pembeli", Description: "NPWP pembeli", Example: "{{buyerNPWP}}"},
											{Token: "sellerName", Label: "Nama Penjual", Description: "Nama PKP penjual", Example: "{{sellerName}}"},
											{Token: "sellerNPWP", Label: "NPWP Penjual", Description: "NPWP PKP penjual", Example: "{{sellerNPWP}}"},
											{Token: "invoiceDate", Label: "Tanggal Faktur", Description: "Tanggal faktur (format YYYY-MM-DD)", Example: "{{invoiceDate}}"},
											{Token: "sourceName", Label: "Nama File Asal", Description: "Nama file upload tanpa ekstensi", Example: "{{sourceName}}"},
											{Token: "documentTag", Label: "Tag Dokumen", Description: "Tag hasil ekstraksi dokumen", Example: "{{documentTag}}"},
										},
										HelpText:    "Gunakan placeholder seperti {{references}} - {{buyerName}}. Ekstensi .pdf akan ditambahkan otomatis.",
										Placeholder: "{{references}} - {{buyerName}}",
										State: FormFieldStateSpec{
											Visible: true,
										},
									},
								},
							},
						},
					},
					Outputs: []ActionOutputSpec{
						{
							Kind:       "file",
							MimeType:   "application/zip",
							Ext:        "zip",
							DownloadOK: true,
						},
					},
				},
				{
					CollectionKind: string(collectionKind),
					ActionType:     "export_tax_invoice_zip",
					Label:          "Export Tax Invoice ZIP",
					Description:    "Export selected tax invoices to ZIP package.",
					State: ActionStateSpec{
						Enabled: false,
						Message: "not implemented yet",
					},
					Presentation: ActionPresentationSpec{
						Mode:  "inline",
						Width: "md",
					},
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  true,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Outputs: []ActionOutputSpec{},
				},
			},
		}, true
	case CollectionKindFPKeluaranCoretax:
		return buildFPCoretaxCollectionSpec(
			collectionKind,
			"FP Keluaran Coretax",
			"Workbook XLSX untuk export MYOB Misc Sales.",
			fpKeluaranMiscSalesActionType,
			"Misc Sales",
			"Export ke MYOB Misc Sales.",
			"Pengaturan Misc Sales",
			"Atur parameter action dan mapping header.",
			"Nama Pembeli",
			[]ActionRequirementSpec{
				{Key: "fpCoretaxDefaultProfile", Label: "Default Profil FP Keluaran", Required: true},
				{Key: "fpCoretaxRegistry", Label: "Customer Registry", Required: true},
			},
		), true
	case CollectionKindFPMasukanCoretax:
		return buildFPCoretaxCollectionSpec(
			collectionKind,
			"FP Masukan Coretax",
			"Workbook XLSX untuk export MYOB Misc Purchases.",
			fpMasukanMiscPurchasesActionType,
			"Misc Purchases",
			"Export ke MYOB Misc Purchases.",
			"Pengaturan Misc Purchases",
			"Atur parameter action dan mapping header.",
			"Nama Penjual",
			[]ActionRequirementSpec{
				{Key: "fpCoretaxDefaultProfile", Label: "Default Profil FP Masukan", Required: true},
				{Key: "fpCoretaxRegistry", Label: "Supplier Registry", Required: true},
			},
		), true
	case CollectionKindBukpotBPPU:
		return buildBukpotCollectionSpec(
			collectionKind,
			"BPPU",
			"Dokumen PDF bukti potong BPPU untuk ekstraksi data bukpot.",
		), true
	case CollectionKindBukpotBP21:
		return buildBukpotCollectionSpec(
			collectionKind,
			"BP21",
			"Dokumen PDF bukti potong BP21 untuk ekstraksi data bukpot.",
		), true
	case CollectionKindBukpotBPA1:
		return buildBukpotCollectionSpec(
			collectionKind,
			"BPA1",
			"Dokumen PDF bukti potong BPA1 untuk ekstraksi data bukpot.",
		), true
	case CollectionKindCashflowImport:
		return CollectionSpec{
			CollectionKind: collectionKind,
			SourceFormat:   SourceFormatXLSX,
			Label:          "Cashflow",
			Description:    "Spreadsheet XLSX cashflow untuk normalisasi data dan action lanjutan.",
			Upload: UploadRuleSpec{
				AcceptExtensions: []string{".xlsx"},
				AcceptMIMETypes: []string{
					"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				},
				MaxChunkMB:       20,
				MaxFilesPerBatch: 200,
			},
			Ingest: IngestRuleSpec{
				KeepRaw:            true,
				DeleteTempAfterRun: true,
				Artifacts: []ArtifactRuleSpec{
					{Kind: "raw", Required: false},
					{Kind: "normalized", Required: true},
				},
			},
			Actions: []ActionSpec{
				{
					CollectionKind: string(collectionKind),
					ActionType:     "export_cashflow_spend_money",
					Label:          "Spend Money",
					Description:    "Konversi cashflow ke format MYOB Spend Money.",
					State: ActionStateSpec{
						Enabled: true,
					},
					Presentation: ActionPresentationSpec{
						Mode:  "inline",
						Width: "xl",
					},
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  true,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Form: &FormSpec{
						Title:       "Pengaturan Spend Money",
						Description: "Pilih format cashflow lalu atur sumber data, parameter, dan mapping pada form yang sama.",
						Sections:    append(buildCashflowMetaSections(false), buildCashflowSections(false)...),
						VariantGroups: []FormVariantGroupSpec{
							buildCashflowVariantGroup(false),
						},
					},
					Detail: &ActionDetailSpec{
						Summary: "Mengubah sheet cashflow menjadi file MYOB Spend Money (.txt).",
						Bullets: []string{
							"Hanya row dengan total negatif yang akan diproses.",
							"Sheet dipilih dari dokumen yang Anda centang saat action dijalankan.",
							"Komponen pajak akan di-resolve memakai daftar nama tax yang didukung melalui Tax Accounts.",
							"Jika nomor cheque dikosongkan, MYOB bisa membuat nomor otomatis.",
						},
						Warnings: []string{
							"Header mapping dan akun default perlu sesuai dengan format cashflow yang dipakai.",
							"Pada format influencer, akun utama memakai Default Influencer/Admin Bank Account Code. Header COA/WHT CoA tidak dipakai untuk menentukan akun utama.",
						},
					},
					Requirements: []ActionRequirementSpec{
						{
							Key:      "cashflowDefaultProfile",
							Label:    "Default Profil Cashflow Spend Money",
							Required: true,
						},
						{
							Key:      "cashflowTaxAccounts",
							Label:    "Tax Accounts",
							Required: true,
						},
					},
					MasterData: map[string]ActionMasterDataSpec{
						"tax": {
							Relative:     "tax_accounts.csv",
							LookupKey:    "name",
							RequiredCols: []string{"name", "account"},
						},
					},
					Outputs: []ActionOutputSpec{
						{
							Kind:       "file",
							MimeType:   "text/plain",
							Ext:        "txt",
							DownloadOK: true,
						},
					},
				},
				{
					CollectionKind: string(collectionKind),
					ActionType:     "export_cashflow_receive_money",
					Label:          "Receive Money",
					Description:    "Konversi cashflow ke format MYOB Receive Money.",
					State: ActionStateSpec{
						Enabled: true,
					},
					Presentation: ActionPresentationSpec{
						Mode:  "inline",
						Width: "xl",
					},
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  true,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Form: &FormSpec{
						Title:       "Pengaturan Receive Money",
						Description: "Pilih format cashflow lalu atur sumber data, parameter, dan mapping pada form yang sama.",
						Sections:    append(buildCashflowMetaSections(true), buildCashflowSections(true)...),
						VariantGroups: []FormVariantGroupSpec{
							buildCashflowVariantGroup(true),
						},
					},
					Detail: &ActionDetailSpec{
						Summary: "Mengubah sheet cashflow menjadi file MYOB Receive Money (.txt).",
						Bullets: []string{
							"Hanya row dengan total positif yang akan diproses.",
							"Sheet dipilih dari dokumen yang Anda centang saat action dijalankan.",
							"Komponen pajak akan di-resolve memakai daftar nama tax yang didukung melalui Tax Accounts.",
							"Kolom akun utama memakai Deposit Account pada template MYOB.",
						},
						Warnings: []string{
							"Header mapping dan akun default perlu sesuai dengan format cashflow yang dipakai.",
							"Pada format influencer, akun utama memakai Default Influencer/Admin Bank Account Code. Header COA/WHT CoA tidak dipakai untuk menentukan akun utama.",
						},
					},
					Requirements: []ActionRequirementSpec{
						{
							Key:      "cashflowDefaultProfile",
							Label:    "Default Profil Cashflow Receive Money",
							Required: true,
						},
						{
							Key:      "cashflowTaxAccounts",
							Label:    "Tax Accounts",
							Required: true,
						},
					},
					MasterData: map[string]ActionMasterDataSpec{
						"tax": {
							Relative:     "tax_accounts.csv",
							LookupKey:    "name",
							RequiredCols: []string{"name", "account"},
						},
					},
					Outputs: []ActionOutputSpec{
						{
							Kind:       "file",
							MimeType:   "text/plain",
							Ext:        "txt",
							DownloadOK: true,
						},
					},
				},
				{
					CollectionKind: string(collectionKind),
					ActionType:     "cashflow_to_pay_bills",
					Label:          "Pay Bills",
					Description:    "Konversi cashflow ke format MYOB Pay Bills.",
					State:          ActionStateSpec{Enabled: true},
					Presentation:   ActionPresentationSpec{Mode: "inline", Width: "xl"},
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  true,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Form: &FormSpec{
						Title:       "Pengaturan Pay Bills",
						Description: "Atur workbook cashflow, upload snapshot ledger, lalu sesuaikan mapping yang dibutuhkan.",
						Sections:    buildCashflowBillSections(false),
					},
					ArtifactInputs: []ActionArtifactInputSpec{
						{
							Key:              "ledgerSnapshotRef",
							ValueType:        "ledger_snapshot_txt",
							Label:            "Purchase / Supplier Detail",
							Required:         true,
							Description:      "Upload export Purchase / Supplier Detail (.txt) dari MYOB untuk mencocokkan supplier dan bill outstanding.",
							AcceptExtensions: []string{".txt"},
							AcceptMIMETypes:  []string{"text/plain"},
						},
					},
					Detail: &ActionDetailSpec{
						Summary: "Mengubah sheet cashflow menjadi file MYOB Pay Bills (.txt).",
						Bullets: []string{
							"Hanya row dengan total negatif yang akan diproses.",
							"Category harus tersedia di registry Category Accounts.",
							"Snapshot ledger .txt wajib di-upload di action ini untuk mencocokkan supplier dan bill outstanding.",
						},
					},
					Requirements: []ActionRequirementSpec{
						{
							Key:      "cashflowBillDefaultProfile",
							Label:    "Default Profil Cashflow Pay Bills",
							Required: true,
						},
						{
							Key:      "cashflowCategoryAccounts",
							Label:    "Category Accounts",
							Required: true,
						},
					},
					MasterData: map[string]ActionMasterDataSpec{
						"category": {
							Relative:     "category_accounts.csv",
							LookupKey:    "name",
							RequiredCols: []string{"name", "account"},
						},
					},
					Outputs: []ActionOutputSpec{
						{
							Kind:       "file",
							MimeType:   "text/plain",
							Ext:        "txt",
							DownloadOK: true,
						},
					},
				},
				{
					CollectionKind: string(collectionKind),
					ActionType:     "cashflow_to_receive_payments",
					Label:          "Receive Payments",
					Description:    "Konversi cashflow ke format MYOB Receive Payments.",
					State:          ActionStateSpec{Enabled: true},
					Presentation:   ActionPresentationSpec{Mode: "inline", Width: "xl"},
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  true,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Form: &FormSpec{
						Title:       "Pengaturan Receive Payments",
						Description: "Atur workbook cashflow, upload snapshot ledger, lalu sesuaikan mapping yang dibutuhkan.",
						Sections:    buildCashflowBillSections(true),
					},
					ArtifactInputs: []ActionArtifactInputSpec{
						{
							Key:              "ledgerSnapshotRef",
							ValueType:        "ledger_snapshot_txt",
							Label:            "Sales / Customer Detail",
							Required:         true,
							Description:      "Upload export Sales / Customer Detail (.txt) dari MYOB untuk mencocokkan customer dan invoice outstanding.",
							AcceptExtensions: []string{".txt"},
							AcceptMIMETypes:  []string{"text/plain"},
						},
					},
					Detail: &ActionDetailSpec{
						Summary: "Mengubah sheet cashflow menjadi file MYOB Receive Payments (.txt).",
						Bullets: []string{
							"Hanya row dengan total positif yang akan diproses.",
							"Category harus tersedia di registry Category Accounts.",
							"Snapshot ledger .txt wajib di-upload di action ini untuk mencocokkan customer dan invoice outstanding.",
						},
					},
					Requirements: []ActionRequirementSpec{
						{
							Key:      "cashflowBillDefaultProfile",
							Label:    "Default Profil Cashflow Receive Payments",
							Required: true,
						},
						{
							Key:      "cashflowCategoryAccounts",
							Label:    "Category Accounts",
							Required: true,
						},
					},
					MasterData: map[string]ActionMasterDataSpec{
						"category": {
							Relative:     "category_accounts.csv",
							LookupKey:    "name",
							RequiredCols: []string{"name", "account"},
						},
					},
					Outputs: []ActionOutputSpec{
						{
							Kind:       "file",
							MimeType:   "text/plain",
							Ext:        "txt",
							DownloadOK: true,
						},
					},
				},
			},
		}, true
	case CollectionKindBukpotRequestGSTDeductionMT:
		return CollectionSpec{
			CollectionKind: collectionKind,
			SourceFormat:   SourceFormatXLSX,
			Label:          "Request Bukpot",
			Description:    "Spreadsheet XLSX GST Deduction MT untuk generate request bukpot Coretax.",
			Upload: UploadRuleSpec{
				AcceptExtensions: []string{".xlsx"},
				AcceptMIMETypes: []string{
					"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
				},
				MaxChunkMB:       20,
				MaxFilesPerBatch: 200,
			},
			Ingest: IngestRuleSpec{
				KeepRaw:            true,
				DeleteTempAfterRun: true,
				Artifacts: []ArtifactRuleSpec{
					{Kind: "raw", Required: true},
					{Kind: "normalized", Required: true},
				},
			},
			Actions: []ActionSpec{
				{
					CollectionKind: string(collectionKind),
					ActionType:     "request_bukpot_gst_deduction_mt",
					Label:          "Request Bukpot GST Deduction MT",
					Description:    "Generate ZIP berisi template Excel dan XML Coretax. Hanya row dengan nilai Entity yang sama dengan Alias profil aktif yang akan diproses.",
					State:          ActionStateSpec{Enabled: true},
					Presentation:   ActionPresentationSpec{Mode: "inline", Width: "xl"},
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  true,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Form: &FormSpec{
						Title:       "Pengaturan Request Bukpot",
						Description: "Default profil akan dipakai otomatis, lalu bisa diubah untuk eksekusi ini saja. Sistem hanya memproses row dengan nilai Entity yang sama dengan Alias profil aktif.",
						Sections: []FormSectionSpec{
							{
								Key:         "source",
								Title:       "Sumber Excel",
								Description: "Pilih sheet dan baris header dari workbook terpilih.",
								Columns:     3,
								Fields: []FormFieldSpec{
									{
										Key:          "sheetName",
										Kind:         FormFieldKindSelect,
										Label:        "Sheet",
										Required:     true,
										DefaultValue: "",
										Span:         2,
										HelpText:     "Pilih dokumen terlebih dahulu untuk melihat sheet yang tersedia.",
										State:        FormFieldStateSpec{Visible: true, Disabled: true},
									},
									buildHeaderRowField("Baris Header", "Dipakai untuk validasi kolom dan pembacaan row data."),
								},
							},
							{
								Key:         "override",
								Title:       "Override Eksekusi Ini",
								Description: "Prefill dari Default Profil. Ubah hanya jika file bulan ini memakai nama header yang berbeda.",
								Columns:     2,
								Fields: []FormFieldSpec{
									{Key: "entity", Kind: FormFieldKindText, Label: "Entity", Required: true, DefaultValue: "Entity", State: FormFieldStateSpec{Visible: true}},
									{Key: "settlementDate", Kind: FormFieldKindText, Label: "Settlement Date", Required: true, DefaultValue: "Settlemet Date", State: FormFieldStateSpec{Visible: true}},
									{Key: "npwp", Kind: FormFieldKindText, Label: "NPWP", Required: true, DefaultValue: "NPWP", State: FormFieldStateSpec{Visible: true}},
									{Key: "nitku", Kind: FormFieldKindText, Label: "NITKU", Required: true, DefaultValue: "NITKU", State: FormFieldStateSpec{Visible: true}},
									{Key: "facility", Kind: FormFieldKindText, Label: "Fasilitas", Required: false, DefaultValue: "Fasilitas", State: FormFieldStateSpec{Visible: true}},
									{Key: "taxObjectCode", Kind: FormFieldKindText, Label: "Kode Objek Pajak", Required: true, DefaultValue: "Kode Objek Pajak", State: FormFieldStateSpec{Visible: true}},
									{Key: "taxBase", Kind: FormFieldKindText, Label: "DPP", Required: true, DefaultValue: "(Rp)Total Invoice (Exc VAT)", State: FormFieldStateSpec{Visible: true}},
									{Key: "withholdingRate", Kind: FormFieldKindText, Label: "Tarif / WHT", Required: true, DefaultValue: "WHT", State: FormFieldStateSpec{Visible: true}},
									{Key: "taxInvoiceNumber", Kind: FormFieldKindText, Label: "Faktur Pajak No", Required: false, DefaultValue: "Faktur Pajak No", State: FormFieldStateSpec{Visible: true}},
									{Key: "referenceNumber", Kind: FormFieldKindText, Label: "Invoice / Kwitansi No", Required: true, DefaultValue: "Invoice / Kwitansi No", State: FormFieldStateSpec{Visible: true}},
									{Key: "referenceDate", Kind: FormFieldKindText, Label: "FP DATE", Required: true, DefaultValue: "FP DATE", State: FormFieldStateSpec{Visible: true}},
								},
							},
						},
					},
					Detail: &ActionDetailSpec{
						Summary: "Membangun ZIP berisi template Excel dan XML Coretax dari workbook GST Deduction MT.",
						Bullets: []string{
							"Hanya row dengan nilai Entity yang sama dengan Alias profil aktif yang akan diproses.",
							"Sheet dan baris header dipilih dari dokumen source yang Anda centang.",
							"Header source akan memakai Default Profil terlebih dahulu, lalu bisa dioverride untuk eksekusi ini.",
						},
					},
					Requirements: []ActionRequirementSpec{
						{
							Key:      "bukpotRequestConfig",
							Label:    "Default Profil Request Bukpot GST Deduction MT",
							Required: true,
						},
					},
					Outputs: []ActionOutputSpec{
						{
							Kind:       "file",
							MimeType:   "application/zip",
							Ext:        "zip",
							DownloadOK: true,
						},
					},
				},
			},
		}, true
	default:
		return CollectionSpec{}, false
	}
}

func BuildCreateCollectionSpec() CreateCollectionSpec {
	kinds := []CollectionKind{
		CollectionKindInvoiceCompany,
		CollectionKindTaxInvoiceCoretax,
		CollectionKindFPKeluaranCoretax,
		CollectionKindFPMasukanCoretax,
		CollectionKindBukpotBPPU,
		CollectionKindBukpotBP21,
		CollectionKindBukpotBPA1,
		CollectionKindBukpotRequestGSTDeductionMT,
		CollectionKindCashflowImport,
	}

	sourceFormatOrder := []SourceFormat{
		SourceFormatPDF,
		SourceFormatXLSX,
		SourceFormatCSV,
	}

	availableFormats := map[SourceFormat]struct{}{}
	kindSpecs := make([]CreateCollectionKindSpec, 0, len(kinds))

	for _, kind := range kinds {
		spec, ok := BuildCollectionSpec(kind)
		if !ok {
			continue
		}

		availableFormats[spec.SourceFormat] = struct{}{}
		primaryActions := make([]string, 0, len(spec.Actions))
		for _, action := range spec.Actions {
			label := strings.TrimSpace(action.Label)
			if label == "" {
				continue
			}
			primaryActions = append(primaryActions, label)
		}

		kindSpecs = append(kindSpecs, CreateCollectionKindSpec{
			CollectionKind: spec.CollectionKind,
			Label:          spec.Label,
			Description:    spec.Description,
			SourceFormats:  []SourceFormat{spec.SourceFormat},
			PrimaryActions: primaryActions,
		})
	}

	sourceFormats := make([]CreateCollectionSourceFormatSpec, 0, len(sourceFormatOrder))
	defaultSourceFormat := SourceFormat("")
	for _, sourceFormat := range sourceFormatOrder {
		if _, ok := availableFormats[sourceFormat]; !ok {
			continue
		}
		if !defaultSourceFormat.IsValid() {
			defaultSourceFormat = sourceFormat
		}
		sourceFormats = append(sourceFormats, CreateCollectionSourceFormatSpec{
			Value:       sourceFormat,
			Label:       sourceFormatCreateLabel(sourceFormat),
			Description: sourceFormatCreateDescription(sourceFormat),
		})
	}

	return CreateCollectionSpec{
		DefaultSourceFormat: defaultSourceFormat,
		SourceFormats:       sourceFormats,
		CollectionKinds:     kindSpecs,
	}
}

func buildBukpotCollectionSpec(
	collectionKind CollectionKind,
	label string,
	description string,
) CollectionSpec {
	actions := []ActionSpec{}
	actions = append(actions, ActionSpec{
		ActionType:  "rename_bukpot",
		Label:       "Rename Bukpot",
		Description: "Ganti nama file bukpot berdasarkan template placeholder dan hasilkan ZIP.",
		State: ActionStateSpec{
			Enabled: true,
		},
		Presentation: ActionPresentationSpec{
			Mode:  "inline",
			Width: "md",
		},
		Selection: ActionSelectionSpec{
			Mode:           "manual",
			AllowCheckAll:  true,
			AllowedStatus:  []string{"ready", "warning"},
			MinDocumentCnt: 1,
		},
		Form: &FormSpec{
			Title: "Pengaturan Rename",
			Sections: []FormSectionSpec{
				{
					Key:     "main",
					Title:   specutil.ParameterActionSectionTitle,
					Columns: 1,
					Fields: []FormFieldSpec{
						{
							Key:          "filenameTemplate",
							Kind:         FormFieldKindTemplate,
							Label:        "Template Nama File",
							Required:     true,
							DefaultValue: "{{nomorBuktiPotong}} - {{namaPenerima}}",
							Suggestions:  buildBukpotTemplateSuggestions(collectionKind),
							HelpText:     "Gunakan placeholder yang tersedia. Ekstensi .pdf akan ditambahkan otomatis.",
							Placeholder:  "{{nomorBuktiPotong}} - {{namaPenerima}}",
							State: FormFieldStateSpec{
								Visible: true,
							},
						},
						{
							Key:          "onlyNormalStatus",
							Kind:         FormFieldKindCheckbox,
							Label:        "Hanya include status Normal",
							Required:     false,
							DefaultValue: true,
							HelpText:     "Jika aktif, bukpot berstatus DIBATALKAN atau PEMBETULAN tidak diproses dan item akan ditandai failed.",
							State: FormFieldStateSpec{
								Visible: true,
							},
						},
					},
				},
			},
		},
		Detail: &ActionDetailSpec{
			Summary: "Mengganti nama file bukpot memakai template lalu menggabungkannya ke ZIP.",
			Warnings: []string{
				"Jika filter status Normal aktif, bukpot berstatus DIBATALKAN atau PEMBETULAN tidak akan diproses.",
				"Dokumen yang statusnya bukan Normal akan muncul sebagai failed pada items table.",
			},
		},
		Outputs: []ActionOutputSpec{
			{
				Kind:       "file",
				MimeType:   "application/zip",
				Ext:        "zip",
				DownloadOK: true,
			},
		},
	})

	if collectionKind == CollectionKindBukpotBPPU || collectionKind == CollectionKindBukpotBP21 {
		actions = append(actions, ActionSpec{
			ActionType:  "rename_by_category",
			Label:       "Rename by Category",
			Description: "Kelompokkan bukpot ke folder kategori dari Dokumen Referensi Nomor lalu hasilkan ZIP.",
			State: ActionStateSpec{
				Enabled: true,
			},
			Presentation: ActionPresentationSpec{
				Mode:  "inline",
				Width: "md",
			},
			Selection: ActionSelectionSpec{
				Mode:           "manual",
				AllowCheckAll:  true,
				AllowedStatus:  []string{"ready", "warning"},
				MinDocumentCnt: 1,
			},
			Form: &FormSpec{
				Title: "Pengaturan Rename by Category",
				Sections: []FormSectionSpec{
					{
						Key:     "filter",
						Title:   "Filter Dokumen",
						Columns: 1,
						Fields: []FormFieldSpec{
							{
								Key:          "onlyNormalStatus",
								Kind:         FormFieldKindCheckbox,
								Label:        "Hanya include status Normal",
								Required:     false,
								DefaultValue: true,
								HelpText:     "Jika aktif, bukpot berstatus DIBATALKAN atau PEMBETULAN tidak diproses dan item akan ditandai failed.",
								State: FormFieldStateSpec{
									Visible: true,
								},
							},
						},
					},
				},
			},
			Detail: &ActionDetailSpec{
				Summary: "Mengelompokkan bukpot ke folder kategori dari Dokumen Referensi Nomor lalu membuat ZIP.",
				Warnings: []string{
					"Jika filter status Normal aktif, bukpot berstatus DIBATALKAN atau PEMBETULAN tidak akan diproses.",
					"Dokumen yang statusnya bukan Normal akan muncul sebagai failed pada items table.",
				},
			},
			Outputs: []ActionOutputSpec{
				{
					Kind:       "file",
					MimeType:   "application/zip",
					Ext:        "zip",
					DownloadOK: true,
				},
			},
		})
	}

	return CollectionSpec{
		CollectionKind: collectionKind,
		SourceFormat:   SourceFormatPDF,
		Label:          label,
		Description:    description,
		Upload: UploadRuleSpec{
			AcceptExtensions: []string{".pdf"},
			AcceptMIMETypes:  []string{"application/pdf"},
			MaxChunkMB:       15,
			MaxFilesPerBatch: 2000,
		},
		Ingest: IngestRuleSpec{
			KeepRaw:            true,
			DeleteTempAfterRun: true,
			Artifacts: []ArtifactRuleSpec{
				{Kind: "raw", Required: false},
				{Kind: "normalized", Required: true},
				{Kind: "audit", Required: false},
			},
		},
		Actions: actions,
	}
}

func buildFPCoretaxCollectionSpec(
	collectionKind CollectionKind,
	label string,
	description string,
	actionType string,
	actionLabel string,
	actionDescription string,
	formTitle string,
	formDescription string,
	partyFieldLabel string,
	requirements []ActionRequirementSpec,
) CollectionSpec {
	outputDefault := "misc_sales"
	accountLabel := "Account Number"
	accountDefault := "41001"
	if collectionKind == CollectionKindFPMasukanCoretax {
		outputDefault = "misc_purchases"
		accountLabel = "Default Account Number"
		accountDefault = "51001"
	}

	return CollectionSpec{
		CollectionKind: collectionKind,
		SourceFormat:   SourceFormatXLSX,
		Label:          label,
		Description:    description,
		Upload: UploadRuleSpec{
			AcceptExtensions: []string{".xlsx"},
			AcceptMIMETypes: []string{
				"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			},
			MaxChunkMB:       20,
			MaxFilesPerBatch: 200,
		},
		Ingest: IngestRuleSpec{
			KeepRaw:            true,
			DeleteTempAfterRun: true,
			Artifacts: []ArtifactRuleSpec{
				{Kind: "raw", Required: true},
				{Kind: "normalized", Required: true},
			},
		},
		Actions: []ActionSpec{
			{
				CollectionKind: string(collectionKind),
				ActionType:     actionType,
				Label:          actionLabel,
				Description:    actionDescription,
				State:          ActionStateSpec{Enabled: true},
				Presentation:   ActionPresentationSpec{Mode: "inline", Width: "xl"},
				Selection: ActionSelectionSpec{
					Mode:           "manual",
					AllowCheckAll:  false,
					AllowedStatus:  []string{"ready", "warning"},
					MinDocumentCnt: 1,
					MaxDocumentCnt: 1,
				},
				Form: &FormSpec{
					Title:       formTitle,
					Description: formDescription,
					Sections: []FormSectionSpec{
						{
							Key:         "source",
							Title:       "Sumber Data",
							Description: "Pilih sheet dan baris header dari workbook terpilih.",
							Columns:     2,
							Fields: []FormFieldSpec{
								{
									Key:          "sheetName",
									Kind:         FormFieldKindSelect,
									Label:        "Sheet",
									Required:     true,
									DefaultValue: "",
									HelpText:     "Pilih dokumen terlebih dahulu untuk melihat sheet yang tersedia.",
									State:        FormFieldStateSpec{Visible: true, Disabled: true},
								},
								buildHeaderRowField("Baris Header", "Baris header workbook untuk membaca data transaksi."),
							},
						},
						{
							Key:         "parameter",
							Title:       specutil.ParameterActionSectionTitle,
							Description: "Parameter default untuk export MYOB.",
							Columns:     2,
							Fields: []FormFieldSpec{
								{
									Key:          "outputFilename",
									Kind:         FormFieldKindText,
									Label:        "Nama Output",
									Required:     true,
									DefaultValue: outputDefault,
									Placeholder:  outputDefault,
									HelpText:     "Tanpa ekstensi file.",
									State:        FormFieldStateSpec{Visible: true},
								},
								{
									Key:          "accountNumber",
									Kind:         FormFieldKindText,
									Label:        accountLabel,
									Required:     true,
									DefaultValue: accountDefault,
									HelpText:     "Dipakai saat registry tidak menyediakan account number.",
									State:        FormFieldStateSpec{Visible: true},
								},
								{
									Key:          "memoTemplate",
									Kind:         FormFieldKindTemplate,
									Label:        "Template Memo",
									Required:     true,
									DefaultValue: "{{nomorFakturPajak}}",
									Placeholder:  "{{nomorFakturPajak}}",
									Suggestions:  buildFPCoretaxTemplateSuggestions(collectionKind),
									HelpText:     "Gunakan placeholder yang tersedia untuk membentuk memo output MYOB.",
									State:        FormFieldStateSpec{Visible: true},
								},
								{
									Key:          "descriptionTemplate",
									Kind:         FormFieldKindTemplate,
									Label:        "Template Description",
									Required:     true,
									DefaultValue: "{{nomorFakturPajak}}",
									Placeholder:  "{{nomorFakturPajak}}",
									Suggestions:  buildFPCoretaxTemplateSuggestions(collectionKind),
									HelpText:     "Gunakan placeholder yang tersedia untuk membentuk description output MYOB.",
									State:        FormFieldStateSpec{Visible: true},
								},
							},
						},
						{
							Key:         "tax",
							Title:       "Tax",
							Description: "Parameter pajak default untuk output MYOB.",
							Columns:     2,
							Fields: []FormFieldSpec{
								{
									Key:          "taxCode",
									Kind:         FormFieldKindText,
									Label:        "Tax Code",
									Required:     true,
									DefaultValue: "PPN",
									HelpText:     "Tax code MYOB untuk setiap baris transaksi.",
									State:        FormFieldStateSpec{Visible: true},
								},
								{
									Key:          "inclusive",
									Kind:         FormFieldKindCheckbox,
									Label:        "Inclusive Tax",
									Required:     false,
									DefaultValue: false,
									HelpText:     "Aktifkan jika nilai total pada source sudah termasuk pajak.",
									State:        FormFieldStateSpec{Visible: true},
								},
							},
						},
						{
							Key:         "mapping",
							Title:       specutil.MappingHeaderSectionTitle,
							Description: "Nama header source untuk membaca kolom faktur pajak.",
							Columns:     2,
							Fields: []FormFieldSpec{
								{Key: "partyName", Kind: FormFieldKindText, Label: partyFieldLabel, Required: true, DefaultValue: "nama", State: FormFieldStateSpec{Visible: true}},
								{Key: "documentNumber", Kind: FormFieldKindText, Label: "Nomor Faktur Pajak", Required: true, DefaultValue: "nomor faktur pajak", State: FormFieldStateSpec{Visible: true}},
								{Key: "date", Kind: FormFieldKindText, Label: "Tanggal Faktur Pajak", Required: true, DefaultValue: "tanggal faktur pajak", State: FormFieldStateSpec{Visible: true}},
								{Key: "taxBase", Kind: FormFieldKindText, Label: "DPP", Required: true, DefaultValue: "harga jual/penggantian/dpp", State: FormFieldStateSpec{Visible: true}},
								{Key: "tax", Kind: FormFieldKindText, Label: "PPN", Required: true, DefaultValue: "ppn", State: FormFieldStateSpec{Visible: true}},
								{Key: "reference", Kind: FormFieldKindText, Label: "Referensi", Required: false, DefaultValue: "referensi", State: FormFieldStateSpec{Visible: true}},
							},
						},
					},
				},
				Detail: &ActionDetailSpec{
					Summary: fmt.Sprintf("Mengubah workbook %s menjadi file MYOB %s (.txt).", strings.ToLower(label), actionLabel),
					Bullets: []string{
						"Sheet dan baris header dipilih dari dokumen source yang Anda centang.",
						"Memo dan Description dibentuk dari template placeholder yang tersedia.",
						"Registry pihak dipakai untuk validasi nama MYOB, dan pada supplier dapat memberi override account number.",
					},
				},
				Requirements: requirements,
				Outputs: []ActionOutputSpec{
					{
						Kind:       "file",
						MimeType:   "text/plain",
						Ext:        "txt",
						DownloadOK: true,
					},
				},
			},
		},
	}
}

func ResolveCollectionSourceFormat(collectionKind CollectionKind) SourceFormat {
	spec, ok := BuildCollectionSpec(collectionKind)
	if !ok || !spec.SourceFormat.IsValid() {
		return SourceFormatPDF
	}
	return spec.SourceFormat
}

func buildBukpotTemplateSuggestions(collectionKind CollectionKind) []FormSuggestionSpec {
	suggestions := []FormSuggestionSpec{
		{Token: "nomorBuktiPotong", Label: "Nomor Bukti Potong", Example: "{{nomorBuktiPotong}}"},
		{Token: "namaPenerima", Label: "Nama Penerima", Example: "{{namaPenerima}}"},
		{Token: "sifatPemotongan", Label: "Sifat Pemotongan", Example: "{{sifatPemotongan}}"},
		{Token: "statusBukti", Label: "Status Bukti", Example: "{{statusBukti}}"},
		{Token: "npwpNikPenerima", Label: "NPWP/NIK Penerima", Example: "{{npwpNikPenerima}}"},
		{Token: "namaPemotong", Label: "Nama Pemotong", Example: "{{namaPemotong}}"},
		{Token: "npwpNikPemotong", Label: "NPWP/NIK Pemotong", Example: "{{npwpNikPemotong}}"},
		{Token: "documentTag", Label: "Tag Dokumen", Example: "{{documentTag}}"},
		{Token: "sourceName", Label: "Nama File Asal", Example: "{{sourceName}}"},
	}

	switch collectionKind {
	case CollectionKindBukpotBPPU, CollectionKindBukpotBP21:
		suggestions = append(suggestions,
			FormSuggestionSpec{Token: "dokumenReferensiNomor", Label: "Nomor Dokumen Referensi", Example: "{{dokumenReferensiNomor}}"},
			FormSuggestionSpec{Token: "dokumenReferensiJenis", Label: "Jenis Dokumen Referensi", Example: "{{dokumenReferensiJenis}}"},
			FormSuggestionSpec{Token: "dokumenReferensiTanggal", Label: "Tanggal Dokumen Referensi", Example: "{{dokumenReferensiTanggal}}"},
			FormSuggestionSpec{Token: "masaPajak", Label: "Masa Pajak", Example: "{{masaPajak}}"},
		)
	case CollectionKindBukpotBPA1:
		suggestions = append(suggestions,
			FormSuggestionSpec{Token: "periodePenghasilan", Label: "Periode Penghasilan", Example: "{{periodePenghasilan}}"},
			FormSuggestionSpec{Token: "posisi", Label: "Posisi", Example: "{{posisi}}"},
			FormSuggestionSpec{Token: "statusPtkp", Label: "Status PTKP", Example: "{{statusPtkp}}"},
		)
	}

	return suggestions
}

func buildFPCoretaxTemplateSuggestions(collectionKind CollectionKind) []FormSuggestionSpec {
	suggestions := []FormSuggestionSpec{
		{Token: "nomorFakturPajak", Label: "Nomor Faktur Pajak", Example: "{{nomorFakturPajak}}"},
		{Token: "tanggalFakturPajak", Label: "Tanggal Faktur Pajak", Example: "{{tanggalFakturPajak}}"},
		{Token: "referensi", Label: "Referensi", Example: "{{referensi}}"},
		{Token: "dpp", Label: "DPP", Example: "{{dpp}}"},
		{Token: "ppn", Label: "PPN", Example: "{{ppn}}"},
		{Token: "total", Label: "Total", Example: "{{total}}"},
		{Token: "sourceName", Label: "Nama File Asal", Example: "{{sourceName}}"},
	}

	if collectionKind == CollectionKindFPMasukanCoretax {
		suggestions = append([]FormSuggestionSpec{
			{Token: "namaPenjual", Label: "Nama Penjual", Example: "{{namaPenjual}}"},
		}, suggestions...)
	} else {
		suggestions = append([]FormSuggestionSpec{
			{Token: "namaPembeli", Label: "Nama Pembeli", Example: "{{namaPembeli}}"},
		}, suggestions...)
	}

	return suggestions
}

func sourceFormatCreateLabel(sourceFormat SourceFormat) string {
	switch sourceFormat {
	case SourceFormatXLSX:
		return "Excel"
	case SourceFormatCSV:
		return "CSV"
	default:
		return "PDF"
	}
}

func sourceFormatCreateDescription(sourceFormat SourceFormat) string {
	switch sourceFormat {
	case SourceFormatXLSX:
		return "Untuk workbook Excel yang diproses sebagai sumber data terstruktur."
	case SourceFormatCSV:
		return "Untuk data tabel CSV yang diproses sebagai sumber data terstruktur."
	default:
		return "Untuk dokumen PDF yang diparse langsung saat upload."
	}
}
