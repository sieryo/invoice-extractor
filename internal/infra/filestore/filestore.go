package filestore

import (
	"context"
)

type TempObject struct {
	ID    string
	JobID string
	Name  string
	Path  string
}

type FinalObject struct {
	Name string
	Path string
}

type FileStore interface {
	SaveTemp(
		ctx context.Context,
		jobID string,
		name string,
		data []byte,
	) (TempObject, error)

	Commit(
		ctx context.Context,
		obj TempObject,
	) (FinalObject, error)

	CleanupTemp(ctx context.Context, jobID string) error
}
