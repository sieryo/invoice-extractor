package document

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/xuri/excelize/v2"
)

type XLSXCashflowProcessor struct {
	fileStore dfile.FileStore
}

func NewXLSXCashflowProcessor(fileStore dfile.FileStore) *XLSXCashflowProcessor {
	return &XLSXCashflowProcessor{
		fileStore: fileStore,
	}
}

func (p *XLSXCashflowProcessor) Type() DocumentType {
	return DocumentTypeXLSXCashflow
}

func (p *XLSXCashflowProcessor) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
	now := time.Now()
	result := IngestResult{
		BatchID:      req.RequestID,
		CollectionID: req.CollectionID,
		DocumentType: string(req.DocumentType),
		Items:        make([]IngestItemResult, 0, len(req.Sources)),
		StartedAt:    now,
	}

	for _, source := range req.Sources {
		item, err := p.buildItem(ctx, req.CollectionID, source)
		if err != nil {
			result.Items = append(result.Items, IngestItemResult{
				SourceID:     source.SourceID,
				OriginalName: source.OriginalName,
				SHA256:       source.SHA256,
				Status:       IngestStatusFailed,
				Message:      "failed to parse cashflow workbook",
				Errors:       []string{err.Error()},
			})
			result.Failed++
			continue
		}
		result.Items = append(result.Items, item)
		switch item.Status {
		case IngestStatusReady:
			result.Success++
		case IngestStatusWarning:
			result.Warning++
		case IngestStatusDuplicate:
			result.Duplicate++
		default:
			result.Failed++
		}
	}

	result.Total = len(result.Items)
	result.FinishedAt = time.Now()
	return result, nil
}

func (p *XLSXCashflowProcessor) RunAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	switch {
	case strings.EqualFold(strings.TrimSpace(req.ActionType), "spreadsheet_to_myob"):
		return p.runSpreadsheetToMYOB(ctx, req)
	case strings.EqualFold(strings.TrimSpace(req.ActionType), "cashflow_to_receive_payments"):
		return p.runCashflowToReceivePayments(ctx, req)
	default:
		return ActionResult{
			ActionID:   req.ActionID,
			ActionType: req.ActionType,
			Status:     "failed",
			StartedAt:  req.RequestedAt,
			FinishedAt: time.Now(),
		}, fmt.Errorf("%w: action %s for %s", ErrProcessorNotImplemented, req.ActionType, p.Type())
	}
}

type xlsxNormalizedPayload struct {
	SourceID     string           `json:"source_id"`
	SourceName   string           `json:"source_name"`
	SourceSHA256 string           `json:"source_sha256"`
	DocumentType string           `json:"document_type"`
	Workbook     xlsxWorkbookMeta `json:"workbook"`
	Preview      map[string]any   `json:"preview,omitempty"`
	Warnings     []string         `json:"warnings,omitempty"`
	ProcessedAt  time.Time        `json:"processed_at"`
}

type xlsxWorkbookMeta struct {
	SheetNames []string `json:"sheet_names"`
	SheetCount int      `json:"sheet_count"`
}

type cashflowReceivePaymentsParams struct {
	TemplateKey               string `json:"templateKey"`
	SheetName                 string `json:"sheetName"`
	HeaderRowNumber           int    `json:"headerRowNumber"`
	RemarkDelimiter           string `json:"remarkDelimiter"`
	LedgerSnapshotRef         string `json:"ledgerSnapshotRef"`
	ReceiveLedgerSnapshotRef  string `json:"receiveLedgerSnapshotRef"`
	PayBillsLedgerSnapshotRef string `json:"payBillsLedgerSnapshotRef"`
	OutputFilename            string `json:"outputFilename"`
}

type cashflowReceivePaymentsSummary struct {
	ActionID          string                                   `json:"action_id"`
	ActionType        string                                   `json:"action_type"`
	CollectionID      string                                   `json:"collection_id"`
	TemplateKey       string                                   `json:"template_key"`
	SheetName         string                                   `json:"sheet_name"`
	HeaderRowNumber   int                                      `json:"header_row_number"`
	LedgerSnapshotRef string                                   `json:"ledger_snapshot_ref"`
	RemarkDelimiter   string                                   `json:"remark_delimiter"`
	GeneratedAt       time.Time                                `json:"generated_at"`
	Documents         []cashflowReceivePaymentsSummaryDocument `json:"documents"`
}

