package document

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	appbukpot "github.com/sieryo/invoice-extractor/internal/app/bukpot"
	appprofile "github.com/sieryo/invoice-extractor/internal/app/profile"
	"github.com/xuri/excelize/v2"

	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
)

const (
	bukpotRequestActionType       = "request_bukpot_gst_deduction_mt"
	bukpotRequestTemplateSheet    = "DATA"
	bukpotRequestTemplateTable    = "Table1"
	bukpotRequestProfileCell      = "C1"
	bukpotRequestDefaultZipPrefix = "BPPU"
)

var bukpotRequestInvalidFilenameCharRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

type bukpotRequestInput struct {
	SheetName        string
	HeaderRowNumber  int
	EntityHeader     string
	SettlementDate   string
	NPWPHeader       string
	NITKUHeader      string
	FacilityHeader   string
	TaxObjectCode    string
	TaxBaseHeader    string
	WithholdingRate  string
	TaxInvoiceNumber string
	ReferenceNumber  string
	ReferenceDate    string
}

type bukpotRequestNormalizedPayload struct {
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

type bukpotRequestRecord struct {
	TaxPeriodMonth                    string
	TaxPeriodYear                     string
	NPWP                              string
	IncomeRecipientBusinessActivityID string
	Facility                          string
	TaxObjectCode                     string
	TaxBase                           float64
	RatePercent                       float64
	RateFraction                      float64
	DocumentType                      string
	DocumentNumber                    string
	DocumentDate                      time.Time
	WithholderBusinessActivityID      string
	GovTreasurerOption                string
	SP2DNumber                        string
	WithholdingDate                   time.Time
	WithholdingAmount                 float64
}

type XLSXBukpotRequestProcessor struct {
	fileStore      dfile.FileStore
	rootDir        string
	requestConfigs *appbukpot.RequestConfigService
}

func NewXLSXBukpotRequestProcessor(
	fileStore dfile.FileStore,
	rootDir string,
	requestConfigs *appbukpot.RequestConfigService,
) *XLSXBukpotRequestProcessor {
	return &XLSXBukpotRequestProcessor{
		fileStore:      fileStore,
		rootDir:        rootDir,
		requestConfigs: requestConfigs,
	}
}

func (p *XLSXBukpotRequestProcessor) Key() ProcessorKey {
	return ProcessorKey{
		CollectionKind: CollectionKindBukpotRequestGSTDeductionMT,
		SourceFormat:   SourceFormatXLSX,
	}
}

func (p *XLSXBukpotRequestProcessor) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
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

func (p *XLSXBukpotRequestProcessor) ingestSource(ctx context.Context, req IngestRequest, source IngestSource) IngestItemResult {
	workbook, warnings, err := ExtractSpreadsheetWorkbook(source.TempPath, source.OriginalName)
	if err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse bukpot request workbook",
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
				Message:      "failed to read bukpot request source file",
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
				Message:      "failed to persist bukpot request source file",
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

	documentTag := strings.TrimSpace(workbook.PrimarySheet)
	payload := bukpotRequestNormalizedPayload{
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
			Message:      "failed to encode normalized bukpot request payload",
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
			Message:      "failed to persist normalized bukpot request payload",
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
		Message:      "bukpot request workbook parsed",
		Warnings:     warnings,
		Artifacts:    artifacts,
	}
}

func (p *XLSXBukpotRequestProcessor) ValidateAction(ctx context.Context, req ActionRequest) error {
	input, err := parseBukpotRequestInput(req.Input)
	if err != nil {
		return err
	}
	for _, doc := range req.SnapshotDocs {
		workbook, err := LoadSpreadsheetWorkbook(ctx, p.fileStore, req.CollectionID, doc.NormalizedRef)
		if err != nil {
			return fmt.Errorf("%s: gagal membaca workbook: %w", doc.SourceName, err)
		}
		sheet, err := FindSpreadsheetSheet(*workbook, input.SheetName)
		if err != nil {
			return fmt.Errorf("%s: %w", doc.SourceName, err)
		}
		if _, _, err := resolveBukpotRequestHeaderIndex(sheet, input); err != nil {
			return fmt.Errorf("%s: %w", doc.SourceName, err)
		}
	}
	return nil
}

func (p *XLSXBukpotRequestProcessor) RunAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	result := ActionResult{
		ActionID:    req.ActionID,
		ActionType:  req.ActionType,
		StartedAt:   req.RequestedAt,
		ItemResults: make([]ActionItemResult, 0, len(req.SnapshotDocs)),
		Outputs:     make([]ActionOutput, 0, 1),
	}

	if strings.TrimSpace(req.ActionType) != bukpotRequestActionType {
		result.Status = "failed"
		result.Message = "unsupported action for bukpot_request_gst_deduction_mt"
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("%w: action %s for %s", ErrProcessorNotImplemented, req.ActionType, p.Key().CollectionKind)
	}

	input, err := parseBukpotRequestInput(req.Input)
	if err != nil {
		result.Status = "failed"
		result.Message = "invalid bukpot request input"
		result.FinishedAt = time.Now()
		return result, err
	}

	profileMeta, err := appprofile.LoadMetadataFromFile(p.rootDir, req.UserID)
	if err != nil {
		result.Status = "failed"
		result.Message = "profil aktif tidak dapat dibaca"
		result.FinishedAt = time.Now()
		return result, err
	}

	records := make([]bukpotRequestRecord, 0)
	for _, doc := range req.SnapshotDocs {
		workbook, loadErr := LoadSpreadsheetWorkbook(ctx, p.fileStore, req.CollectionID, doc.NormalizedRef)
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

		headerRow, headerIndex, resolveErr := resolveBukpotRequestHeaderIndex(sheet, input)
		if resolveErr != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: %s", doc.SourceName, resolveErr.Error()),
				Error:      resolveErr.Error(),
			})
			continue
		}

		docRecords, err := p.buildBukpotRequestRecords(doc.SourceName, sheet, headerRow, headerIndex, input, profileMeta)
		if err != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: %s", doc.SourceName, err.Error()),
				Error:      err.Error(),
			})
			continue
		}

		records = append(records, docRecords...)
		result.ItemResults = append(result.ItemResults, ActionItemResult{
			DocumentID: doc.DocumentID,
			Status:     "success",
			Message:    fmt.Sprintf("%d row diproses", len(docRecords)),
		})
	}

	result.Total = len(req.SnapshotDocs)
	result.Success, result.Warning, result.Failed, result.Skipped = summarizeActionItems(result.ItemResults)
	if len(records) == 0 {
		result.Status = "failed"
		result.Message = "tidak ada row valid untuk request bukpot"
		result.FinishedAt = time.Now()
		return result, errors.New("no bukpot request record was generated")
	}

	xlsxBytes, err := buildBukpotRequestWorkbook(records, profileMeta)
	if err != nil {
		result.Status = "failed"
		result.Message = "gagal membangun file excel request bukpot"
		result.FinishedAt = time.Now()
		return result, err
	}

	xmlBytes, err := buildBukpotRequestXML(records, profileMeta)
	if err != nil {
		result.Status = "failed"
		result.Message = "gagal membangun file xml request bukpot"
		result.FinishedAt = time.Now()
		return result, err
	}

	archiveName := sanitizeBukpotRequestFilename(fmt.Sprintf("%s - %s Deduction MT", bukpotRequestDefaultZipPrefix, profileMeta.Alias))
	if archiveName == "" {
		archiveName = bukpotRequestDefaultZipPrefix
	}

	zipBytes, err := buildBukpotRequestZIP(archiveName, xlsxBytes, xmlBytes)
	if err != nil {
		result.Status = "failed"
		result.Message = "gagal membangun arsip request bukpot"
		result.FinishedAt = time.Now()
		return result, err
	}

	outputName := fmt.Sprintf("%s_%s.zip", archiveName, time.Now().Format("20060102_150405"))
	outputRef, err := p.fileStore.SaveArchive(ctx, req.CollectionID, outputName, zipBytes)
	if err != nil {
		result.Status = "failed"
		result.Message = "gagal menyimpan arsip request bukpot"
		result.FinishedAt = time.Now()
		return result, err
	}

	sum := sha256.Sum256(zipBytes)
	result.Outputs = append(result.Outputs, ActionOutput{
		Kind:      "file",
		Name:      outputName,
		ObjectRef: outputRef,
		MimeType:  "application/zip",
		SizeBytes: int64(len(zipBytes)),
		Checksum:  hex.EncodeToString(sum[:]),
	})

	if result.Failed > 0 {
		result.Status = "partial"
		result.Message = fmt.Sprintf("request bukpot selesai sebagian (%d sukses, %d gagal)", result.Success, result.Failed)
	} else {
		result.Status = "success"
		result.Message = "request bukpot berhasil dibuat"
	}
	result.FinishedAt = time.Now()
	return result, nil
}

