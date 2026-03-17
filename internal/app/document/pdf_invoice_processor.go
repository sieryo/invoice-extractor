package document

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	invoiceextract "github.com/sieryo/invoice-extractor/internal/app/invoice/extract"
	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

type PDFInvoiceProcessor struct {
	extractor      *invoiceextract.InvoiceExtractorService
	invoiceService *invoice.InvoiceService
	fileStore      dfile.FileStore
}

func NewPDFInvoiceProcessor(
	extractor *invoiceextract.InvoiceExtractorService,
	invoiceService *invoice.InvoiceService,
	fileStore dfile.FileStore,
) *PDFInvoiceProcessor {
	return &PDFInvoiceProcessor{
		extractor:      extractor,
		invoiceService: invoiceService,
		fileStore:      fileStore,
	}
}

func (p *PDFInvoiceProcessor) Type() DocumentType {
	return DocumentTypePDFInvoice
}

func (p *PDFInvoiceProcessor) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
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

	batch, err := p.extractor.ExtractBatch(ctx, resolved, nil)
	if err != nil {
		result.FinishedAt = time.Now()
		return result, err
	}

	auditBySourceID := make(map[string]invoiceextract.InvoiceAudit, len(batch.Audits))
	for _, audit := range batch.Audits {
		auditBySourceID[audit.SourceFile.ID] = audit
	}

	seen := make(map[string]bool, len(req.Sources))

	for _, inv := range batch.Invoices {
		if inv == nil || inv.Metadata == nil {
			continue
		}

		sourceID := inv.Metadata.SourceFile.ID
		source, ok := sourceByID[sourceID]
		if !ok {
			continue
		}

		item, err := p.buildSuccessItem(ctx, req.CollectionID, source, inv, auditBySourceID[sourceID])
		if err != nil {
			item = IngestItemResult{
				SourceID:     sourceID,
				OriginalName: source.OriginalName,
				SHA256:       source.SHA256,
				Status:       IngestStatusFailed,
				Message:      "failed to persist normalized invoice",
				Errors:       []string{err.Error()},
			}
		}

		result.Items = append(result.Items, item)
		seen[sourceID] = true
	}

	for _, e := range batch.Errors {
		result.Items = append(result.Items, buildExtractFailedItem(sourceByID, e))
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

func (p *PDFInvoiceProcessor) RunAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	result := ActionResult{
		ActionID:    req.ActionID,
		ActionType:  req.ActionType,
		StartedAt:   req.RequestedAt,
		ItemResults: make([]ActionItemResult, 0, len(req.SnapshotDocs)),
	}

	if strings.TrimSpace(req.ActionType) != "export_faktur_keluaran" {
		result.Status = "failed"
		result.Message = "unsupported action for pdf_invoice"
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("%w: action %s for %s", ErrProcessorNotImplemented, req.ActionType, p.Type())
	}

	if len(req.SnapshotDocs) == 0 {
		result.Status = "failed"
		result.Message = "snapshot is empty"
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("snapshot is empty")
	}

	invoices := make([]*invoice.Invoice, 0, len(req.SnapshotDocs))
	sellerTaxID := ""
	sellerTaxSource := ""
	mismatchFiles := make([]string, 0)
	validationErrors := make([]string, 0)
	hasWarning := false

	for _, doc := range req.SnapshotDocs {
		if strings.TrimSpace(doc.NormalizedRef) == "" {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: normalized artifact is missing", doc.SourceName),
				Error:      "normalized artifact is missing",
			})
			validationErrors = append(validationErrors, doc.SourceName+": missing normalized artifact")
			continue
		}

		inv, err := p.invoiceService.LoadInvoice(ctx, req.CollectionID, doc.NormalizedRef)
		if err != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: failed to read normalized invoice", doc.SourceName),
				Error:      err.Error(),
			})
			validationErrors = append(validationErrors, doc.SourceName+": failed to read normalized invoice")
			continue
		}

		currentTaxID, taxErr := extractSellerTaxID(inv)
		if taxErr != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: %s", doc.SourceName, taxErr.Error()),
				Error:      taxErr.Error(),
			})
			validationErrors = append(validationErrors, doc.SourceName+": "+taxErr.Error())
			continue
		}

		if sellerTaxID == "" {
			sellerTaxID = currentTaxID
			sellerTaxSource = doc.SourceName
		} else if sellerTaxID != currentTaxID {
			msg := fmt.Sprintf(
				"seller tax id mismatch (expected %s from %s, got %s)",
				sellerTaxID,
				sellerTaxSource,
				currentTaxID,
			)
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    msg,
				Error:      "seller tax id mismatch",
			})
			mismatchFiles = append(mismatchFiles, doc.SourceName)
			validationErrors = append(validationErrors, msg)
			continue
		}

		itemStatus := "success"
		itemMessage := "invoice ready for export"
		itemWarnings := []string(nil)
		if inv.Metadata != nil && len(inv.Metadata.Warnings) > 0 {
			itemStatus = "warning"
			itemMessage = "invoice has warnings but eligible for export"
			itemWarnings = append(itemWarnings, inv.Metadata.Warnings...)
			hasWarning = true
		}

		result.ItemResults = append(result.ItemResults, ActionItemResult{
			DocumentID: doc.DocumentID,
			Status:     itemStatus,
			Message:    itemMessage,
			Warnings:   itemWarnings,
		})
		invoices = append(invoices, inv)
	}

	result.Total = len(req.SnapshotDocs)
	result.Success, result.Warning, result.Failed, result.Skipped = summarizeActionItems(result.ItemResults)

	if len(validationErrors) > 0 {
		result.Status = "failed"
		if len(mismatchFiles) > 0 {
			result.Message = fmt.Sprintf(
				"seller tax id mismatch detected on %d file(s): %s",
				len(mismatchFiles),
				strings.Join(mismatchFiles, ", "),
			)
		} else {
			result.Message = "validation failed before export"
		}
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("%s", strings.Join(validationErrors, "; "))
	}

	exported, err := p.invoiceService.Export(ctx, invoices)
	if err != nil {
		result.Status = "failed"
		result.Message = "failed to export e-Faktur file"
		result.FinishedAt = time.Now()
		return result, err
	}

	filename := fmt.Sprintf("%s - %s.xlsx", sellerTaxID, helper.GetIndonesiaDateStr())
	outputRef, err := p.fileStore.SaveArchive(ctx, req.CollectionID, filename, exported)
	if err != nil {
		result.Status = "failed"
		result.Message = "failed to save export output"
		result.FinishedAt = time.Now()
		return result, err
	}

	sum := sha256.Sum256(exported)
	result.Outputs = append(result.Outputs, ActionOutput{
		Kind:      "file",
		Name:      filename,
		ObjectRef: outputRef,
		MimeType:  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		SizeBytes: int64(len(exported)),
		Checksum:  hex.EncodeToString(sum[:]),
	})

	if hasWarning {
		result.Status = "warning"
		result.Message = "export completed with document warnings"
	} else {
		result.Status = "success"
		result.Message = "export completed"
	}
	result.FinishedAt = time.Now()
	return result, nil
}

