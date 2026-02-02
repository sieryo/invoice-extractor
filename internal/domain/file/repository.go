package file

import "context"

type Repository interface {
	Create(ctx context.Context, f *FileObject) error
	CreateBulk(ctx context.Context, files []*FileObject) error

	FindByID(ctx context.Context, id string) (*FileObject, error)
	ListByCollection(ctx context.Context, collectionID string) ([]*FileObject, error)

	UpdateState(ctx context.Context, id string, state FileState) error
	Delete(ctx context.Context, id string) error
	DeleteBulk(ctx context.Context, ids []string) error
	DeleteByCollection(ctx context.Context, collectionID string) error
}
