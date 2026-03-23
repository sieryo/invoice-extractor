package document

import (
	"context"
	"fmt"
)

type NoopProcessor struct {
	key ProcessorKey
}

func NewNoopProcessor(key ProcessorKey) *NoopProcessor {
	return &NoopProcessor{
		key: key,
	}
}

func (n *NoopProcessor) Key() ProcessorKey {
	return n.key
}

func (n *NoopProcessor) Ingest(_ context.Context, req IngestRequest) (IngestResult, error) {
	return IngestResult{
		BatchID:      req.RequestID,
		CollectionID: req.CollectionID,
		DocumentType: string(req.CollectionKind),
		Items:        []IngestItemResult{},
	}, fmt.Errorf("%w: ingest for %s/%s", ErrProcessorNotImplemented, n.key.CollectionKind, n.key.SourceFormat)
}

func (n *NoopProcessor) RunAction(_ context.Context, req ActionRequest) (ActionResult, error) {
	return ActionResult{
		ActionID:   req.ActionID,
		ActionType: req.ActionType,
		Outputs:    []ActionOutput{},
	}, fmt.Errorf("%w: action for %s/%s", ErrProcessorNotImplemented, n.key.CollectionKind, n.key.SourceFormat)
}
