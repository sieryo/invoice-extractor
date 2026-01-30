package collection

import (
	"context"
	"time"

	domain "github.com/sieryo/invoice-extractor/internal/domain/collection"
)

type CollectionService struct {
	collectionRepo domain.Repository
}

func NewCollectionService(
	collectionRepo domain.Repository,
) *CollectionService {
	return &CollectionService{
		collectionRepo: collectionRepo,
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
