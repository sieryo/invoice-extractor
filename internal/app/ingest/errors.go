package ingest

import "errors"

var (
	ErrSessionNotFound    = errors.New("upload session not found")
	ErrSessionNotWritable = errors.New("upload session is not writable")
	ErrChunkNotFound      = errors.New("upload chunk not found")
)
