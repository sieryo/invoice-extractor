package cashflow

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

const cashflowProfileConfigSchemaVersion = "5"

type ProfileConfigKey string

const (
	ProfileConfigSpendMoney   ProfileConfigKey = "spend_money"
	ProfileConfigReceiveMoney ProfileConfigKey = "receive_money"
)

type ProfileConfigSpec struct {
	SchemaVersion    string                     `json:"schemaVersion"`
	ConfigKey        string                     `json:"configKey"`
	Label            string                     `json:"label"`
	Description      string                     `json:"description,omitempty"`
	Defaults         ProfileConfigDefaults      `json:"defaults"`
	HeaderRowOptions []int                      `json:"headerRowOptions,omitempty"`
	FormatOptions    []ProfileConfigFormatSpec  `json:"formatOptions,omitempty"`
	Sections         []configlayout.SectionSpec `json:"sections,omitempty"`
	Fields           []ProfileConfigFieldSpec   `json:"fields"`
	Variants         []ProfileConfigVariantSpec `json:"variants"`
}

type ProfileConfigDefaults struct {
	CashflowFormat  string `json:"cashflowFormat"`
	SheetName       string `json:"sheetName,omitempty"`
	HeaderRowNumber int    `json:"headerRowNumber,omitempty"`
}

type ProfileConfigFormatSpec struct {
	Value string `json:"value"`
	Label string `json:"label"`
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
}

type ProfileConfigFieldOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type ProfileConfigVariantSpec struct {
	Key         string            `json:"key"`
	Label       string            `json:"label"`
	Description string            `json:"description,omitempty"`
	Values      map[string]string `json:"values"`
}

type ProfileConfig struct {
	SchemaVersion string                 `json:"schemaVersion"`
	ConfigKey     string                 `json:"configKey"`
	Defaults      ProfileConfigDefaults  `json:"defaults"`
	Variants      []ProfileConfigVariant `json:"variants"`

	// Legacy flat fields are still read for migration compatibility.
	Fields []ProfileConfigField `json:"fields,omitempty"`
}

type ProfileConfigVariant struct {
	Key         string            `json:"key"`
	Label       string            `json:"label"`
	Description string            `json:"description,omitempty"`
	Values      map[string]string `json:"values"`

	// Legacy variant fields are still read for migration compatibility.
	Fields []ProfileConfigField `json:"fields,omitempty"`
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
			Message:       "Default profil cashflow tidak dapat dibaca saat ini.",
			SchemaVersion: cashflowProfileConfigSchemaVersion,
		}
	}

	standardMissing := profileVariantMissingFields(cfg, StandardFormat)
	influencerMissing := profileVariantMissingFields(cfg, InfluencerFormat)
	if len(standardMissing) > 0 && len(influencerMissing) > 0 {
		return ProfileConfigStatus{
			Configured:    false,
			MissingFields: standardMissing,
			Code:          "NOT_READY",
			Message:       "Default profil cashflow belum lengkap.",
			SchemaVersion: cashflowProfileConfigSchemaVersion,
		}
	}

	return ProfileConfigStatus{
		Configured:    true,
		Code:          "READY",
		Message:       "Minimal satu format cashflow siap digunakan.",
		SchemaVersion: cashflowProfileConfigSchemaVersion,
	}
}

func ResolveProfileConfigValues(cfg ProfileConfig, format string) map[string]string {
	selectedFormat := normalizeProfileConfigFormat(format)
	return variantFieldMap(cfg.Variants, selectedFormat)
}

func ResolveProfileConfigFormValues(cfg ProfileConfig, key ProfileConfigKey, format string) map[string]string {
	return resolveProfileConfigValuesByGroups(cfg, key, format, func(group string) bool {
		return group != "runtime"
	})
}

func ResolveProfileConfigRuntimeValues(cfg ProfileConfig, format string) map[string]string {
	return resolveProfileConfigValuesByGroups(cfg, ProfileConfigSpendMoney, format, func(group string) bool {
		return group == "runtime"
	})
}

