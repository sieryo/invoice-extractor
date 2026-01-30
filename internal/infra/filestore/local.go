package filestore

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type LocalFileStore struct {
	baseDir string
}

func (l *LocalFileStore) SaveTemp(
	ctx context.Context,
	collectionID string,
	name string,
	data []byte,
) (file.FileObject, error) {

	select {
	case <-ctx.Done():
		return file.FileObject{}, ctx.Err()
	default:
	}

	fileID := uuid.New().String()

	fileName := fmt.Sprintf("%s-%s", fileID, name)

	tempDir := filepath.Join(l.baseDir, "temp", collectionID)
	tempPath := filepath.Join(tempDir, fileName)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return file.FileObject{}, err
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return file.FileObject{}, err
	}

	size := int64(len(data))
	mimeType := "application/octet-stream"
	if size > 0 {
		head := data
		if len(head) > 512 {
			head = head[:512]
		}
		mimeType = http.DetectContentType(head)
	}

	return file.NewTempFile(
		fileID,
		collectionID,
		name,
		tempPath,
		size,
		mimeType,
	), nil
}

func (l *LocalFileStore) Commit(
	ctx context.Context,
	obj file.FileObject,
) (file.FileObject, error) {

	select {
	case <-ctx.Done():
		return file.FileObject{}, ctx.Err()
	default:
	}

	finalDir := filepath.Join(l.baseDir, "files", obj.CollectionID)

	if err := os.MkdirAll(finalDir, 0755); err != nil {
		return file.FileObject{}, err
	}

	finalName := obj.Name
	ext := filepath.Ext(finalName)
	base := strings.TrimSuffix(obj.ID, ext)
	attempt := 1

	for {
		finalPath := filepath.Join(finalDir, finalName)
		if _, err := os.Stat(finalPath); os.IsNotExist(err) {
			if err := os.Rename(obj.Path, finalPath); err != nil {
				return file.FileObject{}, err
			}
			return obj.Commit(finalPath)
		}
		attempt++
		finalName = fmt.Sprintf("%s(%d)%s", base, attempt, ext)
	}
}

func (l *LocalFileStore) CleanupTemp(
	ctx context.Context,
	collectionID string,
) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	tempDir := filepath.Join(l.baseDir, "temp", collectionID)
	return os.RemoveAll(tempDir)
}

func (l *LocalFileStore) Read(
	ctx context.Context,
	collectionID string,
	fileID string,
) ([]byte, error) {

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	path := filepath.Join(l.baseDir, "files", collectionID, fileID)
	return os.ReadFile(path)
}

func (l *LocalFileStore) Delete(
	ctx context.Context,
	collectionID string,
	fileID string,
) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	tempPath := filepath.Join(l.baseDir, "temp", collectionID, fileID)
	finalPath := filepath.Join(l.baseDir, "files", collectionID, fileID)

	_ = os.Remove(tempPath)
	_ = os.Remove(finalPath)

	return nil
}

func NewLocalFileStore(baseDir string) *LocalFileStore {
	return &LocalFileStore{
		baseDir: baseDir,
	}
}
