package document

import (
	"archive/zip"
	"bytes"
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

	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax"
	taxextract "github.com/sieryo/invoice-extractor/internal/app/invoice/tax/extract"
	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
)

type PDFTaxInvoiceProcessor struct {
	extractor *taxextract.TaxInvoiceExtractService
	fileStore dfile.FileStore
}

const (
	taxInvoiceRenameActionType       = "rename_tax_invoice"
	defaultTaxInvoiceNameTemplate    = "{{references}} - {{buyerName}}"
	taxInvoiceDefaultFallbackName    = "tax-invoice"
	taxInvoiceNormalizedReadFallback = "invalid normalized tax invoice payload"
)

var taxInvoiceTemplateTokenRegex = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)
var taxInvoiceInvalidFilenameCharRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

type taxInvoiceRenameParams struct {
	FilenameTemplate string `json:"filenameTemplate"`
}

type taxInvoiceNormalizedPayload struct {
	Invoice  *tax.TaxInvoice `json:"invoice"`
	Warnings []string        `json:"warnings,omitempty"`
}

type renamedTaxInvoiceFile struct {
	Name string
	Data []byte
}

func NewPDFTaxInvoiceProcessor(
	extractor *taxextract.TaxInvoiceExtractService,
	fileStore dfile.FileStore,
) *PDFTaxInvoiceProcessor {
	return &PDFTaxInvoiceProcessor{
		extractor: extractor,
		fileStore: fileStore,
	}
}

func (p *PDFTaxInvoiceProcessor) Type() DocumentType {
	return DocumentTypePDFTaxInvoice
}

