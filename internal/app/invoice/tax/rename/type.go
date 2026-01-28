package rename

import (
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
)

type RenamedFile struct {
	FileID    string
	SourceURI string
	OldName   string
	NewName   string
}

type BatchRenameResult struct {
	Files  []RenamedFile
	Errors []shared.FileResultError
}
