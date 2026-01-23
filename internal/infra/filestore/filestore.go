package filestore

import (
	"context"
	"io"
)

type FileStore interface {
	Save(ctx context.Context, name string, r io.Reader) (string, error)
	Delete(path string) error
}