func (p *PDFInvoiceProcessor) buildSuccessItem(
	ctx context.Context,
	collectionID string,
	source IngestSource,
	inv *invoice.Invoice,
	audit invoiceextract.InvoiceAudit,
) (IngestItemResult, error) {
	normalizedName := fmt.Sprintf("normalized_%s.json", source.SourceID)
	b, err := json.Marshal(inv)
	if err != nil {
		return IngestItemResult{}, err
	}

	tempObj, err := p.fileStore.SaveTemp(ctx, collectionID, normalizedName, b)
	if err != nil {
		return IngestItemResult{}, err
	}

	finalObj, err := p.fileStore.Commit(ctx, tempObj)
	if err != nil {
		return IngestItemResult{}, err
	}

	normalizedRef := finalObj.ID + filepath.Ext(normalizedName)

	var auditRef string
	warnings := []string(nil)
	itemStatus := IngestStatusReady
	if inv.Metadata != nil {
		warnings = inv.Metadata.Warnings
	}

	// Metadata warnings are exposed as warning status.
	if len(warnings) > 0 {
		itemStatus = IngestStatusWarning
	}

	auditName := fmt.Sprintf("audit_%s.json", source.SourceID)
	if auditBytes, err := json.Marshal(audit); err == nil {
		if ref, saveErr := p.fileStore.SaveAudit(ctx, collectionID, auditName, auditBytes); saveErr == nil {
			auditRef = ref
		}
	}

	artifacts := []Artifact{
		{
			Kind:     "normalized",
			ObjectID: normalizedRef,
			MimeType: "application/json",
			Size:     int64(len(b)),
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
		DocumentTag:  deriveInvoiceDocumentTag(inv),
		Status:       itemStatus,
		Message:      "invoice parsed",
		Warnings:     warnings,
		Artifacts:    artifacts,
	}, nil
}

func buildExtractFailedItem(
	sourceByID map[string]IngestSource,
	e shared.FileResultError,
) IngestItemResult {
	src, ok := sourceByID[e.FileID]
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
		SourceID:     src.SourceID,
		OriginalName: src.OriginalName,
		SHA256:       src.SHA256,
		Status:       IngestStatusFailed,
		Message:      "extract failed",
		Errors:       []string{e.Error},
	}
}

func summarizeActionItems(items []ActionItemResult) (success int, warning int, failed int, skipped int) {
	for _, item := range items {
		switch strings.ToLower(strings.TrimSpace(item.Status)) {
		case "success":
			success++
		case "warning":
			warning++
		case "failed":
			failed++
		default:
			skipped++
		}
	}
	return
}

func extractSellerTaxID(inv *invoice.Invoice) (string, error) {
	if inv == nil || inv.Seller == nil || inv.Seller.TaxID == nil {
		return "", fmt.Errorf("seller tax id is missing")
	}
	digits := helper.DigitsOnly(*inv.Seller.TaxID)
	if digits == "" {
		return "", fmt.Errorf("seller tax id is empty")
	}
	return digits, nil
}

func deriveInvoiceDocumentTag(inv *invoice.Invoice) string {
	if inv == nil {
		return ""
	}
	if inv.Seller != nil {
		if sellerName := strings.TrimSpace(inv.Seller.Name); sellerName != "" {
			return sellerName
		}
	}
	if inv.Metadata != nil {
		return strings.TrimSpace(inv.Metadata.TemplateID)
	}
	return ""
}
