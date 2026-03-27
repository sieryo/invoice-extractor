package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/document"
	"github.com/sieryo/invoice-extractor/internal/domain/collection"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type IngestService struct {
	sessionRepo    UploadSessionRepository
	chunkRepo      UploadChunkRepository
	documentRepo   DocumentRepository
	historyRepo    CollectionHistoryRepository
	collectionRepo collection.Repository
	fileStore      file.FileStore
	processors     *document.Registry

	queue   chan string
	workers int
	once    sync.Once
}

type uploadChunkPayload struct {
	Sources           []document.IngestSource       `json:"sources"`
	PendingDuplicates []pendingDuplicatePayloadItem `json:"pendingDuplicates,omitempty"`
}

type pendingDuplicatePayloadItem struct {
	Source             document.IngestSource `json:"source"`
	ExistingDocumentID string                `json:"existingDocumentId"`
	ExistingSourceName string                `json:"existingSourceName"`
	ExistingStatus     string                `json:"existingStatus,omitempty"`
}

const (
	sessionHeartbeatTimeout = 2 * time.Minute
	sessionSweepInterval    = 30 * time.Second
)

func NewIngestService(
	sessionRepo UploadSessionRepository,
	chunkRepo UploadChunkRepository,
	documentRepo DocumentRepository,
	historyRepo CollectionHistoryRepository,
	collectionRepo collection.Repository,
	fileStore file.FileStore,
	processors *document.Registry,
	workers int,
) *IngestService {
	if workers < 1 {
		workers = 1
	}

	return &IngestService{
		sessionRepo:    sessionRepo,
		chunkRepo:      chunkRepo,
		documentRepo:   documentRepo,
		historyRepo:    historyRepo,
		collectionRepo: collectionRepo,
		fileStore:      fileStore,
		processors:     processors,
		queue:          make(chan string, 64),
		workers:        workers,
	}
}

func (s *IngestService) StartPool(ctx context.Context) {
	s.once.Do(func() {
		for i := 0; i < s.workers; i++ {
			go s.worker(ctx)
		}
		_ = s.recoverActiveSessions(ctx, true)
		go s.reconcileLoop(ctx)
	})
}

func (s *IngestService) CreateSession(
	ctx context.Context,
	userID string,
	collectionID string,
	clientSessionKey *string,
) (*UploadSession, error) {
	coll, err := s.collectionRepo.FindByID(ctx, collectionID)
	if err != nil {
		return nil, collection.ErrCollectionNotFound
	}

	if coll.UserID != userID {
		return nil, collection.ErrCollectionNotFound
	}

	if !coll.IsCollection() || coll.CollectionKind == nil {
		return nil, collection.ErrInvalidNodeType
	}
	if !document.IsCollectionKindEnabled(document.CollectionKind(*coll.CollectionKind)) {
		return nil, collection.ErrInvalidCollectionKind
	}
	if coll.IsFrozen() {
		return nil, collection.ErrCollectionFrozen
	}

	now := time.Now()
	session := &UploadSession{
		ID:               uuid.NewString(),
		UserID:           userID,
		CollectionID:     collectionID,
		CollectionKind:   document.CollectionKind(*coll.CollectionKind),
		SourceFormat:     document.ResolveCollectionSourceFormat(document.CollectionKind(*coll.CollectionKind)),
		Status:           SessionStatusReceiving,
		TotalChunks:      0,
		UploadedChunks:   0,
		ProcessedChunks:  0,
		FailedChunks:     0,
		DuplicateChunks:  0,
		LastHeartbeatAt:  &now,
		StartedAt:        now,
		ClientSessionKey: clientSessionKey,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, err
	}

	if err := s.collectionRepo.UpdatePhase(ctx, collectionID, collection.PhaseUploading); err != nil {
		return nil, err
	}

	if _, err := s.historyRepo.EnsureUploadHistory(ctx, userID, collectionID, session.ID); err != nil {
		return nil, err
	}

	return session, nil
}

func (s *IngestService) UploadChunk(
	ctx context.Context,
	sessionID string,
	input ChunkUploadInput,
	files []SourceUploadFile,
	sourceOrderStart int,
) (*UploadChunk, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCollectionWritableByID(ctx, session.CollectionID); err != nil {
		return nil, err
	}

	if !isWritableSessionStatus(session.Status) {
		return nil, ErrSessionNotWritable
	}

	existing, err := s.chunkRepo.FindBySessionAndIndex(ctx, sessionID, input.ChunkIndex)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrChunkNotFound) {
		return nil, err
	}

	sources, payloadSize, err := s.prepareChunkSources(ctx, session, files, sourceOrderStart)
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("chunk has no files")
	}

	payloadJSON, err := json.Marshal(uploadChunkPayload{Sources: sources})
	if err != nil {
		return nil, err
	}

	now := time.Now()
	chunk := &UploadChunk{
		ID:              uuid.NewString(),
		SessionID:       session.ID,
		ChunkIndex:      input.ChunkIndex,
		Status:          ChunkStatusUploaded,
		IdempotencyKey:  input.IdempotencyKey,
		RequestChecksum: input.RequestChecksum,
		FileCount:       len(sources),
		SizeBytes:       payloadSize,
		PayloadJSON:     payloadJSON,
		CreatedAt:       now,
	}

	if err := s.chunkRepo.Create(ctx, chunk); err != nil {
		return nil, err
	}

	totalCandidate := input.ChunkIndex + 1
	if err := s.sessionRepo.IncrementUploadedChunk(ctx, session.ID, totalCandidate); err != nil {
		return nil, err
	}
	if err := s.sessionRepo.UpdateStatus(ctx, session.ID, SessionStatusReceiving); err != nil {
		return nil, err
	}

	if err := s.enqueueChunk(ctx, chunk.ID); err != nil {
		return nil, err
	}

	return chunk, nil
}

