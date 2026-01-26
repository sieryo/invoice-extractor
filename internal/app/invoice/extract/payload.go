package extract

import "github.com/sieryo/invoice-extractor/internal/app/job"

type Payload struct {
	JobFiles []job.JobFile `json:"job_files"`
	Template *string       `json:"template,omitempty"`
}
