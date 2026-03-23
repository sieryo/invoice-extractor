package action

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/sieryo/invoice-extractor/internal/app/document"
)

func TestNormalizeAndValidateActionInput_RequiredMissing(t *testing.T) {
	_, err := normalizeAndValidateActionInput(nil, &document.FormSpec{
		Sections: []document.FormSectionSpec{
			{
				Key: "main",
				Fields: []document.FormFieldSpec{
					{
						Key:      "sheetName",
						Kind:     document.FormFieldKindText,
						Required: true,
						State:    document.FormFieldStateSpec{Visible: true},
					},
				},
			},
		},
	})

	if !errors.Is(err, ErrInvalidActionParams) {
		t.Fatalf("expected ErrInvalidActionParams, got %v", err)
	}
}

func TestNormalizeAndValidateActionInput_UnknownField(t *testing.T) {
	raw := json.RawMessage(`{"unknown":"x"}`)
	_, err := normalizeAndValidateActionInput(raw, &document.FormSpec{
		Sections: []document.FormSectionSpec{
			{
				Key: "main",
				Fields: []document.FormFieldSpec{
					{
						Key:      "sheetName",
						Kind:     document.FormFieldKindText,
						Required: true,
						State:    document.FormFieldStateSpec{Visible: true},
					},
				},
			},
		},
	})

	if !errors.Is(err, ErrInvalidActionParams) {
		t.Fatalf("expected ErrInvalidActionParams, got %v", err)
	}
}

func TestNormalizeAndValidateActionInput_NumberCoercion(t *testing.T) {
	raw := json.RawMessage(`{"headerRowNumber":2}`)
	b, err := normalizeAndValidateActionInput(raw, &document.FormSpec{
		Sections: []document.FormSectionSpec{
			{
				Key: "main",
				Fields: []document.FormFieldSpec{
					{
						Key:      "headerRowNumber",
						Kind:     document.FormFieldKindNumber,
						Required: true,
						State:    document.FormFieldStateSpec{Visible: true},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if unmarshalErr := json.Unmarshal(b, &got); unmarshalErr != nil {
		t.Fatalf("unmarshal normalized input failed: %v", unmarshalErr)
	}
	if got["headerRowNumber"] != float64(2) {
		t.Fatalf("expected headerRowNumber=2, got %#v", got["headerRowNumber"])
	}
}

func TestNormalizeAllowedStatuses_InvalidSpec(t *testing.T) {
	_, err := normalizeAllowedStatuses([]string{"ready", "failed"})
	if !errors.Is(err, ErrInvalidActionSpec) {
		t.Fatalf("expected ErrInvalidActionSpec, got %v", err)
	}
}

func TestNormalizeAndValidateActionInput_DefaultApplied(t *testing.T) {
	b, err := normalizeAndValidateActionInput(nil, &document.FormSpec{
		Sections: []document.FormSectionSpec{
			{
				Key: "main",
				Fields: []document.FormFieldSpec{
					{
						Key:          "headerRowNumber",
						Kind:         document.FormFieldKindNumber,
						Required:     true,
						DefaultValue: 1,
						State:        document.FormFieldStateSpec{Visible: true},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got map[string]any
	if unmarshalErr := json.Unmarshal(b, &got); unmarshalErr != nil {
		t.Fatalf("unmarshal normalized input failed: %v", unmarshalErr)
	}
	if got["headerRowNumber"] != float64(1) {
		t.Fatalf("expected headerRowNumber=1, got %#v", got["headerRowNumber"])
	}
}

func TestNormalizeAndValidateActionInput_OptionsRejectInvalid(t *testing.T) {
	raw := json.RawMessage(`{"cashflowFormat":"unknown"}`)
	_, err := normalizeAndValidateActionInput(raw, &document.FormSpec{
		Sections: []document.FormSectionSpec{
			{
				Key: "main",
				Fields: []document.FormFieldSpec{
					{
						Key:      "cashflowFormat",
						Kind:     document.FormFieldKindSelect,
						Required: true,
						State:    document.FormFieldStateSpec{Visible: true},
						Options: []document.FormFieldOption{
							{Label: "Default", Value: "default"},
							{Label: "Influencer", Value: "influencer"},
						},
					},
				},
			},
		},
	})
	if !errors.Is(err, ErrInvalidActionParams) {
		t.Fatalf("expected ErrInvalidActionParams, got %v", err)
	}
}

func TestNormalizeAndValidateActionInput_RequiredIfRule(t *testing.T) {
	form := &document.FormSpec{
		Sections: []document.FormSectionSpec{
			{
				Key: "main",
				Fields: []document.FormFieldSpec{
					{
						Key:      "cashflowFormat",
						Kind:     document.FormFieldKindText,
						Required: true,
						State:    document.FormFieldStateSpec{Visible: true},
					},
					{
						Key:      "defaultIAccountCode",
						Kind:     document.FormFieldKindText,
						Required: false,
						State:    document.FormFieldStateSpec{Visible: true},
						Rules: []document.FormFieldRuleSpec{
							{
								Type:   document.FormFieldRuleRequiredIf,
								Field:  "cashflowFormat",
								Equals: "influencer",
							},
						},
					},
				},
			},
		},
	}

	_, err := normalizeAndValidateActionInput(json.RawMessage(`{"cashflowFormat":"influencer"}`), form)
	if !errors.Is(err, ErrInvalidActionParams) {
		t.Fatalf("expected ErrInvalidActionParams, got %v", err)
	}

	_, okErr := normalizeAndValidateActionInput(
		json.RawMessage(`{"cashflowFormat":"influencer","defaultIAccountCode":"62004"}`),
		form,
	)
	if okErr != nil {
		t.Fatalf("unexpected error: %v", okErr)
	}
}
