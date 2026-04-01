package action

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	appbukpot "github.com/sieryo/invoice-extractor/internal/app/bukpot"
	appbuyer "github.com/sieryo/invoice-extractor/internal/app/buyer"
	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
	appcashflowbill "github.com/sieryo/invoice-extractor/internal/app/cashflowbill"
	"github.com/sieryo/invoice-extractor/internal/app/document"
	dcollection "github.com/sieryo/invoice-extractor/internal/domain/collection"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type BuyerRegistryStatusProvider interface {
	Status(profileID string) appbuyer.BuyerRegistryStatus
}

type CashflowTaxAccountStatusProvider interface {
	Status(profileID string) appcashflow.TaxAccountStatus
}

type CashflowProfileConfigProvider interface {
	Status(profileID string, key appcashflow.ProfileConfigKey) appcashflow.ProfileConfigStatus
	Load(profileID string, key appcashflow.ProfileConfigKey) (appcashflow.ProfileConfig, error)
}

type CashflowBillCategoryStatusProvider interface {
	Status(profileID string) appcashflowbill.CategoryAccountStatus
	Load(profileID string) (map[string]appcashflowbill.CategoryAccount, error)
}

type CashflowBillProfileConfigProvider interface {
	Status(profileID string, key appcashflowbill.ProfileConfigKey) appcashflowbill.ProfileConfigStatus
	Load(profileID string, key appcashflowbill.ProfileConfigKey) (appcashflowbill.ProfileConfig, error)
}

type BukpotRequestConfigProvider interface {
	Status(profileID string) appbukpot.RequestConfigStatus
	Load(profileID string) (appbukpot.RequestConfig, error)
}

type BukpotActionProfileProvider interface {
	Load(profileID string, key appbukpot.ActionProfileKey) (appbukpot.ActionProfile, error)
}

