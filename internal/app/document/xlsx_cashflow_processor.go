package document

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
)

const cashflowNormalizedReadFallback = "invalid normalized cashflow payload"

type CashflowWorkbook struct {
	SourceFile    string          `json:"sourceFile"`
	PrimarySheet  string          `json:"primarySheet,omitempty"`
	SheetCount    int             `json:"sheetCount"`
	TotalRowCount int             `json:"totalRowCount"`
	Sheets        []CashflowSheet `json:"sheets"`
	ExtractedAt   time.Time       `json:"extractedAt"`
}

type CashflowSheet struct {
	Name           string     `json:"name"`
	HeaderRowIndex int        `json:"headerRowIndex"`
	Headers        []string   `json:"headers,omitempty"`
	RawRowNumbers  []int      `json:"rawRowNumbers,omitempty"`
	RawRows        [][]string `json:"rawRows,omitempty"`
	RowNumbers     []int      `json:"rowNumbers,omitempty"`
	Rows           [][]string `json:"rows,omitempty"`
	RowCount       int        `json:"rowCount"`
}

type cashflowNormalizedPayload struct {
	SourceID       string           `json:"sourceId"`
	SourceName     string           `json:"sourceName"`
	SourceSHA256   string           `json:"sourceSha256"`
	CollectionKind string           `json:"collectionKind"`
	SourceFormat   string           `json:"sourceFormat"`
	DocumentTag    string           `json:"documentTag,omitempty"`
	Warnings       []string         `json:"warnings,omitempty"`
	Workbook       CashflowWorkbook `json:"workbook"`
	ProcessedAt    time.Time        `json:"processedAt"`
}

type XLSXCashflowProcessor struct {
	fileStore   dfile.FileStore
	taxAccounts CashflowTaxAccountProvider
}

type CashflowTaxAccountProvider interface {
	Status(profileID string) appcashflow.TaxAccountStatus
	Load(profileID string) (map[string]appcashflow.TaxAccount, error)
}

func NewXLSXCashflowProcessor(
	fileStore dfile.FileStore,
	taxAccounts CashflowTaxAccountProvider,
) *XLSXCashflowProcessor {
	return &XLSXCashflowProcessor{
		fileStore:   fileStore,
		taxAccounts: taxAccounts,
	}
}

func (p *XLSXCashflowProcessor) Key() ProcessorKey {
	return ProcessorKey{
		CollectionKind: CollectionKindCashflowImport,
		SourceFormat:   SourceFormatXLSX,
	}
}

func (p *XLSXCashflowProcessor) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
	startedAt := time.Now()
	result := IngestResult{
		BatchID:      req.RequestID,
		CollectionID: req.CollectionID,
		DocumentType: string(req.CollectionKind),
		Items:        make([]IngestItemResult, 0, len(req.Sources)),
		StartedAt:    startedAt,
	}

	for _, source := range req.Sources {
		item := p.ingestSource(ctx, req, source)
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

func (p *XLSXCashflowProcessor) ingestSource(
	ctx context.Context,
	req IngestRequest,
	source IngestSource,
) IngestItemResult {
	workbook, warnings, err := extractCashflowWorkbook(source.TempPath, source.OriginalName)
	if err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse cashflow workbook",
			Errors:       []string{err.Error()},
		}
	}

	artifacts := make([]Artifact, 0, 2)
	if req.Policy.KeepRaw {
		rawBytes, readErr := os.ReadFile(source.TempPath)
		if readErr != nil {
			return IngestItemResult{
				SourceID:     source.SourceID,
				OriginalName: source.OriginalName,
				SHA256:       source.SHA256,
				Status:       IngestStatusFailed,
				Message:      "failed to read cashflow source file",
				Errors:       []string{readErr.Error()},
			}
		}

		rawName := source.SourceID + filepath.Ext(source.OriginalName)
		if writeErr := p.fileStore.WriteFile(ctx, req.CollectionID, rawName, rawBytes); writeErr != nil {
			return IngestItemResult{
				SourceID:     source.SourceID,
				OriginalName: source.OriginalName,
				SHA256:       source.SHA256,
				Status:       IngestStatusFailed,
				Message:      "failed to persist cashflow source file",
				Errors:       []string{writeErr.Error()},
			}
		}

		artifacts = append(artifacts, Artifact{
			Kind:     "raw",
			ObjectID: rawName,
			MimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			Size:     int64(len(rawBytes)),
		})
	}

	documentTag := deriveCashflowDocumentTag(workbook, source.OriginalName)
	payload := cashflowNormalizedPayload{
		SourceID:       source.SourceID,
		SourceName:     source.OriginalName,
		SourceSHA256:   source.SHA256,
		CollectionKind: string(req.CollectionKind),
		SourceFormat:   string(req.SourceFormat),
		DocumentTag:    documentTag,
		Warnings:       warnings,
		Workbook:       workbook,
		ProcessedAt:    time.Now().UTC(),
	}

	normalizedBytes, err := json.Marshal(payload)
	if err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to encode cashflow normalized payload",
			Errors:       []string{err.Error()},
		}
	}

	normalizedName := fmt.Sprintf("normalized_%s.json", source.SourceID)
	if err := p.fileStore.WriteFile(ctx, req.CollectionID, normalizedName, normalizedBytes); err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to persist cashflow normalized payload",
			Errors:       []string{err.Error()},
		}
	}

	artifacts = append(artifacts, Artifact{
		Kind:     "normalized",
		ObjectID: normalizedName,
		MimeType: "application/json",
		Size:     int64(len(normalizedBytes)),
	})

	status := IngestStatusReady
	if len(warnings) > 0 {
		status = IngestStatusWarning
	}

	return IngestItemResult{
		SourceID:     source.SourceID,
		OriginalName: source.OriginalName,
		SHA256:       source.SHA256,
		DocumentTag:  documentTag,
		Status:       status,
		Message:      "cashflow workbook parsed",
		Warnings:     warnings,
		Artifacts:    artifacts,
	}
}