type cashflowReceivePaymentsSummaryDocument struct {
	DocumentID string `json:"document_id"`
	SourceName string `json:"source_name"`
	SheetName  string `json:"sheet_name"`
	OutputName string `json:"output_name"`
	RowsCopied int    `json:"rows_copied"`
	Status     string `json:"status"`
	Message    string `json:"message"`
}

func (p *XLSXCashflowProcessor) buildItem(
	ctx context.Context,
	collectionID string,
	source IngestSource,
) (IngestItemResult, error) {
	rawBytes, err := os.ReadFile(source.TempPath)
	if err != nil {
		return IngestItemResult{}, err
	}

	rawName := source.SourceID + filepath.Ext(source.OriginalName)
	if err := p.fileStore.WriteFile(ctx, collectionID, rawName, rawBytes); err != nil {
		return IngestItemResult{}, err
	}

	workbook, warnings, err := p.readWorkbookMeta(rawBytes)
	if err != nil {
		return IngestItemResult{}, err
	}

	normalizedPayload := xlsxNormalizedPayload{
		SourceID:     source.SourceID,
		SourceName:   source.OriginalName,
		SourceSHA256: source.SHA256,
		DocumentType: string(DocumentTypeXLSXCashflow),
		Workbook:     workbook,
		Preview: map[string]any{
			"defaultSheet": firstOrEmpty(workbook.SheetNames),
		},
		Warnings:    warnings,
		ProcessedAt: time.Now().UTC(),
	}

	normalizedBytes, err := json.Marshal(normalizedPayload)
	if err != nil {
		return IngestItemResult{}, err
	}

	normalizedName := "normalized_" + source.SourceID + ".json"
	if err := p.fileStore.WriteFile(ctx, collectionID, normalizedName, normalizedBytes); err != nil {
		return IngestItemResult{}, err
	}

	auditName := "audit_" + source.SourceID + ".json"
	auditRef, _ := p.fileStore.SaveAudit(ctx, collectionID, auditName, normalizedBytes)

	status := IngestStatusReady
	if len(workbook.SheetNames) == 0 || len(warnings) > 0 {
		status = IngestStatusWarning
	}

	msg := "cashflow workbook parsed"
	if len(workbook.SheetNames) == 0 {
		msg = "workbook has no sheet"
	}

	artifacts := []Artifact{
		{
			Kind:     "raw",
			ObjectID: rawName,
			MimeType: source.MimeType,
			Size:     int64(len(rawBytes)),
		},
		{
			Kind:     "normalized",
			ObjectID: normalizedName,
			MimeType: "application/json",
			Size:     int64(len(normalizedBytes)),
		},
	}
	if auditRef != "" {
		artifacts = append(artifacts, Artifact{
			Kind:     "audit",
			ObjectID: auditRef,
			MimeType: "application/json",
		})
	}

	return IngestItemResult{
		SourceID:     source.SourceID,
		OriginalName: source.OriginalName,
		SHA256:       source.SHA256,
		Status:       status,
		Message:      msg,
		Warnings:     warnings,
		Artifacts:    artifacts,
	}, nil
}

func (p *XLSXCashflowProcessor) readWorkbookMeta(raw []byte) (xlsxWorkbookMeta, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(raw))
	if err != nil {
		return xlsxWorkbookMeta{}, nil, err
	}
	defer f.Close()

	sheets := f.GetSheetList()
	workbook := xlsxWorkbookMeta{
		SheetNames: sheets,
		SheetCount: len(sheets),
	}

	warnings := make([]string, 0)
	if len(sheets) == 0 {
		warnings = append(warnings, "no sheet detected in workbook")
	}

	return workbook, warnings, nil
}

