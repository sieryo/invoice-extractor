package document

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	invoiceextract "github.com/sieryo/invoice-extractor/internal/app/invoice/extract"
	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
)

type PDFInvoiceProcessor struct {
	extractor *invoiceextract.InvoiceExtractorService
	fileStore dfile.FileStore
}

func NewPDFInvoiceProcessor(
	extractor *invoiceextract.InvoiceExtractorService,
	fileStore dfile.FileStore,
) *PDFInvoiceProcessor {
	return &PDFInvoiceProcessor{
		extractor: extractor,
		fileStore: fileStore,
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
	return ActionResult{
		ActionID:   req.ActionID,
		ActionType: req.ActionType,
		Status:     "failed",
		StartedAt:  req.RequestedAt,
		FinishedAt: time.Now(),
	}, fmt.Errorf("%w: action %s for %s", ErrProcessorNotImplemented, req.ActionType, p.Type())
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
