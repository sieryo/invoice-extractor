package filestore

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type LocalFileStore struct {
	baseDir string
}

func (l *LocalFileStore) actionsDir(collectionID string) string {
	return filepath.Join(l.baseDir, "actions", collectionID)
}

func (l *LocalFileStore) legacyArchiveDir(collectionID string) string {
	return filepath.Join(l.baseDir, "archive", collectionID)
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

	ext := filepath.Ext(obj.Name)
	finalName := obj.ID + ext
	finalPath := filepath.Join(finalDir, finalName)

	if err := os.Rename(obj.Path, finalPath); err != nil {
		return file.FileObject{}, err
	}

	return obj.Commit(finalPath)
}

func (l *LocalFileStore) SaveAudit(
	ctx context.Context,
	collectionID string,
	name string,
	data []byte,
) (string, error) {

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	auditDir := filepath.Join(l.baseDir, "audit", collectionID)
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		return "", err
	}

	auditPath := filepath.Join(auditDir, name)
	if err := os.WriteFile(auditPath, data, 0644); err != nil {
		return "", err
	}

	return auditPath, nil
}

func (l *LocalFileStore) SaveArchive(
	ctx context.Context,
	collectionID string,
	name string,
	data []byte,
) (string, error) {

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	archivePath := filepath.Join(l.actionsDir(collectionID), name)
	if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
		return "", err
	}

	if err := os.WriteFile(archivePath, data, 0644); err != nil {
		return "", err
	}

	return archivePath, nil
}

func (l *LocalFileStore) ReadArchive(
	ctx context.Context,
	collectionID string,
	name string,
) ([]byte, error) {

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	primaryPath := filepath.Join(l.actionsDir(collectionID), name)
	if b, err := os.ReadFile(primaryPath); err == nil {
		return b, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	legacyPath := filepath.Join(l.legacyArchiveDir(collectionID), name)
	return os.ReadFile(legacyPath)
}

func (l *LocalFileStore) ListArchive(
	ctx context.Context,
	collectionID string,
) ([]file.ArchiveInfo, error) {

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	infos := make([]file.ArchiveInfo, 0)
	seen := make(map[string]struct{})

	appendFromDir := func(dir string) error {
		list, err := listArchiveInfosFromDir(dir)
		if err != nil {
			return err
		}
		for _, item := range list {
			if _, exists := seen[item.Name]; exists {
				continue
			}
			seen[item.Name] = struct{}{}
			infos = append(infos, item)
		}
		return nil
	}

	if err := appendFromDir(l.actionsDir(collectionID)); err != nil {
		return nil, err
	}
	if err := appendFromDir(l.legacyArchiveDir(collectionID)); err != nil {
		return nil, err
	}

	return infos, nil
}

func (l *LocalFileStore) ReadAudit(
	ctx context.Context,
	collectionID string,
	name string,
) ([]byte, error) {

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	auditPath := filepath.Join(l.baseDir, "audit", collectionID, name)
	return os.ReadFile(auditPath)
}

func (l *LocalFileStore) ListAudit(
	ctx context.Context,
	collectionID string,
) ([]string, error) {

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	auditDir := filepath.Join(l.baseDir, "audit", collectionID)
	entries, err := os.ReadDir(auditDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}

	return names, nil
}

func (l *LocalFileStore) WriteFile(
	ctx context.Context,
	collectionID string,
	name string,
	data []byte,
) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	filesDir := filepath.Join(l.baseDir, "files", collectionID)
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(filesDir, filepath.Base(name))
	return os.WriteFile(path, data, 0644)
}

func (l *LocalFileStore) GetJobStorageUsage(
	ctx context.Context,
	collectionID string,
) (file.StorageUsage, error) {

	select {
	case <-ctx.Done():
		return file.StorageUsage{}, ctx.Err()
	default:
	}

	tempDir := filepath.Join(l.baseDir, "temp", collectionID)
	filesDir := filepath.Join(l.baseDir, "files", collectionID)
	auditDir := filepath.Join(l.baseDir, "audit", collectionID)

	tempBytes, err := dirSize(ctx, tempDir)
	if err != nil {
		return file.StorageUsage{}, err
	}
	filesBytes, err := dirSize(ctx, filesDir)
	if err != nil {
		return file.StorageUsage{}, err
	}
	auditBytes, err := dirSize(ctx, auditDir)
	if err != nil {
		return file.StorageUsage{}, err
	}

	total := tempBytes + filesBytes + auditBytes
	return file.StorageUsage{
		TempBytes:  tempBytes,
		FilesBytes: filesBytes,
		AuditBytes: auditBytes,
		TotalBytes: total,
	}, nil
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

func (l *LocalFileStore) Cleanup(
	ctx context.Context,
	collectionID string,
) error {

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Remove temp directory if exists
	tempDir := filepath.Join(l.baseDir, "temp", collectionID)
	if err := os.RemoveAll(tempDir); err != nil {
		return err
	}

	// Remove files directory if exists
	filesDir := filepath.Join(l.baseDir, "files", collectionID)
	if err := os.RemoveAll(filesDir); err != nil {
		return err
	}

	// Remove audit directory if exists
	auditDir := filepath.Join(l.baseDir, "audit", collectionID)
	if err := os.RemoveAll(auditDir); err != nil {
		return err
	}

	actionsDir := l.actionsDir(collectionID)
	if err := os.RemoveAll(actionsDir); err != nil {
		return err
	}

	legacyArchiveDir := l.legacyArchiveDir(collectionID)
	if err := os.RemoveAll(legacyArchiveDir); err != nil {
		return err
	}

	return nil
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

func dirSize(ctx context.Context, dir string) (int64, error) {
	var size int64

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	for _, e := range entries {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		info, err := e.Info()
		if err != nil {
			return 0, err
		}

		if e.IsDir() {
			subSize, err := dirSize(ctx, filepath.Join(dir, e.Name()))
			if err != nil {
				return 0, err
			}
			size += subSize
			continue
		}

		size += info.Size()
	}

	return size, nil
}

func listArchiveInfosFromDir(dir string) ([]file.ArchiveInfo, error) {
	infos := make([]file.ArchiveInfo, 0)
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			return infos, nil
		}
		return nil, err
	}

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		infos = append(infos, file.ArchiveInfo{
			Name:    filepath.ToSlash(rel),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return infos, nil
}