func (p *XLSXCashflowProcessor) runCashflowToReceivePayments(
	ctx context.Context,
	req ActionRequest,
) (ActionResult, error) {
	result := ActionResult{
		ActionID:    req.ActionID,
		ActionType:  req.ActionType,
		StartedAt:   req.RequestedAt,
		ItemResults: make([]ActionItemResult, 0, len(req.SnapshotDocs)),
		Outputs:     []ActionOutput{},
	}

	params, err := parseCashflowReceivePaymentsParams(req.Params)
	if err != nil {
		result.Status = "failed"
		result.Message = "invalid action params"
		result.FinishedAt = time.Now()
		return result, err
	}

	if len(req.SnapshotDocs) == 0 {
		result.Status = "failed"
		result.Message = "snapshot is empty"
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("snapshot is empty")
	}

	summary := cashflowReceivePaymentsSummary{
		ActionID:          req.ActionID,
		ActionType:        req.ActionType,
		CollectionID:      req.CollectionID,
		TemplateKey:       params.TemplateKey,
		SheetName:         params.SheetName,
		HeaderRowNumber:   params.HeaderRowNumber,
		LedgerSnapshotRef: params.LedgerSnapshotRef,
		RemarkDelimiter:   params.RemarkDelimiter,
		GeneratedAt:       time.Now().UTC(),
		Documents:         make([]cashflowReceivePaymentsSummaryDocument, 0, len(req.SnapshotDocs)),
	}

	for idx, doc := range req.SnapshotDocs {
		item := ActionItemResult{
			DocumentID: doc.DocumentID,
			Status:     "failed",
			Message:    "failed to process workbook",
		}
		summaryItem := cashflowReceivePaymentsSummaryDocument{
			DocumentID: doc.DocumentID,
			SourceName: doc.SourceName,
			SheetName:  params.SheetName,
			Status:     "failed",
			Message:    "failed to process workbook",
		}

		rawName := filepath.Base(strings.TrimSpace(doc.RawRef))
		if rawName == "" || rawName == "." {
			item.Message = "raw workbook artifact is missing"
			item.Error = "raw workbook artifact is missing"
			summaryItem.Message = item.Message
			result.ItemResults = append(result.ItemResults, item)
			summary.Documents = append(summary.Documents, summaryItem)
			continue
		}

		rawBytes, readErr := p.fileStore.Read(ctx, req.CollectionID, rawName)
		if readErr != nil {
			item.Message = "failed to read workbook artifact"
			item.Error = readErr.Error()
			summaryItem.Message = item.Message
			result.ItemResults = append(result.ItemResults, item)
			summary.Documents = append(summary.Documents, summaryItem)
			continue
		}

		sourceWorkbook, openErr := excelize.OpenReader(bytes.NewReader(rawBytes))
		if openErr != nil {
			item.Message = "failed to open workbook"
			item.Error = openErr.Error()
			summaryItem.Message = item.Message
			result.ItemResults = append(result.ItemResults, item)
			summary.Documents = append(summary.Documents, summaryItem)
			continue
		}

		sheetName, resolveErr := resolveWorkbookSheet(sourceWorkbook, params.SheetName)
		if resolveErr != nil {
			_ = sourceWorkbook.Close()
			item.Message = "target sheet is not available"
			item.Error = resolveErr.Error()
			summaryItem.Message = item.Message
			result.ItemResults = append(result.ItemResults, item)
			summary.Documents = append(summary.Documents, summaryItem)
			continue
		}

		sourceRows, rowErr := sourceWorkbook.GetRows(sheetName)
		_ = sourceWorkbook.Close()
		if rowErr != nil {
			item.Message = "failed to read sheet rows"
			item.Error = rowErr.Error()
			summaryItem.Message = item.Message
			result.ItemResults = append(result.ItemResults, item)
			summary.Documents = append(summary.Documents, summaryItem)
			continue
		}

		outputBytes, copiedRows, buildErr := buildReceivePaymentsWorkbook(sourceRows, doc, sheetName, params)
		if buildErr != nil {
			item.Message = "failed to build output workbook"
			item.Error = buildErr.Error()
			summaryItem.Message = item.Message
			result.ItemResults = append(result.ItemResults, item)
			summary.Documents = append(summary.Documents, summaryItem)
			continue
		}

		outputName := buildReceivePaymentsOutputName(params.OutputFilename, doc.SourceName, idx)
		outputRef, saveErr := p.fileStore.SaveArchive(ctx, req.CollectionID, outputName, outputBytes)
		if saveErr != nil {
			item.Message = "failed to save output workbook"
			item.Error = saveErr.Error()
			summaryItem.Message = item.Message
			result.ItemResults = append(result.ItemResults, item)
			summary.Documents = append(summary.Documents, summaryItem)
			continue
		}

		checksum := sha256.Sum256(outputBytes)
		result.Outputs = append(result.Outputs, ActionOutput{
			Kind:      "file",
			Name:      outputName,
			ObjectRef: outputRef,
			MimeType:  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			SizeBytes: int64(len(outputBytes)),
			Checksum:  hex.EncodeToString(checksum[:]),
		})

		item.Status = "success"
		item.Message = fmt.Sprintf("receive payments workbook generated (%d rows)", copiedRows)
		if copiedRows == 0 {
			item.Status = "warning"
			item.Message = "no rows copied from source sheet"
		}

		summaryItem.Status = item.Status
		summaryItem.Message = item.Message
		summaryItem.OutputName = outputName
		summaryItem.RowsCopied = copiedRows
		summaryItem.SheetName = sheetName

		result.ItemResults = append(result.ItemResults, item)
		summary.Documents = append(summary.Documents, summaryItem)
	}

	result.Total = len(req.SnapshotDocs)
	result.Success, result.Warning, result.Failed, result.Skipped = summarizeActionItems(result.ItemResults)

	summaryBytes, marshalErr := json.Marshal(summary)
	if marshalErr == nil {
		auditName := fmt.Sprintf("action_%s_receive_payments_summary.json", req.ActionID)
		payloadRef, payloadErr := p.fileStore.SaveAudit(ctx, req.CollectionID, auditName, summaryBytes)
		if payloadErr == nil && strings.TrimSpace(payloadRef) != "" {
			sum := sha256.Sum256(summaryBytes)
			result.Outputs = append(result.Outputs, ActionOutput{
				Kind:      "payload",
				Name:      "receive_payments_summary",
				ObjectRef: payloadRef,
				MimeType:  "application/json",
				SizeBytes: int64(len(summaryBytes)),
				Checksum:  hex.EncodeToString(sum[:]),
			})
		}
	}

	switch {
	case result.Failed == 0 && result.Warning == 0:
		result.Status = "success"
		result.Message = "receive payments conversion completed"
	case result.Failed == 0 && result.Warning > 0:
		result.Status = "warning"
		result.Message = "receive payments conversion completed with warnings"
	case result.Success > 0 || result.Warning > 0:
		result.Status = "partial"
		result.Message = "receive payments conversion partially completed"
	default:
		result.Status = "failed"
		result.Message = "receive payments conversion failed"
	}

	result.FinishedAt = time.Now()
	return result, nil
}

