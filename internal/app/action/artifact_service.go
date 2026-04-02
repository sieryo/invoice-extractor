package action

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/document"
	"github.com/sieryo/invoice-extractor/internal/app/ledger"
)

const actionArtifactValueTypeLedgerSnapshotTXT = "ledger_snapshot_txt"

func (s *Service) SaveActionArtifact(
	ctx context.Context,
	userID string,
	collectionID string,
	actionType string,
	artifactSpec document.ActionArtifactInputSpec,
	originalName string,
	mimeType string,
	sizeBytes int64,
	objectRef string,
	data []byte,
) (*CollectionActionArtifact, error) {
	if _, err := s.getOwnedCollection(ctx, userID, collectionID); err != nil {
		return nil, err
	}

	previewJSON, err := buildActionArtifactPreview(strings.TrimSpace(artifactSpec.ValueType), data)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	artifact := &CollectionActionArtifact{
		ID:           uuid.NewString(),
		UserID:       userID,
		CollectionID: collectionID,
		ActionType:   strings.TrimSpace(actionType),
		ArtifactKey:  strings.TrimSpace(artifactSpec.Key),
		ObjectRef:    strings.TrimSpace(objectRef),
		OriginalName: strings.TrimSpace(originalName),
		MimeType:     strings.TrimSpace(mimeType),
		SizeBytes:    sizeBytes,
		PreviewJSON:  previewJSON,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateActionArtifact(ctx, artifact); err != nil {
		return nil, err
	}

	return artifact, nil
}

func (s *Service) GetLatestActionArtifacts(
	ctx context.Context,
	userID string,
	collectionID string,
	actionType string,
) ([]*CollectionActionArtifact, error) {
	if _, err := s.getOwnedCollection(ctx, userID, collectionID); err != nil {
		return nil, err
	}

	items, err := s.repo.ListActionArtifacts(ctx, collectionID, strings.TrimSpace(actionType))
	if err != nil {
		return nil, err
	}

	latestByKey := make(map[string]*CollectionActionArtifact, len(items))
	ordered := make([]*CollectionActionArtifact, 0)
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item.ArtifactKey))
		if key == "" {
			continue
		}
		if _, exists := latestByKey[key]; exists {
			continue
		}
		latestByKey[key] = item
		ordered = append(ordered, item)
	}

	return ordered, nil
}

func buildActionArtifactPreview(valueType string, data []byte) (json.RawMessage, error) {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "", "file_ref":
		return nil, nil
	case actionArtifactValueTypeLedgerSnapshotTXT:
		snapshot, err := ledger.ParseSnapshot(bufio.NewReader(bytes.NewReader(data)))
		if err != nil {
			return nil, err
		}
		preview := ledger.BuildPreview(snapshot, 8)
		b, err := json.Marshal(preview)
		if err != nil {
			return nil, err
		}
		return b, nil
	default:
		return nil, nil
	}
}
