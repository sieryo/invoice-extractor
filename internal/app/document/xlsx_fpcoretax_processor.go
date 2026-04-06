package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	appfpcoretax "github.com/sieryo/invoice-extractor/internal/app/fpcoretax"
	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
)

const (
	fpKeluaranMiscSalesActionType      = "export_fp_keluaran_misc_sales"
	fpKeluaranReturMiscSalesActionType = "export_fp_keluaran_retur_misc_sales"
	fpMasukanMiscPurchasesActionType   = "export_fp_masukan_misc_purchases"
	fpCoretaxDefaultOutputName         = "fp-coretax-myob"
)

type fpCoretaxNormalizedPayload struct {
	SourceID       string              `json:"sourceId"`
	SourceName     string              `json:"sourceName"`
	SourceSHA256   string              `json:"sourceSha256"`
	CollectionKind string              `json:"collectionKind"`
	SourceFormat   string              `json:"sourceFormat"`
	DocumentTag    string              `json:"documentTag,omitempty"`
	Warnings       []string            `json:"warnings,omitempty"`
	Workbook       SpreadsheetWorkbook `json:"workbook"`
	ProcessedAt    time.Time           `json:"processedAt"`
}

type fpCoretaxFieldDefinition struct {
	Key      string
	Required bool
	Aliases  []string
}

type fpCoretaxRowRecord struct {
	RowNumber            int
	PartyName            string
	DocumentNumber       string
	ReturnDocumentNumber string
	Date                 time.Time
	ReturnDate           time.Time
	TaxBase              float64
	Tax                  float64
	Reference            string
}

type FPCoretaxRelationProvider interface {
	Load(profileID string, key appfpcoretax.RelationRegistryKey) (map[string]appfpcoretax.RelationRecord, error)
}

type XLSXFPCoretaxProcessor struct {
	collectionKind CollectionKind
	fileStore      dfile.FileStore
	registry       FPCoretaxRelationProvider
}

var fpCoretaxTemplateTokenRegex = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)
var fpCoretaxInvalidFilenameCharRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

func NewXLSXFPCoretaxProcessor(
	collectionKind CollectionKind,
	fileStore dfile.FileStore,
	registry FPCoretaxRelationProvider,
) *XLSXFPCoretaxProcessor {
	return &XLSXFPCoretaxProcessor{
		collectionKind: collectionKind,
		fileStore:      fileStore,
		registry:       registry,
	}
}

func (p *XLSXFPCoretaxProcessor) Key() ProcessorKey {
	return ProcessorKey{
		CollectionKind: p.collectionKind,
		SourceFormat:   SourceFormatXLSX,
	}
}

func (p *XLSXFPCoretaxProcessor) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
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

func (p *XLSXFPCoretaxProcessor) ingestSource(
	ctx context.Context,
	req IngestRequest,
	source IngestSource,
) IngestItemResult {
	workbook, warnings, err := ExtractSpreadsheetWorkbook(source.TempPath, source.OriginalName)
	if err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse fp coretax workbook",
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
				Message:      "failed to read fp coretax source file",
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
				Message:      "failed to persist fp coretax source file",
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

	documentTag := deriveFPCoretaxDocumentTag(workbook, source.OriginalName)
	payload := fpCoretaxNormalizedPayload{
		SourceID:       source.SourceID,
		SourceName:     source.OriginalName,
		SourceSHA256:   source.SHA256,
		CollectionKind: string(req.CollectionKind),
		SourceFormat:   string(req.SourceFormat),
		DocumentTag:    documentTag,
		Warnings:       warnings,
		Workbook:       CompactSpreadsheetWorkbook(workbook),
		ProcessedAt:    time.Now().UTC(),
	}

	normalizedBytes, err := json.Marshal(payload)
	if err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to encode fp coretax normalized payload",
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
			Message:      "failed to persist fp coretax normalized payload",
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
		Message:      "fp coretax workbook parsed",
		Warnings:     warnings,
		Artifacts:    artifacts,
	}
}

