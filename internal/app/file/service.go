package file

import (
	"context"

	"github.com/sieryo/invoice-extractor/internal/domain/collection"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type FileService struct {
	store          file.FileStore
	fileRepo       file.Repository
	collectionRepo collection.Repository
}

func NewFileService(store file.FileStore, fileRepo file.Repository, collectionRepo collection.Repository) *FileService {
	return &FileService{
		store:          store,
		fileRepo:       fileRepo,
		collectionRepo: collectionRepo,
	}
}

func (s *FileService) UploadFiles(
	ctx context.Context,
	collectionID string,
	inputs []file.UploadInput,
) ([]file.FileObject, error) {

	coll, err := s.collectionRepo.FindByID(ctx, collectionID)
	if err != nil {
		return nil, collection.ErrCollectionNotFound
	}
	if !coll.IsActive() {
		return nil, collection.ErrCollectionNotActive
	}

	files := make([]*file.FileObject, 0, len(inputs))

	for _, in := range inputs {
		obj, err := s.store.SaveTemp(
			ctx,
			collectionID,
			in.Name,
			in.Data,
		)
		if err != nil {

			s.store.CleanupTemp(ctx, collectionID)
			return nil, err
		}

		files = append(files, &obj)
	}

	if err := s.fileRepo.CreateBulk(ctx, files); err != nil {
		return nil, err
	}

	return derefFiles(files), nil
}

func (s *FileService) GetByID(
	ctx context.Context,
	fileID string,
) (*file.FileObject, error) {
	return s.fileRepo.FindByID(ctx, fileID)
}

func (s *FileService) ListByCollection(
	ctx context.Context,
	collectionID string,
) ([]*file.FileObject, error) {

	if _, err := s.collectionRepo.FindByID(ctx, collectionID); err != nil {
		return nil, err
	}

	return s.fileRepo.ListByCollection(ctx, collectionID)
}

func (s *FileService) Delete(
	ctx context.Context,
	id string,
) error {
	f, err := s.fileRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.store.Delete(ctx, f.CollectionID, f.ID); err != nil {
		return err
	}

	return s.fileRepo.Delete(ctx, id)
}

func (s *FileService) DeleteBulk(
	ctx context.Context,
	ids []string,
) error {
	// TODO: NANTI ADA BULK FIND
	for _, id := range ids {
		f, err := s.fileRepo.FindByID(ctx, id)
		if err != nil {
			continue
		}
		_ = s.store.Delete(ctx, f.CollectionID, f.ID)
	}

	return s.fileRepo.DeleteBulk(ctx, ids)
}

// Helper

func derefFiles(files []*file.FileObject) []file.FileObject {
	res := make([]file.FileObject, 0, len(files))
	for _, f := range files {
		if f == nil {
			continue
		}
		res = append(res, *f)
	}
	return res
}
