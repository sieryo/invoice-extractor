package action

import (
	"context"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/document"
)

type Repository interface {
	CreateAction(ctx context.Context, action *CollectionAction) error
	FindActionByID(ctx context.Context, actionID string) (*CollectionAction, error)
	FindActionByIdempotency(
		ctx context.Context,
		collectionID string,
		actionType string,
		idempotencyKey string,
	) (*CollectionAction, error)
	ListActions(
		ctx context.Context,
		collectionID string,
		status string,
		limit int,
		offset int,
	) ([]*CollectionAction, error)
	ListPendingActions(ctx context.Context) ([]*CollectionAction, error)
	ListActionItems(ctx context.Context, actionID string) ([]*CollectionActionItem, error)
	ListActionOutputs(ctx context.Context, actionID string) ([]*CollectionActionOutput, error)
	ListActionArtifacts(
		ctx context.Context,
		collectionID string,
		actionType string,
	) ([]*CollectionActionArtifact, error)
	SetActionRunning(ctx context.Context, actionID string, startedAt time.Time) error
	SetActionFinished(
		ctx context.Context,
		actionID string,
		status Status,
		message string,
		total int,
		success int,
		warning int,
		failed int,
		skipped int,
		finishedAt time.Time,
	) error
	AddActionItems(ctx context.Context, items []*CollectionActionItem) error
	AddActionOutputs(ctx context.Context, outputs []*CollectionActionOutput) error
	CreateActionArtifact(ctx context.Context, artifact *CollectionActionArtifact) error
	ListSnapshotDocuments(
		ctx context.Context,
		collectionID string,
		collectionKind document.CollectionKind,
		sourceFormat document.SourceFormat,
		statuses []string,
	) ([]SnapshotDocument, error)
	ListSnapshotDocumentsByIDs(
		ctx context.Context,
		collectionID string,
		collectionKind document.CollectionKind,
		sourceFormat document.SourceFormat,
		documentIDs []string,
	) ([]SnapshotDocument, error)
}