func (p *PDFTaxInvoiceProcessor) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
	startedAt := time.Now()
	result := IngestResult{
		BatchID:      req.RequestID,
		CollectionID: req.CollectionID,
		DocumentType: string(req.DocumentType),
		Items:        make([]IngestItemResult, 0, len(req.Sources)),
		StartedAt:    startedAt,
	}

	sourceByID := make(map[string]IngestSource, len(req.Sources))
	resolved := make([]dfile.ResolvedFile, 0, len(req.Sources))
	for _, source := range req.Sources {
		sourceByID[source.SourceID] = source
		resolved = append(resolved, dfile.ResolvedFile{
			FileRef: dfile.FileRef{
				ID:           source.SourceID,
				CollectionID: req.CollectionID,
				Name:         source.OriginalName,
			},
			Path: source.TempPath,
		})
	}

	batch, err := p.extractor.ExtractBatch(ctx, resolved)
	if err != nil {
		result.FinishedAt = time.Now()
		return result, err
	}

	seen := make(map[string]bool, len(req.Sources))
	for _, parsed := range batch.Items {
		source, ok := sourceByID[parsed.SourceFile.ID]
		if !ok {
			continue
		}

		item, buildErr := p.buildSuccessItem(ctx, req.CollectionID, source, parsed)
		if buildErr != nil {
			item = IngestItemResult{
				SourceID:     source.SourceID,
				OriginalName: source.OriginalName,
				SHA256:       source.SHA256,
				Status:       IngestStatusFailed,
				Message:      "failed to persist tax invoice artifacts",
				Errors:       []string{buildErr.Error()},
			}
		}

		result.Items = append(result.Items, item)
		seen[source.SourceID] = true
	}

	for _, e := range batch.Errors {
		result.Items = append(result.Items, buildTaxExtractFailedItem(sourceByID, e))
		seen[e.FileID] = true
	}

	for _, source := range req.Sources {
		if seen[source.SourceID] {
			continue
		}

		result.Items = append(result.Items, IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "processor did not return output for source",
			Errors:       []string{"missing result item"},
		})
	}

	for _, item := range result.Items {
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

func (p *PDFTaxInvoiceProcessor) RunAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	result := ActionResult{
		ActionID:    req.ActionID,
		ActionType:  req.ActionType,
		StartedAt:   req.RequestedAt,
		ItemResults: make([]ActionItemResult, 0, len(req.SnapshotDocs)),
		Outputs:     make([]ActionOutput, 0, 1),
	}

	if strings.TrimSpace(req.ActionType) != taxInvoiceRenameActionType {
		result.Status = "failed"
		result.Message = "unsupported action for pdf_tax_invoice"
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("%w: action %s for %s", ErrProcessorNotImplemented, req.ActionType, p.Type())
	}

	if len(req.SnapshotDocs) == 0 {
		result.Status = "failed"
		result.Message = "snapshot is empty"
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("snapshot is empty")
	}

	params, err := parseTaxInvoiceRenameParams(req.Params)
	if err != nil {
		result.Status = "failed"
		result.Message = "invalid action params"
		result.FinishedAt = time.Now()
		return result, err
	}

	hasWarning := false
	outputNameSet := make(map[string]struct{}, len(req.SnapshotDocs))
	renamedFiles := make([]renamedTaxInvoiceFile, 0, len(req.SnapshotDocs))

	for _, doc := range req.SnapshotDocs {
		if strings.TrimSpace(doc.NormalizedRef) == "" {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: normalized artifact is missing", doc.SourceName),
				Error:      "normalized artifact is missing",
			})
			continue
		}
		if strings.TrimSpace(doc.RawRef) == "" {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: raw artifact is missing", doc.SourceName),
				Error:      "raw artifact is missing",
			})
			continue
		}

		payload, loadErr := p.loadTaxInvoiceNormalizedPayload(ctx, req.CollectionID, doc.NormalizedRef)
		if loadErr != nil || payload.Invoice == nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: failed to read normalized tax invoice", doc.SourceName),
				Error:      errorText(loadErr, taxInvoiceNormalizedReadFallback),
			})
			continue
		}

		filename, templateWarnings := renderTaxInvoiceFilename(
			params.FilenameTemplate,
			payload.Invoice,
			doc.SourceName,
		)
		filename = ensureUniqueArchiveFilename(filename, outputNameSet)

		rawBytes, readErr := p.fileStore.Read(ctx, req.CollectionID, doc.RawRef)
		if readErr != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: failed to read raw source file", doc.SourceName),
				Error:      readErr.Error(),
			})
			continue
		}

		itemWarnings := uniqueStrings(append(payload.Warnings, templateWarnings...))
		itemStatus := "success"
		itemMessage := fmt.Sprintf("renamed to %s", filename)
		if len(itemWarnings) > 0 {
			itemStatus = "warning"
			itemMessage = fmt.Sprintf("renamed with warnings to %s", filename)
			hasWarning = true
		}

		result.ItemResults = append(result.ItemResults, ActionItemResult{
			DocumentID: doc.DocumentID,
			Status:     itemStatus,
			Message:    itemMessage,
			Warnings:   itemWarnings,
		})

		renamedFiles = append(renamedFiles, renamedTaxInvoiceFile{
			Name: filename,
			Data: rawBytes,
		})
	}

	result.Total = len(req.SnapshotDocs)
	result.Success, result.Warning, result.Failed, result.Skipped = summarizeActionItems(result.ItemResults)

	if result.Success == 0 && result.Warning == 0 {
		result.Status = "failed"
		result.Message = "rename failed for all selected documents"
		result.FinishedAt = time.Now()
		return result, errors.New("no tax invoice document was renamed successfully")
	}

	zipBytes, zipErr := buildTaxInvoiceZipArchive(renamedFiles)
	if zipErr != nil {
		result.Status = "failed"
		result.Message = "failed to build zip output"
		result.FinishedAt = time.Now()
		return result, zipErr
	}

	zipName := fmt.Sprintf("renamed_tax_invoices_%s.zip", time.Now().Format("20060102_150405"))
	outputRef, saveErr := p.fileStore.SaveArchive(ctx, req.CollectionID, zipName, zipBytes)
	if saveErr != nil {
		result.Status = "failed"
		result.Message = "failed to save zip output"
		result.FinishedAt = time.Now()
		return result, saveErr
	}

	zipSum := sha256.Sum256(zipBytes)
	result.Outputs = append(result.Outputs, ActionOutput{
		Kind:      "file",
		Name:      zipName,
		ObjectRef: outputRef,
		MimeType:  "application/zip",
		SizeBytes: int64(len(zipBytes)),
		Checksum:  hex.EncodeToString(zipSum[:]),
	})

	if result.Failed > 0 {
		result.Status = "partial"
		result.Message = fmt.Sprintf(
			"rename completed with partial results (%d success, %d warning, %d failed)",
			result.Success,
			result.Warning,
			result.Failed,
		)
	} else if hasWarning {
		result.Status = "warning"
		result.Message = "rename completed with warnings"
	} else {
		result.Status = "success"
		result.Message = "rename completed"
	}

	result.FinishedAt = time.Now()
	return result, nil
}

