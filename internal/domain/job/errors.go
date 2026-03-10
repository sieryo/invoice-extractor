package job

import "errors"

var (
	ErrJobNotStartable = errors.New("job is not startable")
	ErrJobNotFound     = errors.New("job not found")
	ErrJobNoFiles      = errors.New("no files found for this job")
	ErrArchiveNotFound = errors.New("archive not found for this job")
)
