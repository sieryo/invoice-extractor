package document

import (
	"context"
	"fmt"
	"time"
)

type XLSXCashflowProcessor struct{}

func NewXLSXCashflowProcessor() *XLSXCashflowProcessor {
	return &XLSXCashflowProcessor{}
}

func (p *XLSXCashflowProcessor) Type() DocumentType {
	return DocumentTypeXLSXCashflow
}

func (p *XLSXCashflowProcessor) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
	now := time.Now()
	result := IngestResult{
		BatchID:      req.RequestID,
		CollectionID: req.CollectionID,
		DocumentType: string(req.DocumentType),
		Items:        make([]IngestItemResult, 0, len(req.Sources)),
		StartedAt:    now,
	}

	for _, source := range req.Sources {
		result.Items = append(result.Items, IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "xlsx cashflow processor is not implemented yet",
			Errors:       []string{ErrProcessorNotImplemented.Error()},
		})
		result.Failed++
	}

	result.Total = len(result.Items)
	result.FinishedAt = time.Now()
	return result, nil
}

func (p *XLSXCashflowProcessor) RunAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	return ActionResult{
		ActionID:   req.ActionID,
		ActionType: req.ActionType,
		Status:     "failed",
		StartedAt:  req.RequestedAt,
		FinishedAt: time.Now(),
	}, fmt.Errorf("%w: action %s for %s", ErrProcessorNotImplemented, req.ActionType, p.Type())
}
