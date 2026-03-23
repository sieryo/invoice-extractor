package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
)

const (
	cashflowMYOBActionType   = "export_cashflow_myob"
	cashflowDefaultTextName  = "cashflow-myob"
	cashflowSheetLookupError = "sheet cashflow tidak ditemukan"
)

var cashflowInvalidFilenameCharRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

type cashflowFieldDefinition struct {
	Key      string
	Required bool
	Aliases  []string
}

type cashflowRowRecord struct {
	RowNumber   int
	Date        time.Time
	Information string
	COA         string
	Remark      string
	OtherCost   float64
	PP23        float64
	PPH15       float64
	PPH21       float64
	PPH23       float64
	PPH42       float64
	PPN         float64
	Total       float64
}

type cashflowComponent struct {
	AccountCode string
	Amount      float64
	Allocation  string
}

func (p *XLSXCashflowProcessor) RunAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	result := ActionResult{
		ActionID:    req.ActionID,
		ActionType:  req.ActionType,
		StartedAt:   req.RequestedAt,
		ItemResults: make([]ActionItemResult, 0, len(req.SnapshotDocs)),
		Outputs:     make([]ActionOutput, 0, 1),
	}

	if strings.TrimSpace(req.ActionType) != cashflowMYOBActionType {
		result.Status = "failed"
		result.Message = "unsupported action for cashflow_import"
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("%w: action %s for %s", ErrProcessorNotImplemented, req.ActionType, p.Key().CollectionKind)
	}
	if len(req.SnapshotDocs) == 0 {
		result.Status = "failed"
		result.Message = "snapshot is empty"
		result.FinishedAt = time.Now()
		return result, errors.New("snapshot is empty")
	}

	input, err := appcashflow.ParseExportMYOBInput(req.Input)
	if err != nil {
		result.Status = "failed"
		result.Message = "invalid cashflow action input"
		result.FinishedAt = time.Now()
		return result, err
	}
	if input.CashflowType != appcashflow.SpendMoneyType {
		result.Status = "failed"
		result.Message = "hanya spend money yang didukung saat ini"
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("cashflow type %q belum diimplementasikan", input.CashflowType)
	}

	if p.taxAccounts == nil {
		result.Status = "failed"
		result.Message = "master data tax accounts belum tersedia"
		result.FinishedAt = time.Now()
		return result, errors.New("cashflow tax account provider is nil")
	}

	taxAccounts, err := p.taxAccounts.Load()
	if err != nil {
		result.Status = "failed"
		result.Message = "master data tax accounts belum siap"
		result.FinishedAt = time.Now()
		return result, err
	}

	rows := make([][]string, 0, len(req.SnapshotDocs)*4)
	rows = append(rows, appcashflow.SpendMoneyHeader())

	nextChequeNumber := 0
	if input.StartingChequeNumber != nil && *input.StartingChequeNumber > 0 {
		nextChequeNumber = *input.StartingChequeNumber
	}
	for _, doc := range req.SnapshotDocs {
		workbook, loadErr := LoadCashflowWorkbook(ctx, p.fileStore, req.CollectionID, doc.NormalizedRef)
		if loadErr != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: gagal membaca workbook cashflow", doc.SourceName),
				Error:      loadErr.Error(),
			})
			continue
		}

		sheet, findErr := findCashflowSheet(*workbook, input.SheetName)
		if findErr != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: %s", doc.SourceName, findErr.Error()),
				Error:      findErr.Error(),
			})
			continue
		}

		entryRows, warnings, processedCount, buildErr := p.buildSpendMoneyDocumentRows(doc.SourceName, sheet, input, taxAccounts, nextChequeNumber)
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
				Error:      "no exportable cashflow rows",
				Warnings:   warnings,
			})
			continue
		}

		if nextChequeNumber > 0 {
			nextChequeNumber += processedCount
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
		result.Message = "export spend money gagal untuk semua dokumen"
		result.FinishedAt = time.Now()
		return result, errors.New("no cashflow document was exported successfully")
	}

	body, err := appcashflow.EncodeTabDelimitedText(rows)
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
		result.Message = "export spend money berhasil"
	}
	result.FinishedAt = time.Now()
	return result, nil
}

