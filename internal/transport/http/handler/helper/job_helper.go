package helper

import (
	"encoding/json"
	"fmt"

	"github.com/sieryo/invoice-extractor/internal/app/invoice/extract"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax/rename"
	jobdomain "github.com/sieryo/invoice-extractor/internal/domain/job"
)

func ValidateJobPayload(jobType jobdomain.JobType, rawPayload json.RawMessage) ([]byte, *string, error) {
	switch jobType {
	case jobdomain.JobTypeExtractInvoice:
		var payload extract.Payload
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return nil, nil, fmt.Errorf("invalid payload for extract invoice: %w", err)
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, err
		}
		return b, ptrIfNotEmpty(payload.CollectionID), nil

	case jobdomain.JobTypeRenameTaxInvoice:
		var payload rename.Payload
		if err := json.Unmarshal(rawPayload, &payload); err != nil {
			return nil, nil, fmt.Errorf("invalid payload for rename tax invoice: %w", err)
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, nil, err
		}
		return b, ptrIfNotEmpty(payload.CollectionID), nil

	default:
		return nil, nil, fmt.Errorf("unsupported job_type: %s", jobType)
	}
}

func ptrIfNotEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
