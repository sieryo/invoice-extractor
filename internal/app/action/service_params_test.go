package action

import (
	"encoding/json"
	"errors"
	"testing"

	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
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
							{Label: "Standard", Value: "standard"},
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

func TestPickPreferredSheetName(t *testing.T) {
	options := []string{"January 2026", "Summary"}

	if got := pickPreferredSheetName(options, "January 2026", "Summary"); got != "January 2026" {
		t.Fatalf("expected preferred sheet to win, got %q", got)
	}

	if got := pickPreferredSheetName(options, "Missing", "Summary"); got != "Summary" {
		t.Fatalf("expected fallback sheet to win, got %q", got)
	}

	if got := pickPreferredSheetName(options, "", ""); got != "January 2026" {
		t.Fatalf("expected first option fallback, got %q", got)
	}
}

func TestResolveCashflowRuntimeInput_MergesProfileRuntimeValues(t *testing.T) {
	service := &Service{
		cashflowProfileConfig: stubCashflowProfileConfigProvider{
			cfg: appcashflow.ProfileConfig{
				Variants: []appcashflow.ProfileConfigVariant{
					{
						Key: "standard",
						Values: map[string]string{
							"informationFilterKeywords": "",
						},
					},
					{
						Key: "influencer",
						Values: map[string]string{
							"informationFilterKeywords": "opening balance\ntransfer",
						},
					},
				},
			},
		},
	}

	input := json.RawMessage(`{"cashflowFormat":"influencer","sheetName":"Cashflow"}`)
	merged := service.resolveCashflowRuntimeInput("user-1", "export_cashflow_spend_money", input)

	var payload map[string]any
	if err := json.Unmarshal(merged, &payload); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if payload["sheetName"] != "Cashflow" {
		t.Fatalf("expected original input to stay intact, got %#v", payload["sheetName"])
	}
	if payload["informationFilterKeywords"] != "opening balance\ntransfer" {
		t.Fatalf("expected runtime keyword config to merge, got %#v", payload["informationFilterKeywords"])
	}
}

type stubCashflowProfileConfigProvider struct {
	cfg appcashflow.ProfileConfig
}

func (s stubCashflowProfileConfigProvider) Status(_ string, _ appcashflow.ProfileConfigKey) appcashflow.ProfileConfigStatus {
	return appcashflow.ProfileConfigStatus{}
}

func (s stubCashflowProfileConfigProvider) Load(_ string, _ appcashflow.ProfileConfigKey) (appcashflow.ProfileConfig, error) {
	return s.cfg, nil
}
