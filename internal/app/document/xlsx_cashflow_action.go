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
	"github.com/sieryo/invoice-extractor/internal/taxcatalog"
)

const (
	cashflowMYOBActionType  = "export_cashflow_myob"
	cashflowSpendActionType = "export_cashflow_spend_money"
	cashflowRecvActionType  = "export_cashflow_receive_money"
	cashflowPayBillsActionType = "cashflow_to_pay_bills"
	cashflowReceivePaymentsActionType = "cashflow_to_receive_payments"
	cashflowDefaultTextName = "cashflow-myob"
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

type cashflowBaseAccountResolution struct {
	AccountCode string
	AccountKind string
	Warnings    []string
}

func (p *XLSXCashflowProcessor) RunAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	result := ActionResult{
		ActionID:    req.ActionID,
		ActionType:  req.ActionType,
		StartedAt:   req.RequestedAt,
		ItemResults: make([]ActionItemResult, 0, len(req.SnapshotDocs)),
		Outputs:     make([]ActionOutput, 0, 1),
	}

	if !isCashflowMYOBAction(req.ActionType) {
		if isCashflowBillActionType(req.ActionType) {
			return p.runCashflowBillAction(ctx, req)
		}
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
	input.CashflowType = cashflowTypeFromAction(req.ActionType)

	if p.taxAccounts == nil {
		result.Status = "failed"
		result.Message = "master data tax accounts belum tersedia"
		result.FinishedAt = time.Now()
		return result, errors.New("cashflow tax account provider is nil")
	}

	taxAccounts, err := p.taxAccounts.Load(req.UserID)
	if err != nil {
		result.Status = "failed"
		result.Message = "master data tax accounts belum siap"
		result.FinishedAt = time.Now()
		return result, err
	}

	rows := make([][]string, 0, len(req.SnapshotDocs)*4)
	if input.CashflowType == appcashflow.ReceiveMoneyType {
		rows = append(rows, appcashflow.ReceiveMoneyHeader())
	} else {
		rows = append(rows, appcashflow.SpendMoneyHeader())
	}

	nextChequeNumber := 0
	if input.StartingChequeNumber != nil && *input.StartingChequeNumber > 0 {
		nextChequeNumber = *input.StartingChequeNumber
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

		entryRows, warnings, processedCount, buildErr := p.buildCashflowDocumentRows(doc.SourceName, sheet, input, taxAccounts, nextChequeNumber)
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
		if input.CashflowType == appcashflow.ReceiveMoneyType {
			result.Message = "export receive money berhasil"
		} else {
			result.Message = "export spend money berhasil"
		}
	}
	result.FinishedAt = time.Now()
	return result, nil
}

func (p *XLSXCashflowProcessor) buildCashflowDocumentRows(
	sourceName string,
	sheet SpreadsheetSheet,
	input appcashflow.ExportMYOBInput,
	taxAccounts map[string]appcashflow.TaxAccount,
	startChequeNumber int,
) ([][]string, []string, int, error) {
	headers, headerIndex, err := resolveCashflowHeaders(sheet, input.HeaderRowNumber, input)
	if err != nil {
		return nil, nil, 0, err
	}
	_ = headers

	rows := make([][]string, 0)
	warnings := make([]string, 0)
	processed := 0

	for idx, rawRow := range sheet.RawRows {
		rowNumber := SpreadsheetRowNumberAt(sheet, idx)
		if rowNumber <= input.HeaderRowNumber {
			continue
		}

		cellRow := spreadsheetCashflowCellRowAt(sheet.RawCellRows, idx)
		record, rowWarnings, recordErr := parseCashflowRow(rawRow, cellRow, rowNumber, headerIndex)
		if recordErr != nil {
			warnings = append(warnings, fmt.Sprintf("row %d: %s", rowNumber, recordErr.Error()))
			continue
		}
		warnings = append(warnings, rowWarnings...)

		if skipReason, shouldSkip := resolveCashflowSkipReason(record, input); shouldSkip {
			warnings = append(warnings, fmt.Sprintf("row %d: %s", rowNumber, skipReason))
			continue
		}
		record = normalizeCashflowRecord(record)

		currentChequeNumber := 0
		if startChequeNumber > 0 {
			currentChequeNumber = startChequeNumber + processed
		}
		entryRows, txWarnings, txErr := buildCashflowRows(record, input, taxAccounts, currentChequeNumber)
		if txErr != nil {
			warnings = append(warnings, fmt.Sprintf("row %d: %s", rowNumber, txErr.Error()))
			continue
		}
		warnings = append(warnings, prefixWarnings(rowNumber, txWarnings)...)
		rows = append(rows, entryRows...)
		processed++
	}

	if processed == 0 {
		return rows, uniqueStrings(warnings), 0, fmt.Errorf("%s: %s", sourceName, "tidak ada row valid yang dapat dikonversi")
	}
	return rows, uniqueStrings(warnings), processed, nil
}

func isCashflowMYOBAction(actionType string) bool {
	switch strings.TrimSpace(actionType) {
	case cashflowMYOBActionType, cashflowSpendActionType, cashflowRecvActionType:
		return true
	default:
		return false
	}
}

func isCashflowBillActionType(actionType string) bool {
	switch strings.TrimSpace(actionType) {
	case cashflowPayBillsActionType, cashflowReceivePaymentsActionType:
		return true
	default:
		return false
	}
}

func cashflowTypeFromAction(actionType string) appcashflow.Type {
	switch strings.TrimSpace(actionType) {
	case cashflowRecvActionType:
		return appcashflow.ReceiveMoneyType
	default:
		return appcashflow.SpendMoneyType
	}
}

func cashflowSkipReason(cashflowType appcashflow.Type) string {
	if cashflowType == appcashflow.ReceiveMoneyType {
		return "total negatif dilewati untuk mode Receive Money"
	}
	return "total positif dilewati untuk mode Spend Money"
}

func resolveCashflowSkipReason(record cashflowRowRecord, input appcashflow.ExportMYOBInput) (string, bool) {
	if keyword, ok := matchCashflowInformationFilterKeyword(record.Information, input.InformationFilterKeywords); ok {
		return fmt.Sprintf("row dilewati karena keyword filter information %q", keyword), true
	}

	if input.CashflowType == appcashflow.ReceiveMoneyType {
		if record.Total < 0 {
			return cashflowSkipReason(input.CashflowType), true
		}
		return "", false
	}

	if record.Total > 0 {
		return cashflowSkipReason(input.CashflowType), true
	}
	return "", false
}

func matchCashflowInformationFilterKeyword(information string, keywords []string) (string, bool) {
	text := strings.ToLower(strings.TrimSpace(information))
	if text == "" || len(keywords) == 0 {
		return "", false
	}

	for _, keyword := range keywords {
		trimmed := strings.TrimSpace(keyword)
		if trimmed == "" {
			continue
		}
		if strings.Contains(text, strings.ToLower(trimmed)) {
			return trimmed, true
		}
	}

	return "", false
}

func normalizeCashflowRecord(record cashflowRowRecord) cashflowRowRecord {
	record.Total = normalizeNearZero(math.Abs(record.Total))
	record.OtherCost = normalizeNearZero(math.Abs(record.OtherCost))
	record.PP23 = normalizeWithholdingTaxAmount(record.PP23)
	record.PPH15 = normalizeWithholdingTaxAmount(record.PPH15)
	record.PPH21 = normalizeWithholdingTaxAmount(record.PPH21)
	record.PPH23 = normalizeWithholdingTaxAmount(record.PPH23)
	record.PPH42 = normalizeWithholdingTaxAmount(record.PPH42)
	record.Information = uppercaseCashflowText(record.Information)
	record.COA = uppercaseCashflowText(record.COA)
	return record
}

func normalizeWithholdingTaxAmount(value float64) float64 {
	amount := normalizeNearZero(math.Abs(value))
	if amount == 0 {
		return 0
	}
	return -amount
}

func buildCashflowRows(
	record cashflowRowRecord,
	input appcashflow.ExportMYOBInput,
	taxAccounts map[string]appcashflow.TaxAccount,
	entryNumber int,
) ([][]string, []string, error) {
	if input.CashflowType == appcashflow.ReceiveMoneyType {
		tx, warnings, err := buildReceiveMoneyTransaction(record, input, taxAccounts, entryNumber)
		if err != nil {
			return nil, warnings, err
		}
		return appcashflow.BuildReceiveMoneyRows(tx, input.ChequeAccount), warnings, nil
	}

	tx, warnings, err := buildSpendMoneyTransaction(record, input, taxAccounts, entryNumber)
	if err != nil {
		return nil, warnings, err
	}
	return appcashflow.BuildSpendMoneyRows(tx, input.ChequeAccount), warnings, nil
}

func resolveCashflowHeaders(sheet SpreadsheetSheet, headerRowNumber int, input appcashflow.ExportMYOBInput) ([]string, map[string]int, error) {
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
	for _, def := range cashflowFieldDefinitions(input) {
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

func parseCashflowRow(row []string, cellRow []SpreadsheetCell, rowNumber int, fieldIndex map[string]int) (cashflowRowRecord, []string, error) {
	record := cashflowRowRecord{RowNumber: rowNumber}
	record.Information = cashflowCellValue(row, cellRow, fieldIndex, "information")
	record.COA = cashflowCellValue(row, cellRow, fieldIndex, "coa")
	record.Remark = cashflowCellValue(row, cellRow, fieldIndex, "remark")

	dateCell := cashflowTypedCell(row, cellRow, fieldIndex, "date")
	dateValue := SpreadsheetCellText(dateCell)
	if dateValue == "" {
		return record, nil, errors.New("tanggal kosong")
	}
	date, ok := SpreadsheetCellDate(dateCell)
	if !ok {
		return record, nil, fmt.Errorf("tanggal tidak valid: %s", dateValue)
	}
	record.Date = date

	totalCell := cashflowTypedCell(row, cellRow, fieldIndex, "total")
	total, ok := SpreadsheetCellMoney(totalCell)
	if !ok {
		return record, nil, fmt.Errorf("total tidak valid")
	}
	record.Total = total
	if strings.TrimSpace(record.Information) == "" {
		return record, nil, errors.New("keterangan kosong")
	}

	record.OtherCost, _ = parseOptionalCashflowTypedAmount(cashflowTypedCell(row, cellRow, fieldIndex, "otherCost"))
	record.PP23, _ = parseOptionalCashflowTypedAmount(cashflowTypedCell(row, cellRow, fieldIndex, "pp23"))
	record.PPH15, _ = parseOptionalCashflowTypedAmount(cashflowTypedCell(row, cellRow, fieldIndex, "pph15"))
	record.PPH21, _ = parseOptionalCashflowTypedAmount(cashflowTypedCell(row, cellRow, fieldIndex, "pph21"))
	record.PPH23, _ = parseOptionalCashflowTypedAmount(cashflowTypedCell(row, cellRow, fieldIndex, "pph23"))
	record.PPH42, _ = parseOptionalCashflowTypedAmount(cashflowTypedCell(row, cellRow, fieldIndex, "pph42"))
	record.PPN, _ = parseOptionalCashflowTypedAmount(cashflowTypedCell(row, cellRow, fieldIndex, "ppn"))

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
	totalAmount := normalizeNearZero(record.Total)
	otherCostAmount := normalizeNearZero(math.Abs(record.OtherCost))

	for _, taxComponent := range []struct {
		Key    string
		Label  string
		Amount float64
	}{
		{Key: "ppn", Label: "PPN", Amount: record.PPN},
		{Key: "pp23", Label: "PP 23", Amount: record.PP23},
		{Key: "pph15", Label: "PPH 15%", Amount: record.PPH15},
		{Key: "pph21", Label: "PPH 21", Amount: record.PPH21},
		{Key: "pph23", Label: "PPH 23", Amount: record.PPH23},
		{Key: "pph42", Label: "PPH 4 (2)", Amount: record.PPH42},
	} {
		amount := normalizeSpendMoneyTaxAmount(taxComponent.Key, taxComponent.Amount)
		if amount == 0 {
			continue
		}
		accountCode := lookupCashflowTaxAccountCode(taxAccounts, taxComponent.Key)
		if accountCode == "" {
			return appcashflow.SpendMoneyTransaction{}, warnings, fmt.Errorf("lookup account untuk %s tidak ditemukan", taxComponent.Label)
		}
		components = append(components, cashflowComponent{
			AccountCode: accountCode,
			Amount:      amount,
			Allocation:  uppercaseCashflowText(taxComponent.Label),
		})
	}

	if otherCostAmount != 0 {
		accountCode := strings.TrimSpace(input.OtherCostsAccountCode)
		if accountCode == "" {
			return appcashflow.SpendMoneyTransaction{}, warnings, errors.New("account code untuk biaya lain belum diatur")
		}
		components = append(components, cashflowComponent{
			AccountCode: accountCode,
			Amount:      otherCostAmount,
			Allocation:  resolveCashflowAllocationMemo(record, input),
		})
	}

	baseAmount := normalizeNearZero(totalAmount - sumCashflowComponents(components))
	if baseAmount == 0 && len(components) == 0 {
		return appcashflow.SpendMoneyTransaction{}, warnings, errors.New("row tidak memiliki komponen transaksi")
	}

	baseAccount, err := resolveCashflowBaseAccountCode(record, input)
	if err != nil {
		return appcashflow.SpendMoneyTransaction{}, warnings, err
	}
	warnings = append(warnings, baseAccount.Warnings...)

	if baseAmount != 0 {
		components = append([]cashflowComponent{{
			AccountCode: baseAccount.AccountCode,
			Amount:      baseAmount,
			Allocation:  resolveCashflowPrimaryAllocation(record, input, baseAccount.AccountKind),
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
		Memo:         resolveCashflowPrimaryMemo(record),
		Amount:       totalAmount,
		Allocation:   resolveCashflowPrimaryAllocation(record, input, baseAccount.AccountKind),
		Items:        items,
	}, uniqueStrings(warnings), nil
}

func normalizeSpendMoneyTaxAmount(internalKey string, rawAmount float64) float64 {
	amount := normalizeNearZero(rawAmount)
	if amount == 0 {
		return 0
	}
	if internalKey == "ppn" {
		return math.Abs(amount)
	}
	return -math.Abs(amount)
}

func buildReceiveMoneyTransaction(
	record cashflowRowRecord,
	input appcashflow.ExportMYOBInput,
	taxAccounts map[string]appcashflow.TaxAccount,
	entryNumber int,
) (appcashflow.ReceiveMoneyTransaction, []string, error) {
	components := make([]cashflowComponent, 0, 8)
	warnings := make([]string, 0)
	otherCostAmount := normalizeNearZero(record.OtherCost)

	for _, taxComponent := range []struct {
		Key    string
		Label  string
		Amount float64
	}{
		{Key: "ppn", Label: "PPN", Amount: record.PPN},
		{Key: "pp23", Label: "PP 23", Amount: record.PP23},
		{Key: "pph15", Label: "PPH 15%", Amount: record.PPH15},
		{Key: "pph21", Label: "PPH 21", Amount: record.PPH21},
		{Key: "pph23", Label: "PPH 23", Amount: record.PPH23},
		{Key: "pph42", Label: "PPH 4 (2)", Amount: record.PPH42},
	} {
		amount := normalizeNearZero(taxComponent.Amount)
		if amount == 0 {
			continue
		}
		accountCode := lookupCashflowTaxAccountCode(taxAccounts, taxComponent.Key)
		if accountCode == "" {
			return appcashflow.ReceiveMoneyTransaction{}, warnings, fmt.Errorf("lookup account untuk %s tidak ditemukan", taxComponent.Label)
		}
		components = append(components, cashflowComponent{
			AccountCode: accountCode,
			Amount:      amount,
			Allocation:  uppercaseCashflowText(taxComponent.Label),
		})
	}

	if otherCostAmount != 0 {
		accountCode := strings.TrimSpace(input.OtherCostsAccountCode)
		if accountCode == "" {
			return appcashflow.ReceiveMoneyTransaction{}, warnings, errors.New("account code untuk biaya lain belum diatur")
		}
		components = append(components, cashflowComponent{
			AccountCode: accountCode,
			Amount:      otherCostAmount,
			Allocation:  resolveCashflowAllocationMemo(record, input),
		})
	}

	baseAmount := normalizeNearZero(record.Total - sumCashflowComponents(components))
	if baseAmount == 0 && len(components) == 0 {
		return appcashflow.ReceiveMoneyTransaction{}, warnings, errors.New("row tidak memiliki komponen transaksi")
	}

	baseAccount, err := resolveCashflowBaseAccountCode(record, input)
	if err != nil {
		return appcashflow.ReceiveMoneyTransaction{}, warnings, err
	}
	warnings = append(warnings, baseAccount.Warnings...)

	if baseAmount != 0 {
		components = append([]cashflowComponent{{
			AccountCode: baseAccount.AccountCode,
			Amount:      baseAmount,
			Allocation:  resolveCashflowPrimaryAllocation(record, input, baseAccount.AccountKind),
		}}, components...)
	}

	items := make([]appcashflow.ReceiveMoneyTransactionItem, 0, len(components))
	for _, component := range components {
		items = append(items, appcashflow.ReceiveMoneyTransactionItem{
			AccountCode: component.AccountCode,
			Amount:      component.Amount,
			Allocation:  component.Allocation,
		})
	}

	var idNumberPtr *int
	if entryNumber > 0 {
		entryNumberCopy := entryNumber
		idNumberPtr = &entryNumberCopy
	}

	return appcashflow.ReceiveMoneyTransaction{
		IDNumber:   idNumberPtr,
		Date:       appcashflow.EnsureNonZeroTime(record.Date),
		Memo:       resolveCashflowPrimaryMemo(record),
		Amount:     record.Total,
		Allocation: resolveCashflowPrimaryAllocation(record, input, baseAccount.AccountKind),
		Items:      items,
	}, uniqueStrings(warnings), nil
}

func resolveCashflowBaseAccountCode(
	record cashflowRowRecord,
	input appcashflow.ExportMYOBInput,
) (cashflowBaseAccountResolution, error) {
	if input.CashflowFormat == appcashflow.InfluencerFormat {
		lowerInfo := strings.ToLower(strings.TrimSpace(record.Information))
		if strings.Contains(lowerInfo, "admin bank") && strings.TrimSpace(input.DefaultBAccountCode) != "" {
			return cashflowBaseAccountResolution{
				AccountCode: input.DefaultBAccountCode,
				AccountKind: "admin_bank",
			}, nil
		}
		if strings.TrimSpace(input.DefaultIAccountCode) != "" {
			return cashflowBaseAccountResolution{
				AccountCode: input.DefaultIAccountCode,
				AccountKind: "influencer",
			}, nil
		}
		return cashflowBaseAccountResolution{}, errors.New("default influencer/admin bank account code belum diatur")
	}

	coa := strings.TrimSpace(record.COA)
	if coa == "" {
		return cashflowBaseAccountResolution{}, errors.New("chart of accounts kosong, row dilewati")
	}
	if looksLikeAccountCode(coa) {
		return cashflowBaseAccountResolution{
			AccountCode: coa,
			AccountKind: "coa",
		}, nil
	}

	return cashflowBaseAccountResolution{}, fmt.Errorf("chart of accounts %q tidak ditemukan, row dilewati", coa)
}

func resolveCashflowPrimaryAllocation(
	record cashflowRowRecord,
	input appcashflow.ExportMYOBInput,
	accountKind string,
) string {
	base := resolveCashflowPrimaryMemo(record)
	if input.CashflowFormat == appcashflow.InfluencerFormat && accountKind == "influencer" && base != "" {
		return "INFLUENCER " + base
	}
	return base
}

func resolveCashflowPrimaryMemo(record cashflowRowRecord) string {
	return uppercaseCashflowText(record.Information)
}

func resolveCashflowAllocationMemo(record cashflowRowRecord, input appcashflow.ExportMYOBInput) string {
	remark := strings.TrimSpace(record.Remark)
	if remark == "" {
		return resolveCashflowPrimaryAllocation(record, input, "")
	}
	delimiter := strings.TrimSpace(input.RemarkDelimiter)
	if delimiter == "" || !strings.Contains(remark, delimiter) {
		return remark
	}
	parts := strings.SplitN(remark, delimiter, 2)
	if len(parts) != 2 {
		return resolveCashflowPrimaryAllocation(record, input, "")
	}
	value := strings.TrimSpace(parts[1])
	if value == "" {
		return resolveCashflowPrimaryAllocation(record, input, "")
	}
	return value
}

func uppercaseCashflowText(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
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

func lookupCashflowTaxAccountCode(accounts map[string]appcashflow.TaxAccount, internalKey string) string {
	canonicalName := taxcatalog.ResolveCanonicalTaxName(internalKey)
	if canonicalName == "" {
		return ""
	}
	return lookupCashflowAccountCode(accounts, canonicalName)
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

func parseOptionalCashflowTypedAmount(cell SpreadsheetCell) (float64, error) {
	if value, ok := SpreadsheetCellMoney(cell); ok {
		return value, nil
	}
	return parseOptionalCashflowAmount(SpreadsheetCellText(cell))
}

func cashflowFieldDefinitions(input appcashflow.ExportMYOBInput) []cashflowFieldDefinition {
	return []cashflowFieldDefinition{
		{Key: "date", Required: true, Aliases: cashflowAliases(input.MappedField("date"), "tanggal", "date")},
		{Key: "information", Required: true, Aliases: cashflowAliases(input.MappedField("information"), "note", "keterangan", "deskripsi", "information")},
		{Key: "coa", Required: input.CashflowFormat != appcashflow.InfluencerFormat, Aliases: cashflowAliases(input.MappedField("coa"), "coa", "chartofaccount", "chartofaccounts")},
		{Key: "otherCost", Required: false, Aliases: cashflowAliases(input.MappedField("otherCost"), "bylainnya", "biayalainnya", "othercost")},
		{Key: "pp23", Required: false, Aliases: cashflowAliases(input.MappedField("pp23"), "pp23", "pp 23")},
		{Key: "pph15", Required: false, Aliases: cashflowAliases(input.MappedField("pph15"), "pph15%", "pph15")},
		{Key: "pph21", Required: false, Aliases: cashflowAliases(input.MappedField("pph21"), "pph21", "pph 21")},
		{Key: "pph23", Required: false, Aliases: cashflowAliases(input.MappedField("pph23"), "pph23", "pph 23")},
		{Key: "pph42", Required: false, Aliases: cashflowAliases(input.MappedField("pph42"), "pph4(2)", "pph4 2", "pph 4 (2)", "pph 4(2)")},
		{Key: "ppn", Required: false, Aliases: cashflowAliases(input.MappedField("ppn"), "ppn")},
		{Key: "remark", Required: false, Aliases: cashflowAliases(input.MappedField("remark"), "catatan", "remark", "memo")},
		{Key: "total", Required: true, Aliases: cashflowAliases(input.MappedField("total"), "idr", "total", "nominal")},
	}
}

func cashflowAliases(primary string, aliases ...string) []string {
	seen := make(map[string]struct{}, len(aliases)+1)
	out := make([]string, 0, len(aliases)+1)

	appendAlias := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		key := normalizeCashflowHeader(trimmed)
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

func spreadsheetCashflowCellRowAt(rows [][]SpreadsheetCell, idx int) []SpreadsheetCell {
	if idx < 0 || idx >= len(rows) {
		return nil
	}
	return rows[idx]
}

func cashflowTypedCell(row []string, cells []SpreadsheetCell, fieldIndex map[string]int, key string) SpreadsheetCell {
	cell := SpreadsheetCellAt(cells, fieldIndex, key)
	if SpreadsheetCellText(cell) != "" || cell.FloatValue != nil || cell.DateValue != "" || cell.BoolValue != nil {
		return cell
	}
	return SpreadsheetCell{
		Display:     cashflowCell(row, fieldIndex, key),
		StringValue: cashflowCell(row, fieldIndex, key),
		ValueType:   SpreadsheetCellValueTypeString,
	}
}

func cashflowCellValue(row []string, cells []SpreadsheetCell, fieldIndex map[string]int, key string) string {
	return SpreadsheetCellText(cashflowTypedCell(row, cells, fieldIndex, key))
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