type Service struct {
	repo                      Repository
	collectionRepo            dcollection.Repository
	processors                *document.Registry
	buyerRegistry             BuyerRegistryStatusProvider
	taxAccounts               CashflowTaxAccountStatusProvider
	cashflowProfileConfig     CashflowProfileConfigProvider
	cashflowBillCategories    CashflowBillCategoryStatusProvider
	cashflowBillProfileConfig CashflowBillProfileConfigProvider
	bukpotRequestConfig       BukpotRequestConfigProvider
	bukpotActionProfiles      BukpotActionProfileProvider
	fileStore                 file.FileStore

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

const maxStoredItemWarnings = 100

func NewService(
	repo Repository,
	collectionRepo dcollection.Repository,
	processors *document.Registry,
	buyerRegistry BuyerRegistryStatusProvider,
	taxAccounts CashflowTaxAccountStatusProvider,
	cashflowProfileConfig CashflowProfileConfigProvider,
	cashflowBillCategories CashflowBillCategoryStatusProvider,
	cashflowBillProfileConfig CashflowBillProfileConfigProvider,
	bukpotRequestConfig BukpotRequestConfigProvider,
	bukpotActionProfiles BukpotActionProfileProvider,
	fileStore file.FileStore,
	workers int,
) *Service {
	if workers < 1 {
		workers = 1
	}

	return &Service{
		repo:                      repo,
		collectionRepo:            collectionRepo,
		processors:                processors,
		buyerRegistry:             buyerRegistry,
		taxAccounts:               taxAccounts,
		cashflowProfileConfig:     cashflowProfileConfig,
		cashflowBillCategories:    cashflowBillCategories,
		cashflowBillProfileConfig: cashflowBillProfileConfig,
		bukpotRequestConfig:       bukpotRequestConfig,
		bukpotActionProfiles:      bukpotActionProfiles,
		fileStore:                 fileStore,
		queue:                     make(chan string, 64),
		workers:                   workers,
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
	if coll.IsFrozen() {
		return nil, dcollection.ErrCollectionFrozen
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

	collectionKind := document.CollectionKind(*coll.CollectionKind)
	sourceFormat := document.ResolveCollectionSourceFormat(collectionKind)
	spec, ok := document.BuildCollectionSpec(collectionKind)
	if !ok {
		return nil, ErrSpecNotFound
	}
	spec = s.applyRuntimeRequirements(spec, coll.UserID)

	actionSpec, ok := spec.FindActionSpec(actionType)
	if !ok {
		return nil, ErrActionNotSupported
	}
	actionSpec = s.applyBukpotActionProfileDefaults(actionSpec, coll.UserID, collectionKind)
	if !actionSpec.State.Enabled {
		reason := strings.TrimSpace(actionSpec.State.Message)
		if reason != "" {
			return nil, &DisabledActionError{Reason: reason}
		}
		return nil, ErrActionDisabled
	}
	if reqErr := validateActionRequirements(actionSpec.Requirements); reqErr != nil {
		return nil, reqErr
	}

	allowedStatuses, err := normalizeAllowedStatuses(actionSpec.Selection.AllowedStatus)
	if err != nil {
		return nil, err
	}

	inputJSON, err := normalizeAndValidateActionInput(req.Input, actionSpec.Form, actionSpec.ArtifactInputs)
	if err != nil {
		return nil, err
	}

	minDocumentCnt := actionSpec.Selection.MinDocumentCnt
	if minDocumentCnt <= 0 {
		minDocumentCnt = 1
	}
	maxDocumentCnt := actionSpec.Selection.MaxDocumentCnt

	selectedDocumentIDs, err := normalizeDocumentIDs(req.DocumentIDs)
	if err != nil {
		return nil, err
	}

	var snapshotDocs []SnapshotDocument
	if len(selectedDocumentIDs) > 0 {
		snapshotDocs, err = s.repo.ListSnapshotDocumentsByIDs(ctx, req.CollectionID, collectionKind, sourceFormat, selectedDocumentIDs)
		if err != nil {
			return nil, err
		}

		if len(snapshotDocs) != len(selectedDocumentIDs) {
			return nil, ErrSnapshotDocNotFound
		}

		if err := validateSnapshotStatuses(snapshotDocs, allowedStatuses); err != nil {
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
		statuses, statusErr := normalizeSnapshotStatuses(req.DocumentStatuses, allowedStatuses)
		if statusErr != nil {
			return nil, statusErr
		}

		snapshotDocs, err = s.repo.ListSnapshotDocuments(ctx, req.CollectionID, collectionKind, sourceFormat, statuses)
		if err != nil {
			return nil, err
		}
	}

	if len(snapshotDocs) == 0 {
		return nil, ErrEmptySnapshot
	}
	if len(snapshotDocs) < minDocumentCnt {
		return nil, ErrMinDocumentsRequired
	}
	if maxDocumentCnt > 0 && len(snapshotDocs) > maxDocumentCnt {
		return nil, &MaxDocumentsError{Limit: maxDocumentCnt}
	}

	processor, ok := s.processors.Get(document.ProcessorKey{
		CollectionKind: collectionKind,
		SourceFormat:   sourceFormat,
	})
	if !ok {
		processor = document.NewNoopProcessor(document.ProcessorKey{
			CollectionKind: collectionKind,
			SourceFormat:   sourceFormat,
		})
	}

	if validator, ok := processor.(document.ActionPreflightValidator); ok {
		preflightDocs := make([]document.ActionSnapshotDocument, 0, len(snapshotDocs))
		for _, doc := range snapshotDocs {
			preflightDocs = append(preflightDocs, document.ActionSnapshotDocument{
				DocumentID:    doc.DocumentID,
				SourceName:    doc.SourceName,
				SourceOrder:   doc.SourceOrder,
				Status:        doc.Status,
				DocumentTag:   doc.DocumentTag,
				SourceSHA256:  doc.SourceSHA256,
				NormalizedRef: doc.NormalizedRef,
				AuditRef:      doc.AuditRef,
				RawRef:        doc.RawRef,
			})
		}
		if err := validator.ValidateAction(ctx, document.ActionRequest{
			ActionType:     actionType,
			UserID:         req.UserID,
			CollectionID:   req.CollectionID,
			CollectionKind: collectionKind,
			SourceFormat:   sourceFormat,
			SnapshotDocs:   preflightDocs,
			Input:          inputJSON,
			RequestedAt:    time.Now(),
		}); err != nil {
			return nil, err
		}
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
		CollectionKind: collectionKind,
		SourceFormat:   sourceFormat,
		ActionType:     actionType,
		Status:         StatusQueued,
		InputJSON:      inputJSON,
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
) (*document.CollectionSpec, error) {
	coll, err := s.getOwnedCollection(ctx, userID, collectionID)
	if err != nil {
		return nil, err
	}

	spec, ok := document.BuildCollectionSpec(document.CollectionKind(*coll.CollectionKind))
	if !ok {
		return nil, ErrSpecNotFound
	}
	spec = s.applyRuntimeRequirements(spec, coll.UserID)
	spec = s.applyBukpotActionProfileCollectionSpec(spec, coll.UserID)
	if coll.IsFrozen() {
		spec = applyFrozenCollectionSpec(spec)
	}

	return &spec, nil
}

func (s *Service) ResolveActionSpec(
	ctx context.Context,
	req ResolveSpecRequest,
) (*document.ActionSpec, error) {
	coll, err := s.getOwnedCollection(ctx, req.UserID, req.CollectionID)
	if err != nil {
		return nil, err
	}

	collectionKind := document.CollectionKind(*coll.CollectionKind)
	sourceFormat := document.ResolveCollectionSourceFormat(collectionKind)

	spec, ok := document.BuildCollectionSpec(collectionKind)
	if !ok {
		return nil, ErrSpecNotFound
	}
	spec = s.applyRuntimeRequirements(spec, coll.UserID)
	spec = s.applyBukpotActionProfileCollectionSpec(spec, coll.UserID)
	if coll.IsFrozen() {
		spec = applyFrozenCollectionSpec(spec)
	}

	actionSpec, ok := spec.FindActionSpec(req.ActionType)
	if !ok {
		return nil, ErrActionNotSupported
	}

	resolved := actionSpec
	if collectionKind == document.CollectionKindCashflowImport && isCashflowExportAction(actionSpec.ActionType) {
		resolved, err = s.resolveCashflowActionSpec(ctx, coll.UserID, req.CollectionID, collectionKind, sourceFormat, actionSpec, req.DocumentIDs, req.Input)
		if err != nil {
			return nil, err
		}
	}
	if collectionKind == document.CollectionKindCashflowImport && isCashflowBillAction(actionSpec.ActionType) {
		resolved, err = s.resolveCashflowBillActionSpec(ctx, coll.UserID, req.CollectionID, collectionKind, sourceFormat, actionSpec, req.DocumentIDs)
		if err != nil {
			return nil, err
		}
	}
	if collectionKind == document.CollectionKindBukpotRequestGSTDeductionMT && strings.EqualFold(strings.TrimSpace(actionSpec.ActionType), "request_bukpot_gst_deduction_mt") {
		resolved, err = s.resolveBukpotRequestActionSpec(ctx, coll.UserID, req.CollectionID, collectionKind, sourceFormat, actionSpec, req.DocumentIDs)
		if err != nil {
			return nil, err
		}
	}

	return &resolved, nil
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
			DocumentTag:   doc.DocumentTag,
			SourceSHA256:  doc.SourceSHA256,
			NormalizedRef: doc.NormalizedRef,
			AuditRef:      doc.AuditRef,
			RawRef:        doc.RawRef,
		})
	}

	processor, ok := s.processors.Get(document.ProcessorKey{
		CollectionKind: action.CollectionKind,
		SourceFormat:   action.SourceFormat,
	})
	if !ok {
		processor = document.NewNoopProcessor(document.ProcessorKey{
			CollectionKind: action.CollectionKind,
			SourceFormat:   action.SourceFormat,
		})
	}

	runInput := action.InputJSON
	if action.CollectionKind == document.CollectionKindCashflowImport {
		runInput = s.resolveCashflowRuntimeInput(action.UserID, action.ActionType, runInput)
		runInput = s.resolveCashflowBillRuntimeInput(action.UserID, action.ActionType, runInput)
	}

	runReq := document.ActionRequest{
		ActionID:       action.ID,
		UserID:         action.UserID,
		CollectionID:   action.CollectionID,
		CollectionKind: action.CollectionKind,
		SourceFormat:   action.SourceFormat,
		ActionType:     action.ActionType,
		SnapshotDocID:  docIDs,
		SnapshotDocs:   snapshotPayload,
		Input:          runInput,
		RequestedAt:    startedAt,
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

	if !coll.IsCollection() || coll.CollectionKind == nil {
		return nil, dcollection.ErrInvalidNodeType
	}

	return coll, nil
}

func applyFrozenCollectionSpec(spec document.CollectionSpec) document.CollectionSpec {
	for idx := range spec.Actions {
		spec.Actions[idx].State.Enabled = false
		spec.Actions[idx].State.Code = "COLLECTION_FROZEN"
		spec.Actions[idx].State.Message = "Collection sudah freeze dan tidak bisa menjalankan action baru."
	}
	return spec
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

		warningJSON, _ := json.Marshal(limitActionWarnings(warnings))
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

func limitActionWarnings(warnings []string) []string {
	if len(warnings) <= maxStoredItemWarnings {
		return warnings
	}

	trimmed := append([]string(nil), warnings[:maxStoredItemWarnings]...)
	trimmed = append(trimmed, fmt.Sprintf("+%d warning lainnya disembunyikan", len(warnings)-maxStoredItemWarnings))
	return trimmed
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

func normalizeAllowedStatuses(input []string) ([]string, error) {
	if len(input) == 0 {
		return nil, ErrInvalidActionSpec
	}
	out := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, raw := range input {
		status := strings.ToLower(strings.TrimSpace(raw))
		switch status {
		case "ready", "warning":
		default:
			return nil, ErrInvalidActionSpec
		}
		if _, exists := seen[status]; exists {
			continue
		}
		seen[status] = struct{}{}
		out = append(out, status)
	}
	if len(out) == 0 {
		return nil, ErrInvalidActionSpec
	}
	return out, nil
}

func normalizeSnapshotStatuses(input []string, allowed []string) ([]string, error) {
	if len(allowed) == 0 {
		return nil, ErrInvalidActionSpec
	}

	if len(input) == 0 {
		return append([]string(nil), allowed...), nil
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, status := range allowed {
		allowedSet[status] = struct{}{}
	}

	selected := make(map[string]struct{}, len(input))
	for _, status := range input {
		normalized := strings.ToLower(strings.TrimSpace(status))
		if _, ok := allowedSet[normalized]; !ok {
			return nil, ErrInvalidDocumentStatus
		}
		selected[normalized] = struct{}{}
	}

	out := make([]string, 0, len(selected))
	for _, status := range allowed {
		if _, ok := selected[status]; ok {
			out = append(out, status)
		}
	}
	return out, nil
}

func normalizeAndValidateActionInput(
	raw json.RawMessage,
	form *document.FormSpec,
	artifactInputs []document.ActionArtifactInputSpec,
) (json.RawMessage, error) {
	fieldSpecs := flattenFormFields(form)
	if len(fieldSpecs) == 0 && len(artifactInputs) == 0 {
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
			return nil, nil
		}

		payload := map[string]any{}
		if err := json.Unmarshal(trimmed, &payload); err != nil {
			return nil, fmt.Errorf("%w: input must be a JSON object", ErrInvalidActionParams)
		}
		if len(payload) > 0 {
			return nil, fmt.Errorf("%w: action does not accept input", ErrInvalidActionParams)
		}
		return nil, nil
	}

	specByKey := make(map[string]document.FormFieldSpec, len(fieldSpecs))
	for _, spec := range fieldSpecs {
		key := strings.TrimSpace(spec.Key)
		if key == "" {
			return nil, fmt.Errorf("%w: field key cannot be empty", ErrInvalidActionSpec)
		}
		specByKey[key] = spec
	}

	artifactByKey := make(map[string]document.ActionArtifactInputSpec, len(artifactInputs))
	for _, artifactInput := range artifactInputs {
		key := strings.TrimSpace(artifactInput.Key)
		if key == "" {
			return nil, fmt.Errorf("%w: artifact input key cannot be empty", ErrInvalidActionSpec)
		}
		artifactByKey[key] = artifactInput
	}

	trimmed := bytes.TrimSpace(raw)
	payload := map[string]any{}
	if !(len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))) {
		if err := json.Unmarshal(trimmed, &payload); err != nil {
			return nil, fmt.Errorf("%w: input must be a JSON object", ErrInvalidActionParams)
		}
		if payload == nil {
			payload = map[string]any{}
		}
	}

	for key := range payload {
		if _, ok := specByKey[key]; ok {
			continue
		}
		if _, ok := artifactByKey[key]; ok {
			continue
		}
		if _, ok := specByKey[key]; !ok {
			return nil, fmt.Errorf("%w: unknown input %q", ErrInvalidActionParams, key)
		}
	}

	normalized := make(map[string]any, len(payload))
	for _, spec := range fieldSpecs {
		value, exists := payload[spec.Key]
		if (!exists || value == nil) && spec.DefaultValue != nil {
			value = spec.DefaultValue
			exists = true
		}
		if !exists || value == nil {
			if spec.Required {
				return nil, fmt.Errorf("%w: missing required input %q", ErrInvalidActionParams, spec.Key)
			}
			continue
		}

		normalizedValue, normalizeErr := normalizeActionInputValue(spec, value)
		if normalizeErr != nil {
			return nil, normalizeErr
		}
		if spec.Required && isEmptyActionInputValue(normalizedValue) {
			return nil, fmt.Errorf("%w: input %q cannot be empty", ErrInvalidActionParams, spec.Key)
		}
		if !spec.Required && isEmptyActionInputValue(normalizedValue) {
			continue
		}
		if optionsErr := validateActionInputOptions(spec, normalizedValue); optionsErr != nil {
			return nil, optionsErr
		}

		normalized[spec.Key] = normalizedValue
	}

	for _, artifactInput := range artifactInputs {
		value, exists := payload[artifactInput.Key]
		if !exists || value == nil {
			if artifactInput.Required {
				return nil, fmt.Errorf("%w: missing required input %q", ErrInvalidActionParams, artifactInput.Key)
			}
			continue
		}

		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%w: input %q must be string", ErrInvalidActionParams, artifactInput.Key)
		}
		s = strings.TrimSpace(s)
		if artifactInput.Required && s == "" {
			return nil, fmt.Errorf("%w: input %q cannot be empty", ErrInvalidActionParams, artifactInput.Key)
		}
		if s == "" {
			continue
		}
		normalized[artifactInput.Key] = s
	}

	if ruleErr := validateActionInputRules(normalized, fieldSpecs); ruleErr != nil {
		return nil, ruleErr
	}

	if len(normalized) == 0 {
		return nil, nil
	}

	b, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to encode normalized input", ErrInvalidActionParams)
	}
	return b, nil
}

func flattenFormFields(form *document.FormSpec) []document.FormFieldSpec {
	if form == nil {
		return nil
	}

	fields := make([]document.FormFieldSpec, 0)
	seen := map[string]struct{}{}
	appendField := func(field document.FormFieldSpec) {
		normalizedKey := strings.ToLower(strings.TrimSpace(field.Key))
		if normalizedKey == "" {
			return
		}
		if _, ok := seen[normalizedKey]; ok {
			return
		}
		seen[normalizedKey] = struct{}{}
		fields = append(fields, field)
	}
	for _, section := range form.Sections {
		for _, field := range section.Fields {
			appendField(field)
		}
	}
	for _, group := range form.VariantGroups {
		for _, variant := range group.Variants {
			for _, section := range variant.Sections {
				for _, field := range section.Fields {
					appendField(field)
				}
			}
		}
	}
	return fields
}

func normalizeActionInputValue(spec document.FormFieldSpec, value any) (any, error) {
	switch strings.ToLower(strings.TrimSpace(spec.Kind)) {
	case document.FormFieldKindText, document.FormFieldKindTextarea, document.FormFieldKindTemplate:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%w: input %q must be string", ErrInvalidActionParams, spec.Key)
		}
		return strings.TrimSpace(s), nil

	case document.FormFieldKindNumber:
		switch n := value.(type) {
		case float64:
			return n, nil
		case int:
			return float64(n), nil
		case int64:
			return float64(n), nil
		case json.Number:
			f, err := n.Float64()
			if err != nil {
				return nil, fmt.Errorf("%w: input %q must be number", ErrInvalidActionParams, spec.Key)
			}
			return f, nil
		default:
			return nil, fmt.Errorf("%w: input %q must be number", ErrInvalidActionParams, spec.Key)
		}

	case document.FormFieldKindCheckbox:
		b, ok := value.(bool)
		if !ok {
			return nil, fmt.Errorf("%w: input %q must be boolean", ErrInvalidActionParams, spec.Key)
		}
		return b, nil

	case document.FormFieldKindSelect:
		s, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("%w: input %q must be string", ErrInvalidActionParams, spec.Key)
		}
		return strings.TrimSpace(s), nil

	default:
		return nil, fmt.Errorf("%w: unsupported field kind %q for key %q", ErrInvalidActionSpec, spec.Kind, spec.Key)
	}
}

