package extract

import (
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type Payload struct {
	CollectionID string
	InputFiles   []file.FileRef `json:"input_files"`
	Template     *string        `json:"template,omitempty"`
}
