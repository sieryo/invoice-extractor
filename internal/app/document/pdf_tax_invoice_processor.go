package document

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	taxextract "github.com/sieryo/invoice-extractor/internal/app/invoice/tax/extract"
	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
)

type PDFTaxInvoiceProcessor struct {
	extractor *taxextract.TaxInvoiceExtractService
	fileStore dfile.FileStore
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
	return ActionResult{
		ActionID:   req.ActionID,
		ActionType: req.ActionType,
		Status:     "failed",
		StartedAt:  req.RequestedAt,
		FinishedAt: time.Now(),
	}, fmt.Errorf("%w: action %s for %s", ErrProcessorNotImplemented, req.ActionType, p.Type())
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