func (p *XLSXFPCoretaxProcessor) RunAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	result := ActionResult{
		ActionID:    req.ActionID,
		ActionType:  req.ActionType,
		StartedAt:   req.RequestedAt,
		ItemResults: make([]ActionItemResult, 0, len(req.SnapshotDocs)),
		Outputs:     make([]ActionOutput, 0, 1),
	}

	if !p.supportsAction(req.ActionType) {
		result.Status = "failed"
		result.Message = "unsupported action for fp coretax"
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("%w: action %s for %s", ErrProcessorNotImplemented, req.ActionType, p.Key().CollectionKind)
	}
	if len(req.SnapshotDocs) == 0 {
		result.Status = "failed"
		result.Message = "snapshot is empty"
		result.FinishedAt = time.Now()
		return result, errors.New("snapshot is empty")
	}

	input, err := appfpcoretax.ParseExportMYOBInput(req.Input)
	if err != nil {
		result.Status = "failed"
		result.Message = "invalid fp coretax action input"
		result.FinishedAt = time.Now()
		return result, err
	}
	if p.collectionKind == CollectionKindFPKeluaranReturCoretax {
		input.IsReturn = true
	}

	if p.registry == nil {
		result.Status = "failed"
		result.Message = "registry provider tidak tersedia"
		result.FinishedAt = time.Now()
		return result, errors.New("fp coretax registry provider is nil")
	}

	registryKey := registryKeyFromCollectionKind(p.collectionKind)
	relations, err := p.registry.Load(req.UserID, registryKey)
	if err != nil {
		result.Status = "failed"
		result.Message = "master data registry belum siap"
		result.FinishedAt = time.Now()
		return result, err
	}

	rows := make([][]string, 0, len(req.SnapshotDocs)*2+1)
	if p.collectionKind == CollectionKindFPMasukanCoretax {
		rows = append(rows, appfpcoretax.MiscPurchasesHeader())
	} else {
		rows = append(rows, appfpcoretax.MiscSalesHeader())
	}

	for _, doc := range req.SnapshotDocs {
		workbook, loadErr := LoadSpreadsheetWorkbookForExecution(ctx, p.fileStore, req.CollectionID, doc.NormalizedRef, doc.RawRef)
		if loadErr != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: gagal membaca workbook", doc.SourceName),
				Error:      loadErr.Error(),
			})
			continue
		}

		sheet, findErr := FindSpreadsheetSheet(*workbook, input.SheetName)
		if findErr != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: %s", doc.SourceName, findErr.Error()),
				Error:      findErr.Error(),
			})
			continue
		}

		entryRows, warnings, processedCount, buildErr := p.buildDocumentRows(doc.SourceName, sheet, input, relations)
		if buildErr != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: %s", doc.SourceName, buildErr.Error()),
				Error:      buildErr.Error(),
				Warnings:   warnings,
			})
			continue
		}
		if processedCount == 0 {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: tidak ada row yang berhasil diexport", doc.SourceName),
				Error:      "no exportable fp coretax rows",
				Warnings:   warnings,
			})
			continue
		}

		rows = append(rows, entryRows...)
		itemStatus := "success"
		itemMessage := fmt.Sprintf("%d row diexport ke format MYOB", processedCount)
		if len(warnings) > 0 {
			itemStatus = "warning"
			itemMessage = fmt.Sprintf("%d row diexport dengan peringatan", processedCount)
		}
		result.ItemResults = append(result.ItemResults, ActionItemResult{
			DocumentID: doc.DocumentID,
			Status:     itemStatus,
			Message:    itemMessage,
			Warnings:   warnings,
		})
	}

	result.Total = len(req.SnapshotDocs)
	result.Success, result.Warning, result.Failed, result.Skipped = summarizeActionItems(result.ItemResults)
	if result.Success == 0 && result.Warning == 0 {
		result.Status = "failed"
		result.Message = "export gagal untuk semua dokumen"
		result.FinishedAt = time.Now()
		return result, errors.New("no fp coretax document was exported successfully")
	}

	body := appfpcoretax.EncodeTabDelimitedText(rows)
	outputName := fmt.Sprintf("%s_%s.txt", sanitizeFPCoretaxOutputFilename(input.OutputFilename), time.Now().Format("20060102_150405"))
	outputRef, err := p.fileStore.SaveArchive(ctx, req.CollectionID, outputName, body)
	if err != nil {
		result.Status = "failed"
		result.Message = "gagal menyimpan output text MYOB"
		result.FinishedAt = time.Now()
		return result, err
	}

	sum := sha256.Sum256(body)
	result.Outputs = append(result.Outputs, ActionOutput{
		Kind:      "file",
		Name:      outputName,
		ObjectRef: outputRef,
		MimeType:  "text/plain; charset=utf-8",
		SizeBytes: int64(len(body)),
		Checksum:  hex.EncodeToString(sum[:]),
	})

	switch {
	case result.Failed > 0:
		result.Status = "partial"
		result.Message = fmt.Sprintf("export selesai sebagian (%d sukses, %d warning, %d gagal)", result.Success, result.Warning, result.Failed)
	case result.Warning > 0:
		result.Status = "warning"
		result.Message = "export selesai dengan peringatan"
	default:
		result.Status = "success"
		if p.collectionKind == CollectionKindFPMasukanCoretax {
			result.Message = "export misc purchases berhasil"
		} else {
			result.Message = "export misc sales berhasil"
		}
	}
	result.FinishedAt = time.Now()
	return result, nil
}

