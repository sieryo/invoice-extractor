package fpcoretax

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/configlayout"
	"github.com/sieryo/invoice-extractor/internal/app/specutil"
	"github.com/sieryo/invoice-extractor/internal/profilepath"
)

const profileConfigSchemaVersion = "1"

type ProfileConfigSpec struct {
	SchemaVersion string                     `json:"schemaVersion"`
	ConfigKey     string                     `json:"configKey"`
	Label         string                     `json:"label"`
	Description   string                     `json:"description,omitempty"`
	Sections      []configlayout.SectionSpec `json:"sections,omitempty"`
	Fields        []ProfileConfigFieldSpec   `json:"fields"`
}

type ProfileConfigFieldSpec struct {
	Key          string                     `json:"key"`
	Label        string                     `json:"label"`
	Required     bool                       `json:"required"`
	DefaultValue string                     `json:"defaultValue,omitempty"`
	Description  string                     `json:"description,omitempty"`
	Kind         string                     `json:"kind,omitempty"`
	Group        string                     `json:"group,omitempty"`
	Options      []ProfileConfigFieldOption `json:"options,omitempty"`
	Suggestions  []ProfileConfigSuggestion  `json:"suggestions,omitempty"`
	HelpText     string                     `json:"helpText,omitempty"`
	Placeholder  string                     `json:"placeholder,omitempty"`
}

type ProfileConfigFieldOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type ProfileConfigSuggestion struct {
	Token       string `json:"token"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
}

type ProfileConfig struct {
	SchemaVersion string               `json:"schemaVersion"`
	ConfigKey     string               `json:"configKey"`
	Fields        []ProfileConfigField `json:"fields"`
}

type ProfileConfigField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Group       string `json:"group,omitempty"`
}

type ProfileConfigStatus struct {
	Configured    bool     `json:"configured"`
	MissingFields []string `json:"missingFields,omitempty"`
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	SchemaVersion string   `json:"schemaVersion"`
}

type ProfileConfigService struct {
	rootDir string
}

func NewProfileConfigService(rootDir string) *ProfileConfigService {
	return &ProfileConfigService{rootDir: rootDir}
}

func (s *ProfileConfigService) Spec(key ProfileConfigKey) ProfileConfigSpec {
	return buildProfileConfigSpec(key)
}

func (s *ProfileConfigService) Load(profileID string, key ProfileConfigKey) (ProfileConfig, error) {
	seed := s.defaultConfig(key)
	path := s.configPath(profileID, key)

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return seed, nil
		}
		return ProfileConfig{}, err
	}

	var cfg ProfileConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return ProfileConfig{}, err
	}
	return s.normalizeConfig(cfg, key), nil
}

func (s *ProfileConfigService) Update(profileID string, key ProfileConfigKey, cfg ProfileConfig) (ProfileConfig, error) {
	normalized := s.normalizeConfig(cfg, key)
	path := s.configPath(profileID, key)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ProfileConfig{}, err
	}
	b, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return ProfileConfig{}, err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return ProfileConfig{}, err
	}
	return normalized, nil
}

func (s *ProfileConfigService) Status(profileID string, key ProfileConfigKey) ProfileConfigStatus {
	cfg, err := s.Load(profileID, key)
	if err != nil {
		return ProfileConfigStatus{
			Configured:    false,
			Code:          "CONFIG_ERROR",
			Message:       "Default profil FP Coretax tidak dapat dibaca saat ini.",
			SchemaVersion: profileConfigSchemaVersion,
		}
	}

	values := ResolveProfileConfigValues(cfg)
	missing := make([]string, 0)
	for _, field := range s.Spec(key).Fields {
		if !field.Required {
			continue
		}
		if strings.TrimSpace(values[field.Key]) == "" {
			missing = append(missing, field.Label)
		}
	}
	if len(missing) > 0 {
		return ProfileConfigStatus{
			Configured:    false,
			MissingFields: missing,
			Code:          "NOT_READY",
			Message:       "Default profil FP Coretax belum lengkap.",
			SchemaVersion: profileConfigSchemaVersion,
		}
	}

	return ProfileConfigStatus{
		Configured:    true,
		Code:          "READY",
		Message:       "Default profil FP Coretax siap digunakan.",
		SchemaVersion: profileConfigSchemaVersion,
	}
}

func buildProfileConfigSpec(key ProfileConfigKey) ProfileConfigSpec {
	label := "FP Keluaran"
	description := "Default parameter action Misc Sales."
	partyLabel := "Nama Pembeli"
	partyDefault := "nama"
	accountLabel := "Account Number"
	accountDefault := "41001"
	if key == ProfileConfigFPMasukanMiscPurchases {
		label = "FP Masukan"
		description = "Default parameter action Misc Purchases."
		partyLabel = "Nama Penjual"
		partyDefault = "nama"
		accountLabel = "Default Account Number"
		accountDefault = "51001"
	}

	return ProfileConfigSpec{
		SchemaVersion: profileConfigSchemaVersion,
		ConfigKey:     string(key),
		Label:         label,
		Description:   description,
		Sections: []configlayout.SectionSpec{
			specutil.ParameterActionSection("Parameter default yang akan dipakai saat action dibuka.", 2, "sheetName", "headerRowNumber", "outputFilename", "accountNumber", "memoTemplate", "descriptionTemplate"),
			specutil.Section("tax", "Tax", "Parameter pajak default untuk output MYOB.", 2, "taxCode", "inclusive"),
			specutil.MappingHeaderSection("Nama header source untuk membaca kolom faktur pajak.", 2, "partyName", "documentNumber", "date", "taxBase", "tax", "reference"),
		},
		Fields: []ProfileConfigFieldSpec{
			textField("sheetName", "Sheet Default", false, "", "Nilai awal sheet untuk action ini.", "parameter"),
			selectField("headerRowNumber", "Baris Header Default", true, "1", "Baris header default workbook.", "parameter", buildHeaderRowOptions(10)),
			textField("outputFilename", "Nama Output", true, defaultOutputFilename(key), "Tanpa ekstensi file.", "parameter"),
			textField("accountNumber", accountLabel, true, accountDefault, "Dipakai saat registry tidak menyediakan account number.", "parameter"),
			templateField("memoTemplate", "Template Memo", true, "{{nomorFakturPajak}}", "Template default memo output MYOB.", "parameter", buildTemplateSuggestions(key)),
			templateField("descriptionTemplate", "Template Description", true, "{{nomorFakturPajak}}", "Template default description output MYOB.", "parameter", buildTemplateSuggestions(key)),
			textField("taxCode", "Tax Code", true, "PPN", "Tax code default untuk baris MYOB.", "tax"),
			checkboxField("inclusive", "Inclusive Tax", false, "false", "Aktifkan jika nilai total pada source sudah termasuk pajak.", "tax"),
			textField("partyName", partyLabel, true, partyDefault, "Header source untuk nama pihak.", "mapping"),
			textField("documentNumber", "Nomor Faktur Pajak", true, "nomor faktur pajak", "Header source nomor faktur pajak.", "mapping"),
			textField("date", "Tanggal Faktur Pajak", true, "tanggal faktur pajak", "Header source tanggal faktur pajak.", "mapping"),
			textField("taxBase", "DPP", true, "harga jual/penggantian/dpp", "Header source nilai DPP.", "mapping"),
			textField("tax", "PPN", true, "ppn", "Header source nilai PPN.", "mapping"),
			textField("reference", "Referensi", false, "referensi", "Header source referensi tambahan.", "mapping"),
		},
	}
}

func buildTemplateSuggestions(key ProfileConfigKey) []ProfileConfigSuggestion {
	partyToken := "namaPembeli"
	partyLabel := "Nama Pembeli"
	if key == ProfileConfigFPMasukanMiscPurchases {
		partyToken = "namaPenjual"
		partyLabel = "Nama Penjual"
	}
	return []ProfileConfigSuggestion{
		{Token: partyToken, Label: partyLabel, Example: "{{" + partyToken + "}}"},
		{Token: "nomorFakturPajak", Label: "Nomor Faktur Pajak", Example: "{{nomorFakturPajak}}"},
		{Token: "tanggalFakturPajak", Label: "Tanggal Faktur Pajak", Example: "{{tanggalFakturPajak}}"},
		{Token: "referensi", Label: "Referensi", Example: "{{referensi}}"},
		{Token: "dpp", Label: "DPP", Example: "{{dpp}}"},
		{Token: "ppn", Label: "PPN", Example: "{{ppn}}"},
		{Token: "total", Label: "Total", Example: "{{total}}"},
		{Token: "sourceName", Label: "Nama File Asal", Example: "{{sourceName}}"},
	}
}

func defaultOutputFilename(key ProfileConfigKey) string {
	if key == ProfileConfigFPMasukanMiscPurchases {
		return "misc_purchases"
	}
	return "misc_sales"
}

func (s *ProfileConfigService) defaultConfig(key ProfileConfigKey) ProfileConfig {
	spec := s.Spec(key)
	fields := make([]ProfileConfigField, 0, len(spec.Fields))
	for _, field := range spec.Fields {
		fields = append(fields, ProfileConfigField{
			Key:         field.Key,
			Label:       field.Label,
			Required:    field.Required,
			Value:       field.DefaultValue,
			Description: field.Description,
			Kind:        field.Kind,
			Group:       field.Group,
		})
	}
	return ProfileConfig{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Fields:        fields,
	}
}

func (s *ProfileConfigService) normalizeConfig(cfg ProfileConfig, key ProfileConfigKey) ProfileConfig {
	spec := s.Spec(key)
	current := make(map[string]ProfileConfigField, len(cfg.Fields))
	for _, field := range cfg.Fields {
		normalizedKey := strings.TrimSpace(field.Key)
		if normalizedKey == "" {
			continue
		}
		current[normalizedKey] = field
	}

	fields := make([]ProfileConfigField, 0, len(spec.Fields))
	for _, fieldSpec := range spec.Fields {
		value := strings.TrimSpace(fieldSpec.DefaultValue)
		if stored, ok := current[fieldSpec.Key]; ok && strings.TrimSpace(stored.Value) != "" {
			value = strings.TrimSpace(stored.Value)
		}
		if fieldSpec.Key == "headerRowNumber" {
			if _, err := strconv.Atoi(strings.TrimSpace(value)); err != nil {
				value = fieldSpec.DefaultValue
			}
		}
		fields = append(fields, ProfileConfigField{
			Key:         fieldSpec.Key,
			Label:       fieldSpec.Label,
			Required:    fieldSpec.Required,
			Value:       value,
			Description: fieldSpec.Description,
			Kind:        fieldSpec.Kind,
			Group:       fieldSpec.Group,
		})
	}

	return ProfileConfig{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Fields:        fields,
	}
}

func (s *ProfileConfigService) configPath(profileID string, key ProfileConfigKey) string {
	switch key {
	case ProfileConfigFPMasukanMiscPurchases:
		return profilepath.FPMasukanMiscPurchasesConfigJSON(s.rootDir, profileID)
	default:
		return profilepath.FPKeluaranMiscSalesConfigJSON(s.rootDir, profileID)
	}
}

func buildHeaderRowOptions(max int) []ProfileConfigFieldOption {
	return specutil.IntOptions(max, func(label string, value string) ProfileConfigFieldOption {
		return ProfileConfigFieldOption{Label: label, Value: value}
	})
}

func textField(key string, label string, required bool, defaultValue string, description string, group string) ProfileConfigFieldSpec {
	return ProfileConfigFieldSpec{
		Key:          key,
		Label:        label,
		Required:     required,
		DefaultValue: defaultValue,
		Description:  description,
		Kind:         "text",
		Group:        group,
	}
}

func selectField(key string, label string, required bool, defaultValue string, description string, group string, options []ProfileConfigFieldOption) ProfileConfigFieldSpec {
	return ProfileConfigFieldSpec{
		Key:          key,
		Label:        label,
		Required:     required,
		DefaultValue: defaultValue,
		Description:  description,
		Kind:         "select",
		Group:        group,
		Options:      options,
	}
}

func checkboxField(key string, label string, required bool, defaultValue string, description string, group string) ProfileConfigFieldSpec {
	return ProfileConfigFieldSpec{
		Key:          key,
		Label:        label,
		Required:     required,
		DefaultValue: defaultValue,
		Description:  description,
		Kind:         "checkbox",
		Group:        group,
	}
}

func templateField(key string, label string, required bool, defaultValue string, description string, group string, suggestions []ProfileConfigSuggestion) ProfileConfigFieldSpec {
	return ProfileConfigFieldSpec{
		Key:          key,
		Label:        label,
		Required:     required,
		DefaultValue: defaultValue,
		Description:  description,
		Kind:         "template",
		Group:        group,
		Placeholder:  defaultValue,
		HelpText:     "Gunakan placeholder yang tersedia untuk membentuk nilai output.",
		Suggestions:  suggestions,
	}
}
