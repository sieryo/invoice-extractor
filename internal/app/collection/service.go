package collection

import (
	"context"
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

	now := time.Now()
	coll := domain.NewCollection(id, userID, name, now)

	if err := s.collectionRepo.Create(ctx, coll); err != nil {
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
	// Clean up local temp files
	if err := s.fileStore.CleanupTemp(ctx, id); err != nil {
		// Log error but continue? Or fail? User said: "clean up local temporary files".
		// Typically cleanup failure shouldn't block logical deletion, but let's return error to be safe.
		return err
	}

	return s.collectionRepo.Delete(ctx, id)
}
