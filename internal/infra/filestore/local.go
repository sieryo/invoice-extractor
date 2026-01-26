package filestore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/shared"
)

type LocalFileStore struct {
	baseDir string
}

func (l *LocalFileStore) SaveTemp(
	ctx context.Context,
	tempDirID string,
	name string,
	data []byte,
) (shared.TempObject, error) {

	fileID := uuid.New().String()
	safeName := fmt.Sprintf("%s_%s", fileID, name)

	tempDir := filepath.Join(l.baseDir, "temp", tempDirID)
	tempPath := filepath.Join(tempDir, safeName)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return shared.TempObject{}, err
	}

	select {
	case <-ctx.Done():
		return shared.TempObject{}, ctx.Err()
	default:
	}

	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return shared.TempObject{}, err
	}

	return shared.TempObject{
		ID:        fileID,
		TempDirID: tempDirID,
		Name:      name,
		Path:      tempPath,
	}, nil
}

func (l *LocalFileStore) Commit(
	ctx context.Context,
	obj shared.TempObject,
) (shared.FinalObject, error) {

	finalDir := filepath.Join(l.baseDir, "jobs", obj.TempDirID)
	finalPath := filepath.Join(finalDir, obj.Name)

	select {
	case <-ctx.Done():
		return shared.FinalObject{}, ctx.Err()
	default:
	}

	if err := os.MkdirAll(finalDir, 0755); err != nil {
		return shared.FinalObject{}, err
	}

	if err := os.Rename(obj.Path, finalPath); err != nil {
		return shared.FinalObject{}, err
	}

	return shared.FinalObject{
		ID:   obj.ID,
		Name: obj.Name,
		Path: finalPath,
	}, nil
}

func (l *LocalFileStore) CleanupTemp(
	ctx context.Context,
	jobID string,
) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	tempDir := filepath.Join(l.baseDir, "temp", jobID)
	return os.RemoveAll(tempDir)
}

func (l *LocalFileStore) Read(
	ctx context.Context,
	jobID string,
	name string,
) ([]byte, error) {

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	path := filepath.Join(l.baseDir, "jobs", jobID, name)
	return os.ReadFile(path)
}

func NewLocalFileStore(baseDir string) *LocalFileStore {
	return &LocalFileStore{
		baseDir: baseDir,
	}
}