func validateActionInputOptions(spec document.FormFieldSpec, value any) error {
	if len(spec.Options) == 0 {
		return nil
	}
	current := fmt.Sprint(value)
	for _, option := range spec.Options {
		if option.Value == current {
			return nil
		}
	}
	return fmt.Errorf("%w: input %q has unsupported value %q", ErrInvalidActionParams, spec.Key, current)
}

func validateActionInputRules(input map[string]any, specs []document.FormFieldSpec) error {
	for _, spec := range specs {
		for _, rule := range spec.Rules {
			switch strings.ToLower(strings.TrimSpace(rule.Type)) {
			case document.FormFieldRuleRequiredIf:
				if !isActionInputRuleConditionMet(input, rule) {
					continue
				}
				value, exists := input[spec.Key]
				if !exists || isEmptyActionInputValue(value) {
					msg := strings.TrimSpace(rule.Message)
					if msg == "" {
						msg = fmt.Sprintf("input %q is required by rule", spec.Key)
					}
					return fmt.Errorf("%w: %s", ErrInvalidActionParams, msg)
				}
			default:
				return fmt.Errorf("%w: unsupported rule type %q for key %q", ErrInvalidActionSpec, rule.Type, spec.Key)
			}
		}
	}
	return nil
}

func isActionInputRuleConditionMet(input map[string]any, rule document.FormFieldRuleSpec) bool {
	field := strings.TrimSpace(rule.Field)
	if field == "" {
		return false
	}
	left, ok := input[field]
	if !ok {
		return false
	}
	expected := strings.TrimSpace(rule.Equals)
	if expected == "" {
		return !isEmptyActionInputValue(left)
	}
	return strings.EqualFold(strings.TrimSpace(fmt.Sprint(left)), expected)
}