func (p *XLSXCashflowProcessor) buildSpendMoneyDocumentRows(
	sourceName string,
	sheet CashflowSheet,
	input appcashflow.ExportMYOBInput,
	taxAccounts map[string]appcashflow.TaxAccount,
	startChequeNumber int,
) ([][]string, []string, int, error) {
	headers, headerIndex, err := resolveCashflowHeaders(sheet, input.HeaderRowNumber)
	if err != nil {
		return nil, nil, 0, err
	}
	_ = headers

	rows := make([][]string, 0)
	warnings := make([]string, 0)
	processed := 0

	for idx, rawRow := range sheet.RawRows {
		rowNumber := sheetRowNumberAt(sheet, idx)
		if rowNumber <= input.HeaderRowNumber {
			continue
		}

		record, rowWarnings, recordErr := parseCashflowRow(rawRow, rowNumber, headerIndex)
		if recordErr != nil {
			warnings = append(warnings, fmt.Sprintf("row %d: %s", rowNumber, recordErr.Error()))
			continue
		}
		warnings = append(warnings, rowWarnings...)

		if input.SkipPositiveTotal && record.Total > 0 {
			warnings = append(warnings, fmt.Sprintf("row %d: total positif dilewati", rowNumber))
			continue
		}

		currentChequeNumber := 0
		if startChequeNumber > 0 {
			currentChequeNumber = startChequeNumber + processed
		}
		tx, txWarnings, txErr := buildSpendMoneyTransaction(record, input, taxAccounts, currentChequeNumber)
		if txErr != nil {
			warnings = append(warnings, fmt.Sprintf("row %d: %s", rowNumber, txErr.Error()))
			continue
		}
		warnings = append(warnings, prefixWarnings(rowNumber, txWarnings)...)
		rows = append(rows, appcashflow.BuildSpendMoneyRows(tx, input.ChequeAccount)...)
		processed++
	}

	if processed == 0 {
		return rows, uniqueStrings(warnings), 0, fmt.Errorf("%s: %s", sourceName, "tidak ada row valid yang dapat dikonversi")
	}
	return rows, uniqueStrings(warnings), processed, nil
}