func parseCashflowReceivePaymentsParams(raw json.RawMessage) (cashflowReceivePaymentsParams, error) {
	params := cashflowReceivePaymentsParams{
		HeaderRowNumber: 1,
		RemarkDelimiter: "*",
		TemplateKey:     "receive_payments",
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return params, fmt.Errorf("params are required")
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return params, err
	}

	params.TemplateKey = strings.TrimSpace(strings.ToLower(params.TemplateKey))
	params.SheetName = strings.TrimSpace(params.SheetName)
	params.LedgerSnapshotRef = strings.TrimSpace(params.LedgerSnapshotRef)
	params.ReceiveLedgerSnapshotRef = strings.TrimSpace(params.ReceiveLedgerSnapshotRef)
	params.PayBillsLedgerSnapshotRef = strings.TrimSpace(params.PayBillsLedgerSnapshotRef)
	params.RemarkDelimiter = strings.TrimSpace(params.RemarkDelimiter)
	params.OutputFilename = strings.TrimSpace(params.OutputFilename)

	if params.TemplateKey == "" {
		params.TemplateKey = "receive_payments"
	}
	if params.SheetName == "" {
		return params, fmt.Errorf("sheetName is required")
	}
	if params.HeaderRowNumber <= 0 {
		params.HeaderRowNumber = 1
	}
	if params.LedgerSnapshotRef == "" {
		switch params.TemplateKey {
		case "pay_bills":
			params.LedgerSnapshotRef = params.PayBillsLedgerSnapshotRef
		default:
			params.LedgerSnapshotRef = params.ReceiveLedgerSnapshotRef
		}
	}
	if params.LedgerSnapshotRef == "" {
		switch params.TemplateKey {
		case "pay_bills":
			return params, fmt.Errorf("payBillsLedgerSnapshotRef is required")
		default:
			return params, fmt.Errorf("receiveLedgerSnapshotRef is required")
		}
	}
	if params.RemarkDelimiter == "" {
		params.RemarkDelimiter = "*"
	}
	if params.OutputFilename == "" {
		if params.TemplateKey == "pay_bills" {
			params.OutputFilename = "OUTPUT_PAY_BILLS"
		} else {
			params.OutputFilename = "OUTPUT_RECEIVE_PAYMENTS"
		}
	}

	return params, nil
}

func resolveWorkbookSheet(workbook *excelize.File, preferred string) (string, error) {
	target := strings.TrimSpace(preferred)
	if target == "" {
		return "", fmt.Errorf("sheet name is required")
	}
	for _, name := range workbook.GetSheetList() {
		if strings.EqualFold(strings.TrimSpace(name), target) {
			return name, nil
		}
	}
	return "", fmt.Errorf("sheet %q not found", target)
}

