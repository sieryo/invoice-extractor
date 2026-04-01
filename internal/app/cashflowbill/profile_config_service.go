package cashflowbill

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/specutil"
	"github.com/sieryo/invoice-extractor/internal/profilepath"
)

const profileConfigSchemaVersion = "1"

type ProfileConfigKey string

const (
	ProfileConfigPayBills        ProfileConfigKey = "pay_bills"
	ProfileConfigReceivePayments ProfileConfigKey = "receive_payments"
)

type ProfileConfigSpec struct {
	SchemaVersion    string                     `json:"schemaVersion"`
	ConfigKey        string                     `json:"configKey"`
	Label            string                     `json:"label"`
	Description      string                     `json:"description,omitempty"`
	Defaults         ProfileConfigDefaults      `json:"defaults"`
	HeaderRowOptions []int                      `json:"headerRowOptions,omitempty"`
	FormatOptions    []ProfileConfigFormatSpec  `json:"formatOptions,omitempty"`
	Fields           []ProfileConfigFieldSpec   `json:"fields"`
	Variants         []ProfileConfigVariantSpec `json:"variants"`
}

type ProfileConfigDefaults struct {
	CashflowFormat string `json:"cashflowFormat"`
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
}

type ProfileConfigVariant struct {
	Key         string            `json:"key"`
	Label       string            `json:"label"`
	Description string            `json:"description,omitempty"`
	Values      map[string]string `json:"values"`
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
			Message:       "Default profil cashflow bills tidak dapat dibaca saat ini.",
			SchemaVersion: profileConfigSchemaVersion,
		}
	}

	missing := profileMissingFields(cfg)
	if len(missing) > 0 {
		return ProfileConfigStatus{
			Configured:    false,
			MissingFields: missing,
			Code:          "NOT_READY",
			Message:       "Default profil cashflow bills belum lengkap.",
			SchemaVersion: profileConfigSchemaVersion,
		}
	}

	return ProfileConfigStatus{
		Configured:    true,
		Code:          "READY",
		Message:       "Default profil cashflow bills siap digunakan.",
		SchemaVersion: profileConfigSchemaVersion,
	}
}

func ResolveProfileConfigValues(cfg ProfileConfig) map[string]string {
	for _, variant := range cfg.Variants {
		return copyProfileValueMap(variant.Values)
	}
	return nil
}

func buildProfileConfigSpec(key ProfileConfigKey) ProfileConfigSpec {
	label := "Cashflow Pay Bills"
	description := "Nilai default untuk action cashflow ke MYOB Pay Bills."
	outputDefault := "pay_bills"
	accountLabel := "Payment Account"
	accountDescription := "Akun payment utama untuk output MYOB."
	if key == ProfileConfigReceivePayments {
		label = "Cashflow Receive Payments"
		description = "Nilai default untuk action cashflow ke MYOB Receive Payments."
		outputDefault = "receive_payments"
		accountLabel = "Deposit Account"
		accountDescription = "Akun deposit utama untuk output MYOB."
	}

	return ProfileConfigSpec{
		SchemaVersion: profileConfigSchemaVersion,
		ConfigKey:     string(key),
		Label:         label,
		Description:   description,
		Defaults: ProfileConfigDefaults{
			CashflowFormat: "default",
		},
		HeaderRowOptions: specutil.HeaderRowNumbers(10),
		Fields: []ProfileConfigFieldSpec{
			{Key: "sheetName", Label: "Sheet Default", Required: false, DefaultValue: "", Description: "Nilai awal sheet untuk action ini.", Group: "source"},
			{Key: "headerRowNumber", Label: "Baris Header Default", Required: true, DefaultValue: "1", Description: "Baris header default workbook.", Kind: "select", Group: "source", Options: buildHeaderRowProfileOptions(10)},
			{Key: "outputFilename", Label: "Nama Output", Required: true, DefaultValue: outputDefault, Description: "Tanpa ekstensi file.", Kind: "text", Group: "parameter"},
			{Key: "chequeAccount", Label: accountLabel, Required: true, DefaultValue: "12021", Description: accountDescription, Kind: "text", Group: "parameter"},
			{Key: "date", Label: "Tanggal", Required: true, DefaultValue: "date", Description: "Header source untuk tanggal transaksi.", Kind: "text", Group: "mapping"},
			{Key: "category", Label: "Category", Required: true, DefaultValue: "Category", Description: "Header source category.", Kind: "text", Group: "mapping"},
			{Key: "information", Label: "Keterangan", Required: true, DefaultValue: "Note", Description: "Header source memo transaksi.", Kind: "text", Group: "mapping"},
			{Key: "partyName", Label: "Nama Customer / Supplier", Required: true, DefaultValue: "nama customer / supplier", Description: "Header source nama pihak.", Kind: "text", Group: "mapping"},
			{Key: "total", Label: "Total", Required: true, DefaultValue: "idr", Description: "Header source nominal pembayaran.", Kind: "text", Group: "mapping"},
		},
		Variants: []ProfileConfigVariantSpec{
			{
				Key:         "default",
				Label:       "Default",
				Description: "Konfigurasi default action cashflow bills.",
				Values: map[string]string{
					"sheetName":       "",
					"headerRowNumber": "1",
					"outputFilename":  outputDefault,
					"chequeAccount":   "12021",
					"date":            "date",
					"category":        "Category",
					"information":     "Note",
					"partyName":       "nama customer / supplier",
					"total":           "idr",
				},
			},
		},
	}
}