func (p *XLSXBukpotRequestProcessor) buildBukpotRequestRecords(
	sourceName string,
	sheet SpreadsheetSheet,
	headerRow int,
	headerIndex map[string]int,
	input bukpotRequestInput,
	profileMeta appprofile.Metadata,
) ([]bukpotRequestRecord, error) {
	records := make([]bukpotRequestRecord, 0)
	for idx, rawRow := range sheet.RawRows {
		rowNumber := SpreadsheetRowNumberAt(sheet, idx)
		if rowNumber <= headerRow {
			continue
		}
		if isSpreadsheetRowEmpty(rawRow) {
			continue
		}

		cellRow := spreadsheetCellRowAt(sheet.RawCellRows, idx)
		entity := strings.TrimSpace(spreadsheetTextValue(rawRow, cellRow, headerIndex, "entity"))
		if entity == "" || !strings.EqualFold(entity, profileMeta.Alias) {
			continue
		}

		record, err := buildBukpotRequestRecord(rawRow, cellRow, headerIndex, profileMeta)
		if err != nil {
			return nil, fmt.Errorf("%s row %d: %w", sourceName, rowNumber, err)
		}
		records = append(records, record)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("%s: tidak ada row yang cocok untuk alias %s", sourceName, profileMeta.Alias)
	}
	return records, nil
}