func (p *PDFTaxInvoiceProcessor) buildSuccessItem(
	ctx context.Context,
	collectionID string,
	source IngestSource,
	parsed taxextract.BatchExtractItem,
) (IngestItemResult, error) {
	rawBytes, err := os.ReadFile(source.TempPath)
	if err != nil {
		return IngestItemResult{}, err
	}

	rawName := source.SourceID + filepath.Ext(source.OriginalName)
	if err := p.fileStore.WriteFile(ctx, collectionID, rawName, rawBytes); err != nil {
		return IngestItemResult{}, err
	}

	normalizedName := fmt.Sprintf("normalized_%s.json", source.SourceID)
	normalizedPayload := map[string]any{
		"source_id":       source.SourceID,
		"source_name":     source.OriginalName,
		"source_sha256":   source.SHA256,
		"document_type":   string(DocumentTypePDFTaxInvoice),
		"normalized_text": parsed.NormalizedText,
		"invoice":         parsed.Invoice,
		"warnings":        parsed.Warnings,
		"processed_at":    time.Now().UTC(),
	}
	normalizedBytes, err := json.Marshal(normalizedPayload)
	if err != nil {
		return IngestItemResult{}, err
	}
	if err := p.fileStore.WriteFile(ctx, collectionID, normalizedName, normalizedBytes); err != nil {
		return IngestItemResult{}, err
	}

	auditPayload := map[string]any{
		"source_file": parsed.SourceFile,
		"warnings":    parsed.Warnings,
		"invoice":     parsed.Invoice,
	}
	auditBytes, _ := json.Marshal(auditPayload)
	auditName := fmt.Sprintf("audit_%s.json", source.SourceID)
	auditRef, _ := p.fileStore.SaveAudit(ctx, collectionID, auditName, auditBytes)

	status := IngestStatusReady
	if len(parsed.Warnings) > 0 {
		status = IngestStatusWarning
	}

	artifacts := []Artifact{
		{
			Kind:     "raw",
			ObjectID: rawName,
			MimeType: "application/pdf",
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
			Size:     int64(len(auditBytes)),
		})
	}

	return IngestItemResult{
		SourceID:     source.SourceID,
		OriginalName: source.OriginalName,
		SHA256:       source.SHA256,
		DocumentTag:  deriveTaxInvoiceDocumentTag(parsed.Invoice),
		Status:       status,
		Message:      "tax invoice parsed",
		Warnings:     parsed.Warnings,
		Artifacts:    artifacts,
	}, nil
}

func buildTaxExtractFailedItem(
	sourceByID map[string]IngestSource,
	e shared.FileResultError,
) IngestItemResult {
	source, ok := sourceByID[e.FileID]
	if !ok {
		return IngestItemResult{
			SourceID:     e.FileID,
			OriginalName: e.FileName,
			Status:       IngestStatusFailed,
			Message:      "extract failed",
			Errors:       []string{e.Error},
		}
	}

	return IngestItemResult{
		SourceID:     source.SourceID,
		OriginalName: source.OriginalName,
		SHA256:       source.SHA256,
		Status:       IngestStatusFailed,
		Message:      "extract failed",
		Errors:       []string{e.Error},
	}
}

