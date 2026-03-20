package document

import "strings"

const (
	ActionParamTypeString  = "string"
	ActionParamTypeInt     = "int"
	ActionParamTypeFloat   = "float"
	ActionParamTypeBoolean = "boolean"

	ActionParamRuleRequiredIf = "required_if"
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

type ActionParamFieldSpec struct {
	Key         string                  `json:"key"`
	Type        string                  `json:"type"`
	Label       string                  `json:"label"`
	Required    bool                    `json:"required"`
	Default     any                     `json:"default,omitempty"`
	Options     []ActionParamOptionSpec `json:"options,omitempty"`
	Rules       []ActionParamRuleSpec   `json:"rules,omitempty"`
	UI          *ActionParamUISpec      `json:"ui,omitempty"`
	Suggestions []ActionParamSuggestion `json:"suggestions,omitempty"`
	Description string                  `json:"description,omitempty"`
	Placeholder string                  `json:"placeholder,omitempty"`
}

type ActionParamOptionSpec struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type ActionParamSuggestion struct {
	Token       string `json:"token"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
}

type ActionParamUISpec struct {
	Editor string `json:"editor,omitempty"`
}

type ActionParamRuleSpec struct {
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
	Key      string                `json:"key"`
	Label    string                `json:"label"`
	Type     string                `json:"type"`
	Required bool                  `json:"required"`
	Aliases  []string              `json:"aliases,omitempty"`
	Group    string                `json:"group,omitempty"`
	Rules    []ActionParamRuleSpec `json:"rules,omitempty"`
}

type ActionSpec struct {
	ActionType     string                    `json:"actionType"`
	Label          string                    `json:"label"`
	Description    string                    `json:"description,omitempty"`
	Enabled        bool                      `json:"enabled"`
	Reason         string                    `json:"reason,omitempty"`
	Selection      ActionSelectionSpec       `json:"selection"`
	Params         []ActionParamFieldSpec    `json:"params"`
	Requirements   []ActionRequirementSpec   `json:"requirements,omitempty"`
	ArtifactInputs []ActionArtifactInputSpec `json:"artifactInputs,omitempty"`
	Columns        []ActionColumnSpec        `json:"columns,omitempty"`
	Outputs        []ActionOutputSpec        `json:"outputs"`
}

type DocumentTypeSpec struct {
	DocumentType DocumentType   `json:"documentType"`
	Label        string         `json:"label"`
	Description  string         `json:"description,omitempty"`
	Upload       UploadRuleSpec `json:"upload"`
	Ingest       IngestRuleSpec `json:"ingest"`
	Actions      []ActionSpec   `json:"actions"`
}

func (s DocumentTypeSpec) FindActionSpec(actionType string) (ActionSpec, bool) {
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

func BuildDocumentTypeSpec(docType DocumentType) (DocumentTypeSpec, bool) {
	switch docType {
	case DocumentTypePDFInvoice:
		return DocumentTypeSpec{
			DocumentType: docType,
			Label:        "Invoice",
			Description:  "PDF invoice document for extraction and e-Faktur export.",
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
					ActionType:  "export_faktur_keluaran",
					Label:       "Export e-Faktur",
					Description: "Export invoice terpilih ke format e-Faktur keluaran Coretax.",
					Enabled:     true,
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  true,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Params: []ActionParamFieldSpec{
						{
							Key:         "filenamePrefix",
							Type:        ActionParamTypeString,
							Label:       "Filename Prefix",
							Required:    false,
							Default:     "faktur-keluaran",
							Description: "Optional prefix for exported file name.",
							Placeholder: "faktur-keluaran",
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
	case DocumentTypePDFTaxInvoice:
		return DocumentTypeSpec{
			DocumentType: docType,
			Label:        "Faktur Pajak",
			Description:  "Dokumen PDF faktur pajak Coretax untuk ekstraksi dan action lanjutan.",
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
					ActionType:  "rename_tax_invoice",
					Label:       "Rename Faktur Pajak",
					Description: "Ganti nama file faktur pajak berdasarkan template placeholder dan hasilkan ZIP.",
					Enabled:     true,
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  true,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Params: []ActionParamFieldSpec{
						{
							Key:      "filenameTemplate",
							Type:     ActionParamTypeString,
							Label:    "Template Nama File",
							Required: true,
							Default:  "{{references}} - {{buyerName}}",
							UI: &ActionParamUISpec{
								Editor: "template",
							},
							Suggestions: []ActionParamSuggestion{
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
							Description: "Gunakan placeholder seperti {{references}} - {{buyerName}}. Ekstensi .pdf akan ditambahkan otomatis.",
							Placeholder: "{{references}} - {{buyerName}}",
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
					ActionType:  "export_tax_invoice_zip",
					Label:       "Export Tax Invoice ZIP",
					Description: "Export selected tax invoices to ZIP package.",
					Enabled:     false,
					Reason:      "not implemented yet",
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  true,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Params:  []ActionParamFieldSpec{},
					Outputs: []ActionOutputSpec{},
				},
			},
		}, true
	case DocumentTypePDFBukpotBPPU:
		return buildBukpotDocumentTypeSpec(
			docType,
			"BPPU",
			"Dokumen PDF bukti potong BPPU untuk ekstraksi data bukpot.",
		), true
	case DocumentTypePDFBukpotBP21:
		return buildBukpotDocumentTypeSpec(
			docType,
			"BP21",
			"Dokumen PDF bukti potong BP21 untuk ekstraksi data bukpot.",
		), true
	case DocumentTypePDFBukpotBPA1:
		return buildBukpotDocumentTypeSpec(
			docType,
			"BPA1",
			"Dokumen PDF bukti potong BPA1 untuk ekstraksi data bukpot.",
		), true
	default:
		return DocumentTypeSpec{}, false
	}
}

func buildBukpotDocumentTypeSpec(
	docType DocumentType,
	label string,
	description string,
) DocumentTypeSpec {
	actions := []ActionSpec{}
	actions = append(actions, ActionSpec{
		ActionType:  "rename_bukpot",
		Label:       "Rename Bukpot",
		Description: "Ganti nama file bukpot berdasarkan template placeholder dan hasilkan ZIP.",
		Enabled:     true,
		Selection: ActionSelectionSpec{
			Mode:           "manual",
			AllowCheckAll:  true,
			AllowedStatus:  []string{"ready", "warning"},
			MinDocumentCnt: 1,
		},
		Params: []ActionParamFieldSpec{
			{
				Key:      "filenameTemplate",
				Type:     ActionParamTypeString,
				Label:    "Template Nama File",
				Required: true,
				Default:  "{{nomorBuktiPotong}} - {{namaPenerima}}",
				UI: &ActionParamUISpec{
					Editor: "template",
				},
				Suggestions: buildBukpotTemplateSuggestions(docType),
				Description: "Gunakan placeholder yang tersedia. Ekstensi .pdf akan ditambahkan otomatis.",
				Placeholder: "{{nomorBuktiPotong}} - {{namaPenerima}}",
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

	if docType == DocumentTypePDFBukpotBPPU || docType == DocumentTypePDFBukpotBP21 {
		actions = append(actions, ActionSpec{
			ActionType:  "rename_by_category",
			Label:       "Rename by Category",
			Description: "Kelompokkan bukpot ke folder kategori dari Dokumen Referensi Nomor lalu hasilkan ZIP.",
			Enabled:     true,
			Selection: ActionSelectionSpec{
				Mode:           "manual",
				AllowCheckAll:  true,
				AllowedStatus:  []string{"ready", "warning"},
				MinDocumentCnt: 1,
			},
			Params: []ActionParamFieldSpec{},
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

	return DocumentTypeSpec{
		DocumentType: docType,
		Label:        label,
		Description:  description,
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

func buildBukpotTemplateSuggestions(docType DocumentType) []ActionParamSuggestion {
	suggestions := []ActionParamSuggestion{
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

	switch docType {
	case DocumentTypePDFBukpotBPPU, DocumentTypePDFBukpotBP21:
		suggestions = append(suggestions,
			ActionParamSuggestion{Token: "dokumenReferensiNomor", Label: "Nomor Dokumen Referensi", Example: "{{dokumenReferensiNomor}}"},
			ActionParamSuggestion{Token: "dokumenReferensiJenis", Label: "Jenis Dokumen Referensi", Example: "{{dokumenReferensiJenis}}"},
			ActionParamSuggestion{Token: "dokumenReferensiTanggal", Label: "Tanggal Dokumen Referensi", Example: "{{dokumenReferensiTanggal}}"},
			ActionParamSuggestion{Token: "masaPajak", Label: "Masa Pajak", Example: "{{masaPajak}}"},
		)
	case DocumentTypePDFBukpotBPA1:
		suggestions = append(suggestions,
			ActionParamSuggestion{Token: "periodePenghasilan", Label: "Periode Penghasilan", Example: "{{periodePenghasilan}}"},
			ActionParamSuggestion{Token: "posisi", Label: "Posisi", Example: "{{posisi}}"},
			ActionParamSuggestion{Token: "statusPtkp", Label: "Status PTKP", Example: "{{statusPtkp}}"},
		)
	}

	return suggestions
}
