package job

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
	jobdomain "github.com/sieryo/invoice-extractor/internal/domain/job"
)

type JobService struct {
	repo      jobdomain.Repository
	runner    jobdomain.JobRunner
	fileStore file.FileStore
}

func NewJobService(repo jobdomain.Repository, runner jobdomain.JobRunner, fileStore file.FileStore) *JobService {
	return &JobService{repo: repo, runner: runner, fileStore: fileStore}
}

func (s *JobService) CreateJob(
	ctx context.Context,
	userID string,
	jobType jobdomain.JobType,
	inputPayload []byte,
	collectionID *string,
) (*jobdomain.Job, error) {

	j := jobdomain.NewJob(
		uuid.NewString(),
		&userID,
		jobType,
		inputPayload,
		collectionID,
	)

	j.Status = jobdomain.JobPending
	j.CreatedAt = time.Now()

	if err := s.repo.Create(ctx, j); err != nil {
		return nil, err
	}

	return j, nil
}

func (s *JobService) StartJob(ctx context.Context, id string) error {
	j, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if j.Status != jobdomain.JobPending && j.Status != jobdomain.JobFailed {
		return jobdomain.ErrJobNotStartable
	}

	now := time.Now()
	j.Status = jobdomain.JobRunning
	j.StartedAt = &now

	if err := s.repo.UpdateStatus(ctx, j.ID, j.Status); err != nil {
		return err
	}

	return s.runner.Run(ctx, j)
}

func (s *JobService) UpdateProgress(ctx context.Context, id string, progress int) error {
	return s.repo.UpdateProgress(ctx, id, progress)
}

func (s *JobService) FinishJob(ctx context.Context, j *jobdomain.Job, success bool, errMsg *string) error {
	now := time.Now()
	j.FinishedAt = &now
	if success {
		j.Status = jobdomain.JobSuccess
	} else {
		j.Status = jobdomain.JobFailed
		j.ErrorMessage = errMsg
	}
	return s.repo.Update(ctx, j)
}

func (s *JobService) GetJobByID(ctx context.Context, id string) (*jobdomain.Job, error) {
	return s.repo.FindByID(ctx, id)
}

// TODO HARUS PAKE USER ID
func (s *JobService) ListJobs(ctx context.Context) ([]*jobdomain.Job, error) {
	return s.repo.List(ctx)
}

func (s *JobService) DeleteJob(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	if err := s.fileStore.Cleanup(ctx, id); err != nil {
		fmt.Printf("Failed to cleanup job files: %v\n", err)
		return err
	}

	return nil
}

type ArchiveResult struct {
	ArchivePath string `json:"archive_path"`
	ArchiveName string `json:"archive_name"`
	SizeBytes   int64  `json:"size_bytes"`
}

type UnarchiveResult struct {
	ArchiveName   string `json:"archive_name"`
	RestoredFiles int    `json:"restored_files"`
	RestoredAudit int    `json:"restored_audit"`
}

