package rename

import (
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type Payload struct {
	CollectionID string         `json:"collection_id"`
	InputFiles   []file.FileRef `json:"input_files"`
}
