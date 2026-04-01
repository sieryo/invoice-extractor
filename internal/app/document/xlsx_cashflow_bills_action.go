package document

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	appcashflowbill "github.com/sieryo/invoice-extractor/internal/app/cashflowbill"
	appledger "github.com/sieryo/invoice-extractor/internal/app/ledger"
)

type cashflowBillFieldDefinition struct {
	Key      string
	Required bool
	Aliases  []string
}

type cashflowBillRowRecord struct {
	RowNumber    int
	Date         time.Time
	Category     string
	Information  string
	PartyName    string
	Total        float64
}

func (p *XLSXCashflowProcessor) runCashflowBillAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	result := ActionResult{
		ActionID:    req.ActionID,
		ActionType:  req.ActionType,
		StartedAt:   req.RequestedAt,
		ItemResults: make([]ActionItemResult, 0, len(req.SnapshotDocs)),
		Outputs:     make([]ActionOutput, 0, 1),
	}

	input, err := appcashflowbill.ParseExportInput(req.Input)
	if err != nil {
		result.Status = "failed"
		result.Message = "invalid cashflow bills action input"
		result.FinishedAt = time.Now()
		return result, err
	}
	if strings.TrimSpace(req.ActionType) == cashflowReceivePaymentsActionType {
		input.CashflowType = appcashflowbill.ReceivePaymentsType
	} else {
		input.CashflowType = appcashflowbill.PayBillsType
	}
	if strings.TrimSpace(input.LedgerSnapshotRef) == "" {
		result.Status = "failed"
		result.Message = "snapshot ledger wajib di-upload"
		result.FinishedAt = time.Now()
		return result, errors.New("ledger snapshot ref is required")
	}
	if p.categoryAccounts == nil {
		result.Status = "failed"
		result.Message = "master data category accounts belum tersedia"
		result.FinishedAt = time.Now()
		return result, errors.New("cashflow category account provider is nil")
	}

	categoryAccounts, err := p.categoryAccounts.Load(req.UserID)
	if err != nil {
		result.Status = "failed"
		result.Message = "master data category accounts belum siap"
		result.FinishedAt = time.Now()
		return result, err
	}

	snapshotBytes, err := p.fileStore.ReadArchive(ctx, req.CollectionID, input.LedgerSnapshotRef)
	if err != nil {
		result.Status = "failed"
		result.Message = "gagal membaca snapshot ledger"
		result.FinishedAt = time.Now()
		return result, err
	}
	snapshot, err := appledger.ParseSnapshot(bufio.NewReader(bytes.NewReader(snapshotBytes)))
	if err != nil {
		result.Status = "failed"
		result.Message = "gagal parse snapshot ledger"
		result.FinishedAt = time.Now()
		return result, err
	}

	rows := make([][]string, 0, len(req.SnapshotDocs)*4)
	if input.CashflowType == appcashflowbill.ReceivePaymentsType {
		rows = append(rows, appcashflowbill.ReceivePaymentsHeader())
	} else {
		rows = append(rows, appcashflowbill.PayBillsHeader())
	}

	for _, doc := range req.SnapshotDocs {
		workbook, loadErr := LoadSpreadsheetWorkbookForExecution(ctx, p.fileStore, req.CollectionID, doc.NormalizedRef, doc.RawRef)
		if loadErr != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: gagal membaca workbook cashflow", doc.SourceName),
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

		entryRows, warnings, processedCount, buildErr := p.buildCashflowBillDocumentRows(sheet, input, snapshot, categoryAccounts)
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
		rows = append(rows, entryRows...)

		itemStatus := "success"
		itemMessage := fmt.Sprintf("%d transaksi diexport", processedCount)
		if len(warnings) > 0 {
			itemStatus = "warning"
			itemMessage = fmt.Sprintf("%d transaksi diexport dengan peringatan", processedCount)
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
		result.Message = "export cashflow bills gagal untuk semua dokumen"
		result.FinishedAt = time.Now()
		return result, errors.New("no cashflow bill document was exported successfully")
	}

	body, err := appcashflowbill.EncodeTabDelimitedText(rows)
	if err != nil {
		result.Status = "failed"
		result.Message = "gagal membangun output text MYOB"
		result.FinishedAt = time.Now()
		return result, err
	}

	outputName := fmt.Sprintf(
		"%s_%s.txt",
		sanitizeCashflowOutputFilename(input.OutputFilename),
		time.Now().Format("20060102_150405"),
	)
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
		if input.CashflowType == appcashflowbill.ReceivePaymentsType {
			result.Message = "export receive payments berhasil"
		} else {
			result.Message = "export pay bills berhasil"
		}
	}
	result.FinishedAt = time.Now()
	return result, nil
}