func buildBukpotRequestRecord(row []string, cellRow []SpreadsheetCell, headerIndex map[string]int, profileMeta appprofile.Metadata) (bukpotRequestRecord, error) {
	settlementCell := spreadsheetTypedCell(row, cellRow, headerIndex, "settlementDate")
	settlementDate, ok := SpreadsheetCellDate(settlementCell)
	if !ok {
		return bukpotRequestRecord{}, fmt.Errorf("Settlement Date tidak valid (%s)", describeSpreadsheetCellValue(settlementCell))
	}

	npwp := strings.TrimSpace(spreadsheetTextValue(row, cellRow, headerIndex, "npwp"))
	if npwp == "" {
		return bukpotRequestRecord{}, errors.New("NPWP kosong")
	}

	nitku := strings.TrimSpace(spreadsheetTextValue(row, cellRow, headerIndex, "nitku"))
	if nitku == "" {
		return bukpotRequestRecord{}, errors.New("NITKU kosong")
	}

	taxObjectCode := strings.TrimSpace(spreadsheetTextValue(row, cellRow, headerIndex, "taxObjectCode"))
	if taxObjectCode == "" {
		return bukpotRequestRecord{}, errors.New("Kode Objek Pajak kosong")
	}

	taxBaseCell := spreadsheetTypedCell(row, cellRow, headerIndex, "taxBase")
	taxBase, ok := SpreadsheetCellFloat(taxBaseCell)
	if !ok {
		return bukpotRequestRecord{}, fmt.Errorf("DPP tidak valid (%s)", describeSpreadsheetCellValue(taxBaseCell))
	}

	rateCell := spreadsheetTypedCell(row, cellRow, headerIndex, "withholdingRate")
	rateFraction, ok := SpreadsheetCellPercent(rateCell)
	if !ok {
		return bukpotRequestRecord{}, fmt.Errorf("WHT tidak valid (%s)", describeSpreadsheetCellValue(rateCell))
	}

	facilitySource := strings.TrimSpace(spreadsheetTextValue(row, cellRow, headerIndex, "facility"))
	facilityValue, finalRate, err := mapBukpotRequestFacility(facilitySource, rateFraction)
	if err != nil {
		return bukpotRequestRecord{}, err
	}

	fakturPajakNo := strings.TrimSpace(spreadsheetTextValue(row, cellRow, headerIndex, "taxInvoiceNumber"))
	invoiceNumber := strings.TrimSpace(spreadsheetTextValue(row, cellRow, headerIndex, "referenceNumber"))
	documentType, documentNumber := resolveBukpotRequestReference(fakturPajakNo, invoiceNumber)
	if documentNumber == "" {
		return bukpotRequestRecord{}, errors.New("Nomor Dok. Referensi kosong")
	}

	documentDateCell := spreadsheetTypedCell(row, cellRow, headerIndex, "referenceDate")
	documentDate, ok := SpreadsheetCellDate(documentDateCell)
	if !ok {
		return bukpotRequestRecord{}, fmt.Errorf("Tanggal Dok. Referensi tidak valid (%s)", describeSpreadsheetCellValue(documentDateCell))
	}

	month := settlementDate.Month()
	if settlementDate.Day() <= profileMeta.CutoffDate {
		if month == time.January {
			month = time.December
		} else {
			month = month - 1
		}
	}

	withholdingAmount := taxBase * finalRate
	return bukpotRequestRecord{
		TaxPeriodMonth:                    strconv.Itoa(int(month)),
		TaxPeriodYear:                     strconv.Itoa(settlementDate.Year()),
		NPWP:                              npwp,
		IncomeRecipientBusinessActivityID: nitku,
		Facility:                          facilityValue,
		TaxObjectCode:                     taxObjectCode,
		TaxBase:                           taxBase,
		RateFraction:                      finalRate,
		RatePercent:                       finalRate * 100,
		DocumentType:                      documentType,
		DocumentNumber:                    formatBukpotDocumentReference(documentNumber),
		DocumentDate:                      documentDate,
		WithholderBusinessActivityID:      profileMeta.TKUID,
		GovTreasurerOption:                "N/A",
		SP2DNumber:                        "",
		WithholdingDate:                   settlementDate,
		WithholdingAmount:                 withholdingAmount,
	}, nil
}