func resolveCashflowHeaders(sheet CashflowSheet, headerRowNumber int) ([]string, map[string]int, error) {
	headerIdx := -1
	for idx := range sheet.RawRows {
		if sheetRowNumberAt(sheet, idx) == headerRowNumber {
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
	for _, def := range cashflowFieldDefinitions() {
		idx, ok := resolveCashflowFieldIndex(def, byNormalized)
		if !ok {
			if def.Required {
				missing = append(missing, def.Key)
			}
			continue
		}
		fieldIndex[def.Key] = idx
	}
	if len(missing) > 0 {
		return nil, nil, fmt.Errorf("kolom wajib cashflow tidak lengkap: %s", strings.Join(missing, ", "))
	}

	return headers, fieldIndex, nil
}

func parseCashflowRow(row []string, rowNumber int, fieldIndex map[string]int) (cashflowRowRecord, []string, error) {
	record := cashflowRowRecord{RowNumber: rowNumber}
	record.Information = cashflowCell(row, fieldIndex, "information")
	record.COA = cashflowCell(row, fieldIndex, "coa")
	record.Remark = cashflowCell(row, fieldIndex, "remark")

	dateValue := cashflowCell(row, fieldIndex, "date")
	if dateValue == "" {
		return record, nil, errors.New("tanggal kosong")
	}
	date, err := parseCashflowDate(dateValue)
	if err != nil {
		return record, nil, fmt.Errorf("tanggal tidak valid: %s", dateValue)
	}
	record.Date = date

	total, err := parseCashflowAmount(cashflowCell(row, fieldIndex, "total"))
	if err != nil {
		return record, nil, fmt.Errorf("total tidak valid")
	}
	record.Total = total
	if strings.TrimSpace(record.Information) == "" {
		return record, nil, errors.New("keterangan kosong")
	}

	record.OtherCost, _ = parseOptionalCashflowAmount(cashflowCell(row, fieldIndex, "otherCost"))
	record.PP23, _ = parseOptionalCashflowAmount(cashflowCell(row, fieldIndex, "pp23"))
	record.PPH15, _ = parseOptionalCashflowAmount(cashflowCell(row, fieldIndex, "pph15"))
	record.PPH21, _ = parseOptionalCashflowAmount(cashflowCell(row, fieldIndex, "pph21"))
	record.PPH23, _ = parseOptionalCashflowAmount(cashflowCell(row, fieldIndex, "pph23"))
	record.PPH42, _ = parseOptionalCashflowAmount(cashflowCell(row, fieldIndex, "pph42"))
	record.PPN, _ = parseOptionalCashflowAmount(cashflowCell(row, fieldIndex, "ppn"))

	return record, nil, nil
}

func buildSpendMoneyTransaction(
	record cashflowRowRecord,
	input appcashflow.ExportMYOBInput,
	taxAccounts map[string]appcashflow.TaxAccount,
	chequeNumber int,
) (appcashflow.SpendMoneyTransaction, []string, error) {
	components := make([]cashflowComponent, 0, 8)
	warnings := make([]string, 0)

	if amount := normalizeNearZero(record.OtherCost); amount != 0 {
		accountCode := strings.TrimSpace(input.OtherCostsAccountCode)
		if accountCode == "" {
			accountCode = lookupCashflowAccountCode(taxAccounts, "Admin Bank")
		}
		if accountCode == "" {
			return appcashflow.SpendMoneyTransaction{}, warnings, errors.New("account code untuk biaya lain tidak ditemukan")
		}
		components = append(components, cashflowComponent{
			AccountCode: accountCode,
			Amount:      amount,
			Allocation:  "Biaya Lainnya",
		})
	}

	for _, taxComponent := range []struct {
		Label  string
		Amount float64
	}{
		{Label: "PP 23", Amount: record.PP23},
		{Label: "PPh 15%", Amount: record.PPH15},
		{Label: "PPH 21", Amount: record.PPH21},
		{Label: "PPH 23", Amount: record.PPH23},
		{Label: "PPH 4 (2)", Amount: record.PPH42},
		{Label: "PPN", Amount: record.PPN},
	} {
		amount := normalizeNearZero(taxComponent.Amount)
		if amount == 0 {
			continue
		}
		accountCode := lookupCashflowAccountCode(taxAccounts, taxComponent.Label)
		if accountCode == "" {
			return appcashflow.SpendMoneyTransaction{}, warnings, fmt.Errorf("lookup account untuk %s tidak ditemukan", taxComponent.Label)
		}
		components = append(components, cashflowComponent{
			AccountCode: accountCode,
			Amount:      amount,
			Allocation:  taxComponent.Label,
		})
	}

	baseAmount := normalizeNearZero(record.Total - sumCashflowComponents(components))
	if baseAmount == 0 && len(components) == 0 {
		return appcashflow.SpendMoneyTransaction{}, warnings, errors.New("row tidak memiliki komponen transaksi")
	}

	baseAccountCode, baseWarnings, err := resolveCashflowBaseAccountCode(record, input, taxAccounts)
	if err != nil {
		return appcashflow.SpendMoneyTransaction{}, warnings, err
	}
	warnings = append(warnings, baseWarnings...)

	if baseAmount != 0 {
		components = append([]cashflowComponent{{
			AccountCode: baseAccountCode,
			Amount:      baseAmount,
			Allocation:  resolveCashflowAllocationMemo(record, input),
		}}, components...)
	}

	items := make([]appcashflow.SpendMoneyTransactionItem, 0, len(components))
	for _, component := range components {
		items = append(items, appcashflow.SpendMoneyTransactionItem{
			AccountCode: component.AccountCode,
			Amount:      component.Amount,
			Allocation:  component.Allocation,
		})
	}

	var chequeNumberPtr *int
	if chequeNumber > 0 {
		chequeNumberCopy := chequeNumber
		chequeNumberPtr = &chequeNumberCopy
	}

	return appcashflow.SpendMoneyTransaction{
		ChequeNumber: chequeNumberPtr,
		Date:         appcashflow.EnsureNonZeroTime(record.Date),
		Memo:         record.Information,
		Amount:       record.Total,
		Allocation:   resolveCashflowAllocationMemo(record, input),
		Items:        items,
	}, uniqueStrings(warnings), nil
}

func resolveCashflowBaseAccountCode(
	record cashflowRowRecord,
	input appcashflow.ExportMYOBInput,
	taxAccounts map[string]appcashflow.TaxAccount,
) (string, []string, error) {
	coa := strings.TrimSpace(record.COA)
	if looksLikeAccountCode(coa) {
		return coa, nil, nil
	}
	if accountCode := lookupCashflowAccountCode(taxAccounts, coa); accountCode != "" {
		return accountCode, nil, nil
	}

	if input.CashflowFormat == appcashflow.InfluencerFormat {
		joined := strings.ToLower(strings.Join([]string{record.COA, record.Information, record.Remark}, " "))
		if strings.Contains(joined, "bank") && strings.TrimSpace(input.DefaultBAccountCode) != "" {
			return input.DefaultBAccountCode, []string{"coa tidak berupa account code, fallback ke Default Bank Account Code"}, nil
		}
		if strings.TrimSpace(input.DefaultIAccountCode) != "" {
			return input.DefaultIAccountCode, []string{"coa tidak berupa account code, fallback ke Default Influencer Account Code"}, nil
		}
	}

	return "", nil, fmt.Errorf("account code untuk COA %q tidak ditemukan", coa)
}

func resolveCashflowAllocationMemo(record cashflowRowRecord, input appcashflow.ExportMYOBInput) string {
	remark := strings.TrimSpace(record.Remark)
	if remark == "" {
		return strings.TrimSpace(record.Information)
	}
	delimiter := strings.TrimSpace(input.RemarkDelimiter)
	if delimiter == "" || !strings.Contains(remark, delimiter) {
		return remark
	}
	parts := strings.Split(remark, delimiter)
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		clean = append(clean, value)
	}
	if len(clean) == 0 {
		return strings.TrimSpace(record.Information)
	}
	return strings.Join(clean, " / ")
}

func lookupCashflowAccountCode(accounts map[string]appcashflow.TaxAccount, key string) string {
	if len(accounts) == 0 {
		return ""
	}
	record, ok := accounts[normalizeCashflowLookupKey(key)]
	if !ok {
		return ""
	}
	return strings.TrimSpace(record.Account)
}

func parseCashflowDate(raw string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, errors.New("empty date")
	}

	layouts := []string{
		"2006-01-02",
		"02/01/2006",
		"2/1/2006",
		"02-01-2006",
		"2-1-2006",
		"02 Jan 2006",
		"2 Jan 2006",
		"02 January 2006",
		"2 January 2006",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}

	if n, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64); err == nil && n > 0 {
		if t, err := excelDateToTime(n); err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.New("unsupported date format")
}