func (p *XLSXCashflowProcessor) buildCashflowBillDocumentRows(
	sheet SpreadsheetSheet,
	input appcashflowbill.ExportInput,
	snapshot *appledger.Snapshot,
	categoryAccounts map[string]appcashflowbill.CategoryAccount,
) ([][]string, []string, int, error) {
	_, headerIndex, err := resolveCashflowBillHeaders(sheet, input.HeaderRowNumber, input)
	if err != nil {
		return nil, nil, 0, err
	}

	warnings := make([]string, 0)
	grouped := make(map[string][]cashflowBillRowRecord)
	displayPartyByKey := make(map[string]string)

	for idx, rawRow := range sheet.RawRows {
		rowNumber := SpreadsheetRowNumberAt(sheet, idx)
		if rowNumber <= input.HeaderRowNumber {
			continue
		}

		cellRow := spreadsheetCashflowCellRowAt(sheet.RawCellRows, idx)
		record, parseErr := parseCashflowBillRow(rawRow, cellRow, rowNumber, headerIndex)
		if parseErr != nil {
			warnings = append(warnings, fmt.Sprintf("row %d: %s", rowNumber, parseErr.Error()))
			continue
		}
		if skipReason, shouldSkip := resolveCashflowBillSkipReason(record, input); shouldSkip {
			warnings = append(warnings, fmt.Sprintf("row %d: %s", rowNumber, skipReason))
			continue
		}
		record = normalizeCashflowBillRecord(record)

		partyKey := normalizeCashflowBillPartyName(record.PartyName)
		if partyKey == "" {
			warnings = append(warnings, fmt.Sprintf("row %d: nama customer / supplier kosong", rowNumber))
			continue
		}

		if _, ok := categoryAccounts[normalizeCashflowBillLookupKey(record.Category)]; !ok {
			continue
		}
		displayPartyByKey[partyKey] = record.PartyName
		grouped[partyKey] = append(grouped[partyKey], record)
	}

	if len(grouped) == 0 {
		return nil, uniqueStrings(warnings), 0, errors.New("tidak ada row valid yang dapat dikonversi")
	}

	partyKeys := make([]string, 0, len(grouped))
	for key := range grouped {
		partyKeys = append(partyKeys, key)
	}
	sort.Strings(partyKeys)

	rows := make([][]string, 0)
	processed := 0
	for _, partyKey := range partyKeys {
		tx, txWarnings, err := buildCashflowBillTransaction(displayPartyByKey[partyKey], grouped[partyKey], snapshot)
		warnings = append(warnings, txWarnings...)
		if err != nil {
			warnings = append(warnings, err.Error())
			continue
		}

		if len(rows) > 0 {
			rows = append(rows, []string{})
		}

		if input.CashflowType == appcashflowbill.ReceivePaymentsType {
			rows = append(rows, appcashflowbill.BuildReceivePaymentsRows(tx, input.ChequeAccount)...)
		} else {
			rows = append(rows, appcashflowbill.BuildPayBillsRows(tx, input.ChequeAccount)...)
		}
		processed++
	}

	if processed == 0 {
		return nil, uniqueStrings(warnings), 0, errors.New("tidak ada transaksi valid yang dapat dikonversi")
	}
	return rows, uniqueStrings(warnings), processed, nil
}

