package cashflow

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/specutil"
	"github.com/sieryo/invoice-extractor/internal/profilepath"
)

const cashflowProfileConfigSchemaVersion = "2"

type ProfileConfigKey string

const (
	ProfileConfigSpendMoney   ProfileConfigKey = "spend_money"
	ProfileConfigReceiveMoney ProfileConfigKey = "receive_money"
)

type ProfileConfigSpec struct {
	SchemaVersion    string                    `json:"schemaVersion"`
	ConfigKey        string                    `json:"configKey"`
	Label            string                    `json:"label"`
	Description      string                    `json:"description,omitempty"`
	Defaults         ProfileConfigDefaults     `json:"defaults"`
	HeaderRowOptions []int                     `json:"headerRowOptions,omitempty"`
	FormatOptions    []ProfileConfigFormatSpec `json:"formatOptions,omitempty"`
	Variants         []ProfileConfigVariantSpec `json:"variants"`
}

type ProfileConfigDefaults struct {
	SheetName            string `json:"sheetName"`
	HeaderRowNumber      int    `json:"headerRowNumber"`
	StartingChequeNumber *int   `json:"startingChequeNumber,omitempty"`
	CashflowFormat       string `json:"cashflowFormat"`
}

type ProfileConfigFormatSpec struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type ProfileConfigFieldSpec struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Description  string `json:"description,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Group        string `json:"group,omitempty"`
}

type ProfileConfigVariantSpec struct {
	Key         string                   `json:"key"`
	Label       string                   `json:"label"`
	Description string                   `json:"description,omitempty"`
	Fields      []ProfileConfigFieldSpec `json:"fields"`
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
	Key         string               `json:"key"`
	Label       string               `json:"label"`
	Description string               `json:"description,omitempty"`
	Fields      []ProfileConfigField `json:"fields"`
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

	format := normalizeProfileConfigFormat(cfg.Defaults.CashflowFormat)
	resolved := ResolveProfileConfigValues(cfg, format)
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
	require("coa", "Chart of Account")
	require("total", "Total")

	if format == string(InfluencerFormat) {
		require("defaultIAccountCode", "Default Influencer Account Code")
		require("defaultBAccountCode", "Default Bank Account Code")
	} else {
		require("remarkDelimiter", "Remark Delimiter")
		require("otherCostsAccountCode", "Kode Akun Biaya Lain")
	}

	if len(missing) > 0 {
		return ProfileConfigStatus{
			Configured:    false,
			MissingFields: missing,
			Code:          "NOT_READY",
			Message:       "Default profil cashflow belum lengkap.",
			SchemaVersion: cashflowProfileConfigSchemaVersion,
		}
	}

	return ProfileConfigStatus{
		Configured:    true,
		Code:          "READY",
		Message:       "Default profil cashflow siap digunakan.",
		SchemaVersion: cashflowProfileConfigSchemaVersion,
	}
}

func ResolveProfileConfigValues(cfg ProfileConfig, format string) map[string]string {
	defaultFields := variantFieldMap(cfg.Variants, string(DefaultFormat))
	selectedFormat := normalizeProfileConfigFormat(format)
	if selectedFormat == string(DefaultFormat) {
		return defaultFields
	}

	merged := make(map[string]string, len(defaultFields))
	for key, value := range defaultFields {
		merged[key] = value
	}
	for key, value := range variantFieldMap(cfg.Variants, selectedFormat) {
		if strings.TrimSpace(value) == "" {
			continue
		}
		merged[key] = strings.TrimSpace(value)
	}
	return merged
}

func buildProfileConfigSpec(key ProfileConfigKey) ProfileConfigSpec {
	label := "Default Profil Cashflow Spend Money"
	description := "Nilai default untuk action export cashflow ke MYOB Spend Money."
	if key == ProfileConfigReceiveMoney {
		label = "Default Profil Cashflow Receive Money"
		description = "Nilai default untuk action export cashflow ke MYOB Receive Money."
	}

	return ProfileConfigSpec{
		SchemaVersion: cashflowProfileConfigSchemaVersion,
		ConfigKey:     string(key),
		Label:         label,
		Description:   description,
		Defaults: ProfileConfigDefaults{
			SheetName:       "",
			HeaderRowNumber: 1,
			CashflowFormat:  string(DefaultFormat),
		},
		HeaderRowOptions: specutil.HeaderRowNumbers(10),
		FormatOptions: []ProfileConfigFormatSpec{
			{Value: string(DefaultFormat), Label: "Default"},
			{Value: string(InfluencerFormat), Label: "Influencer"},
		},
		Variants: []ProfileConfigVariantSpec{
			{
				Key:         string(DefaultFormat),
				Label:       "Format Default",
				Description: "Nilai dasar utama untuk file cashflow reguler.",
				Fields:      buildProfileVariantFieldSpecs(key, DefaultFormat),
			},
			{
				Key:         string(InfluencerFormat),
				Label:       "Format Influencer",
				Description: "Hanya isi field yang memang berbeda. Field kosong akan mengikuti Format Default.",
				Fields:      buildProfileVariantFieldSpecs(key, InfluencerFormat),
			},
		},
	}
}

func buildProfileVariantFieldSpecs(key ProfileConfigKey, format Format) []ProfileConfigFieldSpec {
	defaults := defaultVariantFieldValues(key)
	if format == InfluencerFormat {
		for overrideKey, overrideValue := range influencerVariantOverrides() {
			defaults[overrideKey] = overrideValue
		}
	}

	blueprints := profileFieldBlueprints(key)
	fields := make([]ProfileConfigFieldSpec, 0, len(blueprints))
	for _, blueprint := range blueprints {
		defaultValue := defaults[blueprint.Key]
		if format == InfluencerFormat && !isInfluencerOverrideKey(blueprint.Key) {
			defaultValue = ""
		}
		fields = append(fields, ProfileConfigFieldSpec{
			Key:          blueprint.Key,
			Label:        blueprint.Label,
			Required:     blueprint.Required,
			DefaultValue: defaultValue,
			Description:  blueprint.Description,
			Kind:         blueprint.Kind,
			Group:        blueprint.Group,
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

	return []ProfileConfigFieldSpec{
		{Key: "outputFilename", Label: "Nama Output", Required: true, DefaultValue: outputDefault, Description: "Tanpa ekstensi file.", Group: "parameter"},
		{Key: "chequeAccount", Label: accountLabel, Required: true, DefaultValue: "12021", Description: accountDescription, Group: "parameter"},
		{Key: "remarkDelimiter", Label: "Remark Delimiter", Required: false, DefaultValue: "*", Description: "Dipakai untuk memecah catatan biaya lainnya.", Group: "parameter"},
		{Key: "otherCostsAccountCode", Label: "Kode Akun Biaya Lain", Required: false, DefaultValue: "62099", Description: "Dipakai saat format default memiliki komponen biaya lain.", Group: "parameter"},
		{Key: "defaultIAccountCode", Label: "Default Influencer Account Code", Required: false, DefaultValue: "62004", Description: "Dipakai saat format influencer memilih account influencer.", Group: "parameter"},
		{Key: "defaultBAccountCode", Label: "Default Bank Account Code", Required: false, DefaultValue: "90900", Description: "Dipakai saat format influencer mendeteksi transaksi bank.", Group: "parameter"},
		{Key: "date", Label: "Tanggal", Required: true, DefaultValue: "Date", Description: "Header source untuk tanggal transaksi.", Group: "mapping"},
		{Key: "information", Label: "Keterangan", Required: true, DefaultValue: "Note", Description: "Header source untuk memo utama.", Group: "mapping"},
		{Key: "coa", Label: "Chart of Account", Required: true, DefaultValue: "MYOB", Description: "Header source untuk COA atau nama account.", Group: "mapping"},
		{Key: "otherCost", Label: "Biaya Lainnya", Required: false, DefaultValue: "By Lainnya", Description: "Header source untuk biaya tambahan.", Group: "mapping"},
		{Key: "pp23", Label: "PP 23", Required: false, DefaultValue: "PP 23", Description: "Header source komponen pajak PP 23.", Group: "mapping"},
		{Key: "pph15", Label: "PPh 15%", Required: false, DefaultValue: "PPh 15%", Description: "Header source komponen pajak PPh 15%.", Group: "mapping"},
		{Key: "pph21", Label: "PPH 21", Required: false, DefaultValue: "PPH 21", Description: "Header source komponen pajak PPH 21.", Group: "mapping"},
		{Key: "pph23", Label: "PPH 23", Required: false, DefaultValue: "PPH 23", Description: "Header source komponen pajak PPH 23.", Group: "mapping"},
		{Key: "pph42", Label: "PPH 4 (2)", Required: false, DefaultValue: "pph 4(2)", Description: "Header source komponen pajak PPH 4 (2).", Group: "mapping"},
		{Key: "ppn", Label: "PPN", Required: false, DefaultValue: "PPN", Description: "Header source komponen pajak PPN.", Group: "mapping"},
		{Key: "remark", Label: "Catatan", Required: false, DefaultValue: "Remark", Description: "Header source catatan atau allocation memo.", Group: "mapping"},
		{Key: "total", Label: "Total", Required: true, DefaultValue: "IDR", Description: "Header source total transaksi.", Group: "mapping"},
	}
}

func defaultVariantFieldValues(key ProfileConfigKey) map[string]string {
	values := make(map[string]string)
	for _, field := range profileFieldBlueprints(key) {
		values[field.Key] = field.DefaultValue
	}
	return values
}

func influencerVariantOverrides() map[string]string {
	return map[string]string{
		"defaultIAccountCode": "62004",
		"defaultBAccountCode": "90900",
		"date":                "*Posting Date: # date",
		"information":         "Notes",
		"remark":              "WHT",
		"coa":                 "WHT CoA",
		"pph42":               "PPh 4 (2)",
		"otherCost":           "Biaya Lainnya",
		"pph15":               "Biaya Lainnya",
	}
}

func isInfluencerOverrideKey(key string) bool {
	_, ok := influencerVariantOverrides()[key]
	return ok
}

func variantFieldMap(variants []ProfileConfigVariant, variantKey string) map[string]string {
	out := make(map[string]string)
	for _, variant := range variants {
		if !strings.EqualFold(strings.TrimSpace(variant.Key), strings.TrimSpace(variantKey)) {
			continue
		}
		for _, field := range variant.Fields {
			out[strings.TrimSpace(field.Key)] = strings.TrimSpace(field.Value)
		}
		break
	}
	return out
}

func normalizeProfileConfigFormat(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(InfluencerFormat):
		return string(InfluencerFormat)
	default:
		return string(DefaultFormat)
	}
}

func (s *ProfileConfigService) defaultConfig(key ProfileConfigKey) ProfileConfig {
	spec := s.Spec(key)
	variants := make([]ProfileConfigVariant, 0, len(spec.Variants))
	for _, variantSpec := range spec.Variants {
		fields := make([]ProfileConfigField, 0, len(variantSpec.Fields))
		for _, item := range variantSpec.Fields {
			fields = append(fields, ProfileConfigField{
				Key:         item.Key,
				Label:       item.Label,
				Required:    item.Required,
				Value:       item.DefaultValue,
				Description: item.Description,
				Kind:        item.Kind,
				Group:       item.Group,
			})
		}
		variants = append(variants, ProfileConfigVariant{
			Key:         variantSpec.Key,
			Label:       variantSpec.Label,
			Description: variantSpec.Description,
			Fields:      fields,
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
	if strings.TrimSpace(cfg.Defaults.SheetName) != "" {
		defaults.SheetName = strings.TrimSpace(cfg.Defaults.SheetName)
	}
	if cfg.Defaults.HeaderRowNumber > 0 {
		defaults.HeaderRowNumber = cfg.Defaults.HeaderRowNumber
	}
	if cfg.Defaults.StartingChequeNumber != nil && *cfg.Defaults.StartingChequeNumber > 0 {
		value := *cfg.Defaults.StartingChequeNumber
		defaults.StartingChequeNumber = &value
	}
	if strings.TrimSpace(cfg.Defaults.CashflowFormat) != "" {
		defaults.CashflowFormat = normalizeProfileConfigFormat(cfg.Defaults.CashflowFormat)
	} else if legacyFormat, ok := legacyByKey["cashflowFormat"]; ok && strings.TrimSpace(legacyFormat.Value) != "" {
		defaults.CashflowFormat = normalizeProfileConfigFormat(legacyFormat.Value)
	}

	variants := make([]ProfileConfigVariant, 0, len(spec.Variants))
	for _, variantSpec := range spec.Variants {
		currentVariant, hasCurrentVariant := variantByKey[variantSpec.Key]
		currentFieldByKey := make(map[string]ProfileConfigField, len(currentVariant.Fields))
		if hasCurrentVariant {
			for _, field := range currentVariant.Fields {
				currentFieldByKey[strings.TrimSpace(field.Key)] = field
			}
		}

		fields := make([]ProfileConfigField, 0, len(variantSpec.Fields))
		for _, item := range variantSpec.Fields {
			field := ProfileConfigField{
				Key:         item.Key,
				Label:       item.Label,
				Required:    item.Required,
				Value:       item.DefaultValue,
				Description: item.Description,
				Kind:        item.Kind,
				Group:       item.Group,
			}

			if current, ok := currentFieldByKey[item.Key]; ok {
				field.Value = strings.TrimSpace(current.Value)
			} else if variantSpec.Key == string(DefaultFormat) {
				if legacy, ok := legacyByKey[item.Key]; ok {
					field.Value = strings.TrimSpace(legacy.Value)
				}
			}

			fields = append(fields, field)
		}

		variants = append(variants, ProfileConfigVariant{
			Key:         variantSpec.Key,
			Label:       variantSpec.Label,
			Description: variantSpec.Description,
			Fields:      fields,
		})
	}

	return ProfileConfig{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Defaults:      defaults,
		Variants:      variants,
	}
}

func (s *ProfileConfigService) configPath(profileID string, key ProfileConfigKey) string {
	switch key {
	case ProfileConfigReceiveMoney:
		return profilepath.CashflowReceiveMoneyConfigJSON(s.rootDir, profileID)
	default:
		return profilepath.CashflowSpendMoneyConfigJSON(s.rootDir, profileID)
	}
}
