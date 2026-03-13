package document

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
	Key         string `json:"key"`
	Type        string `json:"type"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
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

type ActionSpec struct {
	ActionType  string                 `json:"actionType"`
	Label       string                 `json:"label"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Reason      string                 `json:"reason,omitempty"`
	Selection   ActionSelectionSpec    `json:"selection"`
	Params      []ActionParamFieldSpec `json:"params"`
	Outputs     []ActionOutputSpec     `json:"outputs"`
}

type DocumentTypeSpec struct {
	DocumentType DocumentType   `json:"documentType"`
	Label        string         `json:"label"`
	Description  string         `json:"description,omitempty"`
	Upload       UploadRuleSpec `json:"upload"`
	Ingest       IngestRuleSpec `json:"ingest"`
	Actions      []ActionSpec   `json:"actions"`
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
					Description: "Export selected invoices to e-Faktur Excel format.",
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
							Type:        "string",
							Label:       "Filename Prefix",
							Required:    false,
							Description: "Optional prefix for exported file name.",
							Placeholder: "faktur-keluaran",
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
			Label:        "Tax Invoice",
			Description:  "PDF tax invoice for extraction and future export actions.",
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
	case DocumentTypeXLSXCashflow:
		return DocumentTypeSpec{
			DocumentType: docType,
			Label:        "Cashflow",
			Description:  "Spreadsheet cashflow document.",
			Upload: UploadRuleSpec{
				AcceptExtensions: []string{".xlsx", ".xls"},
				AcceptMIMETypes:  []string{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.ms-excel"},
				MaxChunkMB:       15,
				MaxFilesPerBatch: 20,
			},
			Ingest: IngestRuleSpec{
				KeepRaw:            true,
				DeleteTempAfterRun: true,
				Artifacts: []ArtifactRuleSpec{
					{Kind: "normalized", Required: true},
					{Kind: "audit", Required: false},
				},
			},
			Actions: []ActionSpec{},
		}, true
	default:
		return DocumentTypeSpec{}, false
	}
}