func resolveCashflowBillHeaders(sheet SpreadsheetSheet, headerRowNumber int, input appcashflowbill.ExportInput) ([]string, map[string]int, error) {
	headerIdx := -1
	for idx := range sheet.RawRows {
		if SpreadsheetRowNumberAt(sheet, idx) == headerRowNumber {
			headerIdx = idx
			break
		}
	}
	if headerIdx < 0 {
		return nil, nil, fmt.Errorf("baris header %d tidak ditemukan pada sheet %q", headerRowNumber, sheet.Name)
	}

	headers := sheet.RawRows[headerIdx]
	byNormalized := make(map[string]int, len(headers))
	for idx, header := range headers {
		key := normalizeCashflowHeader(header)
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
	for _, def := range cashflowBillFieldDefinitions(input) {
		idx, ok := resolveCashflowFieldIndex(cashflowFieldDefinition{
			Key:      def.Key,
			Required: def.Required,
			Aliases:  def.Aliases,
		}, byNormalized)
		if !ok {
			if def.Required {
				missing = append(missing, def.Key)
			}
			continue
		}
		fieldIndex[def.Key] = idx
	}
	if len(missing) > 0 {
		return nil, nil, fmt.Errorf("kolom wajib cashflow bills tidak lengkap: %s", strings.Join(missing, ", "))
	}
	return headers, fieldIndex, nil
}

func cashflowBillFieldDefinitions(input appcashflowbill.ExportInput) []cashflowBillFieldDefinition {
	return []cashflowBillFieldDefinition{
		{Key: "date", Required: true, Aliases: cashflowAliases(input.MappedField("date"), "date", "tanggal")},
		{Key: "category", Required: true, Aliases: cashflowAliases(input.MappedField("category"), "category")},
		{Key: "information", Required: true, Aliases: cashflowAliases(input.MappedField("information"), "note", "keterangan", "information")},
		{Key: "partyName", Required: true, Aliases: cashflowAliases(input.MappedField("partyName"), "nama customer / supplier", "customer", "supplier")},
		{Key: "total", Required: true, Aliases: cashflowAliases(input.MappedField("total"), "idr", "total")},
	}
}

func parseCashflowBillRow(row []string, cells []SpreadsheetCell, rowNumber int, fieldIndex map[string]int) (cashflowBillRowRecord, error) {
	record := cashflowBillRowRecord{RowNumber: rowNumber}
	record.Category = cashflowCellValue(row, cells, fieldIndex, "category")
	record.Information = cashflowCellValue(row, cells, fieldIndex, "information")
	record.PartyName = cashflowCellValue(row, cells, fieldIndex, "partyName")

	dateCell := cashflowTypedCell(row, cells, fieldIndex, "date")
	dateValue := SpreadsheetCellText(dateCell)
	if dateValue == "" {
		return record, errors.New("tanggal kosong")
	}
	date, ok := SpreadsheetCellDate(dateCell)
	if !ok {
		return record, fmt.Errorf("tanggal tidak valid: %s", dateValue)
	}
	record.Date = date

	totalCell := cashflowTypedCell(row, cells, fieldIndex, "total")
	total, ok := SpreadsheetCellMoney(totalCell)
	if !ok {
		return record, errors.New("total tidak valid")
	}
	record.Total = total
	if strings.TrimSpace(record.Information) == "" {
		return record, errors.New("keterangan kosong")
	}
	return record, nil
}

func resolveCashflowBillSkipReason(record cashflowBillRowRecord, input appcashflowbill.ExportInput) (string, bool) {
	if input.CashflowType == appcashflowbill.ReceivePaymentsType {
		if record.Total < 0 {
			return "total negatif dilewati untuk mode Receive Payments", true
		}
		return "", false
	}
	if record.Total > 0 {
		return "total positif dilewati untuk mode Pay Bills", true
	}
	return "", false
}

func normalizeCashflowBillRecord(record cashflowBillRowRecord) cashflowBillRowRecord {
	record.Total = normalizeNearZero(absFloat(record.Total))
	record.Category = strings.TrimSpace(record.Category)
	record.Information = strings.TrimSpace(record.Information)
	record.PartyName = strings.TrimSpace(record.PartyName)
	return record
}

func buildCashflowBillTransaction(
	partyDisplayName string,
	rows []cashflowBillRowRecord,
	snapshot *appledger.Snapshot,
) (appcashflowbill.Transaction, []string, error) {
	if len(rows) == 0 {
		return appcashflowbill.Transaction{}, nil, errors.New("group row kosong")
	}

	partyKey := normalizeCashflowBillPartyName(partyDisplayName)
	party, ok := snapshot.Parties[partyKey]
	if !ok || party == nil {
		return appcashflowbill.Transaction{}, nil, fmt.Errorf("customer / supplier %q tidak ditemukan di snapshot ledger", partyDisplayName)
	}

	items := append([]appledger.SnapshotItem(nil), party.Items...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].Date.Equal(items[j].Date) {
			return items[i].ID < items[j].ID
		}
		return items[i].Date.Before(items[j].Date)
	})

	total := 0.0
	rowNumbers := make([]int, 0, len(rows))
	resultItems := make([]appcashflowbill.TransactionItem, 0)
	paidSoFar := make(map[string]float64)
	warnings := make([]string, 0)

	for _, row := range rows {
		total += row.Total
		rowNumbers = append(rowNumbers, row.RowNumber)
		amountLeft := row.Total

		for _, item := range items {
			remaining := normalizeNearZero(item.Amount - paidSoFar[item.ID])
			if remaining <= 0 {
				continue
			}
			if amountLeft <= 0 {
				break
			}

			pay := remaining
			isPartial := false
			if amountLeft < remaining {
				pay = amountLeft
				isPartial = true
			}

			resultItems = append(resultItems, appcashflowbill.TransactionItem{
				RowSource:  row.RowNumber,
				RefID:      item.ID,
				RefDate:    item.Date,
				Memo:       row.Information,
				ActionDate: row.Date,
				Amount:     pay,
				IsPartial:  isPartial,
			})
			paidSoFar[item.ID] += pay
			amountLeft = normalizeNearZero(amountLeft - pay)
		}

		if amountLeft > 0 {
			warnings = append(warnings, fmt.Sprintf("row %d: payment melebihi saldo outstanding %q sebesar %.2f, sisanya dicatat sebagai credit", row.RowNumber, partyDisplayName, amountLeft))
			resultItems = append(resultItems, appcashflowbill.TransactionItem{
				RowSource:  row.RowNumber,
				RefID:      "",
				RefDate:    row.Date,
				Memo:       row.Information,
				ActionDate: row.Date,
				Amount:     amountLeft,
				IsPartial:  false,
			})
		}
	}

	return appcashflowbill.Transaction{
		Party:      partyDisplayName,
		RowNumbers: rowNumbers,
		Items:      resultItems,
		Total:      total,
	}, warnings, nil
}

func normalizeCashflowBillLookupKey(raw string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(raw))), " ")
}

func normalizeCashflowBillPartyName(raw string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