func parseTaxInvoiceRenameParams(raw json.RawMessage) (taxInvoiceRenameParams, error) {
	params := taxInvoiceRenameParams{
		FilenameTemplate: defaultTaxInvoiceNameTemplate,
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		return params, nil
	}

	if err := json.Unmarshal(raw, &params); err != nil {
		return params, fmt.Errorf("invalid rename params: %w", err)
	}

	params.FilenameTemplate = strings.TrimSpace(params.FilenameTemplate)
	if params.FilenameTemplate == "" {
		params.FilenameTemplate = defaultTaxInvoiceNameTemplate
	}
	return params, nil
}

func (p *PDFTaxInvoiceProcessor) loadTaxInvoiceNormalizedPayload(
	ctx context.Context,
	collectionID string,
	ref string,
) (*taxInvoiceNormalizedPayload, error) {
	b, err := p.fileStore.Read(ctx, collectionID, ref)
	if err != nil {
		return nil, err
	}

	var payload taxInvoiceNormalizedPayload
	if err := json.Unmarshal(b, &payload); err == nil && payload.Invoice != nil {
		payload.Warnings = uniqueStrings(payload.Warnings)
		return &payload, nil
	}

	var inv tax.TaxInvoice
	if err := json.Unmarshal(b, &inv); err == nil {
		if strings.TrimSpace(inv.InvoiceNumber) == "" &&
			strings.TrimSpace(inv.References) == "" &&
			strings.TrimSpace(inv.BuyerName) == "" &&
			strings.TrimSpace(inv.Number) == "" {
			return nil, errors.New(taxInvoiceNormalizedReadFallback)
		}
		return &taxInvoiceNormalizedPayload{
			Invoice:  &inv,
			Warnings: nil,
		}, nil
	}

	return nil, errors.New(taxInvoiceNormalizedReadFallback)
}

func renderTaxInvoiceFilename(
	filenameTemplate string,
	inv *tax.TaxInvoice,
	sourceName string,
) (string, []string) {
	template := strings.TrimSpace(filenameTemplate)
	if template == "" {
		template = defaultTaxInvoiceNameTemplate
	}

	values := buildTaxInvoiceTemplateMap(inv, sourceName)
	missingTokens := make([]string, 0, 2)

	rendered := taxInvoiceTemplateTokenRegex.ReplaceAllStringFunc(template, func(match string) string {
		sub := taxInvoiceTemplateTokenRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return ""
		}

		token := sub[1]
		value, ok := lookupTaxInvoiceTemplateValue(values, token)
		if !ok {
			missingTokens = append(missingTokens, token)
			return ""
		}
		return value
	})

	cleanName := sanitizeTaxInvoiceFilename(rendered)
	warnings := make([]string, 0, 2)
	if cleanName == "" {
		fallbackName := sanitizeTaxInvoiceFilename(values["references"] + " - " + values["buyername"])
		if fallbackName == "" {
			fallbackName = sanitizeTaxInvoiceFilename(values["sourcename"])
		}
		if fallbackName == "" {
			fallbackName = taxInvoiceDefaultFallbackName
		}
		cleanName = fallbackName
		warnings = append(warnings, "template resolved to empty filename, fallback name was used")
	}

	if len(missingTokens) > 0 {
		warnings = append(
			warnings,
			fmt.Sprintf(
				"unknown template placeholder(s): %s",
				strings.Join(uniqueStrings(missingTokens), ", "),
			),
		)
	}

	if !strings.HasSuffix(strings.ToLower(cleanName), ".pdf") {
		cleanName += ".pdf"
	}

	return cleanName, uniqueStrings(warnings)
}