func parseBukpotRequestInput(raw json.RawMessage) (bukpotRequestInput, error) {
	payload := map[string]any{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return bukpotRequestInput{}, err
	}
	readString := func(key string) string {
		value, _ := payload[key]
		if value == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
	readInt := func(key string) int {
		value, ok := payload[key]
		if !ok || value == nil {
			return 0
		}
		switch v := value.(type) {
		case float64:
			return int(v)
		case int:
			return v
		default:
			n, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprint(v)))
			return n
		}
	}

	input := bukpotRequestInput{
		SheetName:        readString("sheetName"),
		HeaderRowNumber:  readInt("headerRowNumber"),
		EntityHeader:     readString("entity"),
		SettlementDate:   readString("settlementDate"),
		NPWPHeader:       readString("npwp"),
		NITKUHeader:      readString("nitku"),
		FacilityHeader:   readString("facility"),
		TaxObjectCode:    readString("taxObjectCode"),
		TaxBaseHeader:    readString("taxBase"),
		WithholdingRate:  readString("withholdingRate"),
		TaxInvoiceNumber: readString("taxInvoiceNumber"),
		ReferenceNumber:  readString("referenceNumber"),
		ReferenceDate:    readString("referenceDate"),
	}

	if strings.TrimSpace(input.SheetName) == "" {
		return bukpotRequestInput{}, errors.New("sheetName wajib diisi")
	}
	if input.HeaderRowNumber <= 0 {
		return bukpotRequestInput{}, errors.New("headerRowNumber wajib lebih dari 0")
	}
	return input, nil
}