func (s *IngestService) FinalizeSession(ctx context.Context, sessionID string) (*UploadSession, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCollectionWritableByID(ctx, session.CollectionID); err != nil {
		return nil, err
	}

	if !isWritableSessionStatus(session.Status) && session.Status != SessionStatusFinalized {
		return nil, ErrSessionNotWritable
	}

	if err := s.sessionRepo.UpdateStatus(ctx, sessionID, SessionStatusFinalized); err != nil {
		return nil, err
	}

	updated, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if err := s.tryCompleteSession(ctx, updated); err != nil {
		return nil, err
	}

	return s.sessionRepo.FindByID(ctx, sessionID)
}

func (s *IngestService) GetSessionDetail(ctx context.Context, sessionID string) (*SessionDetail, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	chunks, err := s.chunkRepo.ListBySession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	return &SessionDetail{
		Session: session,
		Chunks:  chunks,
	}, nil
}

func (s *IngestService) ResolvePendingDuplicates(
	ctx context.Context,
	userID string,
	sessionID string,
	decision ResolveDuplicateDecision,
) (*UploadSession, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.UserID != userID {
		return nil, ErrSessionNotFound
	}
	if session.Status != SessionStatusAwaitingResolution {
		return nil, fmt.Errorf("upload session is not waiting for duplicate resolution")
	}

	chunks, err := s.chunkRepo.ListBySession(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	pendingDuplicates, err := s.collectPendingDuplicates(ctx, chunks)
	if err != nil {
		return nil, err
	}
	if len(pendingDuplicates) == 0 {
		if err := s.sessionRepo.SetFinished(ctx, session.ID, SessionStatusCompleted); err != nil {
			return nil, err
		}
		return s.sessionRepo.FindByID(ctx, session.ID)
	}

	var summary ingestSummary
	switch decision {
	case ResolveDuplicateDecisionSkip:
		summary, err = s.persistPendingDuplicateSkips(ctx, session, pendingDuplicates)
	case ResolveDuplicateDecisionReplace:
		summary, err = s.persistPendingDuplicateReplacements(ctx, session, pendingDuplicates)
	default:
		return nil, fmt.Errorf("invalid duplicate resolution decision")
	}
	if err != nil {
		return nil, err
	}

	if err := s.sessionRepo.UpdateMetadata(ctx, session.ID, UploadSessionMetadata{}); err != nil {
		return nil, err
	}

	finalStatus := SessionStatusCompleted
	if summary.total > 0 && summary.failed == summary.total {
		finalStatus = SessionStatusFailed
	}
	if err := s.sessionRepo.SetFinished(ctx, session.ID, finalStatus); err != nil {
		return nil, err
	}

	history, historyErr := s.historyRepo.EnsureUploadHistory(ctx, session.UserID, session.CollectionID, session.ID)
	if historyErr == nil {
		historyStatus := "success"
		if summary.failed > 0 && summary.failed == summary.total {
			historyStatus = "failed"
		} else if summary.failed > 0 {
			historyStatus = "partial"
		} else if summary.duplicate > 0 {
			historyStatus = "warning"
		}
		_ = s.historyRepo.SetStatus(ctx, history.ID, historyStatus)
	}

	if err := s.reconcileCollectionPhase(ctx, session.CollectionID); err != nil {
		return nil, err
	}

	return s.sessionRepo.FindByID(ctx, session.ID)
}

func (s *IngestService) ListDocuments(
	ctx context.Context,
	userID string,
	collectionID string,
	status string,
	limit int,
	offset int,
) ([]*DocumentRecord, error) {
	if _, err := s.ensureCollectionOwner(ctx, userID, collectionID); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	return s.documentRepo.ListByCollection(ctx, collectionID, status, limit, offset)
}

func (s *IngestService) GetDocument(
	ctx context.Context,
	userID string,
	collectionID string,
	documentID string,
) (*DocumentRecord, error) {
	if _, err := s.ensureCollectionOwner(ctx, userID, collectionID); err != nil {
		return nil, err
	}

	doc, err := s.documentRepo.FindByID(ctx, documentID)
	if err != nil {
		return nil, err
	}
	if doc == nil || doc.CollectionID != collectionID {
		return nil, ErrDocumentNotFound
	}

	return doc, nil
}

func (s *IngestService) ListHistory(
	ctx context.Context,
	userID string,
	collectionID string,
	limit int,
	offset int,
) ([]*CollectionHistory, error) {
	if _, err := s.ensureCollectionOwner(ctx, userID, collectionID); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	return s.historyRepo.ListByCollection(ctx, collectionID, limit, offset)
}

func (s *IngestService) ListHistoryItems(
	ctx context.Context,
	userID string,
	collectionID string,
	historyID string,
	status string,
	limit int,
	offset int,
) ([]*CollectionHistoryItem, error) {
	if _, err := s.ensureCollectionOwner(ctx, userID, collectionID); err != nil {
		return nil, err
	}

	history, err := s.historyRepo.FindByID(ctx, historyID)
	if err != nil {
		return nil, err
	}
	if history.CollectionID != collectionID {
		return nil, ErrHistoryNotFound
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	return s.historyRepo.ListItems(ctx, historyID, status, limit, offset)
}

func (s *IngestService) enqueueChunk(ctx context.Context, chunkID string) error {
	if err := s.chunkRepo.UpdateStatus(ctx, chunkID, ChunkStatusQueued, nil); err != nil {
		return err
	}
	if err := s.chunkRepo.UpdateJobID(ctx, chunkID, chunkID); err != nil {
		return err
	}

	select {
	case s.queue <- chunkID:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *IngestService) ensureCollectionOwner(
	ctx context.Context,
	userID string,
	collectionID string,
) (*collection.Collection, error) {
	coll, err := s.collectionRepo.FindByID(ctx, collectionID)
	if err != nil {
		return nil, collection.ErrCollectionNotFound
	}

	if coll.UserID != userID {
		return nil, collection.ErrCollectionNotFound
	}

	return coll, nil
}

func (s *IngestService) ensureCollectionWritableByID(ctx context.Context, collectionID string) error {
	coll, err := s.collectionRepo.FindByID(ctx, collectionID)
	if err != nil {
		return collection.ErrCollectionNotFound
	}
	if coll.IsFrozen() {
		return collection.ErrCollectionFrozen
	}
	return nil
}

func (s *IngestService) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunkID := <-s.queue:
			_ = s.processChunk(ctx, chunkID)
		}
	}
}

