package ingest

import "context"

type UploadSessionRepository interface {
	Create(ctx context.Context, session *UploadSession) error
	FindByID(ctx context.Context, id string) (*UploadSession, error)
	ListActive(ctx context.Context) ([]*UploadSession, error)
	ListActiveByCollection(ctx context.Context, collectionID string) ([]*UploadSession, error)
	UpdateStatus(ctx context.Context, id string, status SessionStatus) error
	TouchHeartbeat(ctx context.Context, id string) error
	IncrementUploadedChunk(ctx context.Context, id string, totalChunkCandidate int) error
	IncrementProcessedChunk(ctx context.Context, id string, failedDelta int, duplicateDelta int) error
	SetFinished(ctx context.Context, id string, status SessionStatus) error
}

type UploadChunkRepository interface {
	Create(ctx context.Context, chunk *UploadChunk) error
	FindByID(ctx context.Context, id string) (*UploadChunk, error)
	FindBySessionAndIndex(ctx context.Context, sessionID string, chunkIndex int) (*UploadChunk, error)
	ListBySession(ctx context.Context, sessionID string) ([]*UploadChunk, error)
	UpdateStatus(ctx context.Context, id string, status ChunkStatus, errMsg *string) error
	UpdateJobID(ctx context.Context, id string, jobID string) error
}

type DocumentRepository interface {
	FindByID(ctx context.Context, id string) (*DocumentRecord, error)
	FindActiveByHash(
		ctx context.Context,
		collectionID string,
		documentType string,
		sha256 string,
	) (*DocumentRecord, error)
	Create(ctx context.Context, doc *DocumentRecord) error
	ListByCollection(
		ctx context.Context,
		collectionID string,
		status string,
		limit int,
		offset int,
	) ([]*DocumentRecord, error)
}

type CollectionHistoryRepository interface {
	EnsureUploadHistory(
		ctx context.Context,
		userID string,
		collectionID string,
		sessionID string,
	) (*CollectionHistory, error)
	AddItems(ctx context.Context, items []*CollectionHistoryItem) error
	IncrementSummary(
		ctx context.Context,
		historyID string,
		total int,
		ready int,
		warning int,
		failed int,
		duplicate int,
	) error
	SetStatus(ctx context.Context, historyID string, status string) error
	FindByID(ctx context.Context, historyID string) (*CollectionHistory, error)
	ListByCollection(
		ctx context.Context,
		collectionID string,
		limit int,
		offset int,
	) ([]*CollectionHistory, error)
	ListItems(
		ctx context.Context,
		historyID string,
		status string,
		limit int,
		offset int,
	) ([]*CollectionHistoryItem, error)
}