func profileMissingFields(cfg ProfileConfig) []string {
	resolved := ResolveProfileConfigValues(cfg)
	missing := make([]string, 0)
	require := func(fieldKey string, label string) {
		if strings.TrimSpace(resolved[fieldKey]) == "" {
			missing = append(missing, label)
		}
	}

	require("headerRowNumber", "Baris Header Default")
	require("outputFilename", "Nama Output")
	require("chequeAccount", "Akun Utama")
	require("date", "Tanggal")
	require("category", "Category")
	require("information", "Keterangan")
	require("partyName", "Nama Customer / Supplier")
	require("total", "Total")
	return missing
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
	defaults := spec.Defaults
	if strings.TrimSpace(cfg.Defaults.CashflowFormat) != "" {
		defaults.CashflowFormat = strings.TrimSpace(cfg.Defaults.CashflowFormat)
	}

	currentByKey := map[string]string{}
	for _, variant := range cfg.Variants {
		for fieldKey, value := range variant.Values {
			currentByKey[strings.TrimSpace(fieldKey)] = strings.TrimSpace(value)
		}
	}

	values := copyProfileValueMap(spec.Variants[0].Values)
	for _, field := range spec.Fields {
		if current, ok := currentByKey[field.Key]; ok && current != "" {
			values[field.Key] = current
		}
		if field.Key == "headerRowNumber" {
			if _, err := strconv.Atoi(strings.TrimSpace(values[field.Key])); err != nil {
				values[field.Key] = field.DefaultValue
			}
		}
	}

	return ProfileConfig{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Defaults:      defaults,
		Variants: []ProfileConfigVariant{
			{
				Key:         spec.Variants[0].Key,
				Label:       spec.Variants[0].Label,
				Description: spec.Variants[0].Description,
				Values:      values,
			},
		},
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
	case ProfileConfigReceivePayments:
		return profilepath.CashflowReceivePaymentsConfigJSON(s.rootDir, profileID)
	default:
		return profilepath.CashflowPayBillsConfigJSON(s.rootDir, profileID)
	}
}

func buildHeaderRowProfileOptions(max int) []ProfileConfigFieldOption {
	options := make([]ProfileConfigFieldOption, 0, max)
	for _, value := range specutil.HeaderRowNumbers(max) {
		label := strings.TrimSpace(strconv.Itoa(value))
		options = append(options, ProfileConfigFieldOption{
			Label: label,
			Value: label,
		})
	}
	return options
}