func isEmptyActionInputValue(value any) bool {
	if value == nil {
		return true
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
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

func (s *Service) resolveCashflowActionSpec(
	ctx context.Context,
	profileID string,
	collectionID string,
	collectionKind document.CollectionKind,
	sourceFormat document.SourceFormat,
	actionSpec document.ActionSpec,
	documentIDs []string,
	input json.RawMessage,
) (document.ActionSpec, error) {
	if s.cashflowProfileConfig != nil {
		var configKey appcashflow.ProfileConfigKey
		switch strings.TrimSpace(actionSpec.ActionType) {
		case "export_cashflow_receive_money":
			configKey = appcashflow.ProfileConfigReceiveMoney
		default:
			configKey = appcashflow.ProfileConfigSpendMoney
		}

		cfg, err := s.cashflowProfileConfig.Load(profileID, configKey)
		if err == nil {
			selectedFormatRaw := strings.TrimSpace(extractStringInput(input, "cashflowFormat"))
			selectedFormat := appcashflow.NormalizeFormat(selectedFormatRaw)
			if selectedFormatRaw == "" {
				selectedFormat = appcashflow.StandardFormat
				selectedFormatRaw = string(selectedFormat)
			}
			if selectedFormat == "" {
				selectedFormat = appcashflow.StandardFormat
				selectedFormatRaw = string(selectedFormat)
			}

			if field, ok := findFormField(actionSpec.Form, "cashflowFormat"); ok {
				field.DefaultValue = selectedFormatRaw
				updateFormField(actionSpec.Form, field)
			}
			updateFormVariantGroupDefault(actionSpec.Form, "cashflowFormat", selectedFormatRaw)

			for _, variantKey := range []string{string(appcashflow.StandardFormat), string(appcashflow.InfluencerFormat)} {
				updateVariantFormValues(
					actionSpec.Form,
					"cashflowFormat",
					variantKey,
					stringMapToAnyMap(appcashflow.ResolveProfileConfigFormValues(cfg, configKey, variantKey)),
				)
			}
		}
	}

	selectedFormatRaw := strings.TrimSpace(extractStringInput(input, "cashflowFormat"))
	if selectedFormatRaw == "" {
		if field, ok := findFormField(actionSpec.Form, "cashflowFormat"); ok {
			selectedFormatRaw = formFieldDefaultString(field.DefaultValue)
		}
	}
	selectedFormat := appcashflow.NormalizeFormat(selectedFormatRaw)
	if selectedFormat == "" {
		selectedFormat = appcashflow.StandardFormat
	}

	field, ok := findFormField(actionSpec.Form, "sheetName")
	if !ok {
		return actionSpec, nil
	}

	normalizedIDs, err := normalizeDocumentIDs(documentIDs)
	if err != nil {
		return actionSpec, err
	}
	if len(normalizedIDs) == 0 {
		field.Options = nil
		field.HelpText = "Pilih minimal satu dokumen cashflow untuk melihat sheet yang tersedia."
		field.State.Disabled = true
		field.State.Message = "Sheet akan tersedia setelah Anda memilih dokumen cashflow."
		updateFormField(actionSpec.Form, field)
		return actionSpec, nil
	}

	snapshotDocs, err := s.repo.ListSnapshotDocumentsByIDs(ctx, collectionID, collectionKind, sourceFormat, normalizedIDs)
	if err != nil {
		return actionSpec, err
	}
	if len(snapshotDocs) != len(normalizedIDs) {
		return actionSpec, ErrSnapshotDocNotFound
	}
	if allowed, allowedErr := normalizeAllowedStatuses(actionSpec.Selection.AllowedStatus); allowedErr == nil {
		if err := validateSnapshotStatuses(snapshotDocs, allowed); err != nil {
			return actionSpec, err
		}
	}

	commonSheetNames, defaultSheet, defaultHeaderRow, resolveErr := s.resolveCommonCashflowSheets(ctx, collectionID, snapshotDocs)
	if resolveErr != nil {
		return actionSpec, resolveErr
	}

	if len(commonSheetNames) == 0 {
		actionSpec.State.Enabled = false
		actionSpec.State.Code = "CASHFLOW_COMMON_SHEET_NOT_FOUND"
		actionSpec.State.Message = "Dokumen terpilih tidak memiliki nama sheet yang sama untuk diproses bersama."
		field.Options = nil
		field.HelpText = actionSpec.State.Message
		field.State.Disabled = true
		field.State.Message = actionSpec.State.Message
		field.DefaultValue = ""
		updateFormField(actionSpec.Form, field)
		for _, variantKey := range []string{string(appcashflow.StandardFormat), string(appcashflow.InfluencerFormat)} {
			updateVariantFormValues(actionSpec.Form, "cashflowFormat", variantKey, map[string]any{
				"sheetName": "",
			})
		}
		return actionSpec, nil
	}

	field.Options = make([]document.FormFieldOption, 0, len(commonSheetNames))
	for _, sheetName := range commonSheetNames {
		field.Options = append(field.Options, document.FormFieldOption{
			Label: sheetName,
			Value: sheetName,
		})
	}
	field.HelpText = "Sheet yang tersedia pada semua dokumen terpilih."
	field.State.Disabled = false
	field.State.Message = ""
	field.DefaultValue = pickPreferredSheetName(
		commonSheetNames,
		formVariantValueString(actionSpec.Form, "cashflowFormat", string(selectedFormat), "sheetName"),
		defaultSheet,
	)
	updateFormField(actionSpec.Form, field)

	for _, variantKey := range []string{string(appcashflow.StandardFormat), string(appcashflow.InfluencerFormat)} {
		nextValues := map[string]any{
			"sheetName": pickPreferredSheetName(
				commonSheetNames,
				formVariantValueString(actionSpec.Form, "cashflowFormat", variantKey, "sheetName"),
				defaultSheet,
			),
		}
		if defaultHeaderRow > 0 && strings.TrimSpace(formVariantValueString(actionSpec.Form, "cashflowFormat", variantKey, "headerRowNumber")) == "" {
			nextValues["headerRowNumber"] = strconv.Itoa(defaultHeaderRow)
		}
		updateVariantFormValues(actionSpec.Form, "cashflowFormat", variantKey, nextValues)
	}

	if headerField, ok := findFormField(actionSpec.Form, "headerRowNumber"); ok && defaultHeaderRow > 0 {
		if formFieldDefaultString(headerField.DefaultValue) == "" {
			headerField.DefaultValue = strconv.Itoa(defaultHeaderRow)
		}
		updateFormField(actionSpec.Form, headerField)
	}

	return actionSpec, nil
}

func (s *Service) resolveCashflowRuntimeInput(
	profileID string,
	actionType string,
	input json.RawMessage,
) json.RawMessage {
	if !isCashflowExportAction(actionType) {
		return input
	}
	if s.cashflowProfileConfig == nil {
		return input
	}

	var configKey appcashflow.ProfileConfigKey
	switch strings.TrimSpace(actionType) {
	case "export_cashflow_receive_money":
		configKey = appcashflow.ProfileConfigReceiveMoney
	default:
		configKey = appcashflow.ProfileConfigSpendMoney
	}

	cfg, err := s.cashflowProfileConfig.Load(profileID, configKey)
	if err != nil {
		return input
	}

	selectedFormatRaw := strings.TrimSpace(extractStringInput(input, "cashflowFormat"))
	if selectedFormatRaw == "" {
		selectedFormatRaw = string(appcashflow.StandardFormat)
	}
	runtimeValues := appcashflow.ResolveProfileConfigRuntimeValues(cfg, selectedFormatRaw)
	if len(runtimeValues) == 0 {
		return input
	}

	payload := map[string]any{}
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
		if err := json.Unmarshal(trimmed, &payload); err != nil {
			return input
		}
	}

	for key, value := range runtimeValues {
		if strings.TrimSpace(value) == "" {
			continue
		}
		payload[key] = value
	}
	if len(payload) == 0 {
		return input
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return input
	}
	return b
}

func (s *Service) resolveCashflowBillActionSpec(
	ctx context.Context,
	profileID string,
	collectionID string,
	collectionKind document.CollectionKind,
	sourceFormat document.SourceFormat,
	actionSpec document.ActionSpec,
	documentIDs []string,
) (document.ActionSpec, error) {
	if s.cashflowBillProfileConfig != nil {
		configKey := cashflowBillProfileKeyFromAction(actionSpec.ActionType)
		cfg, err := s.cashflowBillProfileConfig.Load(profileID, configKey)
		if err == nil {
			for key, value := range appcashflowbill.ResolveProfileConfigValues(cfg) {
				field, ok := findFormField(actionSpec.Form, key)
				if !ok {
					continue
				}
				field.DefaultValue = strings.TrimSpace(value)
				updateFormField(actionSpec.Form, field)
			}
		}
	}

	field, ok := findFormField(actionSpec.Form, "sheetName")
	if !ok {
		return actionSpec, nil
	}

	normalizedIDs, err := normalizeDocumentIDs(documentIDs)
	if err != nil {
		return actionSpec, err
	}
	if len(normalizedIDs) == 0 {
		field.Options = nil
		field.HelpText = "Pilih minimal satu dokumen cashflow untuk melihat sheet yang tersedia."
		field.State.Disabled = true
		field.State.Message = "Sheet akan tersedia setelah Anda memilih dokumen cashflow."
		updateFormField(actionSpec.Form, field)
		return actionSpec, nil
	}

	snapshotDocs, err := s.repo.ListSnapshotDocumentsByIDs(ctx, collectionID, collectionKind, sourceFormat, normalizedIDs)
	if err != nil {
		return actionSpec, err
	}
	if len(snapshotDocs) != len(normalizedIDs) {
		return actionSpec, ErrSnapshotDocNotFound
	}
	if allowed, allowedErr := normalizeAllowedStatuses(actionSpec.Selection.AllowedStatus); allowedErr == nil {
		if err := validateSnapshotStatuses(snapshotDocs, allowed); err != nil {
			return actionSpec, err
		}
	}

	commonSheetNames, defaultSheet, defaultHeaderRow, resolveErr := s.resolveCommonSpreadsheetSheets(ctx, collectionID, snapshotDocs)
	if resolveErr != nil {
		return actionSpec, resolveErr
	}
	if len(commonSheetNames) == 0 {
		actionSpec.State.Enabled = false
		actionSpec.State.Code = "CASHFLOW_BILLS_COMMON_SHEET_NOT_FOUND"
		actionSpec.State.Message = "Dokumen terpilih tidak memiliki nama sheet yang sama untuk diproses bersama."
		field.Options = nil
		field.HelpText = actionSpec.State.Message
		field.State.Disabled = true
		field.State.Message = actionSpec.State.Message
		field.DefaultValue = ""
		updateFormField(actionSpec.Form, field)
		return actionSpec, nil
	}

	field.Options = make([]document.FormFieldOption, 0, len(commonSheetNames))
	for _, sheetName := range commonSheetNames {
		field.Options = append(field.Options, document.FormFieldOption{
			Label: sheetName,
			Value: sheetName,
		})
	}
	field.HelpText = "Sheet yang tersedia pada semua dokumen terpilih."
	field.State.Disabled = false
	field.State.Message = ""
	field.DefaultValue = pickPreferredSheetName(commonSheetNames, formFieldDefaultString(field.DefaultValue), defaultSheet)
	updateFormField(actionSpec.Form, field)

	if headerField, ok := findFormField(actionSpec.Form, "headerRowNumber"); ok && defaultHeaderRow > 0 {
		if formFieldDefaultString(headerField.DefaultValue) == "" {
			headerField.DefaultValue = strconv.Itoa(defaultHeaderRow)
		}
		updateFormField(actionSpec.Form, headerField)
	}

	return actionSpec, nil
}

func (s *Service) resolveCashflowBillRuntimeInput(
	profileID string,
	actionType string,
	input json.RawMessage,
) json.RawMessage {
	if !isCashflowBillAction(actionType) || s.cashflowBillProfileConfig == nil {
		return input
	}

	cfg, err := s.cashflowBillProfileConfig.Load(profileID, cashflowBillProfileKeyFromAction(actionType))
	if err != nil {
		return input
	}

	values := appcashflowbill.ResolveProfileConfigValues(cfg)
	if len(values) == 0 {
		return input
	}

	payload := map[string]any{}
	trimmed := bytes.TrimSpace(input)
	if len(trimmed) > 0 && !bytes.Equal(trimmed, []byte("null")) {
		if err := json.Unmarshal(trimmed, &payload); err != nil {
			return input
		}
	}

	for key, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, exists := payload[key]; exists && strings.TrimSpace(fmt.Sprint(payload[key])) != "" {
			continue
		}
		payload[key] = value
	}
	if len(payload) == 0 {
		return input
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return input
	}
	return b
}