func ResolveProfileConfigNumber(cfg ProfileConfig, format string, key string) (int, bool) {
	value := strings.TrimSpace(ResolveProfileConfigValues(cfg, format)[strings.TrimSpace(key)])
	if value == "" {
		return 0, false
	}
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func resolveProfileConfigValuesByGroups(
	cfg ProfileConfig,
	key ProfileConfigKey,
	format string,
	includeGroup func(group string) bool,
) map[string]string {
	selectedFormat := normalizeProfileConfigFormat(format)
	values := variantFieldMap(cfg.Variants, selectedFormat)
	if len(values) == 0 {
		return nil
	}

	out := make(map[string]string)
	for _, blueprint := range profileFieldBlueprints(key) {
		if !includeGroup(strings.TrimSpace(blueprint.Group)) {
			continue
		}
		value := strings.TrimSpace(values[blueprint.Key])
		if value == "" {
			continue
		}
		out[blueprint.Key] = value
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildProfileConfigSpec(key ProfileConfigKey) ProfileConfigSpec {
	label := "Cashflow Spend Money"
	description := "Nilai default untuk action export cashflow ke MYOB Spend Money."
	if key == ProfileConfigReceiveMoney {
		label = "Cashflow Receive Money"
		description = "Nilai default untuk action export cashflow ke MYOB Receive Money."
	}

	return ProfileConfigSpec{
		SchemaVersion: cashflowProfileConfigSchemaVersion,
		ConfigKey:     string(key),
		Label:         label,
		Description:   description,
		Defaults: ProfileConfigDefaults{
			CashflowFormat: string(StandardFormat),
		},
		HeaderRowOptions: specutil.HeaderRowNumbers(10),
		FormatOptions: []ProfileConfigFormatSpec{
			{Value: string(StandardFormat), Label: "Standard"},
			{Value: string(InfluencerFormat), Label: "Influencer"},
		},
		Sections: []configlayout.SectionSpec{
			specutil.ParameterActionSection("Parameter default yang akan dipakai saat action dibuka.", 2, "sheetName", "headerRowNumber", "outputFilename", "chequeAccount", "remarkDelimiter", "otherCostsAccountCode", "defaultIAccountCode", "defaultBAccountCode", "informationFilterKeywords"),
			specutil.MappingHeaderSection("Nama header source untuk membaca kolom cashflow.", 2, "date", "information", "coa", "otherCost", "pp23", "pph15", "pph21", "pph23", "pph42", "ppn", "remark", "total"),
		},
		Fields: buildProfileFieldSpecs(key),
		Variants: []ProfileConfigVariantSpec{
			{
				Key:         string(StandardFormat),
				Label:       "Standard",
				Description: "Konfigurasi untuk format cashflow standard.",
				Values:      defaultVariantFieldValues(key, StandardFormat),
			},
			{
				Key:         string(InfluencerFormat),
				Label:       "Influencer",
				Description: "Konfigurasi untuk format cashflow influencer.",
				Values:      defaultVariantFieldValues(key, InfluencerFormat),
			},
		},
	}
}

func buildProfileFieldSpecs(key ProfileConfigKey) []ProfileConfigFieldSpec {
	blueprints := profileFieldBlueprints(key)
	fields := make([]ProfileConfigFieldSpec, 0, len(blueprints))
	for _, blueprint := range blueprints {
		fields = append(fields, ProfileConfigFieldSpec{
			Key:          blueprint.Key,
			Label:        blueprint.Label,
			Required:     blueprint.Required,
			DefaultValue: blueprint.DefaultValue,
			Description:  blueprint.Description,
			Kind:         blueprint.Kind,
			Group:        blueprint.Group,
			Options:      blueprint.Options,
		})
	}
	return fields
}

func profileFieldBlueprints(key ProfileConfigKey) []ProfileConfigFieldSpec {
	outputDefault := "spend_money"
	accountLabel := "Cheque Account"
	accountDescription := "Akun cheque utama untuk output MYOB."
	if key == ProfileConfigReceiveMoney {
		outputDefault = "receive_money"
		accountLabel = "Deposit Account"
		accountDescription = "Akun deposit utama untuk output MYOB."
	}

	fields := []ProfileConfigFieldSpec{
		cashflowTextField("sheetName", "Sheet Default", false, "", "Nilai awal sheet untuk format ini.", "source"),
		cashflowSelectField("headerRowNumber", "Baris Header Default", true, "1", "Baris header default untuk format ini.", "source", buildHeaderRowProfileOptions(10)),
		cashflowTextField("outputFilename", "Nama Output", true, outputDefault, "Tanpa ekstensi file.", "parameter"),
		cashflowTextField("chequeAccount", accountLabel, true, "12021", accountDescription, "parameter"),
		cashflowTextField("remarkDelimiter", "Remark Delimiter", false, "*", "Dipakai untuk memecah catatan biaya lainnya.", "parameter"),
		cashflowTextField("otherCostsAccountCode", "Kode Akun Biaya Lain", false, "62099", "Dipakai saat format default memiliki komponen biaya lain.", "parameter"),
		cashflowTextField("defaultIAccountCode", "Default Influencer Account Code", false, "", "Dipakai saat format influencer memilih account influencer.", "parameter"),
		cashflowTextField("defaultBAccountCode", "Default Bank Account Code", false, "", "Dipakai saat format influencer mendeteksi transaksi bank.", "parameter"),
		cashflowTextareaField("informationFilterKeywords", "Keyword Filter Information", false, "", "Satu baris satu keyword. Row akan di-skip bila kolom Information mengandung keyword ini.", "runtime"),
		cashflowTextField("date", "Tanggal", true, "Tanggal", "Header source untuk tanggal transaksi.", "mapping"),
		cashflowTextField("information", "Keterangan", true, "note", "Header source untuk memo utama.", "mapping"),
		cashflowTextField("coa", "Chart of Account", false, "coa", "Wajib untuk format standard. Pada format influencer, header ini tidak dipakai untuk menentukan akun utama.", "mapping"),
		cashflowTextField("otherCost", "Biaya Lainnya", false, "By Lainnya", "Header source untuk biaya tambahan.", "mapping"),
		cashflowTextField("pp23", "PP 23", false, "PP 23", "Header source komponen pajak PP 23.", "mapping"),
		cashflowTextField("pph15", "PPh 15%", false, "PPh 15%", "Header source komponen pajak PPh 15%.", "mapping"),
		cashflowTextField("pph21", "PPH 21", false, "PPH 21", "Header source komponen pajak PPH 21.", "mapping"),
		cashflowTextField("pph23", "PPH 23", false, "PPH 23", "Header source komponen pajak PPH 23.", "mapping"),
		cashflowTextField("pph42", "PPH 4 (2)", false, "PPH 4(2)", "Header source komponen pajak PPH 4 (2).", "mapping"),
		cashflowTextField("ppn", "PPN", false, "PPN", "Header source komponen pajak PPN.", "mapping"),
		cashflowTextField("remark", "Catatan", false, "catatan", "Header source catatan atau allocation memo.", "mapping"),
		cashflowTextField("total", "Total", true, "idr", "Header source total transaksi.", "mapping"),
	}

	return fields
}

func defaultVariantFieldValues(key ProfileConfigKey, format Format) map[string]string {
	values := make(map[string]string)
	for _, field := range profileFieldBlueprints(key) {
		values[field.Key] = field.DefaultValue
	}
	if format == InfluencerFormat {
		values["defaultIAccountCode"] = "62004"
		values["defaultBAccountCode"] = "90900"
		values["informationFilterKeywords"] = "opening balance\ntransfer"
		values["date"] = "*Posting Date: # date"
		values["information"] = "Notes"
		values["remark"] = "WHT"
		values["coa"] = "WHT CoA"
		values["pph42"] = "PPh 4 (2)"
		values["otherCost"] = "Biaya Lainnya"
	}
	return values
}

func variantFieldMap(variants []ProfileConfigVariant, variantKey string) map[string]string {
	out := make(map[string]string)
	for _, variant := range variants {
		if !strings.EqualFold(strings.TrimSpace(variant.Key), strings.TrimSpace(variantKey)) {
			continue
		}
		for key, value := range variant.Values {
			normalizedKey := strings.TrimSpace(key)
			if normalizedKey == "" {
				continue
			}
			out[normalizedKey] = strings.TrimSpace(value)
		}
		for _, field := range variant.Fields {
			normalizedKey := strings.TrimSpace(field.Key)
			if normalizedKey == "" {
				continue
			}
			out[normalizedKey] = strings.TrimSpace(field.Value)
		}
		break
	}
	return out
}

func profileVariantMissingFields(cfg ProfileConfig, format Format) []string {
	resolved := ResolveProfileConfigValues(cfg, string(format))
	missing := make([]string, 0)
	require := func(fieldKey string, label string) {
		if strings.TrimSpace(resolved[fieldKey]) == "" {
			missing = append(missing, label)
		}
	}

	require("outputFilename", "Nama Output")
	require("chequeAccount", "Akun Utama")
	require("date", "Tanggal")
	require("information", "Keterangan")
	require("total", "Total")

	if format == InfluencerFormat {
		require("defaultIAccountCode", "Default Influencer Account Code")
		require("defaultBAccountCode", "Default Bank Account Code")
	} else {
		require("coa", "Chart of Account")
		require("remarkDelimiter", "Remark Delimiter")
		require("otherCostsAccountCode", "Kode Akun Biaya Lain")
	}

	return missing
}

func normalizeProfileConfigFormat(raw string) string {
	switch NormalizeFormat(raw) {
	case InfluencerFormat:
		return string(InfluencerFormat)
	default:
		return string(StandardFormat)
	}
}

func (s *ProfileConfigService) defaultConfig(key ProfileConfigKey) ProfileConfig {
	spec := s.Spec(key)
	variants := make([]ProfileConfigVariant, 0, len(spec.Variants))
	for _, variantSpec := range spec.Variants {
		variants = append(variants, ProfileConfigVariant{
			Key:         variantSpec.Key,
			Label:       variantSpec.Label,
			Description: variantSpec.Description,
			Values:      copyProfileValueMap(variantSpec.Values),
		})
	}

	return ProfileConfig{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Defaults:      spec.Defaults,
		Variants:      variants,
	}
}

func (s *ProfileConfigService) normalizeConfig(cfg ProfileConfig, key ProfileConfigKey) ProfileConfig {
	spec := s.Spec(key)
	legacyByKey := make(map[string]ProfileConfigField, len(cfg.Fields))
	for _, field := range cfg.Fields {
		normalized := strings.TrimSpace(field.Key)
		if normalized == "" {
			continue
		}
		legacyByKey[normalized] = field
	}

	variantByKey := make(map[string]ProfileConfigVariant, len(cfg.Variants))
	for _, variant := range cfg.Variants {
		normalized := strings.TrimSpace(variant.Key)
		if normalized == "" {
			continue
		}
		variantByKey[normalized] = variant
	}

	defaults := spec.Defaults
	if strings.TrimSpace(cfg.Defaults.CashflowFormat) != "" {
		defaults.CashflowFormat = normalizeProfileConfigFormat(cfg.Defaults.CashflowFormat)
	} else if legacyFormat, ok := legacyByKey["cashflowFormat"]; ok && strings.TrimSpace(legacyFormat.Value) != "" {
		defaults.CashflowFormat = normalizeProfileConfigFormat(legacyFormat.Value)
	}

	variants := make([]ProfileConfigVariant, 0, len(spec.Variants))
	for _, variantSpec := range spec.Variants {
		currentVariant, hasCurrentVariant := variantByKey[variantSpec.Key]
		if !hasCurrentVariant && variantSpec.Key == string(StandardFormat) {
			currentVariant, hasCurrentVariant = variantByKey["default"]
		}
		currentValueByKey := make(map[string]string, len(currentVariant.Values)+len(currentVariant.Fields))
		if hasCurrentVariant {
			for key, value := range currentVariant.Values {
				normalizedKey := strings.TrimSpace(key)
				if normalizedKey == "" {
					continue
				}
				currentValueByKey[normalizedKey] = strings.TrimSpace(value)
			}
			for _, field := range currentVariant.Fields {
				normalizedKey := strings.TrimSpace(field.Key)
				if normalizedKey == "" {
					continue
				}
				currentValueByKey[normalizedKey] = strings.TrimSpace(field.Value)
			}
		}

		values := copyProfileValueMap(variantSpec.Values)
		for _, item := range spec.Fields {
			value := strings.TrimSpace(values[item.Key])
			if current, ok := currentValueByKey[item.Key]; ok && current != "" {
				value = current
			} else if legacy, ok := legacyByKey[item.Key]; ok && strings.TrimSpace(legacy.Value) != "" {
				value = strings.TrimSpace(legacy.Value)
			}
			if item.Key == "sheetName" && value == "" && strings.TrimSpace(cfg.Defaults.SheetName) != "" {
				value = strings.TrimSpace(cfg.Defaults.SheetName)
			}
			if item.Key == "headerRowNumber" && value == "" && cfg.Defaults.HeaderRowNumber > 0 {
				value = strconv.Itoa(cfg.Defaults.HeaderRowNumber)
			}
			values[item.Key] = value
		}

		variants = append(variants, ProfileConfigVariant{
			Key:         variantSpec.Key,
			Label:       variantSpec.Label,
			Description: variantSpec.Description,
			Values:      values,
		})
	}

	return ProfileConfig{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Defaults: ProfileConfigDefaults{
			CashflowFormat: defaults.CashflowFormat,
		},
		Variants: variants,
	}
}

func copyProfileValueMap(source map[string]string) map[string]string {
	out := make(map[string]string, len(source))
	for key, value := range source {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		out[normalizedKey] = strings.TrimSpace(value)
	}
	return out
}

func (s *ProfileConfigService) configPath(profileID string, key ProfileConfigKey) string {
	switch key {
	case ProfileConfigReceiveMoney:
		return profilepath.CashflowReceiveMoneyConfigJSON(s.rootDir, profileID)
	default:
		return profilepath.CashflowSpendMoneyConfigJSON(s.rootDir, profileID)
	}
}

func buildHeaderRowProfileOptions(max int) []ProfileConfigFieldOption {
	return specutil.IntOptions(max, func(label string, value string) ProfileConfigFieldOption {
		return ProfileConfigFieldOption{Label: label, Value: value}
	})
}

func cashflowTextField(key string, label string, required bool, defaultValue string, description string, group string) ProfileConfigFieldSpec {
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

func cashflowSelectField(key string, label string, required bool, defaultValue string, description string, group string, options []ProfileConfigFieldOption) ProfileConfigFieldSpec {
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

func cashflowTextareaField(key string, label string, required bool, defaultValue string, description string, group string) ProfileConfigFieldSpec {
	return ProfileConfigFieldSpec{
		Key:          key,
		Label:        label,
		Required:     required,
		DefaultValue: defaultValue,
		Description:  description,
		Kind:         "textarea",
		Group:        group,
	}
}