func parseCashflowAmount(raw string) (float64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("empty amount")
	}
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "\u00a0", "")

	if strings.Contains(value, ",") && strings.Contains(value, ".") {
		lastComma := strings.LastIndex(value, ",")
		lastDot := strings.LastIndex(value, ".")
		if lastComma > lastDot {
			value = strings.ReplaceAll(value, ".", "")
			value = strings.ReplaceAll(value, ",", ".")
		} else {
			value = strings.ReplaceAll(value, ",", "")
		}
	} else if strings.Count(value, ",") == 1 && !strings.Contains(value, ".") {
		value = strings.ReplaceAll(value, ",", ".")
	} else {
		value = strings.ReplaceAll(value, ",", "")
	}

	return strconv.ParseFloat(value, 64)
}

func parseOptionalCashflowAmount(raw string) (float64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}
	return parseCashflowAmount(value)
}

func cashflowFieldDefinitions() []cashflowFieldDefinition {
	return []cashflowFieldDefinition{
		{Key: "date", Required: true, Aliases: []string{"tanggal", "date"}},
		{Key: "information", Required: true, Aliases: []string{"note", "keterangan", "deskripsi", "information"}},
		{Key: "coa", Required: true, Aliases: []string{"coa", "chartofaccount", "chartofaccounts"}},
		{Key: "otherCost", Required: false, Aliases: []string{"bylainnya", "biayalainnya", "othercost"}},
		{Key: "pp23", Required: false, Aliases: []string{"pp23", "pp 23"}},
		{Key: "pph15", Required: false, Aliases: []string{"pph15%", "pph15"}},
		{Key: "pph21", Required: false, Aliases: []string{"pph21", "pph 21"}},
		{Key: "pph23", Required: false, Aliases: []string{"pph23", "pph 23"}},
		{Key: "pph42", Required: false, Aliases: []string{"pph4(2)", "pph4 2", "pph 4 (2)", "pph 4(2)"}},
		{Key: "ppn", Required: false, Aliases: []string{"ppn"}},
		{Key: "remark", Required: false, Aliases: []string{"catatan", "remark", "memo"}},
		{Key: "total", Required: true, Aliases: []string{"idr", "total", "nominal"}},
	}
}