func (p *XLSXFPCoretaxProcessor) buildDocumentRows(
	sourceName string,
	sheet SpreadsheetSheet,
	input appfpcoretax.ExportMYOBInput,
	relations map[string]appfpcoretax.RelationRecord,
) ([][]string, []string, int, error) {
	headers, headerIndex, err := resolveFPCoretaxHeaders(sheet, input)
	if err != nil {
		return nil, nil, 0, err
	}
	_ = headers

	rows := make([][]string, 0)
	warnings := make([]string, 0)
	processed := 0

	for idx, rawRow := range sheet.RawRows {
		rowNumber := SpreadsheetRowNumberAt(sheet, idx)
		if rowNumber <= input.HeaderRowNumber || isSpreadsheetRowEmpty(rawRow) {
			continue
		}

		cellRow := spreadsheetCashflowCellRowAt(sheet.RawCellRows, idx)
		record, rowWarnings, recordErr := parseFPCoretaxRow(rawRow, cellRow, rowNumber, headerIndex)
		if recordErr != nil {
			warnings = append(warnings, fmt.Sprintf("row %d: %s", rowNumber, recordErr.Error()))
			continue
		}
		warnings = append(warnings, rowWarnings...)

		entry, entryWarnings, entryErr := p.buildEntry(record, input, relations, sourceName)
		if entryErr != nil {
			warnings = append(warnings, fmt.Sprintf("row %d: %s", rowNumber, entryErr.Error()))
			continue
		}
		warnings = append(warnings, prefixWarnings(rowNumber, entryWarnings)...)

		if p.collectionKind == CollectionKindFPMasukanCoretax {
			rows = append(rows, appfpcoretax.BuildMiscPurchasesRow(entry), []string{})
		} else {
			rows = append(rows, appfpcoretax.BuildMiscSalesRow(entry), []string{})
		}
		processed++
	}

	if processed == 0 {
		return rows, uniqueStrings(warnings), 0, fmt.Errorf("%s: tidak ada row valid yang dapat dikonversi", sourceName)
	}
	return rows, uniqueStrings(warnings), processed, nil
}