func resolveBukpotRequestHeaderIndex(sheet SpreadsheetSheet, input bukpotRequestInput) (int, map[string]int, error) {
	headerIdx := -1
	for idx := range sheet.RawRows {
		if SpreadsheetRowNumberAt(sheet, idx) == input.HeaderRowNumber {
			headerIdx = idx
			break
		}
	}
	if headerIdx < 0 {
		return 0, nil, fmt.Errorf("baris header %d tidak ditemukan pada sheet %q", input.HeaderRowNumber, sheet.Name)
	}

	headers := sheet.RawRows[headerIdx]
	byNormalized := make(map[string]int, len(headers))
	for idx, header := range headers {
		key := normalizeBukpotRequestHeader(header)
		if key == "" {
			continue
		}
		if _, exists := byNormalized[key]; !exists {
			byNormalized[key] = idx
		}
	}

	mapping := map[string]string{
		"entity":           input.EntityHeader,
		"settlementDate":   input.SettlementDate,
		"npwp":             input.NPWPHeader,
		"nitku":            input.NITKUHeader,
		"facility":         input.FacilityHeader,
		"taxObjectCode":    input.TaxObjectCode,
		"taxBase":          input.TaxBaseHeader,
		"withholdingRate":  input.WithholdingRate,
		"taxInvoiceNumber": input.TaxInvoiceNumber,
		"referenceNumber":  input.ReferenceNumber,
		"referenceDate":    input.ReferenceDate,
	}

	requiredKeys := map[string]bool{
		"entity":          true,
		"settlementDate":  true,
		"npwp":            true,
		"nitku":           true,
		"taxObjectCode":   true,
		"taxBase":         true,
		"withholdingRate": true,
		"referenceNumber": true,
		"referenceDate":   true,
	}

	indexes := make(map[string]int, len(mapping))
	missing := make([]string, 0)
	for key, headerName := range mapping {
		headerName = strings.TrimSpace(headerName)
		if headerName == "" {
			if requiredKeys[key] {
				missing = append(missing, key)
			}
			continue
		}
		idx, ok := byNormalized[normalizeBukpotRequestHeader(headerName)]
		if !ok {
			if requiredKeys[key] {
				missing = append(missing, headerName)
			}
			continue
		}
		indexes[key] = idx
	}

	if len(missing) > 0 {
		return 0, nil, fmt.Errorf("kolom wajib tidak ditemukan: %s", strings.Join(missing, ", "))
	}
	return input.HeaderRowNumber, indexes, nil
}