func (s *IngestService) processChunk(ctx context.Context, chunkID string) error {
	chunk, err := s.chunkRepo.FindByID(ctx, chunkID)
	if err != nil {
		return err
	}

	if isTerminalChunkStatus(chunk.Status) {
		return nil
	}

	session, err := s.sessionRepo.FindByID(ctx, chunk.SessionID)
	if err != nil {
		return err
	}

	if err := s.chunkRepo.UpdateStatus(ctx, chunk.ID, ChunkStatusProcessing, nil); err != nil {
		return err
	}
	if err := s.sessionRepo.UpdateStatus(ctx, session.ID, SessionStatusProcessing); err != nil {
		return err
	}
	if err := s.collectionRepo.UpdatePhase(ctx, session.CollectionID, collection.PhaseProcessing); err != nil {
		return err
	}

	var payload uploadChunkPayload
	if err := json.Unmarshal(chunk.PayloadJSON, &payload); err != nil {
		msg := "failed to decode chunk payload"
		_ = s.chunkRepo.UpdateStatus(ctx, chunk.ID, ChunkStatusFailed, &msg)
		_ = s.sessionRepo.IncrementProcessedChunk(ctx, session.ID, 1, 0)
		return err
	}
	sources := payload.Sources

	dupCandidates, uniqueSources, sourceByID := s.filterDuplicateSources(ctx, session, sources)
	payload.PendingDuplicates = dupCandidates
	if nextPayloadJSON, err := json.Marshal(payload); err == nil {
		_ = s.chunkRepo.UpdatePayload(ctx, chunk.ID, nextPayloadJSON)
	}

	processor, ok := s.processors.Get(document.ProcessorKey{
		CollectionKind: session.CollectionKind,
		SourceFormat:   session.SourceFormat,
	})
	if !ok {
		processor = document.NewNoopProcessor(document.ProcessorKey{
			CollectionKind: session.CollectionKind,
			SourceFormat:   session.SourceFormat,
		})
	}

	ingestItems := make([]document.IngestItemResult, 0, len(uniqueSources))
	if len(uniqueSources) > 0 {
		req := document.IngestRequest{
			RequestID:      chunk.ID,
			UserID:         session.UserID,
			CollectionID:   session.CollectionID,
			CollectionKind: session.CollectionKind,
			SourceFormat:   session.SourceFormat,
			Sources:        uniqueSources,
			Policy:         defaultIngestPolicy(session.CollectionKind),
			RequestedAt:    time.Now(),
		}

		res, err := processor.Ingest(ctx, req)
		if err != nil && len(res.Items) == 0 {
			for _, src := range uniqueSources {
				ingestItems = append(ingestItems, document.IngestItemResult{
					SourceID:     src.SourceID,
					OriginalName: src.OriginalName,
					SHA256:       src.SHA256,
					Status:       document.IngestStatusFailed,
					Message:      "processor failed",
					Errors:       []string{err.Error()},
				})
			}
		} else {
			ingestItems = normalizeIngestItems(uniqueSources, res.Items)
		}
	}

	summary, persistErr := s.persistChunkResult(ctx, session, ingestItems, sourceByID)
	if persistErr != nil {
		msg := persistErr.Error()
		_ = s.chunkRepo.UpdateStatus(ctx, chunk.ID, ChunkStatusFailed, &msg)
		_ = s.sessionRepo.IncrementProcessedChunk(ctx, session.ID, 1, 0)
		s.cleanupTempSources(sources, defaultIngestPolicy(session.CollectionKind))
		return persistErr
	}

	chunkStatus := ChunkStatusDone
	chunkFailedDelta := 0
	chunkDuplicateDelta := 0

	if len(dupCandidates) > 0 && len(uniqueSources) == 0 && summary.failed == 0 {
		chunkStatus = ChunkStatusDuplicate
		chunkDuplicateDelta = 1
	} else if len(uniqueSources) > 0 && summary.total > 0 && summary.failed == summary.total {
		chunkStatus = ChunkStatusFailed
		chunkFailedDelta = 1
	}

	if err := s.chunkRepo.UpdateStatus(ctx, chunk.ID, chunkStatus, nil); err != nil {
		return err
	}
	if err := s.sessionRepo.IncrementProcessedChunk(ctx, session.ID, chunkFailedDelta, chunkDuplicateDelta); err != nil {
		return err
	}

	if len(dupCandidates) == 0 {
		s.cleanupTempSources(sources, defaultIngestPolicy(session.CollectionKind))
	} else {
		s.cleanupTempSources(uniqueSources, defaultIngestPolicy(session.CollectionKind))
	}

	latestSession, err := s.sessionRepo.FindByID(ctx, session.ID)
	if err == nil {
		_ = s.tryCompleteSession(ctx, latestSession)
	}

	return nil
}