func (p *XLSXFPCoretaxProcessor) buildEntry(
	record fpCoretaxRowRecord,
	input appfpcoretax.ExportMYOBInput,
	relations map[string]appfpcoretax.RelationRecord,
	sourceName string,
) (appfpcoretax.TransactionEntry, []string, error) {
	warnings := make([]string, 0)
	accountNumber := strings.TrimSpace(input.AccountNumber)

	if relation, ok := relations[normalizeFPCoretaxLookupKey(record.PartyName)]; ok {
		if p.collectionKind == CollectionKindFPMasukanCoretax && strings.TrimSpace(relation.Account) != "" {
			accountNumber = strings.TrimSpace(relation.Account)
		}
	} else {
		label := "customer"
		if p.collectionKind == CollectionKindFPMasukanCoretax {
			label = "supplier"
		}
		warnings = append(warnings, fmt.Sprintf("%s %q tidak ditemukan di registry", label, record.PartyName))
	}

	if accountNumber == "" {
		return appfpcoretax.TransactionEntry{}, warnings, errors.New("account number kosong")
	}

	tokenValues := buildFPCoretaxTemplateValues(p.collectionKind, record, sourceName)
	memo, memoWarnings := renderFPCoretaxTemplate(input.MemoTemplate, tokenValues)
	description, descriptionWarnings := renderFPCoretaxTemplate(input.DescriptionTemplate, tokenValues)
	warnings = append(warnings, memoWarnings...)
	warnings = append(warnings, descriptionWarnings...)

	return appfpcoretax.TransactionEntry{
		PartyName:     record.PartyName,
		AccountNumber: accountNumber,
		Date:          resolveFPCoretaxEntryDate(record),
		Memo:          memo,
		Description:   description,
		Amount:        resolveFPCoretaxAmount(record.TaxBase, input),
		GSTAmount:     resolveFPCoretaxAmount(record.Tax, input),
		IncTaxAmount:  resolveFPCoretaxAmount(record.TaxBase+record.Tax, input),
		TaxCode:       input.TaxCode,
		Inclusive:     input.Inclusive,
	}, uniqueStrings(warnings), nil
}

func (p *XLSXFPCoretaxProcessor) supportsAction(actionType string) bool {
	switch strings.TrimSpace(actionType) {
	case fpKeluaranMiscSalesActionType:
		return p.collectionKind == CollectionKindFPKeluaranCoretax
	case fpKeluaranReturMiscSalesActionType:
		return p.collectionKind == CollectionKindFPKeluaranReturCoretax
	case fpMasukanMiscPurchasesActionType:
		return p.collectionKind == CollectionKindFPMasukanCoretax
	default:
		return false
	}
}

func registryKeyFromCollectionKind(collectionKind CollectionKind) appfpcoretax.RelationRegistryKey {
	if collectionKind == CollectionKindFPMasukanCoretax {
		return appfpcoretax.RelationRegistrySupplier
	}
	return appfpcoretax.RelationRegistryCustomer
}

func deriveFPCoretaxDocumentTag(workbook SpreadsheetWorkbook, sourceName string) string {
	if strings.TrimSpace(workbook.PrimarySheet) != "" {
		return strings.TrimSpace(workbook.PrimarySheet)
	}
	return strings.TrimSpace(strings.TrimSuffix(sourceName, filepath.Ext(sourceName)))
}

func resolveFPCoretaxHeaders(sheet SpreadsheetSheet, input appfpcoretax.ExportMYOBInput) ([]string, map[string]int, error) {
	headerIdx := -1
	for idx := range sheet.RawRows {
		if SpreadsheetRowNumberAt(sheet, idx) == input.HeaderRowNumber {
			headerIdx = idx
			break
		}
	}
	if headerIdx < 0 {
		return nil, nil, fmt.Errorf("baris header %d tidak ditemukan pada sheet %q", input.HeaderRowNumber, sheet.Name)
	}

	headers := sheet.RawRows[headerIdx]
	byNormalized := make(map[string]int, len(headers))
	for idx, header := range headers {
		key := normalizeFPCoretaxHeader(header)
		if key == "" {
			continue
		}
		if _, exists := byNormalized[key]; exists {
			continue
		}
		byNormalized[key] = idx
	}

	fieldIndex := make(map[string]int)
	missing := make([]string, 0)
	for _, def := range fpCoretaxFieldDefinitions(input) {
		idx, ok := resolveFPCoretaxFieldIndex(def, byNormalized)
		if !ok {
			if def.Required {
				missing = append(missing, def.Key)
			}
			continue
		}
		fieldIndex[def.Key] = idx
	}
	if len(missing) > 0 {
		return nil, nil, fmt.Errorf("kolom wajib fp coretax tidak lengkap: %s", strings.Join(missing, ", "))
	}
	return headers, fieldIndex, nil
}