func isCashflowExportAction(actionType string) bool {
	switch strings.TrimSpace(actionType) {
	case "export_cashflow_myob", "export_cashflow_spend_money", "export_cashflow_receive_money":
		return true
	default:
		return false
	}
}

func isCashflowBillAction(actionType string) bool {
	switch strings.TrimSpace(actionType) {
	case "cashflow_to_pay_bills", "cashflow_to_receive_payments":
		return true
	default:
		return false
	}
}

func cashflowBillProfileKeyFromAction(actionType string) appcashflowbill.ProfileConfigKey {
	switch strings.TrimSpace(actionType) {
	case "cashflow_to_receive_payments":
		return appcashflowbill.ProfileConfigReceivePayments
	default:
		return appcashflowbill.ProfileConfigPayBills
	}
}

func (s *Service) resolveBukpotRequestActionSpec(
	ctx context.Context,
	profileID string,
	collectionID string,
	collectionKind document.CollectionKind,
	sourceFormat document.SourceFormat,
	actionSpec document.ActionSpec,
	documentIDs []string,
) (document.ActionSpec, error) {
	if s.bukpotRequestConfig != nil {
		cfg, err := s.bukpotRequestConfig.Load(profileID)
		if err == nil {
			for _, item := range cfg.Fields {
				field, ok := findFormField(actionSpec.Form, item.Key)
				if !ok {
					continue
				}
				field.DefaultValue = strings.TrimSpace(item.Value)
				updateFormField(actionSpec.Form, field)
			}
			if field, ok := findFormField(actionSpec.Form, "headerRowNumber"); ok && cfg.Defaults.HeaderRowNumber > 0 {
				field.DefaultValue = strconv.Itoa(cfg.Defaults.HeaderRowNumber)
				updateFormField(actionSpec.Form, field)
			}
			if field, ok := findFormField(actionSpec.Form, "sheetName"); ok && strings.TrimSpace(cfg.Defaults.SheetName) != "" {
				field.DefaultValue = strings.TrimSpace(cfg.Defaults.SheetName)
				updateFormField(actionSpec.Form, field)
			}
		}
	}

	field, ok := findFormField(actionSpec.Form, "sheetName")
	if !ok {
		return actionSpec, nil
	}
	preferredSheetName := formFieldDefaultString(field.DefaultValue)

	normalizedIDs, err := normalizeDocumentIDs(documentIDs)
	if err != nil {
		return actionSpec, err
	}
	if len(normalizedIDs) == 0 {
		field.Options = nil
		field.HelpText = "Pilih minimal satu dokumen request bukpot untuk melihat sheet yang tersedia."
		field.State.Disabled = true
		field.State.Message = "Sheet akan tersedia setelah Anda memilih dokumen source."
		updateFormField(actionSpec.Form, field)
		return actionSpec, nil
	}

	snapshotDocs, err := s.repo.ListSnapshotDocumentsByIDs(ctx, collectionID, collectionKind, sourceFormat, normalizedIDs)
	if err != nil {
		return actionSpec, err
	}
	if len(snapshotDocs) != len(normalizedIDs) {
		return actionSpec, ErrSnapshotDocNotFound
	}
	if allowed, allowedErr := normalizeAllowedStatuses(actionSpec.Selection.AllowedStatus); allowedErr == nil {
		if err := validateSnapshotStatuses(snapshotDocs, allowed); err != nil {
			return actionSpec, err
		}
	}

	commonSheetNames, defaultSheet, defaultHeaderRow, resolveErr := s.resolveCommonSpreadsheetSheets(ctx, collectionID, snapshotDocs)
	if resolveErr != nil {
		return actionSpec, resolveErr
	}
	if len(commonSheetNames) == 0 {
		actionSpec.State.Enabled = false
		actionSpec.State.Code = "BUKPOT_REQUEST_COMMON_SHEET_NOT_FOUND"
		actionSpec.State.Message = "Dokumen terpilih tidak memiliki nama sheet yang sama untuk diproses bersama."
		field.Options = nil
		field.HelpText = actionSpec.State.Message
		field.State.Disabled = true
		field.State.Message = actionSpec.State.Message
		field.DefaultValue = ""
		updateFormField(actionSpec.Form, field)
		return actionSpec, nil
	}

	field.Options = make([]document.FormFieldOption, 0, len(commonSheetNames))
	for _, sheetName := range commonSheetNames {
		field.Options = append(field.Options, document.FormFieldOption{
			Label: sheetName,
			Value: sheetName,
		})
	}
	field.HelpText = "Sheet yang tersedia pada semua dokumen terpilih."
	field.State.Disabled = false
	field.State.Message = ""
	field.DefaultValue = pickPreferredSheetName(commonSheetNames, preferredSheetName, defaultSheet)
	updateFormField(actionSpec.Form, field)

	if headerField, ok := findFormField(actionSpec.Form, "headerRowNumber"); ok && defaultHeaderRow > 0 {
		headerField.DefaultValue = strconv.Itoa(defaultHeaderRow)
		updateFormField(actionSpec.Form, headerField)
	}

	return actionSpec, nil
}

