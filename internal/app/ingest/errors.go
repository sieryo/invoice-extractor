package ingest

import "errors"

var (
	ErrSessionNotFound    = errors.New("upload session not found")
	ErrSessionNotWritable = errors.New("upload session is not writable")
	ErrChunkNotFound      = errors.New("upload chunk not found")
	ErrHistoryNotFound    = errors.New("collection history not found")
	ErrDocumentNotFound   = errors.New("collection document not found")
	ErrReplaceSourceNotSupported = errors.New("replace source is not supported for this document")
)
