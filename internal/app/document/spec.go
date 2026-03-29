package document

import (
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
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Sections    []FormSectionSpec `json:"sections"`
}

type FormSectionSpec struct {
	Key         string          `json:"key"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Columns     int             `json:"columns,omitempty"`
	Fields      []FormFieldSpec `json:"fields"`
}

type FormFieldSpec struct {
	Key          string               `json:"key"`
	Kind         string               `json:"kind"`
	Label        string               `json:"label"`
	Required     bool                 `json:"required"`
	DefaultValue any                  `json:"defaultValue,omitempty"`
	Options      []FormFieldOption    `json:"options,omitempty"`
	Rules        []FormFieldRuleSpec  `json:"rules,omitempty"`
	State        FormFieldStateSpec   `json:"state"`
	Suggestions  []FormSuggestionSpec `json:"suggestions,omitempty"`
	HelpText     string               `json:"helpText,omitempty"`
	Placeholder  string               `json:"placeholder,omitempty"`
	Span         int                  `json:"span,omitempty"`
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
	Upload         UploadRuleSpec `json:"upload"`
	Ingest         IngestRuleSpec `json:"ingest"`
	Actions        []ActionSpec   `json:"actions"`
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
						MaxDocumentCnt: 1,
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
						AllowCheckAll:  false,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
						MaxDocumentCnt: 1,
					},
					Form: &FormSpec{
						Title: "Pengaturan Rename",
						Sections: []FormSectionSpec{
							{
								Key:     "main",
								Title:   "Template Nama File",
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
					ActionType:     "export_cashflow_myob",
					Label:          "Export MYOB",
					Description:    "Konversi cashflow ke format CSV MYOB Spend Money.",
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
						Title:       "Pengaturan Export MYOB",
						Description: "Pilih sheet sumber dan atur parameter export cashflow.",
						Sections: []FormSectionSpec{
							{
								Key:     "source",
								Title:   "Sumber Data",
								Columns: 3,
								Fields: []FormFieldSpec{
									{
										Key:          "sheetName",
										Kind:         FormFieldKindSelect,
										Label:        "Sheet",
										Required:     true,
										DefaultValue: "",
										HelpText:     "Pilih dokumen cashflow terlebih dahulu untuk melihat sheet yang tersedia.",
										Span:         2,
										State: FormFieldStateSpec{
											Visible:  true,
											Disabled: true,
										},
									},
									buildHeaderRowField("Baris Header", "Nomor baris header pada sheet yang dipilih."),
									{
										Key:         "startingChequeNumber",
										Kind:        FormFieldKindNumber,
										Label:       "Nomor Awal Cheque",
										Required:    false,
										HelpText:    "Opsional. Kosongkan jika nomor cheque ingin dibuat otomatis oleh MYOB.",
										Placeholder: "17500",
										State: FormFieldStateSpec{
											Visible: true,
										},
									},
								},
							},
							{
								Key:     "output",
								Title:   "Output",
								Columns: 3,
								Fields: []FormFieldSpec{
									{
										Key:          "outputFilename",
										Kind:         FormFieldKindText,
										Label:        "Nama Output",
										Required:     true,
										DefaultValue: "spend_money",
										Placeholder:  "cashflow-myob",
										HelpText:     "Tanpa ekstensi file.",
										State: FormFieldStateSpec{
											Visible: true,
										},
									},
									{
										Key:          "chequeAccount",
										Kind:         FormFieldKindText,
										Label:        "Cheque Account",
										Required:     true,
										DefaultValue: "12021",
										HelpText:     "Akun cheque utama untuk file MYOB.",
										State:        FormFieldStateSpec{Visible: true},
									},
									{
										Key:          "cashflowFormat",
										Kind:         FormFieldKindSelect,
										Label:        "Format Cashflow",
										Required:     true,
										DefaultValue: "default",
										Options: []FormFieldOption{
											{Label: "Default", Value: "default"},
											{Label: "Influencer", Value: "influencer"},
										},
										State: FormFieldStateSpec{Visible: true},
									},
									{
										Key:          "cashflowType",
										Kind:         FormFieldKindSelect,
										Label:        "Tipe Cashflow",
										Required:     true,
										DefaultValue: "spend_money",
										Options: []FormFieldOption{
											{Label: "Spend Money", Value: "spend_money"},
										},
										HelpText: "Sementara hanya Spend Money yang didukung.",
										State:    FormFieldStateSpec{Visible: true},
									},
									{
										Key:          "skipPositiveTotal",
										Kind:         FormFieldKindCheckbox,
										Label:        "Lewati Total Positif",
										Required:     false,
										DefaultValue: true,
										HelpText:     "Abaikan baris dengan total positif.",
										State:        FormFieldStateSpec{Visible: true},
									},
								},
							},
							{
								Key:     "default_format",
								Title:   "Format Default",
								Columns: 2,
								Fields: []FormFieldSpec{
									{
										Key:          "remarkDelimiter",
										Kind:         FormFieldKindText,
										Label:        "Remark Delimiter",
										Required:     false,
										DefaultValue: "*",
										Placeholder:  "*",
										HelpText:     "Wajib untuk format default.",
										Rules: []FormFieldRuleSpec{
											{Type: FormFieldRuleRequiredIf, Field: "cashflowFormat", Equals: "default", Message: "Remark Delimiter wajib diisi"},
										},
										State: FormFieldStateSpec{Visible: true},
									},
									{
										Key:          "otherCostsAccountCode",
										Kind:         FormFieldKindText,
										Label:        "Kode Akun Biaya Lain",
										Required:     false,
										DefaultValue: "62099",
										HelpText:     "Wajib untuk format default.",
										Rules: []FormFieldRuleSpec{
											{Type: FormFieldRuleRequiredIf, Field: "cashflowFormat", Equals: "default", Message: "Kode akun biaya lain wajib diisi"},
										},
										State: FormFieldStateSpec{Visible: true},
									},
								},
							},
							{
								Key:     "influencer_format",
								Title:   "Format Influencer",
								Columns: 2,
								Fields: []FormFieldSpec{
									{
										Key:          "defaultIAccountCode",
										Kind:         FormFieldKindText,
										Label:        "Default Influencer Account Code",
										Required:     false,
										DefaultValue: "62004",
										Rules: []FormFieldRuleSpec{
											{Type: FormFieldRuleRequiredIf, Field: "cashflowFormat", Equals: "influencer", Message: "Default Influencer Account Code wajib diisi"},
										},
										State: FormFieldStateSpec{Visible: true},
									},
									{
										Key:          "defaultBAccountCode",
										Kind:         FormFieldKindText,
										Label:        "Default Bank Account Code",
										Required:     false,
										DefaultValue: "90900",
										Rules: []FormFieldRuleSpec{
											{Type: FormFieldRuleRequiredIf, Field: "cashflowFormat", Equals: "influencer", Message: "Default Bank Account Code wajib diisi"},
										},
										State: FormFieldStateSpec{Visible: true},
									},
								},
							},
						},
					},
					Requirements: []ActionRequirementSpec{
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
					Requirements: []ActionRequirementSpec{
						{
							Key:      "bukpotRequestConfig",
							Label:    "Default Profil Request Bukpot",
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
					Title:   "Template Nama File",
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