func parseFPCoretaxRow(row []string, cellRow []SpreadsheetCell, rowNumber int, fieldIndex map[string]int) (fpCoretaxRowRecord, []string, error) {
	record := fpCoretaxRowRecord{RowNumber: rowNumber}
	record.PartyName = strings.TrimSpace(cashflowCellValue(row, cellRow, fieldIndex, "partyName"))
	record.DocumentNumber = strings.TrimSpace(cashflowCellValue(row, cellRow, fieldIndex, "documentNumber"))
	record.ReturnDocumentNumber = strings.TrimSpace(cashflowCellValue(row, cellRow, fieldIndex, "returnDocumentNumber"))
	record.Reference = strings.TrimSpace(cashflowCellValue(row, cellRow, fieldIndex, "reference"))

	dateCell := cashflowTypedCell(row, cellRow, fieldIndex, "date")
	date, ok := SpreadsheetCellDate(dateCell)
	if !ok {
		return record, nil, fmt.Errorf("tanggal faktur pajak tidak valid (%s)", describeSpreadsheetCellValue(dateCell))
	}
	record.Date = date

	if returnDateCell, ok := optionalCashflowTypedCell(row, cellRow, fieldIndex, "returnDate"); ok {
		returnDate, dateOK := SpreadsheetCellDate(returnDateCell)
		if !dateOK {
			return record, nil, fmt.Errorf("tanggal retur tidak valid (%s)", describeSpreadsheetCellValue(returnDateCell))
		}
		record.ReturnDate = returnDate
	}

	taxBaseCell := cashflowTypedCell(row, cellRow, fieldIndex, "taxBase")
	taxBase, ok := SpreadsheetCellMoney(taxBaseCell)
	if !ok {
		return record, nil, fmt.Errorf("DPP tidak valid (%s)", describeSpreadsheetCellValue(taxBaseCell))
	}
	record.TaxBase = taxBase

	taxCell := cashflowTypedCell(row, cellRow, fieldIndex, "tax")
	tax, ok := SpreadsheetCellMoney(taxCell)
	if !ok {
		return record, nil, fmt.Errorf("PPN tidak valid (%s)", describeSpreadsheetCellValue(taxCell))
	}
	record.Tax = tax

	if record.PartyName == "" {
		return record, nil, errors.New("nama pihak kosong")
	}
	if record.DocumentNumber == "" {
		return record, nil, errors.New("nomor faktur pajak kosong")
	}
	return record, nil, nil
}

func fpCoretaxFieldDefinitions(input appfpcoretax.ExportMYOBInput) []fpCoretaxFieldDefinition {
	defs := []fpCoretaxFieldDefinition{
		{Key: "partyName", Required: true, Aliases: fpCoretaxAliases(input.MappedField("partyName"), "nama", "nama pembeli", "nama penjual")},
		{Key: "documentNumber", Required: true, Aliases: fpCoretaxAliases(input.MappedField("documentNumber"), "nomor faktur pajak")},
		{Key: "date", Required: true, Aliases: fpCoretaxAliases(input.MappedField("date"), "tanggal faktur pajak", "tanggal")},
		{Key: "taxBase", Required: true, Aliases: fpCoretaxAliases(input.MappedField("taxBase"), "harga jual/penggantian/dpp", "dpp")},
		{Key: "tax", Required: true, Aliases: fpCoretaxAliases(input.MappedField("tax"), "ppn")},
		{Key: "reference", Required: false, Aliases: fpCoretaxAliases(input.MappedField("reference"), "referensi", "reference")},
	}

	if input.IsReturn {
		defs = append(defs,
			fpCoretaxFieldDefinition{Key: "returnDocumentNumber", Required: true, Aliases: fpCoretaxAliases(input.MappedField("returnDocumentNumber"), "nomor retur", "return number")},
			fpCoretaxFieldDefinition{Key: "returnDate", Required: true, Aliases: fpCoretaxAliases(input.MappedField("returnDate"), "tanggal retur", "return date")},
		)
	}

	return defs
}