func buildBukpotRequestWorkbook(records []bukpotRequestRecord, profileMeta appprofile.Metadata) ([]byte, error) {
	f, err := excelize.OpenReader(bytes.NewReader(bukpotRequestTemplate))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if err := f.SetCellValue(bukpotRequestTemplateSheet, bukpotRequestProfileCell, profileMeta.NPWP); err != nil {
		return nil, err
	}
	if err := f.SetCellStyle(bukpotRequestTemplateSheet, bukpotRequestProfileCell, bukpotRequestProfileCell, 0); err != nil {
		// ignore style reset if template doesn't need it
	}

	tables, err := f.GetTables(bukpotRequestTemplateSheet)
	if err != nil {
		return nil, err
	}

	var table *excelize.Table
	for idx := range tables {
		if tables[idx].Name == bukpotRequestTemplateTable {
			table = &tables[idx]
			break
		}
	}
	if table == nil {
		return nil, fmt.Errorf("tabel %s tidak ditemukan di sheet %s", bukpotRequestTemplateTable, bukpotRequestTemplateSheet)
	}

	startCol, startRow, err := excelize.CellNameToCoordinates(strings.Split(table.Range, ":")[0])
	if err != nil {
		return nil, err
	}
	endCol, _, err := excelize.CellNameToCoordinates(strings.Split(table.Range, ":")[1])
	if err != nil {
		return nil, err
	}
	headerRow := startRow
	dataRowStart := headerRow + 1

	headers := []string{
		"Masa Pajak",
		"Tahun Pajak",
		"NPWP",
		"ID TKU Penerima Penghasilan",
		"Fasilitas",
		"Kode Objek Pajak",
		"DPP",
		"Tarif",
		"Jenis Dok. Referensi",
		"Nomor Dok. Referensi",
		"Tanggal Dok. Referensi",
		"ID TKU Pemotong",
		"Opsi Pembayaran (IP)",
		"Nomor SP2D (IP)",
		"Tanggal Pemotongan",
	}

	for idx, record := range records {
		row := dataRowStart + idx
		values := []any{
			record.TaxPeriodMonth,
			record.TaxPeriodYear,
			record.NPWP,
			record.IncomeRecipientBusinessActivityID,
			record.Facility,
			record.TaxObjectCode,
			record.TaxBase,
			record.RatePercent,
			record.DocumentType,
			record.DocumentNumber,
			formatBukpotWorkbookDate(record.DocumentDate),
			record.WithholderBusinessActivityID,
			record.GovTreasurerOption,
			record.SP2DNumber,
			formatBukpotWorkbookDate(record.WithholdingDate),
		}
		cell, _ := excelize.CoordinatesToCellName(startCol, row)
		if err := f.SetSheetRow(bukpotRequestTemplateSheet, cell, &values); err != nil {
			return nil, err
		}

		// Python versi lama mengisi kolom 18 dengan WHT dan kolom 19 dengan kode akun.
		if err := f.SetCellValue(bukpotRequestTemplateSheet, fmt.Sprintf("R%d", row), record.WithholdingAmount); err != nil {
			return nil, err
		}
		if err := f.SetCellValue(bukpotRequestTemplateSheet, fmt.Sprintf("S%d", row), "411124-100"); err != nil {
			return nil, err
		}
	}

	newEndRow := dataRowStart + len(records) - 1
	if newEndRow < dataRowStart {
		newEndRow = dataRowStart
	}
	endCell, _ := excelize.CoordinatesToCellName(endCol, newEndRow)
	if err := f.DeleteTable(bukpotRequestTemplateTable); err == nil {
		table.Range = fmt.Sprintf("%s:%s", strings.Split(table.Range, ":")[0], endCell)
		if err := f.AddTable(bukpotRequestTemplateSheet, table); err != nil {
			return nil, err
		}
	}

	for colIndex, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(startCol+colIndex, headerRow)
		if err := f.SetCellValue(bukpotRequestTemplateSheet, cell, header); err != nil {
			return nil, err
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func buildBukpotRequestXML(records []bukpotRequestRecord, profileMeta appprofile.Metadata) ([]byte, error) {
	type bpu struct {
		TaxPeriodMonth                       string  `xml:"TaxPeriodMonth"`
		TaxPeriodYear                        string  `xml:"TaxPeriodYear"`
		CounterpartTIN                       string  `xml:"CounterpartTin"`
		IDPlaceOfBusinessActivityOfRecipient string  `xml:"IDPlaceOfBusinessActivityOfIncomeRecipient"`
		TaxCertificate                       string  `xml:"TaxCertificate"`
		TaxObjectCode                        string  `xml:"TaxObjectCode"`
		TaxBase                              int64   `xml:"TaxBase"`
		Rate                                 int64   `xml:"Rate"`
		Document                             string  `xml:"Document"`
		DocumentNumber                       string  `xml:"DocumentNumber"`
		DocumentDate                         string  `xml:"DocumentDate"`
		IDPlaceOfBusinessActivity            string  `xml:"IDPlaceOfBusinessActivity"`
		GovTreasurerOpt                      string  `xml:"GovTreasurerOpt"`
		SP2DNumber                           *string `xml:"SP2DNumber,omitempty"`
		WithholdingDate                      string  `xml:"WithholdingDate"`
	}
	type root struct {
		XMLName   xml.Name `xml:"BpuBulk"`
		XmlnsXSI  string   `xml:"xmlns:xsi,attr"`
		TIN       string   `xml:"TIN"`
		ListOfBpu []bpu    `xml:"ListOfBpu>Bpu"`
	}

	doc := root{
		XmlnsXSI:  "http://www.w3.org/2001/XMLSchema-instance",
		TIN:       profileMeta.NPWP,
		ListOfBpu: make([]bpu, 0, len(records)),
	}

	for _, record := range records {
		var sp2d *string
		if strings.TrimSpace(record.SP2DNumber) != "" {
			value := record.SP2DNumber
			sp2d = &value
		}
		doc.ListOfBpu = append(doc.ListOfBpu, bpu{
			TaxPeriodMonth:                       record.TaxPeriodMonth,
			TaxPeriodYear:                        record.TaxPeriodYear,
			CounterpartTIN:                       record.NPWP,
			IDPlaceOfBusinessActivityOfRecipient: record.IncomeRecipientBusinessActivityID,
			TaxCertificate:                       record.Facility,
			TaxObjectCode:                        record.TaxObjectCode,
			TaxBase:                              int64(record.TaxBase),
			Rate:                                 int64(record.RatePercent),
			Document:                             record.DocumentType,
			DocumentNumber:                       record.DocumentNumber,
			DocumentDate:                         record.DocumentDate.Format("2006-01-02"),
			IDPlaceOfBusinessActivity:            record.WithholderBusinessActivityID,
			GovTreasurerOpt:                      record.GovTreasurerOption,
			SP2DNumber:                           sp2d,
			WithholdingDate:                      record.WithholdingDate.Format("2006-01-02"),
		})
	}

	b, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), b...), nil
}