func (s *IngestService) tryCompleteSession(ctx context.Context, session *UploadSession) error {
	if session.Status != SessionStatusFinalized {
		return nil
	}

	chunks, err := s.chunkRepo.ListBySession(ctx, session.ID)
	if err != nil {
		return err
	}

	if len(chunks) > 0 {
		for _, ch := range chunks {
			if !isTerminalChunkStatus(ch.Status) {
				return nil
			}
		}
	}

	pendingDuplicates, err := s.collectPendingDuplicates(ctx, chunks)
	if err != nil {
		return err
	}
	if err := s.sessionRepo.UpdateMetadata(ctx, session.ID, UploadSessionMetadata{
		PendingDuplicates: summarizePendingDuplicates(pendingDuplicates),
	}); err != nil {
		return err
	}
	if len(pendingDuplicates) > 0 {
		if err := s.sessionRepo.UpdateStatus(ctx, session.ID, SessionStatusAwaitingResolution); err != nil {
			return err
		}
		return s.reconcileCollectionPhase(ctx, session.CollectionID)
	}

	finalStatus := SessionStatusCompleted
	if len(chunks) > 0 && session.FailedChunks > 0 && session.FailedChunks == len(chunks) {
		finalStatus = SessionStatusFailed
	}

	if err := s.sessionRepo.SetFinished(ctx, session.ID, finalStatus); err != nil {
		return err
	}

	history, err := s.historyRepo.EnsureUploadHistory(ctx, session.UserID, session.CollectionID, session.ID)
	if err == nil {
		historyStatus := "success"
		if session.FailedChunks > 0 {
			historyStatus = "partial"
		}
		if finalStatus == SessionStatusFailed {
			historyStatus = "failed"
		}
		_ = s.historyRepo.SetStatus(ctx, history.ID, historyStatus)
	}

	return s.reconcileCollectionPhase(ctx, session.CollectionID)
}

func (s *IngestService) reconcileLoop(ctx context.Context) {
	ticker := time.NewTicker(sessionSweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.recoverActiveSessions(ctx, false)
		}
	}
}

