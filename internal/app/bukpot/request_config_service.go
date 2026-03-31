package bukpot

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

const bukpotRequestConfigSchemaVersion = "1"

type RequestConfigSpec struct {
	SchemaVersion    string                   `json:"schemaVersion"`
	Defaults         RequestConfigDefaults    `json:"defaults"`
	HeaderRowOptions []int                    `json:"headerRowOptions,omitempty"`
	DefaultFields    []RequestConfigFieldSpec `json:"defaultFields,omitempty"`
	Fields           []RequestConfigFieldSpec `json:"fields"`
}

type RequestConfigDefaults struct {
	SheetName       string `json:"sheetName"`
	HeaderRowNumber int    `json:"headerRowNumber"`
}

type RequestConfigFieldSpec struct {
	Key          string `json:"key"`
	Label        string `json:"label"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"defaultValue,omitempty"`
	Description  string `json:"description,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Group        string `json:"group,omitempty"`
	Options      []RequestConfigFieldOption `json:"options,omitempty"`
}

type RequestConfigFieldOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type RequestConfig struct {
	SchemaVersion string                `json:"schemaVersion"`
	Defaults      RequestConfigDefaults `json:"defaults"`
	Fields        []RequestConfigField  `json:"fields"`
}

type RequestConfigField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Kind        string `json:"kind,omitempty"`
	Group       string `json:"group,omitempty"`
}

type RequestConfigStatus struct {
	Configured    bool     `json:"configured"`
	MissingFields []string `json:"missingFields,omitempty"`
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	SchemaVersion string   `json:"schemaVersion"`
}

type RequestConfigService struct {
	rootDir string
}

func NewRequestConfigService(rootDir string) *RequestConfigService {
	return &RequestConfigService{rootDir: rootDir}
}

func (s *RequestConfigService) Spec() RequestConfigSpec {
	return RequestConfigSpec{
		SchemaVersion: bukpotRequestConfigSchemaVersion,
		Defaults: RequestConfigDefaults{
			SheetName:       "",
			HeaderRowNumber: 1,
		},
		HeaderRowOptions: specutil.HeaderRowNumbers(10),
		DefaultFields: []RequestConfigFieldSpec{
			{Key: "sheetName", Label: "Sheet Default", Required: false, DefaultValue: "", Description: "Nilai awal sheet saat action dibuka.", Kind: "text", Group: "source"},
			{Key: "headerRowNumber", Label: "Baris Header Default", Required: true, DefaultValue: "1", Description: "Baris header default untuk source Excel.", Kind: "select", Group: "source", Options: buildHeaderRowRequestOptions(10)},
		},
		Fields: []RequestConfigFieldSpec{
			{Key: "entity", Label: "Entity", Required: true, DefaultValue: "Entity", Description: "Dipakai untuk filter alias profile.", Kind: "text", Group: "mapping"},
			{Key: "settlementDate", Label: "Settlement Date", Required: true, DefaultValue: "Settlemet Date", Description: "Dipakai untuk masa pajak, tahun pajak, dan tanggal pemotongan.", Kind: "text", Group: "mapping"},
			{Key: "npwp", Label: "NPWP", Required: true, DefaultValue: "NPWP", Description: "NPWP penerima penghasilan dari source Excel.", Kind: "text", Group: "mapping"},
			{Key: "nitku", Label: "NITKU", Required: true, DefaultValue: "NITKU", Description: "ID TKU penerima penghasilan.", Kind: "text", Group: "mapping"},
			{Key: "facility", Label: "Fasilitas", Required: false, DefaultValue: "Fasilitas", Description: "Dipakai jika logic fasilitas dan tarif aktif.", Kind: "text", Group: "mapping"},
			{Key: "taxObjectCode", Label: "Kode Objek Pajak", Required: true, DefaultValue: "Kode Objek Pajak", Description: "Kode objek pajak untuk output Coretax.", Kind: "text", Group: "mapping"},
			{Key: "taxBase", Label: "DPP", Required: true, DefaultValue: "(Rp)Total Invoice (Exc VAT)", Description: "Nilai dasar pengenaan pajak.", Kind: "text", Group: "mapping"},
			{Key: "withholdingRate", Label: "Tarif / WHT", Required: true, DefaultValue: "WHT", Description: "Tarif pemotongan dari source Excel.", Kind: "text", Group: "mapping"},
			{Key: "taxInvoiceNumber", Label: "Faktur Pajak No", Required: false, DefaultValue: "Faktur Pajak No", Description: "Dipakai untuk jenis dan nomor dokumen referensi.", Kind: "text", Group: "mapping"},
			{Key: "referenceNumber", Label: "Invoice / Kwitansi No", Required: true, DefaultValue: "Invoice / Kwitansi No", Description: "Fallback nomor dokumen referensi jika faktur pajak kosong.", Kind: "text", Group: "mapping"},
			{Key: "referenceDate", Label: "FP DATE", Required: true, DefaultValue: "FP DATE", Description: "Tanggal dokumen referensi.", Kind: "text", Group: "mapping"},
		},
	}
}

func (s *RequestConfigService) Load(profileID string) (RequestConfig, error) {
	seed := s.defaultConfig()
	path := profilepath.BukpotRequestConfigJSON(s.rootDir, profileID)

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return seed, nil
		}
		return RequestConfig{}, err
	}

	var cfg RequestConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return RequestConfig{}, err
	}

	return s.normalizeConfig(cfg), nil
}

func (s *RequestConfigService) Update(profileID string, cfg RequestConfig) (RequestConfig, error) {
	normalized := s.normalizeConfig(cfg)
	path := profilepath.BukpotRequestConfigJSON(s.rootDir, profileID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return RequestConfig{}, err
	}
	b, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return RequestConfig{}, err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return RequestConfig{}, err
	}
	return normalized, nil
}

func (s *RequestConfigService) Status(profileID string) RequestConfigStatus {
	cfg, err := s.Load(profileID)
	if err != nil {
		return RequestConfigStatus{
			Configured:    false,
			Code:          "CONFIG_ERROR",
			Message:       "Konfigurasi request bukpot tidak dapat dibaca saat ini.",
			SchemaVersion: bukpotRequestConfigSchemaVersion,
		}
	}

	missing := make([]string, 0)
	for _, field := range cfg.Fields {
		if field.Required && strings.TrimSpace(field.Value) == "" {
			missing = append(missing, field.Label)
		}
	}

	if len(missing) > 0 {
		return RequestConfigStatus{
			Configured:    false,
			MissingFields: missing,
			Code:          "NOT_READY",
			Message:       "Default profil request bukpot belum lengkap.",
			SchemaVersion: bukpotRequestConfigSchemaVersion,
		}
	}

	return RequestConfigStatus{
		Configured:    true,
		Code:          "READY",
		Message:       "Default profil request bukpot siap digunakan.",
		SchemaVersion: bukpotRequestConfigSchemaVersion,
	}
}

func (s *RequestConfigService) defaultConfig() RequestConfig {
	spec := s.Spec()
	fields := make([]RequestConfigField, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		fields = append(fields, RequestConfigField{
			Key:         item.Key,
			Label:       item.Label,
			Required:    item.Required,
			Value:       item.DefaultValue,
			Description: item.Description,
			Kind:        item.Kind,
			Group:       item.Group,
		})
	}
	return RequestConfig{
		SchemaVersion: spec.SchemaVersion,
		Defaults:      spec.Defaults,
		Fields:        fields,
	}
}

func (s *RequestConfigService) normalizeConfig(cfg RequestConfig) RequestConfig {
	spec := s.Spec()
	byKey := make(map[string]RequestConfigField, len(cfg.Fields))
	for _, field := range cfg.Fields {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			continue
		}
		byKey[key] = field
	}

	fields := make([]RequestConfigField, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		field := RequestConfigField{
			Key:         item.Key,
			Label:       item.Label,
			Required:    item.Required,
			Value:       item.DefaultValue,
			Description: item.Description,
			Kind:        item.Kind,
			Group:       item.Group,
		}
		if current, ok := byKey[item.Key]; ok {
			if value := strings.TrimSpace(current.Value); value != "" {
				field.Value = value
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

	return RequestConfig{
		SchemaVersion: spec.SchemaVersion,
		Defaults:      defaults,
		Fields:        fields,
	}
}

func buildHeaderRowRequestOptions(max int) []RequestConfigFieldOption {
	options := make([]RequestConfigFieldOption, 0, max)
	for _, value := range specutil.HeaderRowNumbers(max) {
		label := strings.TrimSpace(strconv.Itoa(value))
		options = append(options, RequestConfigFieldOption{
			Label: label,
			Value: label,
		})
	}
	return options
}