func buildTaxInvoiceTemplateMap(
	inv *tax.TaxInvoice,
	sourceName string,
) map[string]string {
	values := map[string]string{
		"sourcename": sanitizeTaxInvoiceFilename(strings.TrimSuffix(sourceName, filepath.Ext(sourceName))),
	}
	if inv == nil {
		return values
	}

	invoiceDate := ""
	if !inv.InvoiceDate.IsZero() {
		invoiceDate = inv.InvoiceDate.Format("2006-01-02")
	}

	values["reference"] = strings.TrimSpace(inv.References)
	values["references"] = strings.TrimSpace(inv.References)
	values["invoicenumber"] = strings.TrimSpace(inv.InvoiceNumber)
	values["buyername"] = strings.TrimSpace(inv.BuyerName)
	values["buyernpwp"] = strings.TrimSpace(inv.BuyerNPWP)
	values["sellername"] = strings.TrimSpace(inv.SellerName)
	values["sellernpwp"] = strings.TrimSpace(inv.SellerNPWP)
	values["invoicedate"] = invoiceDate
	values["documenttag"] = deriveTaxInvoiceDocumentTag(inv)

	if inv.Buyer != nil {
		if values["buyername"] == "" {
			values["buyername"] = strings.TrimSpace(inv.Buyer.Name)
		}
		if inv.Buyer.TaxID != nil && values["buyernpwp"] == "" {
			values["buyernpwp"] = strings.TrimSpace(*inv.Buyer.TaxID)
		}
	}

	if inv.Number != "" && values["references"] == "" {
		values["references"] = strings.TrimSpace(inv.Number)
		values["reference"] = strings.TrimSpace(inv.Number)
	}

	return values
}

func lookupTaxInvoiceTemplateValue(values map[string]string, rawToken string) (string, bool) {
	token := normalizeTaxInvoiceTemplateToken(rawToken)
	value, ok := values[token]
	if !ok {
		return "", false
	}
	return strings.TrimSpace(value), true
}

func normalizeTaxInvoiceTemplateToken(token string) string {
	normalized := strings.ToLower(strings.TrimSpace(token))
	normalized = strings.ReplaceAll(normalized, "_", "")
	return normalized
}

func sanitizeTaxInvoiceFilename(raw string) string {
	value := strings.TrimSpace(raw)
	value = taxInvoiceInvalidFilenameCharRegex.ReplaceAllString(value, " ")
	value = strings.ReplaceAll(value, "..", " ")
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Trim(value, ".-_ ")
	return value
}

func ensureUniqueArchiveFilename(name string, used map[string]struct{}) string {
	candidate := strings.TrimSpace(name)
	if candidate == "" {
		candidate = taxInvoiceDefaultFallbackName + ".pdf"
	}

	ext := filepath.Ext(candidate)
	base := strings.TrimSuffix(candidate, ext)
	if ext == "" {
		ext = ".pdf"
	}

	lowered := strings.ToLower(candidate)
	if _, exists := used[lowered]; !exists {
		used[lowered] = struct{}{}
		return candidate
	}

	for i := 2; ; i++ {
		next := fmt.Sprintf("%s (%d)%s", base, i, ext)
		key := strings.ToLower(next)
		if _, exists := used[key]; exists {
			continue
		}
		used[key] = struct{}{}
		return next
	}
}

func uniqueStrings(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	out := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, raw := range input {
		value := strings.TrimSpace(raw)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func deriveTaxInvoiceDocumentTag(inv *tax.TaxInvoice) string {
	if inv == nil {
		return ""
	}
	if sellerName := strings.TrimSpace(inv.SellerName); sellerName != "" {
		return sellerName
	}
	if inv.Buyer != nil {
		if buyerName := strings.TrimSpace(inv.Buyer.Name); buyerName != "" {
			return buyerName
		}
	}
	if buyerName := strings.TrimSpace(inv.BuyerName); buyerName != "" {
		return buyerName
	}
	return strings.TrimSpace(inv.References)
}

func errorText(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	msg := strings.TrimSpace(err.Error())
	if msg == "" {
		return fallback
	}
	return msg
}

func buildTaxInvoiceZipArchive(files []renamedTaxInvoiceFile) ([]byte, error) {
	if len(files) == 0 {
		return nil, errors.New("no renamed tax invoice files to archive")
	}

	buf := bytes.NewBuffer(nil)
	zipWriter := zip.NewWriter(buf)

	for _, item := range files {
		entryName := strings.TrimSpace(item.Name)
		if entryName == "" {
			entryName = taxInvoiceDefaultFallbackName + ".pdf"
		}

		entry, err := zipWriter.Create(filepath.Base(entryName))
		if err != nil {
			_ = zipWriter.Close()
			return nil, err
		}
		if _, err := entry.Write(item.Data); err != nil {
			_ = zipWriter.Close()
			return nil, err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
