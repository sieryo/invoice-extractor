package document

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
)

type cashflowNormalizedPayload struct {
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

type XLSXCashflowProcessor struct {
	fileStore   dfile.FileStore
	taxAccounts CashflowTaxAccountProvider
}

type CashflowTaxAccountProvider interface {
	Status(profileID string) appcashflow.TaxAccountStatus
	Load(profileID string) (map[string]appcashflow.TaxAccount, error)
}

func NewXLSXCashflowProcessor(
	fileStore dfile.FileStore,
	taxAccounts CashflowTaxAccountProvider,
) *XLSXCashflowProcessor {
	return &XLSXCashflowProcessor{
		fileStore:   fileStore,
		taxAccounts: taxAccounts,
	}
}

func (p *XLSXCashflowProcessor) Key() ProcessorKey {
	return ProcessorKey{
		CollectionKind: CollectionKindCashflowImport,
		SourceFormat:   SourceFormatXLSX,
	}
}

func (p *XLSXCashflowProcessor) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
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

func (p *XLSXCashflowProcessor) ingestSource(
	ctx context.Context,
	req IngestRequest,
	source IngestSource,
) IngestItemResult {
	workbook, warnings, err := ExtractSpreadsheetWorkbook(source.TempPath, source.OriginalName)
	if err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse cashflow workbook",
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
				Message:      "failed to read cashflow source file",
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
				Message:      "failed to persist cashflow source file",
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

	documentTag := deriveCashflowDocumentTag(workbook, source.OriginalName)
	payload := cashflowNormalizedPayload{
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
			Message:      "failed to encode cashflow normalized payload",
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
			Message:      "failed to persist cashflow normalized payload",
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
		Message:      "cashflow workbook parsed",
		Warnings:     warnings,
		Artifacts:    artifacts,
	}
}

func deriveCashflowDocumentTag(workbook SpreadsheetWorkbook, sourceName string) string {
	if strings.TrimSpace(workbook.PrimarySheet) != "" {
		return strings.TrimSpace(workbook.PrimarySheet)
	}
	return strings.TrimSpace(strings.TrimSuffix(sourceName, filepath.Ext(sourceName)))
}
