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

const cashflowProfileConfigSchemaVersion = "1"

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
	Fields           []ProfileConfigFieldSpec  `json:"fields"`
}

type ProfileConfigDefaults struct {
	SheetName            string `json:"sheetName"`
	HeaderRowNumber      int    `json:"headerRowNumber"`
	StartingChequeNumber *int   `json:"startingChequeNumber,omitempty"`
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

type ProfileConfig struct {
	SchemaVersion string                `json:"schemaVersion"`
	ConfigKey     string                `json:"configKey"`
	Defaults      ProfileConfigDefaults `json:"defaults"`
	Fields        []ProfileConfigField  `json:"fields"`
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
	switch key {
	case ProfileConfigReceiveMoney:
		return ProfileConfigSpec{
			SchemaVersion: cashflowProfileConfigSchemaVersion,
			ConfigKey:     string(key),
			Label:         "Default Profil Cashflow Receive Money",
			Description:   "Nilai default untuk action export cashflow ke MYOB Receive Money.",
			Defaults: ProfileConfigDefaults{
				SheetName:       "",
				HeaderRowNumber: 1,
			},
			HeaderRowOptions: specutil.HeaderRowNumbers(10),
			FormatOptions: []ProfileConfigFormatSpec{
				{Value: string(DefaultFormat), Label: "Default"},
				{Value: string(InfluencerFormat), Label: "Influencer"},
			},
			Fields: []ProfileConfigFieldSpec{
				{Key: "outputFilename", Label: "Nama Output", Required: true, DefaultValue: "receive_money", Description: "Tanpa ekstensi file.", Group: "parameter"},
				{Key: "chequeAccount", Label: "Deposit Account", Required: true, DefaultValue: "12021", Description: "Akun deposit utama untuk output MYOB.", Group: "parameter"},
				{Key: "cashflowFormat", Label: "Format Cashflow", Required: true, DefaultValue: string(DefaultFormat), Description: "Pilih format source cashflow.", Kind: "select", Group: "parameter"},
				{Key: "remarkDelimiter", Label: "Remark Delimiter", Required: false, DefaultValue: "*", Description: "Wajib jika format cashflow menggunakan mode default.", Group: "parameter"},
				{Key: "otherCostsAccountCode", Label: "Kode Akun Biaya Lain", Required: false, DefaultValue: "62099", Description: "Wajib untuk mode default.", Group: "parameter"},
				{Key: "defaultIAccountCode", Label: "Default Influencer Account Code", Required: false, DefaultValue: "62004", Description: "Wajib untuk mode influencer.", Group: "parameter"},
				{Key: "defaultBAccountCode", Label: "Default Bank Account Code", Required: false, DefaultValue: "90900", Description: "Wajib untuk mode influencer.", Group: "parameter"},
				{Key: "date", Label: "Tanggal", Required: true, DefaultValue: "Tanggal", Description: "Header source untuk tanggal transaksi.", Group: "mapping"},
				{Key: "information", Label: "Keterangan", Required: true, DefaultValue: "note", Description: "Header source untuk memo utama.", Group: "mapping"},
				{Key: "coa", Label: "Chart of Account", Required: true, DefaultValue: "coa", Description: "Header source untuk COA atau nama account.", Group: "mapping"},
				{Key: "otherCost", Label: "Biaya Lainnya", Required: false, DefaultValue: "By Lainnya", Description: "Header source untuk biaya tambahan.", Group: "mapping"},
				{Key: "pp23", Label: "PP 23", Required: false, DefaultValue: "PP 23", Description: "Header source komponen pajak PP 23.", Group: "mapping"},
				{Key: "pph15", Label: "PPh 15%", Required: false, DefaultValue: "PPh 15%", Description: "Header source komponen pajak PPh 15%.", Group: "mapping"},
				{Key: "pph21", Label: "PPH 21", Required: false, DefaultValue: "PPH 21", Description: "Header source komponen pajak PPH 21.", Group: "mapping"},
				{Key: "pph23", Label: "PPH 23", Required: false, DefaultValue: "PPH 23", Description: "Header source komponen pajak PPH 23.", Group: "mapping"},
				{Key: "pph42", Label: "PPH 4 (2)", Required: false, DefaultValue: "PPH 4(2)", Description: "Header source komponen pajak PPH 4 (2).", Group: "mapping"},
				{Key: "ppn", Label: "PPN", Required: false, DefaultValue: "PPN", Description: "Header source komponen pajak PPN.", Group: "mapping"},
				{Key: "remark", Label: "Catatan", Required: false, DefaultValue: "catatan", Description: "Header source catatan atau allocation memo.", Group: "mapping"},
				{Key: "total", Label: "Total", Required: true, DefaultValue: "idr", Description: "Header source total transaksi.", Group: "mapping"},
			},
		}
	default:
		return ProfileConfigSpec{
			SchemaVersion: cashflowProfileConfigSchemaVersion,
			ConfigKey:     string(ProfileConfigSpendMoney),
			Label:         "Default Profil Cashflow Spend Money",
			Description:   "Nilai default untuk action export cashflow ke MYOB Spend Money.",
			Defaults: ProfileConfigDefaults{
				SheetName:       "",
				HeaderRowNumber: 1,
			},
			HeaderRowOptions: specutil.HeaderRowNumbers(10),
			FormatOptions: []ProfileConfigFormatSpec{
				{Value: string(DefaultFormat), Label: "Default"},
				{Value: string(InfluencerFormat), Label: "Influencer"},
			},
			Fields: []ProfileConfigFieldSpec{
				{Key: "outputFilename", Label: "Nama Output", Required: true, DefaultValue: "spend_money", Description: "Tanpa ekstensi file.", Group: "parameter"},
				{Key: "chequeAccount", Label: "Cheque Account", Required: true, DefaultValue: "12021", Description: "Akun cheque utama untuk output MYOB.", Group: "parameter"},
				{Key: "cashflowFormat", Label: "Format Cashflow", Required: true, DefaultValue: string(DefaultFormat), Description: "Pilih format source cashflow.", Kind: "select", Group: "parameter"},
				{Key: "remarkDelimiter", Label: "Remark Delimiter", Required: false, DefaultValue: "*", Description: "Wajib jika format cashflow menggunakan mode default.", Group: "parameter"},
				{Key: "otherCostsAccountCode", Label: "Kode Akun Biaya Lain", Required: false, DefaultValue: "62099", Description: "Wajib untuk mode default.", Group: "parameter"},
				{Key: "defaultIAccountCode", Label: "Default Influencer Account Code", Required: false, DefaultValue: "62004", Description: "Wajib untuk mode influencer.", Group: "parameter"},
				{Key: "defaultBAccountCode", Label: "Default Bank Account Code", Required: false, DefaultValue: "90900", Description: "Wajib untuk mode influencer.", Group: "parameter"},
				{Key: "date", Label: "Tanggal", Required: true, DefaultValue: "Tanggal", Description: "Header source untuk tanggal transaksi.", Group: "mapping"},
				{Key: "information", Label: "Keterangan", Required: true, DefaultValue: "note", Description: "Header source untuk memo utama.", Group: "mapping"},
				{Key: "coa", Label: "Chart of Account", Required: true, DefaultValue: "coa", Description: "Header source untuk COA atau nama account.", Group: "mapping"},
				{Key: "otherCost", Label: "Biaya Lainnya", Required: false, DefaultValue: "By Lainnya", Description: "Header source untuk biaya tambahan.", Group: "mapping"},
				{Key: "pp23", Label: "PP 23", Required: false, DefaultValue: "PP 23", Description: "Header source komponen pajak PP 23.", Group: "mapping"},
				{Key: "pph15", Label: "PPh 15%", Required: false, DefaultValue: "PPh 15%", Description: "Header source komponen pajak PPh 15%.", Group: "mapping"},
				{Key: "pph21", Label: "PPH 21", Required: false, DefaultValue: "PPH 21", Description: "Header source komponen pajak PPH 21.", Group: "mapping"},
				{Key: "pph23", Label: "PPH 23", Required: false, DefaultValue: "PPH 23", Description: "Header source komponen pajak PPH 23.", Group: "mapping"},
				{Key: "pph42", Label: "PPH 4 (2)", Required: false, DefaultValue: "PPH 4(2)", Description: "Header source komponen pajak PPH 4 (2).", Group: "mapping"},
				{Key: "ppn", Label: "PPN", Required: false, DefaultValue: "PPN", Description: "Header source komponen pajak PPN.", Group: "mapping"},
				{Key: "remark", Label: "Catatan", Required: false, DefaultValue: "catatan", Description: "Header source catatan atau allocation memo.", Group: "mapping"},
				{Key: "total", Label: "Total", Required: true, DefaultValue: "idr", Description: "Header source total transaksi.", Group: "mapping"},
			},
		}
	}
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

	byKey := make(map[string]string, len(cfg.Fields))
	for _, field := range cfg.Fields {
		byKey[field.Key] = strings.TrimSpace(field.Value)
	}

	missing := make([]string, 0)
	require := func(fieldKey string, label string) {
		if strings.TrimSpace(byKey[fieldKey]) == "" {
			missing = append(missing, label)
		}
	}

	require("outputFilename", "Nama Output")
	require("chequeAccount", "Akun Utama")
	require("date", "Tanggal")
	require("information", "Keterangan")
	require("coa", "Chart of Account")
	require("total", "Total")
	format := strings.TrimSpace(byKey["cashflowFormat"])
	if format == "" {
		missing = append(missing, "Format Cashflow")
	} else if strings.EqualFold(format, string(InfluencerFormat)) {
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

func (s *ProfileConfigService) defaultConfig(key ProfileConfigKey) ProfileConfig {
	spec := s.Spec(key)
	fields := make([]ProfileConfigField, 0, len(spec.Fields))
	for _, item := range spec.Fields {
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

	return ProfileConfig{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Defaults:      spec.Defaults,
		Fields:        fields,
	}
}

func (s *ProfileConfigService) normalizeConfig(cfg ProfileConfig, key ProfileConfigKey) ProfileConfig {
	spec := s.Spec(key)
	byKey := make(map[string]ProfileConfigField, len(cfg.Fields))
	for _, field := range cfg.Fields {
		normalized := strings.TrimSpace(field.Key)
		if normalized == "" {
			continue
		}
		byKey[normalized] = field
	}

	fields := make([]ProfileConfigField, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		field := ProfileConfigField{
			Key:         item.Key,
			Label:       item.Label,
			Required:    item.Required,
			Value:       item.DefaultValue,
			Description: item.Description,
			Kind:        item.Kind,
			Group:       item.Group,
		}
		if current, ok := byKey[item.Key]; ok {
			trimmed := strings.TrimSpace(current.Value)
			if trimmed != "" {
				field.Value = trimmed
			} else if !item.Required {
				field.Value = ""
			}
		}
		fields = append(fields, field)
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

	return ProfileConfig{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Defaults:      defaults,
		Fields:        fields,
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