func extractCashflowWorkbook(path string, sourceName string) (CashflowWorkbook, []string, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return CashflowWorkbook{}, nil, err
	}
	defer file.Close()

	sheetNames := file.GetSheetList()
	if len(sheetNames) == 0 {
		return CashflowWorkbook{}, nil, errors.New("workbook has no sheets")
	}

	workbook := CashflowWorkbook{
		SourceFile:  sourceName,
		SheetCount:  len(sheetNames),
		Sheets:      make([]CashflowSheet, 0, len(sheetNames)),
		ExtractedAt: time.Now().UTC(),
	}

	warnings := make([]string, 0, 2)
	for _, sheetName := range sheetNames {
		rows, err := file.GetRows(sheetName)
		if err != nil {
			return CashflowWorkbook{}, nil, fmt.Errorf("failed to read sheet %q: %w", sheetName, err)
		}

		sheet, sheetWarnings := buildCashflowSheet(sheetName, rows)
		if workbook.PrimarySheet == "" && len(sheet.RawRows) > 0 {
			workbook.PrimarySheet = sheet.Name
		}
		workbook.TotalRowCount += sheet.RowCount
		workbook.Sheets = append(workbook.Sheets, sheet)
		warnings = append(warnings, sheetWarnings...)
	}

	if workbook.PrimarySheet == "" {
		return CashflowWorkbook{}, nil, errors.New("workbook has no readable rows")
	}

	return workbook, uniqueStrings(warnings), nil
}

func buildCashflowSheet(sheetName string, rows [][]string) (CashflowSheet, []string) {
	sheet := CashflowSheet{
		Name: sheetName,
	}
	warnings := make([]string, 0, 2)

	headerFound := false
	for rowIndex, row := range rows {
		trimmedRow := trimTrailingEmptyCells(row)
		if len(trimmedRow) == 0 {
			continue
		}

		clonedRow := append([]string(nil), trimmedRow...)
		sheet.RawRowNumbers = append(sheet.RawRowNumbers, rowIndex+1)
		sheet.RawRows = append(sheet.RawRows, clonedRow)

		if !headerFound {
			headerFound = true
			sheet.HeaderRowIndex = rowIndex + 1
			sheet.Headers = append([]string(nil), clonedRow...)
			continue
		}

		sheet.RowNumbers = append(sheet.RowNumbers, rowIndex+1)
		sheet.Rows = append(sheet.Rows, clonedRow)
	}

	sheet.RowCount = len(sheet.Rows)
	if len(sheet.RawRows) == 0 {
		warnings = append(warnings, fmt.Sprintf("sheet %q is empty", sheetName))
		return sheet, warnings
	}
	if sheet.HeaderRowIndex == 0 || len(sheet.Headers) == 0 {
		warnings = append(warnings, fmt.Sprintf("sheet %q has no header row", sheetName))
	}
	if sheet.RowCount == 0 {
		warnings = append(warnings, fmt.Sprintf("sheet %q has no data rows", sheetName))
	}

	return sheet, warnings
}

func trimTrailingEmptyCells(row []string) []string {
	lastNonEmpty := -1
	for idx, cell := range row {
		if strings.TrimSpace(cell) != "" {
			lastNonEmpty = idx
		}
	}
	if lastNonEmpty < 0 {
		return nil
	}

	trimmed := make([]string, 0, lastNonEmpty+1)
	for _, cell := range row[:lastNonEmpty+1] {
		trimmed = append(trimmed, strings.TrimSpace(cell))
	}
	return trimmed
}

func deriveCashflowDocumentTag(workbook CashflowWorkbook, sourceName string) string {
	if strings.TrimSpace(workbook.PrimarySheet) != "" {
		return strings.TrimSpace(workbook.PrimarySheet)
	}
	return strings.TrimSpace(strings.TrimSuffix(sourceName, filepath.Ext(sourceName)))
}

func LoadCashflowWorkbook(
	ctx context.Context,
	fileStore dfile.FileStore,
	collectionID string,
	ref string,
) (*CashflowWorkbook, error) {
	if strings.TrimSpace(ref) == "" {
		return nil, errors.New(cashflowNormalizedReadFallback)
	}

	b, err := fileStore.Read(ctx, collectionID, ref)
	if err != nil {
		return nil, err
	}

	var payload cashflowNormalizedPayload
	if err := json.Unmarshal(b, &payload); err == nil && len(payload.Workbook.Sheets) > 0 {
		return &payload.Workbook, nil
	}

	var workbook CashflowWorkbook
	if err := json.Unmarshal(b, &workbook); err == nil && len(workbook.Sheets) > 0 {
		return &workbook, nil
	}

	return nil, errors.New(cashflowNormalizedReadFallback)
}
