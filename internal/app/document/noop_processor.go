package document

import (
	"context"
	"fmt"
)

type NoopProcessor struct {
	docType DocumentType
}

func NewNoopProcessor(docType DocumentType) *NoopProcessor {
	return &NoopProcessor{
		docType: docType,
	}
}

func (n *NoopProcessor) Type() DocumentType {
	return n.docType
}

func (n *NoopProcessor) Ingest(_ context.Context, req IngestRequest) (IngestResult, error) {
	return IngestResult{
		BatchID:      req.RequestID,
		CollectionID: req.CollectionID,
		DocumentType: string(req.DocumentType),
		Items:        []IngestItemResult{},
	}, fmt.Errorf("%w: ingest for %s", ErrProcessorNotImplemented, n.docType)
}

func (n *NoopProcessor) RunAction(_ context.Context, req ActionRequest) (ActionResult, error) {
	return ActionResult{
		ActionID:   req.ActionID,
		ActionType: req.ActionType,
		Outputs:    []ActionOutput{},
	}, fmt.Errorf("%w: action for %s", ErrProcessorNotImplemented, n.docType)
}
