package helper

import (
	"encoding/json"
	"fmt"

	"github.com/sieryo/invoice-extractor/internal/app/invoice/extract"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax/rename"
	jobdomain "github.com/sieryo/invoice-extractor/internal/domain/job"
)

func ValidateJobPayload(jobType jobdomain.JobType, rawPayload json.RawMessage) ([]byte, error) {
	switch jobType {
	case jobdomain.JobTypeExtractInvoice:
		var payload extract.Payload
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return nil, fmt.Errorf("invalid payload for extract invoice: %w", err)
		}
		return json.Marshal(payload)

	case jobdomain.JobTypeRenameTaxInvoice:
		var payload rename.Payload
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return nil, fmt.Errorf("invalid payload for rename tax invoice: %w", err)
		}
		return json.Marshal(payload)

	default:
		return nil, fmt.Errorf("unsupported job_type: %s", jobType)
	}
}
