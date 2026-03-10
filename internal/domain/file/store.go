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

	SaveAudit(
		ctx context.Context,
		collectionID string,
		name string,
		data []byte,
	) (string, error)

	ReadAudit(
		ctx context.Context,
		collectionID string,
		name string,
	) ([]byte, error)

	ListAudit(
		ctx context.Context,
		collectionID string,
	) ([]string, error)

	SaveArchive(
		ctx context.Context,
		collectionID string,
		name string,
		data []byte,
	) (string, error)

	ReadArchive(
		ctx context.Context,
		collectionID string,
		name string,
	) ([]byte, error)

	ListArchive(
		ctx context.Context,
		collectionID string,
	) ([]ArchiveInfo, error)

	WriteFile(
		ctx context.Context,
		collectionID string,
		name string,
		data []byte,
	) error

	GetJobStorageUsage(
		ctx context.Context,
		collectionID string,
	) (StorageUsage, error)

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