func (s *IngestService) recoverActiveSessions(ctx context.Context, requeuePending bool) error {
	sessions, err := s.sessionRepo.ListActive(ctx)
	if err != nil {
		return err
	}

	collections := make(map[string]struct{}, len(sessions))
	var firstErr error

	for _, session := range sessions {
		collections[session.CollectionID] = struct{}{}
		if err := s.recoverSession(ctx, session, requeuePending); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	for collectionID := range collections {
		if err := s.reconcileCollectionPhase(ctx, collectionID); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (s *IngestService) recoverSession(ctx context.Context, session *UploadSession, requeuePending bool) error {
	chunks, err := s.chunkRepo.ListBySession(ctx, session.ID)
	if err != nil {
		return err
	}

	if requeuePending {
		for _, chunk := range chunks {
			if isTerminalChunkStatus(chunk.Status) {
				continue
			}

			if len(chunk.PayloadJSON) == 0 {
				msg := "chunk payload missing for recovery"
				_ = s.chunkRepo.UpdateStatus(ctx, chunk.ID, ChunkStatusFailed, &msg)
				_ = s.sessionRepo.IncrementProcessedChunk(ctx, session.ID, 1, 0)
				continue
			}

			if err := s.enqueueChunk(ctx, chunk.ID); err != nil {
				return err
			}
		}
	}

	latest, err := s.sessionRepo.FindByID(ctx, session.ID)
	if err != nil {
		return err
	}

	if latest.Status == SessionStatusFinalized {
		if err := s.tryCompleteSession(ctx, latest); err != nil {
			return err
		}
		latest, err = s.sessionRepo.FindByID(ctx, session.ID)
		if err != nil {
			return err
		}
	}

	latestChunks, err := s.chunkRepo.ListBySession(ctx, session.ID)
	if err != nil {
		return err
	}

	if shouldInterruptSession(latest, latestChunks, time.Now()) {
		if err := s.markSessionInterrupted(ctx, latest); err != nil {
			return err
		}
	}

	return nil
}

func (s *IngestService) markSessionInterrupted(ctx context.Context, session *UploadSession) error {
	if err := s.sessionRepo.SetFinished(ctx, session.ID, SessionStatusInterrupted); err != nil {
		return err
	}

	history, err := s.historyRepo.EnsureUploadHistory(ctx, session.UserID, session.CollectionID, session.ID)
	if err == nil {
		historyStatus := "failed"
		if session.ProcessedChunks > 0 {
			historyStatus = "partial"
		}
		_ = s.historyRepo.SetStatus(ctx, history.ID, historyStatus)
	}

	return nil
}

func (s *IngestService) reconcileCollectionPhase(ctx context.Context, collectionID string) error {
	activeSessions, err := s.sessionRepo.ListActiveByCollection(ctx, collectionID)
	if err != nil {
		return err
	}

	targetPhase := collection.PhaseReady
	if len(activeSessions) > 0 {
		targetPhase = collection.PhaseUploading
		for _, session := range activeSessions {
			chunks, chunkErr := s.chunkRepo.ListBySession(ctx, session.ID)
			if chunkErr != nil {
				return chunkErr
			}
			if session.Status == SessionStatusProcessing || hasPendingChunks(chunks) {
				targetPhase = collection.PhaseProcessing
				break
			}
		}
	}

	return s.collectionRepo.UpdatePhase(ctx, collectionID, targetPhase)
}

func (s *IngestService) prepareChunkSources(
	ctx context.Context,
	session *UploadSession,
	filesIn []SourceUploadFile,
	sourceOrderStart int,
) ([]document.IngestSource, int64, error) {
	sources := make([]document.IngestSource, 0, len(filesIn))
	var payloadSize int64

	for i, f := range filesIn {
		if !isAllowedUploadBySpec(session.CollectionKind, f.Name) {
			return nil, 0, fmt.Errorf("file %s is not allowed for collection kind %s", f.Name, session.CollectionKind)
		}

		obj, err := s.fileStore.SaveTemp(ctx, session.CollectionID, f.Name, f.Data)
		if err != nil {
			return nil, 0, err
		}

		sum := sha256.Sum256(f.Data)
		sha := hex.EncodeToString(sum[:])
		now := time.Now()

		sources = append(sources, document.IngestSource{
			SourceID:     obj.ID,
			OriginalName: f.Name,
			MimeType:     obj.MimeType,
			SizeBytes:    int64(len(f.Data)),
			SHA256:       sha,
			SourceOrder:  sourceOrderStart + i,
			TempPath:     obj.Path,
			UploadedAt:   now,
		})
		payloadSize += int64(len(f.Data))
	}

	return sources, payloadSize, nil
}

func (s *IngestService) filterDuplicateSources(
	ctx context.Context,
	session *UploadSession,
	sources []document.IngestSource,
) ([]pendingDuplicatePayloadItem, []document.IngestSource, map[string]document.IngestSource) {
	dupItems := make([]pendingDuplicatePayloadItem, 0)
	unique := make([]document.IngestSource, 0, len(sources))
	sourceByID := make(map[string]document.IngestSource, len(sources))

	for _, src := range sources {
		sourceByID[src.SourceID] = src

		existing, err := s.documentRepo.FindActiveByHash(
			ctx,
			session.CollectionID,
			string(session.CollectionKind),
			src.SHA256,
		)
		if err != nil || existing == nil {
			unique = append(unique, src)
			continue
		}

		dupItems = append(dupItems, pendingDuplicatePayloadItem{
			Source:             src,
			ExistingDocumentID: existing.ID,
			ExistingSourceName: existing.SourceName,
			ExistingStatus:     existing.Status,
		})
	}

	return dupItems, unique, sourceByID
}

type ingestSummary struct {
	total     int
	ready     int
	warning   int
	failed    int
	duplicate int
}

func (s *IngestService) persistChunkResult(
	ctx context.Context,
	session *UploadSession,
	items []document.IngestItemResult,
	sourceByID map[string]document.IngestSource,
) (ingestSummary, error) {
	summary := ingestSummary{}
	history, err := s.historyRepo.EnsureUploadHistory(ctx, session.UserID, session.CollectionID, session.ID)
	if err != nil {
		return summary, err
	}

	historyItems := make([]*CollectionHistoryItem, 0, len(items))

	for _, item := range items {
		summary.total++
		src := sourceByID[item.SourceID]
		status := string(item.Status)
		docID := item.DocumentID
		dupOf := item.DuplicateOfID
		dupKey := ptrIfNotEmpty(item.SHA256)
		msg := item.Message

		switch item.Status {
		case document.IngestStatusReady, document.IngestStatusWarning:
			normalizedRef := pickArtifactRef(item.Artifacts, "normalized")
			if normalizedRef == "" {
				status = string(document.IngestStatusFailed)
				msg = "normalized artifact missing"
				summary.failed++
				break
			}

			auditRef := ptrIfNotEmpty(pickArtifactRef(item.Artifacts, "audit"))
			rawRef := ptrIfNotEmpty(pickArtifactRef(item.Artifacts, "raw"))
			documentTag := strings.TrimSpace(item.DocumentTag)
			newDocID := uuid.NewString()
			doc := &DocumentRecord{
				ID:              newDocID,
				UserID:          session.UserID,
				CollectionID:    session.CollectionID,
				CollectionKind:  session.CollectionKind,
				SourceFormat:    session.SourceFormat,
				DocumentTag:     documentTag,
				SourceName:      src.OriginalName,
				SourceSizeBytes: src.SizeBytes,
				SourceMIME:      src.MimeType,
				SourceSHA256:    src.SHA256,
				SourceOrder:     src.SourceOrder,
				Status:          status,
				Message:         msg,
				NormalizedRef:   normalizedRef,
				AuditRef:        auditRef,
				RawRef:          rawRef,
			}

			err := s.documentRepo.Create(ctx, doc)
			if err != nil && isDedupUniqueError(err) {
				existing, findErr := s.documentRepo.FindActiveByHash(
					ctx,
					session.CollectionID,
					string(session.CollectionKind),
					src.SHA256,
				)
				if findErr == nil && existing != nil {
					status = string(document.IngestStatusDuplicate)
					dupOf = &existing.ID
					docID = nil
					summary.duplicate++
				} else {
					status = string(document.IngestStatusFailed)
					msg = "duplicate detected but existing document lookup failed"
					summary.failed++
				}
			} else if err != nil {
				status = string(document.IngestStatusFailed)
				msg = err.Error()
				summary.failed++
			} else {
				docID = &newDocID
				if item.Status == document.IngestStatusWarning {
					summary.warning++
				} else {
					summary.ready++
				}
			}

		case document.IngestStatusDuplicate:
			summary.duplicate++

		default:
			summary.failed++
		}

		warningsJSON, _ := json.Marshal(item.Warnings)
		errorsJSON, _ := json.Marshal(item.Errors)
		historyItems = append(historyItems, &CollectionHistoryItem{
			ID:              uuid.NewString(),
			HistoryID:       history.ID,
			UserID:          session.UserID,
			CollectionID:    session.CollectionID,
			CollectionKind:  session.CollectionKind,
			SourceFormat:    session.SourceFormat,
			SourceName:      src.OriginalName,
			SourceSizeBytes: src.SizeBytes,
			SourceMIME:      src.MimeType,
			SourceSHA256:    src.SHA256,
			SourceOrder:     src.SourceOrder,
			ItemStatus:      status,
			Message:         msg,
			DocumentID:      docID,
			DuplicateOfID:   dupOf,
			DuplicateKey:    dupKey,
			WarningsJSON:    warningsJSON,
			ErrorsJSON:      errorsJSON,
		})
	}

	if err := s.historyRepo.AddItems(ctx, historyItems); err != nil {
		return summary, err
	}

	if err := s.historyRepo.IncrementSummary(
		ctx,
		history.ID,
		summary.total,
		summary.ready,
		summary.warning,
		summary.failed,
		summary.duplicate,
	); err != nil {
		return summary, err
	}

	coll, err := s.collectionRepo.FindByID(ctx, session.CollectionID)
	if err == nil {
		_ = s.collectionRepo.UpdateSummary(
			ctx,
			session.CollectionID,
			coll.TotalCount+summary.total,
			coll.ReadyCount+summary.ready,
			coll.WarningCount+summary.warning,
			coll.FailedCount+summary.failed,
			coll.DuplicateCount+summary.duplicate,
		)
	}

	return summary, nil
}

func (s *IngestService) cleanupTempSources(sources []document.IngestSource, policy document.IngestPolicy) {
	if !policy.DeleteTempAfterRun {
		return
	}

	for _, src := range sources {
		_ = os.Remove(src.TempPath)
	}
}

func normalizeIngestItems(
	sources []document.IngestSource,
	items []document.IngestItemResult,
) []document.IngestItemResult {
	out := make([]document.IngestItemResult, 0, len(sources))
	byID := make(map[string]document.IngestItemResult, len(items))
	for _, item := range items {
		byID[item.SourceID] = item
	}

	for _, src := range sources {
		if item, ok := byID[src.SourceID]; ok {
			out = append(out, item)
			continue
		}

		out = append(out, document.IngestItemResult{
			SourceID:     src.SourceID,
			OriginalName: src.OriginalName,
			SHA256:       src.SHA256,
			Status:       document.IngestStatusFailed,
			Message:      "processor returned no item",
			Errors:       []string{"missing item result"},
		})
	}

	return out
}

func (s *IngestService) collectPendingDuplicates(
	ctx context.Context,
	chunks []*UploadChunk,
) ([]pendingDuplicatePayloadItem, error) {
	out := make([]pendingDuplicatePayloadItem, 0)
	for _, chunk := range chunks {
		if len(chunk.PayloadJSON) == 0 {
			continue
		}

		var payload uploadChunkPayload
		if err := json.Unmarshal(chunk.PayloadJSON, &payload); err != nil {
			return nil, err
		}
		if len(payload.PendingDuplicates) == 0 {
			continue
		}
		out = append(out, payload.PendingDuplicates...)
	}

	return out, nil
}

func (s *IngestService) persistPendingDuplicateSkips(
	ctx context.Context,
	session *UploadSession,
	duplicates []pendingDuplicatePayloadItem,
) (ingestSummary, error) {
	summary := ingestSummary{}
	history, err := s.historyRepo.EnsureUploadHistory(ctx, session.UserID, session.CollectionID, session.ID)
	if err != nil {
		return summary, err
	}

	historyItems := make([]*CollectionHistoryItem, 0, len(duplicates))
	for _, candidate := range duplicates {
		summary.total++
		summary.duplicate++

		existingID := candidate.ExistingDocumentID
		source := candidate.Source
		historyItems = append(historyItems, &CollectionHistoryItem{
			ID:              uuid.NewString(),
			HistoryID:       history.ID,
			UserID:          session.UserID,
			CollectionID:    session.CollectionID,
			CollectionKind:  session.CollectionKind,
			SourceFormat:    session.SourceFormat,
			SourceName:      source.OriginalName,
			SourceSizeBytes: source.SizeBytes,
			SourceMIME:      source.MimeType,
			SourceSHA256:    source.SHA256,
			SourceOrder:     source.SourceOrder,
			ItemStatus:      string(document.IngestStatusDuplicate),
			Message:         "duplicate source skipped",
			DuplicateOfID:   &existingID,
			DuplicateKey:    ptrIfNotEmpty(source.SHA256),
		})
	}

	if err := s.historyRepo.AddItems(ctx, historyItems); err != nil {
		return summary, err
	}
	if err := s.historyRepo.IncrementSummary(ctx, history.ID, summary.total, 0, 0, 0, summary.duplicate); err != nil {
		return summary, err
	}

	if coll, err := s.collectionRepo.FindByID(ctx, session.CollectionID); err == nil {
		_ = s.collectionRepo.UpdateSummary(
			ctx,
			session.CollectionID,
			coll.TotalCount+summary.total,
			coll.ReadyCount,
			coll.WarningCount,
			coll.FailedCount,
			coll.DuplicateCount+summary.duplicate,
		)
	}

	s.cleanupTempSources(pendingDuplicateSources(duplicates), defaultIngestPolicy(session.CollectionKind))
	return summary, nil
}

func (s *IngestService) persistPendingDuplicateReplacements(
	ctx context.Context,
	session *UploadSession,
	duplicates []pendingDuplicatePayloadItem,
) (ingestSummary, error) {
	summary := ingestSummary{}
	if len(duplicates) == 0 {
		return summary, nil
	}

	processor, ok := s.processors.Get(document.ProcessorKey{
		CollectionKind: session.CollectionKind,
		SourceFormat:   session.SourceFormat,
	})
	if !ok {
		processor = document.NewNoopProcessor(document.ProcessorKey{
			CollectionKind: session.CollectionKind,
			SourceFormat:   session.SourceFormat,
		})
	}

	sources := pendingDuplicateSources(duplicates)
	req := document.IngestRequest{
		RequestID:      session.ID,
		UserID:         session.UserID,
		CollectionID:   session.CollectionID,
		CollectionKind: session.CollectionKind,
		SourceFormat:   session.SourceFormat,
		Sources:        sources,
		Policy:         defaultIngestPolicy(session.CollectionKind),
		RequestedAt:    time.Now(),
	}

	res, err := processor.Ingest(ctx, req)
	items := res.Items
	if err != nil && len(items) == 0 {
		items = make([]document.IngestItemResult, 0, len(sources))
		for _, src := range sources {
			items = append(items, document.IngestItemResult{
				SourceID:     src.SourceID,
				OriginalName: src.OriginalName,
				SHA256:       src.SHA256,
				Status:       document.IngestStatusFailed,
				Message:      "processor failed",
				Errors:       []string{err.Error()},
			})
		}
	}
	items = normalizeIngestItems(sources, items)

	history, err := s.historyRepo.EnsureUploadHistory(ctx, session.UserID, session.CollectionID, session.ID)
	if err != nil {
		return summary, err
	}

	candidateBySourceID := make(map[string]pendingDuplicatePayloadItem, len(duplicates))
	for _, candidate := range duplicates {
		candidateBySourceID[candidate.Source.SourceID] = candidate
	}

	historyItems := make([]*CollectionHistoryItem, 0, len(items))
	coll, _ := s.collectionRepo.FindByID(ctx, session.CollectionID)
	updatedSummary := false

	for _, item := range items {
		summary.total++
		candidate, ok := candidateBySourceID[item.SourceID]
		if !ok {
			summary.failed++
			continue
		}

		src := candidate.Source
		status := string(item.Status)
		msg := item.Message
		docID := &candidate.ExistingDocumentID
		dupKey := ptrIfNotEmpty(src.SHA256)
		warningsJSON, _ := json.Marshal(item.Warnings)
		errorsJSON, _ := json.Marshal(item.Errors)

		switch item.Status {
		case document.IngestStatusReady, document.IngestStatusWarning:
			existingDoc, findErr := s.documentRepo.FindByID(ctx, candidate.ExistingDocumentID)
			if findErr != nil || existingDoc == nil {
				status = string(document.IngestStatusFailed)
				msg = "existing document not found for replacement"
				summary.failed++
				docID = nil
				break
			}

			normalizedRef := pickArtifactRef(item.Artifacts, "normalized")
			if normalizedRef == "" {
				status = string(document.IngestStatusFailed)
				msg = "normalized artifact missing"
				summary.failed++
				docID = nil
				break
			}

			oldStatus := existingDoc.Status
			oldNormalizedRef := existingDoc.NormalizedRef
			oldAuditRef := existingDoc.AuditRef
			oldRawRef := existingDoc.RawRef

			existingDoc.DocumentTag = strings.TrimSpace(item.DocumentTag)
			existingDoc.SourceName = src.OriginalName
			existingDoc.SourceSizeBytes = src.SizeBytes
			existingDoc.SourceMIME = src.MimeType
			existingDoc.SourceSHA256 = src.SHA256
			existingDoc.SourceOrder = src.SourceOrder
			existingDoc.Status = status
			existingDoc.Message = msg
			existingDoc.NormalizedRef = normalizedRef
			existingDoc.AuditRef = ptrIfNotEmpty(pickArtifactRef(item.Artifacts, "audit"))
			existingDoc.RawRef = ptrIfNotEmpty(pickArtifactRef(item.Artifacts, "raw"))

			if updateErr := s.documentRepo.Update(ctx, existingDoc); updateErr != nil {
				status = string(document.IngestStatusFailed)
				msg = updateErr.Error()
				summary.failed++
				docID = nil
				break
			}

			if coll != nil {
				if oldStatus == "ready" && status == "warning" {
					coll.ReadyCount--
					coll.WarningCount++
					updatedSummary = true
				} else if oldStatus == "warning" && status == "ready" {
					coll.WarningCount--
					coll.ReadyCount++
					updatedSummary = true
				}
			}

			deleteReplacedArtifact(ctx, s.fileStore, session.CollectionID, oldNormalizedRef, existingDoc.NormalizedRef)
			deleteReplacedArtifact(ctx, s.fileStore, session.CollectionID, stringPtrValue(oldAuditRef), stringPtrValue(existingDoc.AuditRef))
			deleteReplacedArtifact(ctx, s.fileStore, session.CollectionID, stringPtrValue(oldRawRef), stringPtrValue(existingDoc.RawRef))

			if item.Status == document.IngestStatusWarning {
				summary.warning++
			} else {
				summary.ready++
			}

		default:
			summary.failed++
			docID = nil
		}

		historyItems = append(historyItems, &CollectionHistoryItem{
			ID:              uuid.NewString(),
			HistoryID:       history.ID,
			UserID:          session.UserID,
			CollectionID:    session.CollectionID,
			CollectionKind:  session.CollectionKind,
			SourceFormat:    session.SourceFormat,
			SourceName:      src.OriginalName,
			SourceSizeBytes: src.SizeBytes,
			SourceMIME:      src.MimeType,
			SourceSHA256:    src.SHA256,
			SourceOrder:     src.SourceOrder,
			ItemStatus:      status,
			Message:         msg,
			DocumentID:      docID,
			DuplicateKey:    dupKey,
			WarningsJSON:    warningsJSON,
			ErrorsJSON:      errorsJSON,
		})
	}

	if err := s.historyRepo.AddItems(ctx, historyItems); err != nil {
		return summary, err
	}
	if err := s.historyRepo.IncrementSummary(ctx, history.ID, summary.total, summary.ready, summary.warning, summary.failed, 0); err != nil {
		return summary, err
	}

	if coll != nil && updatedSummary {
		_ = s.collectionRepo.UpdateSummary(
			ctx,
			session.CollectionID,
			coll.TotalCount,
			coll.ReadyCount,
			coll.WarningCount,
			coll.FailedCount,
			coll.DuplicateCount,
		)
	}

	s.cleanupTempSources(sources, defaultIngestPolicy(session.CollectionKind))
	return summary, nil
}

func decodeUploadSessionMetadata(raw json.RawMessage) UploadSessionMetadata {
	if len(raw) == 0 {
		return UploadSessionMetadata{}
	}

	var metadata UploadSessionMetadata
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return UploadSessionMetadata{}
	}
	return metadata
}

func pendingDuplicateSources(duplicates []pendingDuplicatePayloadItem) []document.IngestSource {
	out := make([]document.IngestSource, 0, len(duplicates))
	for _, candidate := range duplicates {
		out = append(out, candidate.Source)
	}
	return out
}

func summarizePendingDuplicates(items []pendingDuplicatePayloadItem) []PendingDuplicateResolution {
	out := make([]PendingDuplicateResolution, 0, len(items))
	for _, item := range items {
		out = append(out, PendingDuplicateResolution{
			Source:             newPendingDuplicateSource(item.Source),
			ExistingDocumentID: item.ExistingDocumentID,
			ExistingSourceName: item.ExistingSourceName,
			ExistingStatus:     item.ExistingStatus,
		})
	}
	return out
}

func newPendingDuplicateSource(src document.IngestSource) PendingDuplicateSource {
	return PendingDuplicateSource{
		SourceID:     src.SourceID,
		OriginalName: src.OriginalName,
		MimeType:     src.MimeType,
		SizeBytes:    src.SizeBytes,
		SHA256:       src.SHA256,
		SourceOrder:  src.SourceOrder,
		TempPath:     src.TempPath,
		UploadedAt:   src.UploadedAt,
	}
}

func defaultIngestPolicy(collectionKind document.CollectionKind) document.IngestPolicy {
	spec, ok := document.BuildCollectionSpec(collectionKind)
	if ok {
		return document.IngestPolicy{
			KeepRaw:            spec.Ingest.KeepRaw,
			DeleteTempAfterRun: spec.Ingest.DeleteTempAfterRun,
		}
	}

	return document.IngestPolicy{
		KeepRaw:            false,
		DeleteTempAfterRun: true,
	}
}

func isWritableSessionStatus(status SessionStatus) bool {
	switch status {
	case SessionStatusCreated, SessionStatusReceiving, SessionStatusProcessing, SessionStatusFinalized:
		return true
	default:
		return false
	}
}

func isTerminalChunkStatus(status ChunkStatus) bool {
	switch status {
	case ChunkStatusDone, ChunkStatusFailed, ChunkStatusDuplicate:
		return true
	default:
		return false
	}
}

func hasPendingChunks(chunks []*UploadChunk) bool {
	for _, chunk := range chunks {
		if !isTerminalChunkStatus(chunk.Status) {
			return true
		}
	}
	return false
}

func isTerminalSessionStatus(status SessionStatus) bool {
	switch status {
	case SessionStatusAwaitingResolution, SessionStatusCompleted, SessionStatusFailed, SessionStatusInterrupted, SessionStatusExpired:
		return true
	default:
		return false
	}
}

func shouldInterruptSession(session *UploadSession, chunks []*UploadChunk, now time.Time) bool {
	if session == nil || isTerminalSessionStatus(session.Status) || session.Status == SessionStatusFinalized {
		return false
	}

	if hasPendingChunks(chunks) {
		return false
	}

	last := session.StartedAt
	if session.LastHeartbeatAt != nil {
		last = *session.LastHeartbeatAt
	}

	return now.Sub(last) >= sessionHeartbeatTimeout
}

func pickArtifactRef(artifacts []document.Artifact, kind string) string {
	for _, a := range artifacts {
		if strings.EqualFold(a.Kind, kind) {
			return a.ObjectID
		}
	}
	return ""
}

func ptrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func deleteReplacedArtifact(
	ctx context.Context,
	store file.FileStore,
	collectionID string,
	oldRef string,
	newRef string,
) {
	if oldRef == "" || oldRef == newRef {
		return
	}
	_ = store.Delete(ctx, collectionID, oldRef)
}

func isDedupUniqueError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "documents.collection_id") &&
		strings.Contains(msg, "documents.document_type") &&
		strings.Contains(msg, "documents.source_sha256")
}

func isAllowedUploadBySpec(collectionKind document.CollectionKind, fileName string) bool {
	spec, ok := document.BuildCollectionSpec(collectionKind)
	if !ok {
		return false
	}
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext == "" {
		return false
	}
	for _, allowed := range spec.Upload.AcceptExtensions {
		if strings.EqualFold(ext, allowed) {
			return true
		}
	}
	return false
}
