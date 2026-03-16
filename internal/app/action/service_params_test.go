package action

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/sieryo/invoice-extractor/internal/app/document"
)

func TestNormalizeAndValidateActionParams_RequiredMissing(t *testing.T) {
	_, err := normalizeAndValidateActionParams(nil, []document.ActionParamFieldSpec{
		{
			Key:      "sheetName",
			Type:     document.ActionParamTypeString,
			Required: true,
		},
	})

	if !errors.Is(err, ErrInvalidActionParams) {
		t.Fatalf("expected ErrInvalidActionParams, got %v", err)
	}
}

func TestNormalizeAndValidateActionParams_UnknownParam(t *testing.T) {
	raw := json.RawMessage(`{"unknown":"x"}`)
	_, err := normalizeAndValidateActionParams(raw, []document.ActionParamFieldSpec{
		{
			Key:      "sheetName",
			Type:     document.ActionParamTypeString,
			Required: true,
		},
	})

	if !errors.Is(err, ErrInvalidActionParams) {
		t.Fatalf("expected ErrInvalidActionParams, got %v", err)
	}
}

func TestNormalizeAndValidateActionParams_IntCoercion(t *testing.T) {
	raw := json.RawMessage(`{"headerRowNumber":2}`)
	b, err := normalizeAndValidateActionParams(raw, []document.ActionParamFieldSpec{
		{
			Key:      "headerRowNumber",
			Type:     document.ActionParamTypeInt,
			Required: true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if unmarshalErr := json.Unmarshal(b, &got); unmarshalErr != nil {
		t.Fatalf("unmarshal normalized params failed: %v", unmarshalErr)
	}
	if got["headerRowNumber"] != float64(2) {
		t.Fatalf("expected headerRowNumber=2, got %#v", got["headerRowNumber"])
	}
}

func TestNormalizeAndValidateActionParams_IntRejectDecimal(t *testing.T) {
	raw := json.RawMessage(`{"headerRowNumber":2.5}`)
	_, err := normalizeAndValidateActionParams(raw, []document.ActionParamFieldSpec{
		{
			Key:      "headerRowNumber",
			Type:     document.ActionParamTypeInt,
			Required: true,
		},
	})
	if !errors.Is(err, ErrInvalidActionParams) {
		t.Fatalf("expected ErrInvalidActionParams, got %v", err)
	}
}

func TestNormalizeAllowedStatuses_InvalidSpec(t *testing.T) {
	_, err := normalizeAllowedStatuses([]string{"ready", "failed"})
	if !errors.Is(err, ErrInvalidActionSpec) {
		t.Fatalf("expected ErrInvalidActionSpec, got %v", err)
	}
}

func TestNormalizeAndValidateActionParams_DefaultApplied(t *testing.T) {
	b, err := normalizeAndValidateActionParams(nil, []document.ActionParamFieldSpec{
		{
			Key:      "headerRowNumber",
			Type:     document.ActionParamTypeInt,
			Required: true,
			Default:  1,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if unmarshalErr := json.Unmarshal(b, &got); unmarshalErr != nil {
		t.Fatalf("unmarshal normalized params failed: %v", unmarshalErr)
	}
	if got["headerRowNumber"] != float64(1) {
		t.Fatalf("expected headerRowNumber=1, got %#v", got["headerRowNumber"])
	}
}

func TestNormalizeAndValidateActionParams_OptionsRejectInvalid(t *testing.T) {
	raw := json.RawMessage(`{"cashflowFormat":"unknown"}`)
	_, err := normalizeAndValidateActionParams(raw, []document.ActionParamFieldSpec{
		{
			Key:      "cashflowFormat",
			Type:     document.ActionParamTypeString,
			Required: true,
			Options: []document.ActionParamOptionSpec{
				{Label: "Default", Value: "default"},
				{Label: "Influencer", Value: "influencer"},
			},
		},
	})
	if !errors.Is(err, ErrInvalidActionParams) {
		t.Fatalf("expected ErrInvalidActionParams, got %v", err)
	}
}

func TestNormalizeAndValidateActionParams_RequiredIfRule(t *testing.T) {
	specs := []document.ActionParamFieldSpec{
		{
			Key:      "cashflowFormat",
			Type:     document.ActionParamTypeString,
			Required: true,
		},
		{
			Key:      "defaultIAccountCode",
			Type:     document.ActionParamTypeString,
			Required: false,
			Rules: []document.ActionParamRuleSpec{
				{
					Type:   document.ActionParamRuleRequiredIf,
					Field:  "cashflowFormat",
					Equals: "influencer",
				},
			},
		},
	}

	_, err := normalizeAndValidateActionParams(json.RawMessage(`{"cashflowFormat":"influencer"}`), specs)
	if !errors.Is(err, ErrInvalidActionParams) {
		t.Fatalf("expected ErrInvalidActionParams, got %v", err)
	}

	_, okErr := normalizeAndValidateActionParams(
		json.RawMessage(`{"cashflowFormat":"influencer","defaultIAccountCode":"62004"}`),
		specs,
	)
	if okErr != nil {
		t.Fatalf("unexpected error: %v", okErr)
	}
}
