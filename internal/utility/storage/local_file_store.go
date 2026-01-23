package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type LocalFileStore struct {
	baseDir string
}

func (l *LocalFileStore) Save(
	ctx context.Context,
	name string,
	r io.Reader,
) (string, error) {

	finalPath := filepath.Join(l.baseDir, name)
	tmpPath := finalPath + ".tmp"

	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		return "", err
	}

	f, err := os.Create(tmpPath)
	if err != nil {
		return "", err
	}

	writeErr := error(nil)

	defer func() {
		if writeErr != nil {
			_ = f.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	buf := make([]byte, 32*1024)

	for {
		if err := ctx.Err(); err != nil {
			writeErr = err
			return "", err
		}

		n, readErr := r.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				writeErr = err
				return "", err
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			writeErr = readErr
			return "", readErr
		}
	}

	if err = f.Sync(); err != nil {
		writeErr = err
		return "", err
	}

	if err = f.Close(); err != nil {
		writeErr = err
		return "", err
	}

	if err = os.Rename(tmpPath, finalPath); err != nil {
		writeErr = err
		return "", err
	}

	return finalPath, nil
}

func NewLocalFileStore(baseDir string) *LocalFileStore {
	return &LocalFileStore{
		baseDir: baseDir,
	}
}
