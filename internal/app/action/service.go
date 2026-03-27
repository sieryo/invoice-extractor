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
	appbuyer "github.com/sieryo/invoice-extractor/internal/app/buyer"
	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
	"github.com/sieryo/invoice-extractor/internal/app/document"
	dcollection "github.com/sieryo/invoice-extractor/internal/domain/collection"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type BuyerRegistryStatusProvider interface {
	Status() appbuyer.BuyerRegistryStatus
}

type CashflowTaxAccountStatusProvider interface {
	Status() appcashflow.TaxAccountStatus
}

type Service struct {
	repo           Repository
	collectionRepo dcollection.Repository
	processors     *document.Registry
	buyerRegistry  BuyerRegistryStatusProvider
	taxAccounts    CashflowTaxAccountStatusProvider
	fileStore      file.FileStore

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
	buyerRegistry BuyerRegistryStatusProvider,
	taxAccounts CashflowTaxAccountStatusProvider,
	fileStore file.FileStore,
	workers int,
) *Service {
	if workers < 1 {
		workers = 1
	}

	return &Service{
		repo:           repo,
		collectionRepo: collectionRepo,
		processors:     processors,
		buyerRegistry:  buyerRegistry,
		taxAccounts:    taxAccounts,
		fileStore:      fileStore,
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
	spec = s.applyRuntimeRequirements(spec)

	actionSpec, ok := spec.FindActionSpec(actionType)
	if !ok {
		return nil, ErrActionNotSupported
	}
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

	inputJSON, err := normalizeAndValidateActionInput(req.Input, actionSpec.Form)
	if err != nil {
		return nil, err
	}

	minDocumentCnt := actionSpec.Selection.MinDocumentCnt
	if minDocumentCnt <= 0 {
		minDocumentCnt = 1
	}

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
	spec = s.applyRuntimeRequirements(spec)
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
	spec = s.applyRuntimeRequirements(spec)
	if coll.IsFrozen() {
		spec = applyFrozenCollectionSpec(spec)
	}

	actionSpec, ok := spec.FindActionSpec(req.ActionType)
	if !ok {
		return nil, ErrActionNotSupported
	}

	resolved := actionSpec
	if collectionKind == document.CollectionKindCashflowImport && strings.EqualFold(strings.TrimSpace(actionSpec.ActionType), "export_cashflow_myob") {
		resolved, err = s.resolveCashflowActionSpec(ctx, req.CollectionID, collectionKind, sourceFormat, actionSpec, req.DocumentIDs)
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

	runReq := document.ActionRequest{
		ActionID:       action.ID,
		UserID:         action.UserID,
		CollectionID:   action.CollectionID,
		CollectionKind: action.CollectionKind,
		SourceFormat:   action.SourceFormat,
		ActionType:     action.ActionType,
		SnapshotDocID:  docIDs,
		SnapshotDocs:   snapshotPayload,
		Input:          action.InputJSON,
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
) (json.RawMessage, error) {
	fieldSpecs := flattenFormFields(form)
	if len(fieldSpecs) == 0 {
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
	if form == nil || len(form.Sections) == 0 {
		return nil
	}

	fields := make([]document.FormFieldSpec, 0)
	for _, section := range form.Sections {
		fields = append(fields, section.Fields...)
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
	collectionID string,
	collectionKind document.CollectionKind,
	sourceFormat document.SourceFormat,
	actionSpec document.ActionSpec,
	documentIDs []string,
) (document.ActionSpec, error) {
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
		field.DefaultValue = ""
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
	if defaultSheet != "" {
		field.DefaultValue = defaultSheet
	} else {
		field.DefaultValue = ""
	}
	updateFormField(actionSpec.Form, field)

	if headerField, ok := findFormField(actionSpec.Form, "headerRowNumber"); ok {
		if defaultHeaderRow > 0 {
			headerField.DefaultValue = defaultHeaderRow
		}
		updateFormField(actionSpec.Form, headerField)
	}

	return actionSpec, nil
}

func (s *Service) resolveCommonCashflowSheets(
	ctx context.Context,
	collectionID string,
	snapshotDocs []SnapshotDocument,
) ([]string, string, int, error) {
	var common []string
	headerBySheet := map[string]int{}
	for idx, doc := range snapshotDocs {
		workbook, err := document.LoadCashflowWorkbook(ctx, s.fileStore, collectionID, doc.NormalizedRef)
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

func (s *Service) applyRuntimeRequirements(spec document.CollectionSpec) document.CollectionSpec {
	actions := make([]document.ActionSpec, 0, len(spec.Actions))
	for _, item := range spec.Actions {
		actionSpec := item
		actionSpec.Requirements = append([]document.ActionRequirementSpec(nil), item.Requirements...)

		if spec.CollectionKind == document.CollectionKindInvoiceCompany &&
			strings.EqualFold(strings.TrimSpace(actionSpec.ActionType), "export_faktur_keluaran") {
			actionSpec = s.applyBuyerRegistryRequirement(actionSpec)
		}
		if spec.CollectionKind == document.CollectionKindCashflowImport &&
			strings.EqualFold(strings.TrimSpace(actionSpec.ActionType), "export_cashflow_myob") {
			actionSpec = s.applyCashflowTaxAccountRequirement(actionSpec)
		}

		actions = append(actions, actionSpec)
	}
	spec.Actions = actions
	return spec
}

func (s *Service) applyBuyerRegistryRequirement(actionSpec document.ActionSpec) document.ActionSpec {
	status := appbuyer.BuyerRegistryStatus{
		Loaded:  false,
		Code:    "BUYER_REGISTRY_STATUS_UNAVAILABLE",
		Message: "Buyer registry checker tidak tersedia.",
	}
	if s.buyerRegistry != nil {
		status = s.buyerRegistry.Status()
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

func (s *Service) applyCashflowTaxAccountRequirement(actionSpec document.ActionSpec) document.ActionSpec {
	status := appcashflow.TaxAccountStatus{
		Loaded:  false,
		Code:    "CASHFLOW_TAX_ACCOUNTS_UNAVAILABLE",
		Message: "Master data tax accounts belum tersedia.",
	}
	if s.taxAccounts != nil {
		status = s.taxAccounts.Status()
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