func (s *Service) resolveCommonCashflowSheets(
	ctx context.Context,
	collectionID string,
	snapshotDocs []SnapshotDocument,
) ([]string, string, int, error) {
	return s.resolveCommonSpreadsheetSheets(ctx, collectionID, snapshotDocs)
}

func (s *Service) resolveCommonSpreadsheetSheets(
	ctx context.Context,
	collectionID string,
	snapshotDocs []SnapshotDocument,
) ([]string, string, int, error) {
	var common []string
	headerBySheet := map[string]int{}
	for idx, doc := range snapshotDocs {
		workbook, err := document.LoadSpreadsheetWorkbook(ctx, s.fileStore, collectionID, doc.NormalizedRef)
		if err != nil {
			return nil, "", 0, err
		}

		currentNames := make([]string, 0, len(workbook.Sheets))
		currentHeaders := map[string]int{}
		for _, sheet := range workbook.Sheets {
			name := strings.TrimSpace(sheet.Name)
			if name == "" {
				continue
			}
			currentNames = append(currentNames, name)
			currentHeaders[name] = sheet.HeaderRowIndex
		}

		if idx == 0 {
			common = append(common, currentNames...)
			for name, row := range currentHeaders {
				headerBySheet[name] = row
			}
			continue
		}

		nextCommon := make([]string, 0, len(common))
		for _, existing := range common {
			row, ok := currentHeaders[existing]
			if !ok {
				continue
			}
			nextCommon = append(nextCommon, existing)
			if headerBySheet[existing] <= 0 || row <= 0 || headerBySheet[existing] != row {
				headerBySheet[existing] = 1
			}
		}
		common = nextCommon
	}

	if len(common) == 0 {
		return nil, "", 0, nil
	}

	defaultSheet := common[0]
	defaultHeader := headerBySheet[defaultSheet]
	if defaultHeader <= 0 {
		defaultHeader = 1
	}

	return common, defaultSheet, defaultHeader, nil
}

func findFormField(form *document.FormSpec, key string) (document.FormFieldSpec, bool) {
	if form == nil {
		return document.FormFieldSpec{}, false
	}
	for _, section := range form.Sections {
		for _, field := range section.Fields {
			if strings.EqualFold(strings.TrimSpace(field.Key), strings.TrimSpace(key)) {
				return field, true
			}
		}
	}
	return document.FormFieldSpec{}, false
}

func findVariantFormField(
	form *document.FormSpec,
	groupFieldKey string,
	variantKey string,
	fieldKey string,
) (document.FormFieldSpec, bool) {
	if form == nil {
		return document.FormFieldSpec{}, false
	}
	for _, group := range form.VariantGroups {
		if !strings.EqualFold(strings.TrimSpace(group.FieldKey), strings.TrimSpace(groupFieldKey)) {
			continue
		}
		for _, variant := range group.Variants {
			if !strings.EqualFold(strings.TrimSpace(variant.Key), strings.TrimSpace(variantKey)) {
				continue
			}
			for _, field := range variant.Sections {
				for _, item := range field.Fields {
					if strings.EqualFold(strings.TrimSpace(item.Key), strings.TrimSpace(fieldKey)) {
						return item, true
					}
				}
			}
		}
	}
	return document.FormFieldSpec{}, false
}

func updateFormField(form *document.FormSpec, updated document.FormFieldSpec) {
	if form == nil {
		return
	}
	for sectionIndex := range form.Sections {
		for fieldIndex := range form.Sections[sectionIndex].Fields {
			if strings.EqualFold(strings.TrimSpace(form.Sections[sectionIndex].Fields[fieldIndex].Key), strings.TrimSpace(updated.Key)) {
				form.Sections[sectionIndex].Fields[fieldIndex] = updated
				return
			}
		}
	}
}

func updateFormVariantGroupDefault(form *document.FormSpec, groupFieldKey string, defaultVariantKey string) {
	if form == nil {
		return
	}
	for groupIndex := range form.VariantGroups {
		if strings.EqualFold(strings.TrimSpace(form.VariantGroups[groupIndex].FieldKey), strings.TrimSpace(groupFieldKey)) {
			form.VariantGroups[groupIndex].DefaultVariantKey = strings.TrimSpace(defaultVariantKey)
			return
		}
	}
}

func updateVariantFormValues(
	form *document.FormSpec,
	groupFieldKey string,
	variantKey string,
	updated map[string]any,
) {
	if form == nil {
		return
	}
	for groupIndex := range form.VariantGroups {
		if !strings.EqualFold(strings.TrimSpace(form.VariantGroups[groupIndex].FieldKey), strings.TrimSpace(groupFieldKey)) {
			continue
		}
		for variantIndex := range form.VariantGroups[groupIndex].Variants {
			if !strings.EqualFold(strings.TrimSpace(form.VariantGroups[groupIndex].Variants[variantIndex].Key), strings.TrimSpace(variantKey)) {
				continue
			}
			if form.VariantGroups[groupIndex].Variants[variantIndex].Values == nil {
				form.VariantGroups[groupIndex].Variants[variantIndex].Values = map[string]any{}
			}
			for key, value := range updated {
				normalizedKey := strings.TrimSpace(key)
				if normalizedKey == "" {
					continue
				}
				form.VariantGroups[groupIndex].Variants[variantIndex].Values[normalizedKey] = value
			}
			return
		}
	}
}

func formVariantValueString(
	form *document.FormSpec,
	groupFieldKey string,
	variantKey string,
	fieldKey string,
) string {
	if form == nil {
		return ""
	}
	for _, group := range form.VariantGroups {
		if !strings.EqualFold(strings.TrimSpace(group.FieldKey), strings.TrimSpace(groupFieldKey)) {
			continue
		}
		for _, variant := range group.Variants {
			if !strings.EqualFold(strings.TrimSpace(variant.Key), strings.TrimSpace(variantKey)) {
				continue
			}
			if variant.Values != nil {
				if value, ok := variant.Values[strings.TrimSpace(fieldKey)]; ok {
					return formFieldDefaultString(value)
				}
			}
			for _, section := range variant.Sections {
				for _, field := range section.Fields {
					if strings.EqualFold(strings.TrimSpace(field.Key), strings.TrimSpace(fieldKey)) {
						return formFieldDefaultString(field.DefaultValue)
					}
				}
			}
			return ""
		}
	}
	return ""
}

