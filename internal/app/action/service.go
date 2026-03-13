package action

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/document"
	dcollection "github.com/sieryo/invoice-extractor/internal/domain/collection"
)

type Service struct {
	repo           Repository
	collectionRepo dcollection.Repository
	processors     *document.Registry

	queue   chan string
	workers int
	once    sync.Once
}

type summaryCount struct {
	total   int
	success int
	warning int
	failed  int
	skipped int
}

func NewService(
	repo Repository,
	collectionRepo dcollection.Repository,
	processors *document.Registry,
	workers int,
) *Service {
	if workers < 1 {
		workers = 1
	}

	return &Service{
		repo:           repo,
		collectionRepo: collectionRepo,
		processors:     processors,
		queue:          make(chan string, 64),
		workers:        workers,
	}
}

func (s *Service) StartPool(ctx context.Context) {
	s.once.Do(func() {
		for i := 0; i < s.workers; i++ {
			go s.worker(ctx)
		}
		_ = s.recoverPendingActions(ctx)
	})
}

func (s *Service) RunAction(ctx context.Context, req RunRequest) (*CollectionAction, error) {
	actionType := strings.TrimSpace(req.ActionType)
	if actionType == "" {
		return nil, ErrInvalidActionType
	}

	coll, err := s.getOwnedCollection(ctx, req.UserID, req.CollectionID)
	if err != nil {
		return nil, err
	}

	idempotencyKey := normalizePtrString(req.IdempotencyKey)
	if idempotencyKey != nil {
		existing, findErr := s.repo.FindActionByIdempotency(
			ctx,
			req.CollectionID,
			actionType,
			*idempotencyKey,
		)
		if findErr == nil && existing != nil {
			return existing, nil
		}
	}

	docType := document.DocumentType(*coll.DocumentType)
	selectedDocumentIDs, err := normalizeDocumentIDs(req.DocumentIDs)
	if err != nil {
		return nil, err
	}

	var snapshotDocs []SnapshotDocument
	if len(selectedDocumentIDs) > 0 {
		snapshotDocs, err = s.repo.ListSnapshotDocumentsByIDs(ctx, req.CollectionID, docType, selectedDocumentIDs)
		if err != nil {
			return nil, err
		}

		if len(snapshotDocs) != len(selectedDocumentIDs) {
			return nil, ErrSnapshotDocNotFound
		}

		if err := validateSnapshotStatuses(snapshotDocs, []string{"ready", "warning"}); err != nil {
			return nil, err
		}
	} else if req.RerunOfActionID != nil && strings.TrimSpace(*req.RerunOfActionID) != "" && len(req.DocumentStatuses) == 0 {
		baseAction, findErr := s.repo.FindActionByID(ctx, strings.TrimSpace(*req.RerunOfActionID))
		if findErr != nil {
			return nil, ErrActionNotFound
		}
		if baseAction.CollectionID != req.CollectionID {
			return nil, ErrActionNotFound
		}
		if err := json.Unmarshal(baseAction.SnapshotJSON, &snapshotDocs); err != nil {
			return nil, ErrEmptySnapshot
		}
	} else {
		statuses, statusErr := normalizeSnapshotStatuses(req.DocumentStatuses)
		if statusErr != nil {
			return nil, statusErr
		}

		snapshotDocs, err = s.repo.ListSnapshotDocuments(ctx, req.CollectionID, docType, statuses)
		if err != nil {
			return nil, err
		}
	}

	if len(snapshotDocs) == 0 {
		return nil, ErrEmptySnapshot
	}

	snapshotJSON, err := json.Marshal(snapshotDocs)
	if err != nil {
		return nil, err
	}

	sum := sha256.Sum256(snapshotJSON)
	hash := hex.EncodeToString(sum[:])

	now := time.Now()
	action := &CollectionAction{
		ID:             uuid.NewString(),
		UserID:         req.UserID,
		CollectionID:   req.CollectionID,
		DocumentType:   docType,
		ActionType:     actionType,
		Status:         StatusQueued,
		ParamsJSON:     req.Params,
		SnapshotJSON:   snapshotJSON,
		SnapshotHash:   hash,
		SnapshotTotal:  len(snapshotDocs),
		RerunOfAction:  req.RerunOfActionID,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.CreateAction(ctx, action); err != nil {
		if action.IdempotencyKey != nil && isActionIdempotencyUniqueError(err) {
			existing, findErr := s.repo.FindActionByIdempotency(
				ctx,
				req.CollectionID,
				actionType,
				*action.IdempotencyKey,
			)
			if findErr == nil && existing != nil {
				return existing, nil
			}
		}
		return nil, err
	}

	if err := s.enqueue(ctx, action.ID); err != nil {
		return nil, err
	}

	return action, nil
}

func (s *Service) ListActions(ctx context.Context, req ListRequest) ([]*CollectionAction, error) {
	if _, err := s.getOwnedCollection(ctx, req.UserID, req.CollectionID); err != nil {
		return nil, err
	}

	limit := req.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := req.Offset
	if offset < 0 {
		offset = 0
	}

	return s.repo.ListActions(ctx, req.CollectionID, req.Status, limit, offset)
}

func (s *Service) GetActionSpec(
	ctx context.Context,
	userID string,
	collectionID string,
) (*document.DocumentTypeSpec, error) {
	coll, err := s.getOwnedCollection(ctx, userID, collectionID)
	if err != nil {
		return nil, err
	}

	spec, ok := document.BuildDocumentTypeSpec(document.DocumentType(*coll.DocumentType))
	if !ok {
		return nil, ErrSpecNotFound
	}

	return &spec, nil
}

func (s *Service) GetActionDetail(
	ctx context.Context,
	userID string,
	collectionID string,
	actionID string,
) (*ActionDetail, error) {
	if _, err := s.getOwnedCollection(ctx, userID, collectionID); err != nil {
		return nil, err
	}

	action, err := s.repo.FindActionByID(ctx, actionID)
	if err != nil {
		return nil, err
	}
	if action.CollectionID != collectionID {
		return nil, ErrActionNotFound
	}

	items, err := s.repo.ListActionItems(ctx, actionID)
	if err != nil {
		return nil, err
	}

	outputs, err := s.repo.ListActionOutputs(ctx, actionID)
	if err != nil {
		return nil, err
	}

	return &ActionDetail{
		Action:  action,
		Items:   items,
		Outputs: outputs,
	}, nil
}

func (s *Service) enqueue(ctx context.Context, actionID string) error {
	select {
	case s.queue <- actionID:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *Service) recoverPendingActions(ctx context.Context) error {
	pending, err := s.repo.ListPendingActions(ctx)
	if err != nil {
		return err
	}

	for _, act := range pending {
		if err := s.enqueue(ctx, act.ID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case actionID := <-s.queue:
			_ = s.process(ctx, actionID)
		}
	}
}

func (s *Service) process(ctx context.Context, actionID string) error {
	action, err := s.repo.FindActionByID(ctx, actionID)
	if err != nil {
		return err
	}

	if isTerminalActionStatus(action.Status) {
		return nil
	}

	startedAt := time.Now()
	if err := s.repo.SetActionRunning(ctx, action.ID, startedAt); err != nil {
		return err
	}

	var snapshotDocs []SnapshotDocument
	if err := json.Unmarshal(action.SnapshotJSON, &snapshotDocs); err != nil {
		return s.finishAction(
			ctx,
			action.ID,
			StatusFailed,
			"invalid snapshot payload",
			summaryCount{},
		)
	}

	docIDs := make([]string, 0, len(snapshotDocs))
	snapshotPayload := make([]document.ActionSnapshotDocument, 0, len(snapshotDocs))
	for _, doc := range snapshotDocs {
		docIDs = append(docIDs, doc.DocumentID)
		snapshotPayload = append(snapshotPayload, document.ActionSnapshotDocument{
			DocumentID:    doc.DocumentID,
			SourceName:    doc.SourceName,
			SourceOrder:   doc.SourceOrder,
			Status:        doc.Status,
			SourceSHA256:  doc.SourceSHA256,
			NormalizedRef: doc.NormalizedRef,
			AuditRef:      doc.AuditRef,
			RawRef:        doc.RawRef,
		})
	}

	processor, ok := s.processors.Get(action.DocumentType)
	if !ok {
		processor = document.NewNoopProcessor(action.DocumentType)
	}

	runReq := document.ActionRequest{
		ActionID:      action.ID,
		UserID:        action.UserID,
		CollectionID:  action.CollectionID,
		DocumentType:  action.DocumentType,
		ActionType:    action.ActionType,
		SnapshotDocID: docIDs,
		SnapshotDocs:  snapshotPayload,
		Params:        action.ParamsJSON,
		RequestedAt:   startedAt,
	}

	result, runErr := processor.RunAction(ctx, runReq)

	items, counts := buildActionItems(action.ID, snapshotDocs, result.ItemResults, runErr)
	if err := s.repo.AddActionItems(ctx, items); err != nil {
		return err
	}

	outputs := buildActionOutputs(action.ID, result.Outputs)
	if err := s.repo.AddActionOutputs(ctx, outputs); err != nil {
		return err
	}

	status := deriveFinalStatus(result.Status, counts, runErr)
	message := strings.TrimSpace(result.Message)
	if runErr != nil {
		if message == "" {
			message = runErr.Error()
		} else {
			message += ": " + runErr.Error()
		}
	}

	return s.finishAction(ctx, action.ID, status, message, counts)
}

func (s *Service) finishAction(
	ctx context.Context,
	actionID string,
	status Status,
	message string,
	counts summaryCount,
) error {
	return s.repo.SetActionFinished(
		ctx,
		actionID,
		status,
		message,
		counts.total,
		counts.success,
		counts.warning,
		counts.failed,
		counts.skipped,
		time.Now(),
	)
}

func (s *Service) getOwnedCollection(
	ctx context.Context,
	userID string,
	collectionID string,
) (*dcollection.Collection, error) {
	coll, err := s.collectionRepo.FindByID(ctx, collectionID)
	if err != nil {
		return nil, dcollection.ErrCollectionNotFound
	}

	if coll.UserID != userID {
		return nil, dcollection.ErrCollectionNotFound
	}

	if !coll.IsCollection() || coll.DocumentType == nil {
		return nil, dcollection.ErrInvalidNodeType
	}

	return coll, nil
}

func buildActionItems(
	actionID string,
	snapshotDocs []SnapshotDocument,
	input []document.ActionItemResult,
	runErr error,
) ([]*CollectionActionItem, summaryCount) {
	now := time.Now()
	items := make([]*CollectionActionItem, 0, len(snapshotDocs))
	resultByDoc := make(map[string]document.ActionItemResult, len(input))
	for _, item := range input {
		resultByDoc[item.DocumentID] = item
	}

	counts := summaryCount{}
	for _, snapshotDoc := range snapshotDocs {
		counts.total++
		docID := snapshotDoc.DocumentID
		resultItem, ok := resultByDoc[docID]

		status := ItemStatusSkipped
		message := "document not processed"
		errText := ""
		warnings := []string(nil)

		if runErr != nil && len(input) == 0 {
			status = ItemStatusFailed
			message = "processor failed"
			errText = runErr.Error()
		} else if ok {
			status = normalizeItemStatus(resultItem.Status, resultItem.Error)
			message = strings.TrimSpace(resultItem.Message)
			errText = strings.TrimSpace(resultItem.Error)
			warnings = resultItem.Warnings
		}

		switch status {
		case ItemStatusSuccess:
			counts.success++
		case ItemStatusWarning:
			counts.warning++
		case ItemStatusFailed:
			counts.failed++
		default:
			counts.skipped++
		}

		warningJSON, _ := json.Marshal(warnings)
		items = append(items, &CollectionActionItem{
			ID:           uuid.NewString(),
			ActionID:     actionID,
			DocumentID:   &docID,
			Status:       status,
			Message:      message,
			WarningsJSON: warningJSON,
			Error:        errText,
			CreatedAt:    now,
		})
	}

	return items, counts
}

func buildActionOutputs(actionID string, outputs []document.ActionOutput) []*CollectionActionOutput {
	now := time.Now()
	out := make([]*CollectionActionOutput, 0, len(outputs))
	for i, output := range outputs {
		if strings.TrimSpace(output.ObjectRef) == "" {
			continue
		}

		kind := OutputKindPayload
		switch strings.ToLower(strings.TrimSpace(output.Kind)) {
		case string(OutputKindFile):
			kind = OutputKindFile
		case string(OutputKindLink):
			kind = OutputKindLink
		case string(OutputKindPayload):
			kind = OutputKindPayload
		}

		name := strings.TrimSpace(output.Name)
		if name == "" {
			name = "output_" + strconv.Itoa(i+1)
		}

		out = append(out, &CollectionActionOutput{
			ID:        uuid.NewString(),
			ActionID:  actionID,
			Kind:      kind,
			Name:      name,
			ObjectRef: output.ObjectRef,
			MimeType:  output.MimeType,
			SizeBytes: output.SizeBytes,
			Checksum:  output.Checksum,
			CreatedAt: now,
		})
	}

	return out
}

func deriveFinalStatus(raw string, counts summaryCount, runErr error) Status {
	if runErr != nil {
		if counts.failed == 0 && (counts.success > 0 || counts.warning > 0 || counts.skipped > 0) {
			return StatusPartial
		}
		return StatusFailed
	}

	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch Status(normalized) {
	case StatusSuccess, StatusWarning, StatusPartial, StatusFailed, StatusCanceled:
		return Status(normalized)
	}

	if counts.failed > 0 {
		if counts.success > 0 || counts.warning > 0 || counts.skipped > 0 {
			return StatusPartial
		}
		return StatusFailed
	}
	if counts.warning > 0 {
		return StatusWarning
	}

	return StatusSuccess
}

func isTerminalActionStatus(status Status) bool {
	switch status {
	case StatusSuccess, StatusWarning, StatusPartial, StatusFailed, StatusCanceled:
		return true
	default:
		return false
	}
}

func normalizeSnapshotStatuses(input []string) ([]string, error) {
	if len(input) == 0 {
		return []string{"ready", "warning"}, nil
	}

	set := map[string]struct{}{}
	for _, status := range input {
		normalized := strings.ToLower(strings.TrimSpace(status))
		switch normalized {
		case "ready", "warning":
			set[normalized] = struct{}{}
		default:
			return nil, ErrInvalidDocumentStatus
		}
	}

	out := make([]string, 0, len(set))
	if _, ok := set["ready"]; ok {
		out = append(out, "ready")
	}
	if _, ok := set["warning"]; ok {
		out = append(out, "warning")
	}
	return out, nil
}

func normalizeItemStatus(raw string, errText string) ItemStatus {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case string(ItemStatusSuccess):
		return ItemStatusSuccess
	case string(ItemStatusWarning):
		return ItemStatusWarning
	case string(ItemStatusFailed):
		return ItemStatusFailed
	case string(ItemStatusSkipped):
		return ItemStatusSkipped
	case string(ItemStatusCanceled):
		return ItemStatusCanceled
	default:
		if strings.TrimSpace(errText) != "" {
			return ItemStatusFailed
		}
		return ItemStatusSuccess
	}
}

func normalizePtrString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func isActionIdempotencyUniqueError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "collection_actions.collection_id") &&
		strings.Contains(msg, "collection_actions.action_type") &&
		strings.Contains(msg, "collection_actions.idempotency_key")
}

func normalizeDocumentIDs(input []string) ([]string, error) {
	if len(input) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(input))
	out := make([]string, 0, len(input))
	for _, id := range input {
		normalized := strings.TrimSpace(id)
		if normalized == "" {
			return nil, ErrInvalidDocumentIDs
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out, nil
}

func validateSnapshotStatuses(snapshotDocs []SnapshotDocument, allowed []string) error {
	allow := make(map[string]struct{}, len(allowed))
	for _, status := range allowed {
		allow[status] = struct{}{}
	}
	for _, doc := range snapshotDocs {
		status := strings.ToLower(strings.TrimSpace(doc.Status))
		if _, ok := allow[status]; !ok {
			return ErrSnapshotDocStatus
		}
	}
	return nil
}