func resolveCashflowFieldIndex(def cashflowFieldDefinition, byNormalized map[string]int) (int, bool) {
	for _, alias := range def.Aliases {
		idx, ok := byNormalized[normalizeCashflowHeader(alias)]
		if ok {
			return idx, true
		}
	}
	return 0, false
}

func normalizeCashflowHeader(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}
	replacer := strings.NewReplacer(" ", "", "_", "", "-", "", "/", "", ".", "", "(", "", ")", "", "%", "%")
	return replacer.Replace(value)
}

func normalizeCashflowLookupKey(raw string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(raw))), " ")
}

func cashflowCell(row []string, fieldIndex map[string]int, key string) string {
	idx, ok := fieldIndex[key]
	if !ok || idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func sheetRowNumberAt(sheet CashflowSheet, idx int) int {
	if idx >= 0 && idx < len(sheet.RawRowNumbers) && sheet.RawRowNumbers[idx] > 0 {
		return sheet.RawRowNumbers[idx]
	}
	if idx < len(sheet.RawRows) {
		return sheet.HeaderRowIndex + idx
	}
	return idx + 1
}

func findCashflowSheet(workbook CashflowWorkbook, sheetName string) (CashflowSheet, error) {
	target := strings.TrimSpace(sheetName)
	if target == "" {
		target = strings.TrimSpace(workbook.PrimarySheet)
	}
	for _, sheet := range workbook.Sheets {
		if strings.EqualFold(strings.TrimSpace(sheet.Name), target) {
			return sheet, nil
		}
	}
	return CashflowSheet{}, fmt.Errorf("%s: %s", cashflowSheetLookupError, target)
}

func prefixWarnings(rowNumber int, warnings []string) []string {
	if len(warnings) == 0 {
		return nil
	}
	out := make([]string, 0, len(warnings))
	for _, warning := range warnings {
		if strings.TrimSpace(warning) == "" {
			continue
		}
		out = append(out, fmt.Sprintf("row %d: %s", rowNumber, warning))
	}
	return out
}

func sanitizeCashflowOutputFilename(raw string) string {
	value := strings.TrimSpace(raw)
	value = cashflowInvalidFilenameCharRegex.ReplaceAllString(value, "_")
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Trim(value, ".-_ ")
	if value == "" {
		return cashflowDefaultTextName
	}
	return strings.TrimSuffix(value, filepath.Ext(value))
}

func looksLikeAccountCode(raw string) bool {
	value := strings.TrimSpace(raw)
	if value == "" {
		return false
	}
	digits := 0
	for _, char := range value {
		if char >= '0' && char <= '9' {
			digits++
			continue
		}
		if char == '-' || char == '/' || char == ' ' {
			continue
		}
		return false
	}
	return digits >= 3
}

func sumCashflowComponents(components []cashflowComponent) float64 {
	total := 0.0
	for _, component := range components {
		total += component.Amount
	}
	return total
}

func normalizeNearZero(value float64) float64 {
	if math.Abs(value) < 0.00001 {
		return 0
	}
	return value
}

func excelDateToTime(serial float64) (time.Time, error) {
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	if serial < 1 {
		return time.Time{}, errors.New("invalid excel serial")
	}
	days := int(serial)
	fraction := serial - float64(days)
	seconds := int(math.Round(fraction * 24 * 60 * 60))
	return base.AddDate(0, 0, days).Add(time.Duration(seconds) * time.Second), nil
}