func stringMapToAnyMap(values map[string]string) map[string]any {
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func updateVariantFormField(
	form *document.FormSpec,
	groupFieldKey string,
	variantKey string,
	updated document.FormFieldSpec,
) {
	if form == nil {
		return
	}
	for groupIndex := range form.VariantGroups {
		if !strings.EqualFold(strings.TrimSpace(form.VariantGroups[groupIndex].FieldKey), strings.TrimSpace(groupFieldKey)) {
			continue
		}
		for variantIndex := range form.VariantGroups[groupIndex].Variants {
			if !strings.EqualFold(strings.TrimSpace(form.VariantGroups[groupIndex].Variants[variantIndex].Key), strings.TrimSpace(variantKey)) {
				continue
			}
			for sectionIndex := range form.VariantGroups[groupIndex].Variants[variantIndex].Sections {
				for fieldIndex := range form.VariantGroups[groupIndex].Variants[variantIndex].Sections[sectionIndex].Fields {
					if strings.EqualFold(strings.TrimSpace(form.VariantGroups[groupIndex].Variants[variantIndex].Sections[sectionIndex].Fields[fieldIndex].Key), strings.TrimSpace(updated.Key)) {
						form.VariantGroups[groupIndex].Variants[variantIndex].Sections[sectionIndex].Fields[fieldIndex] = updated
						return
					}
				}
			}
		}
	}
}

func pickPreferredSheetName(options []string, preferred string, fallback string) string {
	preferred = strings.TrimSpace(preferred)
	if preferred != "" {
		for _, option := range options {
			if strings.EqualFold(strings.TrimSpace(option), preferred) {
				return option
			}
		}
	}

	fallback = strings.TrimSpace(fallback)
	if fallback != "" {
		for _, option := range options {
			if strings.EqualFold(strings.TrimSpace(option), fallback) {
				return option
			}
		}
	}

	if len(options) == 0 {
		return ""
	}

	return options[0]
}

func formFieldDefaultString(value any) string {
	if value == nil {
		return ""
	}

	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func extractStringInput(input json.RawMessage, key string) string {
	if len(input) == 0 {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal(input, &payload); err != nil {
		return ""
	}

	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}

	return strings.TrimSpace(fmt.Sprint(raw))
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

func (s *Service) applyRuntimeRequirements(spec document.CollectionSpec, profileID string) document.CollectionSpec {
	actions := make([]document.ActionSpec, 0, len(spec.Actions))
	for _, item := range spec.Actions {
		actionSpec := item
		actionSpec.Requirements = append([]document.ActionRequirementSpec(nil), item.Requirements...)

		if spec.CollectionKind == document.CollectionKindInvoiceCompany &&
			strings.EqualFold(strings.TrimSpace(actionSpec.ActionType), "export_faktur_keluaran") {
			actionSpec = s.applyBuyerRegistryRequirement(actionSpec, profileID)
		}
		if spec.CollectionKind == document.CollectionKindCashflowImport &&
			isCashflowExportAction(actionSpec.ActionType) {
			actionSpec = s.applyCashflowTaxAccountRequirement(actionSpec, profileID)
			actionSpec = s.applyCashflowProfileRequirement(actionSpec, profileID)
		}
		if spec.CollectionKind == document.CollectionKindCashflowImport &&
			isCashflowBillAction(actionSpec.ActionType) {
			actionSpec = s.applyCashflowBillCategoryRequirement(actionSpec, profileID)
			actionSpec = s.applyCashflowBillProfileRequirement(actionSpec, profileID)
		}
		if spec.CollectionKind == document.CollectionKindBukpotRequestGSTDeductionMT &&
			strings.EqualFold(strings.TrimSpace(actionSpec.ActionType), "request_bukpot_gst_deduction_mt") {
			actionSpec = s.applyBukpotRequestConfigRequirement(actionSpec, profileID)
		}

		actions = append(actions, actionSpec)
	}
	spec.Actions = actions
	return spec
}

func (s *Service) applyBukpotActionProfileCollectionSpec(
	spec document.CollectionSpec,
	profileID string,
) document.CollectionSpec {
	if s.bukpotActionProfiles == nil {
		return spec
	}

	actions := make([]document.ActionSpec, 0, len(spec.Actions))
	for _, item := range spec.Actions {
		actions = append(actions, s.applyBukpotActionProfileDefaults(item, profileID, spec.CollectionKind))
	}
	spec.Actions = actions
	return spec
}

func (s *Service) applyBukpotActionProfileDefaults(
	actionSpec document.ActionSpec,
	profileID string,
	collectionKind document.CollectionKind,
) document.ActionSpec {
	if s.bukpotActionProfiles == nil || actionSpec.Form == nil {
		return actionSpec
	}

	profileKey, ok := appbukpot.ResolveActionProfileKey(string(collectionKind), actionSpec.ActionType)
	if !ok {
		return actionSpec
	}

	cfg, err := s.bukpotActionProfiles.Load(profileID, profileKey)
	if err != nil {
		return actionSpec
	}

	for _, item := range cfg.Fields {
		field, found := findFormField(actionSpec.Form, item.Key)
		if !found {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(field.Kind)) {
		case document.FormFieldKindCheckbox:
			field.DefaultValue = strings.EqualFold(strings.TrimSpace(item.Value), "true")
		case document.FormFieldKindNumber:
			if n, convErr := strconv.Atoi(strings.TrimSpace(item.Value)); convErr == nil {
				field.DefaultValue = float64(n)
			}
		default:
			field.DefaultValue = strings.TrimSpace(item.Value)
		}
		updateFormField(actionSpec.Form, field)
	}

	return actionSpec
}

func (s *Service) applyBuyerRegistryRequirement(actionSpec document.ActionSpec, profileID string) document.ActionSpec {
	status := appbuyer.BuyerRegistryStatus{
		Loaded:  false,
		Code:    "BUYER_REGISTRY_STATUS_UNAVAILABLE",
		Message: "Buyer registry checker tidak tersedia.",
	}
	if s.buyerRegistry != nil {
		status = s.buyerRegistry.Status(profileID)
	}

	updated := false
	for idx, requirement := range actionSpec.Requirements {
		if !strings.EqualFold(strings.TrimSpace(requirement.Key), "buyerRegistry") {
			continue
		}
		requirement.Satisfied = status.Loaded
		requirement.Code = strings.TrimSpace(status.Code)
		requirement.Message = strings.TrimSpace(status.Message)
		actionSpec.Requirements[idx] = requirement
		updated = true
	}

	if !updated {
		actionSpec.Requirements = append(actionSpec.Requirements, document.ActionRequirementSpec{
			Key:       "buyerRegistry",
			Label:     "Buyer Registry",
			Required:  true,
			Satisfied: status.Loaded,
			Code:      strings.TrimSpace(status.Code),
			Message:   strings.TrimSpace(status.Message),
		})
	}

	if !status.Loaded {
		actionSpec.State.Enabled = false
		actionSpec.State.Code = strings.TrimSpace(status.Code)
		reason := strings.TrimSpace(status.Message)
		if reason == "" {
			reason = "Buyer registry belum siap."
		}
		actionSpec.State.Message = reason
	}

	return actionSpec
}

func (s *Service) applyBukpotRequestConfigRequirement(actionSpec document.ActionSpec, profileID string) document.ActionSpec {
	status := appbukpot.RequestConfigStatus{
		Configured:    false,
		Code:          "BUKPOT_REQUEST_CONFIG_UNAVAILABLE",
		Message:       "Konfigurasi default profil request bukpot belum tersedia.",
		SchemaVersion: "1",
	}
	if s.bukpotRequestConfig != nil {
		status = s.bukpotRequestConfig.Status(profileID)
	}

	updated := false
	for idx, requirement := range actionSpec.Requirements {
		if !strings.EqualFold(strings.TrimSpace(requirement.Key), "bukpotRequestConfig") {
			continue
		}
		requirement.Satisfied = status.Configured
		requirement.Code = strings.TrimSpace(status.Code)
		requirement.Message = strings.TrimSpace(status.Message)
		actionSpec.Requirements[idx] = requirement
		updated = true
	}

	if !updated {
		actionSpec.Requirements = append(actionSpec.Requirements, document.ActionRequirementSpec{
			Key:       "bukpotRequestConfig",
			Label:     "Default Profil Request Bukpot GST Deduction MT",
			Required:  true,
			Satisfied: status.Configured,
			Code:      strings.TrimSpace(status.Code),
			Message:   strings.TrimSpace(status.Message),
		})
	}

	if !status.Configured {
		actionSpec.State.Enabled = false
		actionSpec.State.Code = strings.TrimSpace(status.Code)
		actionSpec.State.Message = strings.TrimSpace(status.Message)
	}

	return actionSpec
}

func (s *Service) applyCashflowTaxAccountRequirement(actionSpec document.ActionSpec, profileID string) document.ActionSpec {
	status := appcashflow.TaxAccountStatus{
		Loaded:  false,
		Code:    "CASHFLOW_TAX_ACCOUNTS_UNAVAILABLE",
		Message: "Master data tax accounts belum tersedia.",
	}
	if s.taxAccounts != nil {
		status = s.taxAccounts.Status(profileID)
	}

	updated := false
	for idx, requirement := range actionSpec.Requirements {
		if !strings.EqualFold(strings.TrimSpace(requirement.Key), "cashflowTaxAccounts") {
			continue
		}
		requirement.Satisfied = status.Loaded
		requirement.Code = strings.TrimSpace(status.Code)
		requirement.Message = strings.TrimSpace(status.Message)
		actionSpec.Requirements[idx] = requirement
		updated = true
	}

	if !updated {
		actionSpec.Requirements = append(actionSpec.Requirements, document.ActionRequirementSpec{
			Key:       "cashflowTaxAccounts",
			Label:     "Tax Accounts",
			Required:  true,
			Satisfied: status.Loaded,
			Code:      strings.TrimSpace(status.Code),
			Message:   strings.TrimSpace(status.Message),
		})
	}

	if !status.Loaded {
		actionSpec.State.Enabled = false
		actionSpec.State.Code = strings.TrimSpace(status.Code)
		actionSpec.State.Message = strings.TrimSpace(status.Message)
		if actionSpec.State.Message == "" {
			actionSpec.State.Message = "Master data tax accounts belum siap."
		}
	}

	return actionSpec
}

func (s *Service) applyCashflowProfileRequirement(actionSpec document.ActionSpec, profileID string) document.ActionSpec {
	configKey := appcashflow.ProfileConfigSpendMoney
	label := "Default Profil Cashflow Spend Money"
	defaultCode := "CASHFLOW_SPEND_PROFILE_UNAVAILABLE"
	defaultMessage := "Default profil cashflow spend money belum tersedia."

	if strings.EqualFold(strings.TrimSpace(actionSpec.ActionType), "export_cashflow_receive_money") {
		configKey = appcashflow.ProfileConfigReceiveMoney
		label = "Default Profil Cashflow Receive Money"
		defaultCode = "CASHFLOW_RECEIVE_PROFILE_UNAVAILABLE"
		defaultMessage = "Default profil cashflow receive money belum tersedia."
	}

	status := appcashflow.ProfileConfigStatus{
		Configured:    false,
		Code:          defaultCode,
		Message:       defaultMessage,
		SchemaVersion: "1",
	}
	if s.cashflowProfileConfig != nil {
		status = s.cashflowProfileConfig.Status(profileID, configKey)
	}

	updated := false
	for idx, requirement := range actionSpec.Requirements {
		if !strings.EqualFold(strings.TrimSpace(requirement.Key), "cashflowDefaultProfile") {
			continue
		}
		requirement.Label = label
		requirement.Satisfied = status.Configured
		requirement.Code = strings.TrimSpace(status.Code)
		requirement.Message = strings.TrimSpace(status.Message)
		actionSpec.Requirements[idx] = requirement
		updated = true
	}

	if !updated {
		actionSpec.Requirements = append(actionSpec.Requirements, document.ActionRequirementSpec{
			Key:       "cashflowDefaultProfile",
			Label:     label,
			Required:  true,
			Satisfied: status.Configured,
			Code:      strings.TrimSpace(status.Code),
			Message:   strings.TrimSpace(status.Message),
		})
	}

	if !status.Configured {
		actionSpec.State.Enabled = false
		actionSpec.State.Code = strings.TrimSpace(status.Code)
		if strings.TrimSpace(actionSpec.State.Message) == "" {
			actionSpec.State.Message = strings.TrimSpace(status.Message)
		}
	}

	return actionSpec
}

func (s *Service) applyCashflowBillCategoryRequirement(actionSpec document.ActionSpec, profileID string) document.ActionSpec {
	status := appcashflowbill.CategoryAccountStatus{
		Loaded:  false,
		Code:    "CASHFLOW_CATEGORY_ACCOUNTS_UNAVAILABLE",
		Message: "Master data category accounts belum tersedia.",
	}
	if s.cashflowBillCategories != nil {
		status = s.cashflowBillCategories.Status(profileID)
	}

	updated := false
	for idx, requirement := range actionSpec.Requirements {
		if !strings.EqualFold(strings.TrimSpace(requirement.Key), "cashflowCategoryAccounts") {
			continue
		}
		requirement.Satisfied = status.Loaded
		requirement.Code = strings.TrimSpace(status.Code)
		requirement.Message = strings.TrimSpace(status.Message)
		actionSpec.Requirements[idx] = requirement
		updated = true
	}

	if !updated {
		actionSpec.Requirements = append(actionSpec.Requirements, document.ActionRequirementSpec{
			Key:       "cashflowCategoryAccounts",
			Label:     "Category Accounts",
			Required:  true,
			Satisfied: status.Loaded,
			Code:      strings.TrimSpace(status.Code),
			Message:   strings.TrimSpace(status.Message),
		})
	}

	if !status.Loaded {
		actionSpec.State.Enabled = false
		actionSpec.State.Code = strings.TrimSpace(status.Code)
		actionSpec.State.Message = strings.TrimSpace(status.Message)
		if actionSpec.State.Message == "" {
			actionSpec.State.Message = "Master data category accounts belum siap."
		}
	}

	return actionSpec
}

func (s *Service) applyCashflowBillProfileRequirement(actionSpec document.ActionSpec, profileID string) document.ActionSpec {
	configKey := cashflowBillProfileKeyFromAction(actionSpec.ActionType)
	label := "Default Profil Cashflow Pay Bills"
	defaultCode := "CASHFLOW_PAY_BILLS_PROFILE_UNAVAILABLE"
	defaultMessage := "Default profil cashflow pay bills belum tersedia."

	if configKey == appcashflowbill.ProfileConfigReceivePayments {
		label = "Default Profil Cashflow Receive Payments"
		defaultCode = "CASHFLOW_RECEIVE_PAYMENTS_PROFILE_UNAVAILABLE"
		defaultMessage = "Default profil cashflow receive payments belum tersedia."
	}

	status := appcashflowbill.ProfileConfigStatus{
		Configured:    false,
		Code:          defaultCode,
		Message:       defaultMessage,
		SchemaVersion: "1",
	}
	if s.cashflowBillProfileConfig != nil {
		status = s.cashflowBillProfileConfig.Status(profileID, configKey)
	}

	updated := false
	for idx, requirement := range actionSpec.Requirements {
		if !strings.EqualFold(strings.TrimSpace(requirement.Key), "cashflowBillDefaultProfile") {
			continue
		}
		requirement.Label = label
		requirement.Satisfied = status.Configured
		requirement.Code = strings.TrimSpace(status.Code)
		requirement.Message = strings.TrimSpace(status.Message)
		actionSpec.Requirements[idx] = requirement
		updated = true
	}

	if !updated {
		actionSpec.Requirements = append(actionSpec.Requirements, document.ActionRequirementSpec{
			Key:       "cashflowBillDefaultProfile",
			Label:     label,
			Required:  true,
			Satisfied: status.Configured,
			Code:      strings.TrimSpace(status.Code),
			Message:   strings.TrimSpace(status.Message),
		})
	}

	if !status.Configured {
		actionSpec.State.Enabled = false
		actionSpec.State.Code = strings.TrimSpace(status.Code)
		if strings.TrimSpace(actionSpec.State.Message) == "" {
			actionSpec.State.Message = strings.TrimSpace(status.Message)
		}
	}

	return actionSpec
}

func validateActionRequirements(requirements []document.ActionRequirementSpec) error {
	for _, requirement := range requirements {
		if !requirement.Required || requirement.Satisfied {
			continue
		}
		return &RequirementError{
			Code:    strings.TrimSpace(requirement.Code),
			Message: strings.TrimSpace(requirement.Message),
		}
	}
	return nil
}