func (s *JobService) ArchiveJob(ctx context.Context, jobID string, deleteOriginal bool) (ArchiveResult, error) {
	j, err := s.repo.FindByID(ctx, jobID)
	if err != nil {
		return ArchiveResult{}, jobdomain.ErrJobNotFound
	}

	if j.OutputManifest == nil || len(j.OutputManifest.Files) == 0 {
		return ArchiveResult{}, jobdomain.ErrJobNoFiles
	}

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, f := range j.OutputManifest.Files {
		if f.Status != jobdomain.OutputFileReady && f.Status != jobdomain.OutputFileWarning {
			continue
		}

		name := f.StorageName
		if name == "" {
			name = f.Name
		}

		fileContent, err := s.fileStore.Read(ctx, j.ID, name)
		if err != nil {
			continue
		}

		entryName := "files/" + name
		fWriter, err := zipWriter.Create(entryName)
		if err != nil {
			return ArchiveResult{}, err
		}

		if _, err := fWriter.Write(fileContent); err != nil {
			return ArchiveResult{}, err
		}
	}

	auditNames, err := s.fileStore.ListAudit(ctx, j.ID)
	if err != nil {
		return ArchiveResult{}, err
	}

	for _, name := range auditNames {
		data, err := s.fileStore.ReadAudit(ctx, j.ID, name)
		if err != nil {
			continue
		}

		fWriter, err := zipWriter.Create("audit/" + name)
		if err != nil {
			return ArchiveResult{}, err
		}

		if _, err := fWriter.Write(data); err != nil {
			return ArchiveResult{}, err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return ArchiveResult{}, err
	}

	filename := fmt.Sprintf("job_%s_%s.zip", j.ID, time.Now().Format("20060102_150405"))
	archivePath, err := s.fileStore.SaveArchive(ctx, j.ID, filename, buf.Bytes())
	if err != nil {
		return ArchiveResult{}, err
	}

	if deleteOriginal {
		if err := s.fileStore.Cleanup(ctx, j.ID); err != nil {
			return ArchiveResult{}, err
		}
	}

	return ArchiveResult{
		ArchivePath: archivePath,
		ArchiveName: filename,
		SizeBytes:   int64(buf.Len()),
	}, nil
}

func (s *JobService) UnarchiveJob(ctx context.Context, jobID string, archiveName *string) (UnarchiveResult, error) {
	j, err := s.repo.FindByID(ctx, jobID)
	if err != nil {
		return UnarchiveResult{}, jobdomain.ErrJobNotFound
	}

	name := ""
	if archiveName != nil && *archiveName != "" {
		name = *archiveName
	} else {
		archives, err := s.fileStore.ListArchive(ctx, jobID)
		if err != nil {
			return UnarchiveResult{}, err
		}
		if len(archives) == 0 {
			return UnarchiveResult{}, jobdomain.ErrArchiveNotFound
		}
		sort.Slice(archives, func(i, k int) bool {
			return archives[i].ModTime.After(archives[k].ModTime)
		})
		name = archives[0].Name
	}

	zipBytes, err := s.fileStore.ReadArchive(ctx, jobID, name)
	if err != nil {
		return UnarchiveResult{}, err
	}

	readerAt := bytes.NewReader(zipBytes)
	zr, err := zip.NewReader(readerAt, int64(len(zipBytes)))
	if err != nil {
		return UnarchiveResult{}, err
	}

	manifestNameToStorage := make(map[string]string)
	if j.OutputManifest != nil {
		for _, f := range j.OutputManifest.Files {
			if f.Name != "" && f.StorageName != "" {
				manifestNameToStorage[f.Name] = f.StorageName
			}
		}
	}

	var restoredFiles int
	var restoredAudit int

	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() {
			continue
		}

		rc, err := zf.Open()
		if err != nil {
			return UnarchiveResult{}, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return UnarchiveResult{}, err
		}

		name := zf.Name
		if strings.HasPrefix(name, "audit/") {
			base := strings.TrimPrefix(name, "audit/")
			if _, err := s.fileStore.SaveAudit(ctx, jobID, filepath.Base(base), data); err != nil {
				return UnarchiveResult{}, err
			}
			restoredAudit++
			continue
		}

		if strings.HasPrefix(name, "files/") {
			base := strings.TrimPrefix(name, "files/")
			if err := s.fileStore.WriteFile(ctx, jobID, base, data); err != nil {
				return UnarchiveResult{}, err
			}
			restoredFiles++
			continue
		}

		dest := name
		if mapped, ok := manifestNameToStorage[name]; ok {
			dest = mapped
		}
		if err := s.fileStore.WriteFile(ctx, jobID, dest, data); err != nil {
			return UnarchiveResult{}, err
		}
		restoredFiles++
	}

	return UnarchiveResult{
		ArchiveName:   name,
		RestoredFiles: restoredFiles,
		RestoredAudit: restoredAudit,
	}, nil
}

func (s *JobService) GetArchiveBytes(ctx context.Context, jobID string, archiveName *string) ([]byte, string, error) {
	if _, err := s.repo.FindByID(ctx, jobID); err != nil {
		return nil, "", jobdomain.ErrJobNotFound
	}

	name := ""
	if archiveName != nil && *archiveName != "" {
		name = *archiveName
	} else {
		archives, err := s.fileStore.ListArchive(ctx, jobID)
		if err != nil {
			return nil, "", err
		}
		if len(archives) == 0 {
			return nil, "", jobdomain.ErrArchiveNotFound
		}
		sort.Slice(archives, func(i, k int) bool {
			return archives[i].ModTime.After(archives[k].ModTime)
		})
		name = archives[0].Name
	}

	data, err := s.fileStore.ReadArchive(ctx, jobID, name)
	if err != nil {
		return nil, "", err
	}

	return data, name, nil
}