func fpCoretaxAliases(primary string, aliases ...string) []string {
	seen := make(map[string]struct{}, len(aliases)+1)
	out := make([]string, 0, len(aliases)+1)

	appendAlias := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		key := normalizeFPCoretaxHeader(trimmed)
		if key == "" {
			return
		}
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}

	appendAlias(primary)
	for _, alias := range aliases {
		appendAlias(alias)
	}
	return out
}

func resolveFPCoretaxFieldIndex(def fpCoretaxFieldDefinition, byNormalized map[string]int) (int, bool) {
	for _, alias := range def.Aliases {
		idx, ok := byNormalized[normalizeFPCoretaxHeader(alias)]
		if ok {
			return idx, true
		}
	}
	return 0, false
}

func normalizeFPCoretaxHeader(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "", "_", "", "-", "", "/", "", ".", "", "(", "", ")", "", "%", "")
	return replacer.Replace(value)
}

func normalizeFPCoretaxLookupKey(raw string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(raw))), " ")
}

func buildFPCoretaxTemplateValues(collectionKind CollectionKind, record fpCoretaxRowRecord, sourceName string) map[string]string {
	partyKey := "namapembeli"
	if collectionKind == CollectionKindFPMasukanCoretax {
		partyKey = "namapenjual"
	}
	return map[string]string{
		partyKey:             strings.TrimSpace(record.PartyName),
		"namapihak":          strings.TrimSpace(record.PartyName),
		"nomorfakturpajak":   strings.TrimSpace(record.DocumentNumber),
		"tanggalfakturpajak": record.Date.Format("2006-01-02"),
		"nomorretur":         strings.TrimSpace(record.ReturnDocumentNumber),
		"tanggalretur":       formatOptionalFPCoretaxDate(record.ReturnDate),
		"referensi":          strings.TrimSpace(record.Reference),
		"dpp":                appfpcoretax.FormatMYOBTemplateNumber(record.TaxBase),
		"ppn":                appfpcoretax.FormatMYOBTemplateNumber(record.Tax),
		"total":              appfpcoretax.FormatMYOBTemplateNumber(record.TaxBase + record.Tax),
		"sourcename":         strings.TrimSpace(strings.TrimSuffix(sourceName, filepath.Ext(sourceName))),
	}
}

func renderFPCoretaxTemplate(template string, values map[string]string) (string, []string) {
	template = strings.TrimSpace(template)
	if template == "" {
		return "", nil
	}

	missingTokens := make([]string, 0, 2)
	rendered := fpCoretaxTemplateTokenRegex.ReplaceAllStringFunc(template, func(match string) string {
		sub := fpCoretaxTemplateTokenRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return ""
		}
		token := strings.ToLower(strings.TrimSpace(sub[1]))
		value, ok := values[token]
		if !ok {
			missingTokens = append(missingTokens, token)
			return ""
		}
		return strings.TrimSpace(value)
	})
	rendered = strings.Join(strings.Fields(rendered), " ")

	warnings := make([]string, 0, 1)
	if len(missingTokens) > 0 {
		warnings = append(warnings, fmt.Sprintf("unknown template placeholder(s): %s", strings.Join(uniqueStrings(missingTokens), ", ")))
	}
	return rendered, warnings
}

func optionalCashflowTypedCell(row []string, cellRow []SpreadsheetCell, fieldIndex map[string]int, key string) (SpreadsheetCell, bool) {
	idx, ok := fieldIndex[key]
	if !ok || idx < 0 {
		return SpreadsheetCell{}, false
	}
	return cashflowTypedCell(row, cellRow, fieldIndex, key), true
}

func resolveFPCoretaxEntryDate(record fpCoretaxRowRecord) time.Time {
	return record.Date
}

func resolveFPCoretaxAmount(value float64, input appfpcoretax.ExportMYOBInput) float64 {
	if input.IsReturn {
		return -value
	}
	return value
}

func formatOptionalFPCoretaxDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func sanitizeFPCoretaxOutputFilename(raw string) string {
	value := strings.TrimSpace(raw)
	value = fpCoretaxInvalidFilenameCharRegex.ReplaceAllString(value, "_")
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Trim(value, ".-_ ")
	if value == "" {
		return fpCoretaxDefaultOutputName
	}
	return strings.TrimSuffix(value, filepath.Ext(value))
}
