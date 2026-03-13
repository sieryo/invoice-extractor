package document

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
)

type PDFTaxInvoiceProcessor struct {
	fileStore dfile.FileStore
}

func NewPDFTaxInvoiceProcessor(fileStore dfile.FileStore) *PDFTaxInvoiceProcessor {
	return &PDFTaxInvoiceProcessor{
		fileStore: fileStore,
	}
}

func (p *PDFTaxInvoiceProcessor) Type() DocumentType {
	return DocumentTypePDFTaxInvoice
}

func (p *PDFTaxInvoiceProcessor) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
	now := time.Now()
	result := IngestResult{
		BatchID:      req.RequestID,
		CollectionID: req.CollectionID,
		DocumentType: string(req.DocumentType),
		Items:        make([]IngestItemResult, 0, len(req.Sources)),
		StartedAt:    now,
	}

	for _, source := range req.Sources {
		item := IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
		}

		data, err := os.ReadFile(source.TempPath)
		if err != nil {
			item.Status = IngestStatusFailed
			item.Message = "failed to read temporary source"
			item.Errors = []string{err.Error()}
			result.Items = append(result.Items, item)
			result.Failed++
			continue
		}

		rawName := source.SourceID + filepath.Ext(source.OriginalName)
		if err := p.fileStore.WriteFile(ctx, req.CollectionID, rawName, data); err != nil {
			item.Status = IngestStatusFailed
			item.Message = "failed to persist raw file"
			item.Errors = []string{err.Error()}
			result.Items = append(result.Items, item)
			result.Failed++
			continue
		}

		normalizedName := fmt.Sprintf("normalized_%s.json", source.SourceID)
		normalized := map[string]any{
			"source_id":     source.SourceID,
			"source_name":   source.OriginalName,
			"source_sha256": source.SHA256,
			"document_type": string(req.DocumentType),
			"processed_at":  time.Now().UTC(),
			"status":        "ingested",
		}
		normBytes, _ := json.Marshal(normalized)
		if err := p.fileStore.WriteFile(ctx, req.CollectionID, normalizedName, normBytes); err != nil {
			item.Status = IngestStatusFailed
			item.Message = "failed to persist normalized data"
			item.Errors = []string{err.Error()}
			result.Items = append(result.Items, item)
			result.Failed++
			continue
		}

		auditName := fmt.Sprintf("audit_%s.json", source.SourceID)
		auditRef, _ := p.fileStore.SaveAudit(ctx, req.CollectionID, auditName, normBytes)

		item.Status = IngestStatusReady
		item.Message = "tax invoice ingested"
		item.Artifacts = []Artifact{
			{
				Kind:     "raw",
				ObjectID: rawName,
				MimeType: "application/pdf",
				Size:     int64(len(data)),
			},
			{
				Kind:     "normalized",
				ObjectID: normalizedName,
				MimeType: "application/json",
				Size:     int64(len(normBytes)),
			},
		}
		if auditRef != "" {
			item.Artifacts = append(item.Artifacts, Artifact{
				Kind:     "audit",
				ObjectID: auditRef,
				MimeType: "application/json",
			})
		}

		result.Items = append(result.Items, item)
		result.Success++
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
