package file

import "context"

type FileStore interface {
	SaveTemp(
		ctx context.Context,
		collectionID string,
		name string,
		data []byte,
	) (FileObject, error)

	Commit(
		ctx context.Context,
		obj FileObject,
	) (FileObject, error)

	Read(
		ctx context.Context,
		collectionID string,
		fileID string,
	) ([]byte, error)

	Delete(
		ctx context.Context,
		collectionID string,
		fileID string,
	) error

	CleanupTemp(ctx context.Context, collectionID string) error
	Cleanup(ctx context.Context, collectionID string) error
}