func buildBukpotRequestZIP(baseName string, xlsxBytes []byte, xmlBytes []byte) ([]byte, error) {
	buf := new(bytes.Buffer)
	zw := zip.NewWriter(buf)

	files := map[string][]byte{
		baseName + ".xlsx": xlsxBytes,
		baseName + ".xml":  xmlBytes,
	}
	for name, data := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(data); err != nil {
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func mapBukpotRequestFacility(source string, defaultRate float64) (string, float64, error) {
	switch strings.TrimSpace(source) {
	case "Surat Keterangan Bebas (SKB) Pemotongan PPh Pasal 23":
		return "TESTING", 0, nil
	case "Tanpa Fasilitas", "":
		return "N/A", defaultRate, nil
	default:
		return "", 0, fmt.Errorf("nilai Fasilitas tidak didukung: %s", source)
	}
}

func resolveBukpotRequestReference(taxInvoiceNumber string, invoiceNumber string) (string, string) {
	if !isBukpotReferenceEmpty(taxInvoiceNumber) {
		return "TaxInvoice", strings.TrimSpace(taxInvoiceNumber)
	}
	return "CommercialInvoice", strings.TrimSpace(invoiceNumber)
}

func isBukpotReferenceEmpty(value string) bool {
	trimmed := strings.TrimSpace(value)
	switch trimmed {
	case "", "0", "0.0", "00", "000":
		return true
	default:
		return false
	}
}

func formatBukpotDocumentReference(value string) string {
	if value == "" || !strings.Contains(value, ".") || !strings.Contains(value, "-") {
		return value
	}
	if !regexp.MustCompile(`^[0-9.\-]+$`).MatchString(value) {
		return value
	}
	cleaned := strings.NewReplacer(".", "", "-", "").Replace(value)
	if len(cleaned) > 4 {
		cleaned = cleaned[:4] + "9" + cleaned[4:]
	}
	if len(cleaned) < 17 {
		cleaned = cleaned + strings.Repeat("0", 17-len(cleaned))
	}
	if len(cleaned) > 17 {
		cleaned = cleaned[:17]
	}
	return cleaned
}

func normalizeBukpotRequestHeader(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), "")
}

func spreadsheetCell(row []string, indexes map[string]int, key string) string {
	idx, ok := indexes[key]
	if !ok || idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func isSpreadsheetRowEmpty(row []string) bool {
	for _, item := range row {
		if strings.TrimSpace(item) != "" {
			return false
		}
	}
	return true
}

func spreadsheetCellRowAt(rows [][]SpreadsheetCell, idx int) []SpreadsheetCell {
	if idx < 0 || idx >= len(rows) {
		return nil
	}
	return rows[idx]
}

func spreadsheetTypedCell(row []string, cells []SpreadsheetCell, indexes map[string]int, key string) SpreadsheetCell {
	cell := SpreadsheetCellAt(cells, indexes, key)
	if SpreadsheetCellText(cell) != "" || cell.FloatValue != nil || cell.DateValue != "" || cell.BoolValue != nil {
		return cell
	}
	return SpreadsheetCell{
		Display:     spreadsheetCell(row, indexes, key),
		StringValue: spreadsheetCell(row, indexes, key),
		ValueType:   SpreadsheetCellValueTypeString,
	}
}

func spreadsheetTextValue(row []string, cells []SpreadsheetCell, indexes map[string]int, key string) string {
	return SpreadsheetCellText(spreadsheetTypedCell(row, cells, indexes, key))
}

func describeSpreadsheetCellValue(cell SpreadsheetCell) string {
	display := strings.TrimSpace(cell.Display)
	raw := strings.TrimSpace(cell.Raw)

	switch {
	case display != "" && raw != "" && display != raw:
		return fmt.Sprintf("display=%q, raw=%q", display, raw)
	case display != "":
		return fmt.Sprintf("value=%q", display)
	case raw != "":
		return fmt.Sprintf("raw=%q", raw)
	default:
		return "nilai kosong"
	}
}

func formatBukpotWorkbookDate(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format("02/01/2006")
}

func sanitizeBukpotRequestFilename(raw string) string {
	value := strings.TrimSpace(raw)
	value = bukpotRequestInvalidFilenameCharRegex.ReplaceAllString(value, "_")
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Trim(value, ".-_ ")
	return strings.TrimSuffix(value, filepath.Ext(value))
}
