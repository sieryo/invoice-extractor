package rename

import "github.com/sieryo/invoice-extractor/internal/app/job"

type Payload struct {
	InputFiles []job.InputFile `json:"input_files"`
}
