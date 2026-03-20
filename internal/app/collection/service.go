package collection

import (
	"context"
	"fmt"
	"strings"
	"time"

	domain "github.com/sieryo/invoice-extractor/internal/domain/collection"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type CollectionService struct {
	collectionRepo domain.Repository
	fileStore      file.FileStore
}

func NewCollectionService(
	collectionRepo domain.Repository,
	fileStore file.FileStore,
) *CollectionService {
	return &CollectionService{
		collectionRepo: collectionRepo,
		fileStore:      fileStore,
	}
}

func (s *CollectionService) Create(
	ctx context.Context,
	id string,
	name string,
	userID string,
) (*domain.Collection, error) {
	normalizedName, err := normalizeCollectionName(name)
	if err != nil {
		return nil, err
	}
	if err := s.ensureUniqueName(ctx, userID, nil, normalizedName, nil); err != nil {
		return nil, err
	}

	now := time.Now()
	coll := domain.NewCollection(id, userID, normalizedName, now)

	if err := s.collectionRepo.Create(ctx, coll); err != nil {
		if isUniqueCollectionNameConstraint(err) {
			return nil, domain.ErrCollectionNameConflict
		}
		return nil, err
	}

	return coll, nil
}

func (s *CollectionService) CreateFolder(
	ctx context.Context,
	id string,
	name string,
	userID string,
	parentID *string,
) (*domain.Collection, error) {
	normalizedName, err := normalizeCollectionName(name)
	if err != nil {
		return nil, err
	}
	if err := s.ensureUniqueName(ctx, userID, parentID, normalizedName, nil); err != nil {
		return nil, err
	}

	now := time.Now()
	folder := domain.NewFolder(id, userID, parentID, normalizedName, now)

	if err := s.collectionRepo.Create(ctx, folder); err != nil {
		if isUniqueCollectionNameConstraint(err) {
			return nil, domain.ErrCollectionNameConflict
		}
		return nil, err
	}

	return folder, nil
}

func (s *CollectionService) CreateTypedCollection(
	ctx context.Context,
	id string,
	name string,
	userID string,
	parentID *string,
	documentType domain.DocumentType,
) (*domain.Collection, error) {
	if !documentType.IsValid() {
		return nil, domain.ErrInvalidDocumentType
	}
	normalizedName, err := normalizeCollectionName(name)
	if err != nil {
		return nil, err
	}
	if err := s.ensureUniqueName(ctx, userID, parentID, normalizedName, nil); err != nil {
		return nil, err
	}

	now := time.Now()
	coll := domain.NewTypedCollection(
		id,
		userID,
		parentID,
		normalizedName,
		documentType,
		now,
	)

	if err := s.collectionRepo.Create(ctx, coll); err != nil {
		if isUniqueCollectionNameConstraint(err) {
			return nil, domain.ErrCollectionNameConflict
		}
		return nil, err
	}

	return coll, nil
}

func (s *CollectionService) GetByID(
	ctx context.Context,
	id string,
) (*domain.Collection, error) {
	coll, err := s.collectionRepo.FindByID(ctx, id)
	if err != nil {
		return nil, domain.ErrCollectionNotFound
	}

	return coll, nil
}

func (s *CollectionService) ListChildren(
	ctx context.Context,
	userID string,
	parentID *string,
) ([]*domain.Collection, error) {
	collections, err := s.collectionRepo.ListChildren(ctx, userID, parentID)
	if err != nil {
		return nil, err
	}

	return collections, nil
}

func (s *CollectionService) ListByUser(
	ctx context.Context,
	userID string,
) ([]*domain.Collection, error) {
	collections, err := s.collectionRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return collections, nil
}

func (s *CollectionService) Delete(
	ctx context.Context,
	id string,
) error {
	// Keep cleanup best-effort while collection still exists.
	if err := s.fileStore.CleanupTemp(ctx, id); err != nil {
		return err
	}

	return s.collectionRepo.Delete(ctx, id)
}

func (s *CollectionService) Rename(
	ctx context.Context,
	id string,
	userID string,
	name string,
) (*domain.Collection, error) {
	normalizedName, err := normalizeCollectionName(name)
	if err != nil {
		return nil, err
	}

	current, err := s.collectionRepo.FindByID(ctx, id)
	if err != nil || current == nil {
		return nil, domain.ErrCollectionNotFound
	}
	if current.UserID != userID || current.DeletedAt != nil {
		return nil, domain.ErrCollectionNotFound
	}

	if strings.EqualFold(strings.TrimSpace(current.Name), normalizedName) {
		return current, nil
	}

	if err := s.ensureUniqueName(ctx, userID, current.Parent, normalizedName, &id); err != nil {
		return nil, err
	}

	if err := s.collectionRepo.UpdateName(ctx, id, normalizedName); err != nil {
		if isUniqueCollectionNameConstraint(err) {
			return nil, domain.ErrCollectionNameConflict
		}
		return nil, err
	}

	updated, err := s.collectionRepo.FindByID(ctx, id)
	if err != nil || updated == nil {
		return nil, domain.ErrCollectionNotFound
	}

	return updated, nil
}

func (s *CollectionService) GetPath(
	ctx context.Context,
	userID string,
	id string,
) ([]*domain.Collection, error) {
	currentID := strings.TrimSpace(id)
	if currentID == "" {
		return nil, domain.ErrCollectionNotFound
	}

	path := make([]*domain.Collection, 0, 8)
	visited := make(map[string]struct{}, 8)

	for currentID != "" {
		if _, exists := visited[currentID]; exists {
			return nil, fmt.Errorf("collection path cycle detected at id %s", currentID)
		}
		visited[currentID] = struct{}{}

		coll, err := s.collectionRepo.FindByID(ctx, currentID)
		if err != nil || coll == nil {
			return nil, domain.ErrCollectionNotFound
		}
		if coll.UserID != userID || coll.DeletedAt != nil {
			return nil, domain.ErrCollectionNotFound
		}

		path = append(path, coll)

		if coll.Parent == nil {
			break
		}
		parentID := strings.TrimSpace(*coll.Parent)
		if parentID == "" {
			break
		}
		currentID = parentID
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path, nil
}

func (s *CollectionService) ensureUniqueName(
	ctx context.Context,
	userID string,
	parentID *string,
	name string,
	excludeID *string,
) error {
	children, err := s.collectionRepo.ListChildren(ctx, userID, parentID)
	if err != nil {
		return err
	}
	exclude := ""
	if excludeID != nil {
		exclude = strings.TrimSpace(*excludeID)
	}
	for _, child := range children {
		if child == nil || child.DeletedAt != nil {
			continue
		}
		if exclude != "" && child.ID == exclude {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(child.Name), name) {
			return domain.ErrCollectionNameConflict
		}
	}
	return nil
}

func normalizeCollectionName(name string) (string, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return "", domain.ErrInvalidCollectionName
	}
	return normalized, nil
}

func isUniqueCollectionNameConstraint(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_collections_unique_name_active") ||
		(strings.Contains(message, "unique constraint failed") &&
			strings.Contains(message, "collections"))
}