func buildReceivePaymentsWorkbook(
	sourceRows [][]string,
	doc ActionSnapshotDocument,
	sourceSheet string,
	params cashflowReceivePaymentsParams,
) ([]byte, int, error) {
	out := excelize.NewFile()
	outSheet := "ReceivePayments"
	baseSheet := out.GetSheetName(out.GetActiveSheetIndex())
	if err := out.SetSheetName(baseSheet, outSheet); err != nil {
		_ = out.Close()
		return nil, 0, err
	}

	_ = out.SetCellStr(outSheet, "A1", "Source Document")
	_ = out.SetCellStr(outSheet, "B1", doc.SourceName)
	_ = out.SetCellStr(outSheet, "A2", "Source Sheet")
	_ = out.SetCellStr(outSheet, "B2", sourceSheet)
	_ = out.SetCellStr(outSheet, "A3", "Ledger Snapshot Ref")
	_ = out.SetCellStr(outSheet, "B3", params.LedgerSnapshotRef)
	_ = out.SetCellStr(outSheet, "A4", "Remark Delimiter")
	_ = out.SetCellStr(outSheet, "B4", params.RemarkDelimiter)
	_ = out.SetCellStr(outSheet, "A6", "Extracted Sheet Rows")

	startRow := params.HeaderRowNumber
	if startRow <= 0 {
		startRow = 1
	}

	targetRow := 7
	copiedRows := 0
	for rowIdx := startRow - 1; rowIdx < len(sourceRows); rowIdx++ {
		row := sourceRows[rowIdx]
		for colIdx, cellValue := range row {
			cellName, err := excelize.CoordinatesToCellName(colIdx+1, targetRow)
			if err != nil {
				_ = out.Close()
				return nil, copiedRows, err
			}
			if err := out.SetCellStr(outSheet, cellName, cellValue); err != nil {
				_ = out.Close()
				return nil, copiedRows, err
			}
		}
		targetRow++
		copiedRows++
	}

	buffer, err := out.WriteToBuffer()
	_ = out.Close()
	if err != nil {
		return nil, copiedRows, err
	}

	return buffer.Bytes(), copiedRows, nil
}

func buildReceivePaymentsOutputName(prefix string, sourceName string, index int) string {
	base := strings.TrimSpace(prefix)
	if base == "" {
		base = "OUTPUT_RECEIVE_PAYMENTS"
	}
	base = strings.ReplaceAll(base, "/", "_")
	base = strings.ReplaceAll(base, "\\", "_")
	base = strings.ReplaceAll(base, ":", "_")
	base = strings.TrimSpace(base)
	if base == "" {
		base = "OUTPUT_RECEIVE_PAYMENTS"
	}

	stem := strings.TrimSuffix(filepath.Base(sourceName), filepath.Ext(sourceName))
	stem = strings.ReplaceAll(stem, "/", "_")
	stem = strings.ReplaceAll(stem, "\\", "_")
	stem = strings.ReplaceAll(stem, ":", "_")
	stem = strings.TrimSpace(stem)
	if stem == "" {
		stem = fmt.Sprintf("document_%d", index+1)
	}

	return fmt.Sprintf("%s_%s.xlsx", base, stem)
}

func firstOrEmpty(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (p *XLSXCashflowProcessor) runSpreadsheetToMYOB(
	ctx context.Context,
	req ActionRequest,
) (ActionResult, error) {
	templateKey, err := extractTemplateKey(req.Params)
	if err != nil {
		return ActionResult{
			ActionID:   req.ActionID,
			ActionType: req.ActionType,
			Status:     "failed",
			Message:    "invalid template key",
			StartedAt:  req.RequestedAt,
			FinishedAt: time.Now(),
		}, err
	}

	switch templateKey {
	case "", "receive_payments":
		return p.runCashflowToReceivePayments(ctx, req)
	case "pay_bills":
		return ActionResult{
			ActionID:   req.ActionID,
			ActionType: req.ActionType,
			Status:     "failed",
			Message:    "template pay_bills is not implemented yet",
			StartedAt:  req.RequestedAt,
			FinishedAt: time.Now(),
		}, fmt.Errorf("%w: template %s", ErrProcessorNotImplemented, templateKey)
	default:
		return ActionResult{
			ActionID:   req.ActionID,
			ActionType: req.ActionType,
			Status:     "failed",
			Message:    "unsupported template key",
			StartedAt:  req.RequestedAt,
			FinishedAt: time.Now(),
		}, fmt.Errorf("unsupported template key %q", templateKey)
	}
}

func extractTemplateKey(raw json.RawMessage) (string, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return "", nil
	}

	var payload struct {
		TemplateKey string `json:"templateKey"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.ToLower(payload.TemplateKey)), nil
}
