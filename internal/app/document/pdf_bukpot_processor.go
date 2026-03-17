package document

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/bukpot"
	bukpotdomain "github.com/sieryo/invoice-extractor/internal/domain/bukpot"
	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
)

type PDFBukpotProcessor struct {
	docType    DocumentType
	forcedKind bukpotdomain.Kind
	service    *bukpot.Service
	fileStore  dfile.FileStore
}

func NewPDFBukpotProcessor(
	docType DocumentType,
	service *bukpot.Service,
	fileStore dfile.FileStore,
) *PDFBukpotProcessor {
	return &PDFBukpotProcessor{
		docType:    docType,
		forcedKind: mapBukpotKindFromDocType(docType),
		service:    service,
		fileStore:  fileStore,
	}
}

func (p *PDFBukpotProcessor) Type() DocumentType {
	return p.docType
}

func (p *PDFBukpotProcessor) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
	startedAt := time.Now()
	result := IngestResult{
		BatchID:      req.RequestID,
		CollectionID: req.CollectionID,
		DocumentType: string(req.DocumentType),
		Items:        make([]IngestItemResult, 0, len(req.Sources)),
		StartedAt:    startedAt,
	}

	if !p.forcedKind.IsValid() {
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("%w: unsupported bukpot document type %s", ErrProcessorNotImplemented, p.docType)
	}

	for _, source := range req.Sources {
		item := p.ingestSource(ctx, req, source)
		result.Items = append(result.Items, item)
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

func (p *PDFBukpotProcessor) RunAction(_ context.Context, req ActionRequest) (ActionResult, error) {
	return ActionResult{
		ActionID:    req.ActionID,
		ActionType:  req.ActionType,
		Outputs:     []ActionOutput{},
		ItemResults: []ActionItemResult{},
	}, fmt.Errorf("%w: action for %s", ErrProcessorNotImplemented, p.docType)
}

func (p *PDFBukpotProcessor) ingestSource(
	ctx context.Context,
	req IngestRequest,
	source IngestSource,
) IngestItemResult {
	forcedKind := p.forcedKind
	parsed, err := p.service.ParseFile(ctx, bukpotdomain.FileInput{
		UploadIndex: source.SourceOrder,
		SourceName:  source.OriginalName,
		Path:        source.TempPath,
	}, &forcedKind)
	if err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse bukpot",
			Errors:       []string{err.Error()},
		}
	}

	if parsed == nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse bukpot",
			Errors:       []string{"empty parser response"},
		}
	}

	if parsed.Error != nil && strings.TrimSpace(*parsed.Error) != "" {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse bukpot",
			Errors:       []string{strings.TrimSpace(*parsed.Error)},
		}
	}

	if parsed.Data == nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse bukpot",
			Errors:       []string{"parsed document is empty"},
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
				Message:      "failed to read bukpot source file",
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
				Message:      "failed to persist bukpot raw file",
				Errors:       []string{writeErr.Error()},
			}
		}

		artifacts = append(artifacts, Artifact{
			Kind:     "raw",
			ObjectID: rawName,
			MimeType: "application/pdf",
			Size:     int64(len(rawBytes)),
		})
	}

	normalizedPayload := map[string]any{
		"source_id":     source.SourceID,
		"source_name":   source.OriginalName,
		"source_sha256": source.SHA256,
		"document_type": string(req.DocumentType),
		"document_tag":  strings.TrimSpace(parsed.Data.DocumentTag),
		"bukpot":        parsed.Data,
		"processed_at":  time.Now().UTC(),
	}
	normalizedBytes, err := json.Marshal(normalizedPayload)
	if err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to encode bukpot normalized payload",
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
			Message:      "failed to persist bukpot normalized payload",
			Errors:       []string{err.Error()},
		}
	}

	artifacts = append(artifacts, Artifact{
		Kind:     "normalized",
		ObjectID: normalizedName,
		MimeType: "application/json",
		Size:     int64(len(normalizedBytes)),
	})

	return IngestItemResult{
		SourceID:     source.SourceID,
		OriginalName: source.OriginalName,
		SHA256:       source.SHA256,
		DocumentTag:  strings.TrimSpace(parsed.Data.DocumentTag),
		Status:       IngestStatusReady,
		Message:      "bukpot parsed",
		Artifacts:    artifacts,
	}
}

func mapBukpotKindFromDocType(docType DocumentType) bukpotdomain.Kind {
	switch docType {
	case DocumentTypePDFBukpotBPPU:
		return bukpotdomain.KindBPPU
	case DocumentTypePDFBukpotBP21:
		return bukpotdomain.KindBP21
	case DocumentTypePDFBukpotBPA1:
		return bukpotdomain.KindBPA1
	default:
		return ""
	}
}
