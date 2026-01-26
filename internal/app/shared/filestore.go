package shared

import (
	"context"
)

type TempObject struct {
	ID        string
	TempDirID string
	Name      string
	Path      string
}

type FinalObject struct {
	ID   string
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

	Read(
		ctx context.Context,
		jobID string,
		name string,
	) ([]byte, error)

	CleanupTemp(ctx context.Context, jobID string) error
}
