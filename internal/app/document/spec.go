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
	Description string                  `json:"description,omitempty"`
	Placeholder string                  `json:"placeholder,omitempty"`
}

type ActionParamOptionSpec struct {
	Label string `json:"label"`
	Value string `json:"value"`
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
							Type:        ActionParamTypeString,
							Label:       "Filename Prefix",
							Required:    false,
							Default:     "faktur-keluaran",
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
					{Kind: "raw", Required: true},
					{Kind: "normalized", Required: true},
					{Kind: "audit", Required: false},
				},
			},
			Actions: []ActionSpec{
				{
					ActionType:  "cashflow_to_pay_bills",
					Label:       "Cashflow to Pay Bills",
					Description: "Convert selected workbook to pay bills import package.",
					Enabled:     false,
					Reason:      "not implemented yet",
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  false,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Params: []ActionParamFieldSpec{
						{
							Key:         "sheetName",
							Type:        ActionParamTypeString,
							Label:       "Worksheet Name",
							Required:    true,
							Description: "Worksheet name to convert.",
							Placeholder: "Sheet1",
						},
						{
							Key:         "outputFilename",
							Type:        ActionParamTypeString,
							Label:       "Output Filename",
							Required:    false,
							Default:     "OUTPUT",
							Description: "Output file name prefix.",
							Placeholder: "OUTPUT",
						},
						{
							Key:         "headerRowNumber",
							Type:        ActionParamTypeInt,
							Label:       "Header Row Number",
							Required:    true,
							Default:     1,
							Description: "Detected or selected header row number.",
							Placeholder: "1",
						},
						{
							Key:         "skipPositiveTotal",
							Type:        ActionParamTypeBoolean,
							Label:       "Skip Positive Total",
							Required:    false,
							Default:     true,
							Description: "If true, rows with positive total can be skipped by converter.",
						},
						{
							Key:      "cashflowFormat",
							Type:     ActionParamTypeString,
							Label:    "Cashflow Format",
							Required: true,
							Default:  "default",
							Options: []ActionParamOptionSpec{
								{Label: "Default", Value: "default"},
								{Label: "Influencer", Value: "influencer"},
							},
						},
						{
							Key:      "cashflowType",
							Type:     ActionParamTypeString,
							Label:    "Output Type",
							Required: true,
							Default:  "spend_money",
							Options: []ActionParamOptionSpec{
								{Label: "Spend Money", Value: "spend_money"},
								{Label: "Receive Money", Value: "receive_money"},
							},
						},
						{
							Key:         "ledgerSnapshotId",
							Type:        ActionParamTypeString,
							Label:       "Ledger Snapshot Id",
							Required:    true,
							Description: "Uploaded ledger snapshot artifact id.",
							Placeholder: "ledger-202603",
						},
						{
							Key:      "remarkDelimiter",
							Type:     ActionParamTypeString,
							Label:    "Remark Delimiter",
							Required: false,
							Default:  "*",
							Rules: []ActionParamRuleSpec{
								{
									Type:    ActionParamRuleRequiredIf,
									Field:   "cashflowFormat",
									Equals:  "default",
									Message: "Remark Delimiter is required when cashflow format is default",
								},
							},
						},
						{
							Key:      "otherCostsAccountCode",
							Type:     ActionParamTypeString,
							Label:    "Other Costs Account Code",
							Required: false,
							Default:  "62099",
							Rules: []ActionParamRuleSpec{
								{
									Type:    ActionParamRuleRequiredIf,
									Field:   "cashflowFormat",
									Equals:  "default",
									Message: "Other Costs Account Code is required when cashflow format is default",
								},
							},
						},
						{
							Key:      "defaultIAccountCode",
							Type:     ActionParamTypeString,
							Label:    "Default Influencer Account Code",
							Required: false,
							Default:  "62004",
							Rules: []ActionParamRuleSpec{
								{
									Type:    ActionParamRuleRequiredIf,
									Field:   "cashflowFormat",
									Equals:  "influencer",
									Message: "Default Influencer Account Code is required when cashflow format is influencer",
								},
							},
						},
						{
							Key:      "defaultBAccountCode",
							Type:     ActionParamTypeString,
							Label:    "Default Bank Account Code",
							Required: false,
							Default:  "90900",
							Rules: []ActionParamRuleSpec{
								{
									Type:    ActionParamRuleRequiredIf,
									Field:   "cashflowFormat",
									Equals:  "influencer",
									Message: "Default Bank Account Code is required when cashflow format is influencer",
								},
							},
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
				{
					ActionType:  "spreadsheet_to_myob",
					Label:       "Spreadsheet to MYOB",
					Description: "Convert selected spreadsheet to MYOB format using selected template.",
					Enabled:     true,
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  false,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Params: []ActionParamFieldSpec{
						{
							Key:      "templateKey",
							Type:     ActionParamTypeString,
							Label:    "MYOB Template",
							Required: true,
							Default:  "receive_payments",
							Options: []ActionParamOptionSpec{
								{Label: "Receive Payments", Value: "receive_payments"},
								{Label: "Pay Bills", Value: "pay_bills"},
							},
							Description: "Choose MYOB template profile.",
						},
						{
							Key:         "sheetName",
							Type:        ActionParamTypeString,
							Label:       "Worksheet Name",
							Required:    true,
							Description: "Worksheet name to convert.",
							Placeholder: "Sheet1",
						},
						{
							Key:         "headerRowNumber",
							Type:        ActionParamTypeInt,
							Label:       "Header Row Number",
							Required:    true,
							Default:     1,
							Description: "Detected or selected header row number.",
							Placeholder: "1",
						},
						{
							Key:         "remarkDelimiter",
							Type:        ActionParamTypeString,
							Label:       "Remark Delimiter",
							Required:    false,
							Default:     "*",
							Description: "Delimiter used to split memo segments (for example PPN*MATERAI).",
							Placeholder: "*",
						},
						{
							Key:         "receiveLedgerSnapshotRef",
							Type:        ActionParamTypeString,
							Label:       "Sales Ledger Snapshot Reference",
							Required:    false,
							Description: "Reference ID for sales ledger snapshot artifact (Receive Payments).",
							Placeholder: "sales-ledger-2026-03",
							Rules: []ActionParamRuleSpec{
								{
									Type:    ActionParamRuleRequiredIf,
									Field:   "templateKey",
									Equals:  "receive_payments",
									Message: "Sales Ledger Snapshot Reference is required for Receive Payments template",
								},
							},
						},
						{
							Key:         "payBillsLedgerSnapshotRef",
							Type:        ActionParamTypeString,
							Label:       "Payables Ledger Snapshot Reference",
							Required:    false,
							Description: "Reference ID for payables ledger snapshot artifact (Pay Bills).",
							Placeholder: "payables-ledger-2026-03",
							Rules: []ActionParamRuleSpec{
								{
									Type:    ActionParamRuleRequiredIf,
									Field:   "templateKey",
									Equals:  "pay_bills",
									Message: "Payables Ledger Snapshot Reference is required for Pay Bills template",
								},
							},
						},
						{
							Key:         "outputFilename",
							Type:        ActionParamTypeString,
							Label:       "Output Filename",
							Required:    false,
							Default:     "OUTPUT_MYOB",
							Description: "Output file name prefix.",
							Placeholder: "OUTPUT_MYOB",
						},
						{
							Key:      "rowSelectionMode",
							Type:     ActionParamTypeString,
							Label:    "Row Selection Mode",
							Required: false,
							Default:  "all",
							Options: []ActionParamOptionSpec{
								{Label: "All Rows", Value: "all"},
								{Label: "Manual", Value: "manual"},
							},
							Description: "Selection mode from spreadsheet setup.",
						},
						{
							Key:         "selectedRowRanges",
							Type:        ActionParamTypeString,
							Label:       "Selected Row Ranges",
							Required:    false,
							Description: "Compressed selected row ranges, for example 2-120,130,140-145.",
							Placeholder: "2-120,130,140-145",
						},
						{
							Key:         "columnMapJson",
							Type:        ActionParamTypeString,
							Label:       "Column Map JSON",
							Required:    false,
							Description: "Spreadsheet header mapping stored from setup.",
							Placeholder: "{\"date\":\"2\",\"total\":\"8\"}",
						},
					},
					ArtifactInputs: []ActionArtifactInputSpec{
						{
							Key:              "receiveLedgerSnapshotRef",
							ValueType:        "file_ref",
							Label:            "Sales Ledger Snapshot",
							Required:         false,
							Description:      "Required for Receive Payments template; upload .txt sales ledger snapshot and use its reference.",
							AcceptExtensions: []string{".txt"},
							AcceptMIMETypes:  []string{"text/plain"},
						},
						{
							Key:              "payBillsLedgerSnapshotRef",
							ValueType:        "file_ref",
							Label:            "Payables Ledger Snapshot",
							Required:         false,
							Description:      "Required for Pay Bills template; upload .txt payables ledger snapshot and use its reference.",
							AcceptExtensions: []string{".txt"},
							AcceptMIMETypes:  []string{"text/plain"},
						},
					},
					Columns: []ActionColumnSpec{
						{
							Key:      "date",
							Label:    "Tanggal",
							Type:     "date",
							Required: true,
							Aliases:  []string{"Tanggal", "Date"},
							Group:    "GENERAL",
						},
						{
							Key:      "information",
							Label:    "Keterangan",
							Type:     "string",
							Required: true,
							Aliases:  []string{"Keterangan", "Note", "Notes"},
							Group:    "GENERAL",
						},
						{
							Key:      "partyName",
							Label:    "Nama Customer / Supplier",
							Type:     "string",
							Required: false,
							Aliases:  []string{"Nama Customer / Supplier", "Customer", "Supplier"},
							Group:    "GENERAL",
							Rules: []ActionParamRuleSpec{
								{
									Type:   ActionParamRuleRequiredIf,
									Field:  "templateKey",
									Equals: "receive_payments",
								},
								{
									Type:   ActionParamRuleRequiredIf,
									Field:  "templateKey",
									Equals: "pay_bills",
								},
							},
						},
						{
							Key:      "total",
							Label:    "Total",
							Type:     "float",
							Required: true,
							Aliases:  []string{"IDR", "idr", "Total"},
							Group:    "GENERAL",
						},
						{
							Key:      "remark",
							Label:    "Catatan",
							Type:     "string",
							Required: false,
							Aliases:  []string{"catatan", "Remark", "WHT"},
							Group:    "GENERAL",
						},
					},
					Outputs: []ActionOutputSpec{
						{
							Kind:       "file",
							MimeType:   "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
							Ext:        "xlsx",
							DownloadOK: true,
						},
						{
							Kind:       "payload",
							DownloadOK: false,
						},
					},
				},
				{
					ActionType:  "cashflow_to_receive_payments",
					Label:       "Cashflow to Receive Payments",
					Description: "Legacy action. Use Spreadsheet to MYOB with Receive Payments template.",
					Enabled:     false,
					Reason:      "deprecated: use spreadsheet_to_myob",
					Selection: ActionSelectionSpec{
						Mode:           "manual",
						AllowCheckAll:  false,
						AllowedStatus:  []string{"ready", "warning"},
						MinDocumentCnt: 1,
					},
					Params: []ActionParamFieldSpec{
						{
							Key:         "sheetName",
							Type:        ActionParamTypeString,
							Label:       "Worksheet Name",
							Required:    true,
							Description: "Worksheet name to convert.",
							Placeholder: "Sheet1",
						},
						{
							Key:         "headerRowNumber",
							Type:        ActionParamTypeInt,
							Label:       "Header Row Number",
							Required:    true,
							Default:     1,
							Description: "Detected or selected header row number.",
							Placeholder: "1",
						},
						{
							Key:         "remarkDelimiter",
							Type:        ActionParamTypeString,
							Label:       "Remark Delimiter",
							Required:    false,
							Default:     "*",
							Description: "Delimiter used to split memo segments (for example PPN*MATERAI).",
							Placeholder: "*",
						},
						{
							Key:         "ledgerSnapshotRef",
							Type:        ActionParamTypeString,
							Label:       "Ledger Snapshot Reference",
							Required:    true,
							Description: "Reference ID to uploaded ledger snapshot artifact.",
							Placeholder: "ledger-snapshot-2026-03",
						},
						{
							Key:         "outputFilename",
							Type:        ActionParamTypeString,
							Label:       "Output Filename",
							Required:    false,
							Default:     "OUTPUT_RECEIVE_PAYMENTS",
							Description: "Output file name prefix.",
							Placeholder: "OUTPUT_RECEIVE_PAYMENTS",
						},
					},
					ArtifactInputs: []ActionArtifactInputSpec{
						{
							Key:              "ledgerSnapshotRef",
							ValueType:        "file_ref",
							Label:            "Ledger Snapshot",
							Required:         true,
							Description:      "Ledger snapshot input file used as lookup/reference for receive payments.",
							AcceptExtensions: []string{".txt"},
							AcceptMIMETypes:  []string{"text/plain"},
						},
					},
					Columns: []ActionColumnSpec{
						{
							Key:      "date",
							Label:    "Tanggal",
							Type:     "date",
							Required: true,
							Aliases:  []string{"Tanggal", "Date"},
							Group:    "GENERAL",
						},
						{
							Key:      "information",
							Label:    "Keterangan",
							Type:     "string",
							Required: true,
							Aliases:  []string{"Keterangan", "Note", "Notes"},
							Group:    "GENERAL",
						},
						{
							Key:      "partyName",
							Label:    "Nama Customer / Supplier",
							Type:     "string",
							Required: false,
							Aliases:  []string{"Nama Customer / Supplier", "Customer", "Supplier"},
							Group:    "GENERAL",
						},
						{
							Key:      "total",
							Label:    "Total",
							Type:     "float",
							Required: true,
							Aliases:  []string{"IDR", "idr", "Total"},
							Group:    "GENERAL",
						},
						{
							Key:      "remark",
							Label:    "Catatan",
							Type:     "string",
							Required: false,
							Aliases:  []string{"catatan", "Remark", "WHT"},
							Group:    "GENERAL",
						},
					},
					Outputs: []ActionOutputSpec{
						{
							Kind:       "file",
							MimeType:   "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
							Ext:        "xlsx",
							DownloadOK: true,
						},
						{
							Kind:       "payload",
							DownloadOK: false,
						},
					},
				},
			},
		}, true
	default:
		return DocumentTypeSpec{}, false
	}
}
