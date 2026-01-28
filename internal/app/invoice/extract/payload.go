package extract

import "github.com/sieryo/invoice-extractor/internal/app/job"

type Payload struct {
	InputFiles []job.InputFile `json:"input_files"`
	Template   *string         `json:"template,omitempty"`
}
