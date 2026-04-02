package bukpot

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/configlayout"
	"github.com/sieryo/invoice-extractor/internal/app/specutil"
	"github.com/sieryo/invoice-extractor/internal/profilepath"
)

const bukpotRequestConfigSchemaVersion = "1"

type RequestConfigSpec struct {
	SchemaVersion    string                     `json:"schemaVersion"`
	Defaults         RequestConfigDefaults      `json:"defaults"`
	HeaderRowOptions []int                      `json:"headerRowOptions,omitempty"`
	Sections         []configlayout.SectionSpec `json:"sections,omitempty"`
	DefaultFields    []RequestConfigFieldSpec   `json:"defaultFields,omitempty"`
	Fields           []RequestConfigFieldSpec   `json:"fields"`
}

type RequestConfigDefaults struct {
	SheetName       string `json:"sheetName"`
	HeaderRowNumber int    `json:"headerRowNumber"`
}

type RequestConfigFieldSpec struct {
	Key          string                     `json:"key"`
	Label        string                     `json:"label"`
	Required     bool                       `json:"required"`
	DefaultValue string                     `json:"defaultValue,omitempty"`
	Description  string                     `json:"description,omitempty"`
	Kind         string                     `json:"kind,omitempty"`
	Group        string                     `json:"group,omitempty"`
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
		Sections: []configlayout.SectionSpec{
			specutil.ParameterActionSection(
				"Parameter default yang akan dipakai saat action dibuka.",
				2,
				"sheetName",
				"headerRowNumber",
			),
			specutil.MappingHeaderSection(
				"Nama header source yang akan dipakai sebagai nilai awal action.",
				2,
				"entity",
				"settlementDate",
				"npwp",
				"nitku",
				"facility",
				"taxObjectCode",
				"taxBase",
				"withholdingRate",
				"taxInvoiceNumber",
				"referenceNumber",
				"referenceDate",
			),
		},
		DefaultFields: []RequestConfigFieldSpec{
			requestTextField("sheetName", "Sheet Default", false, "", "Nilai awal sheet saat action dibuka.", "source"),
			requestSelectField("headerRowNumber", "Baris Header Default", true, "1", "Baris header default untuk source Excel.", "source", buildHeaderRowRequestOptions(10)),
		},
		Fields: []RequestConfigFieldSpec{
			requestTextField("entity", "Entity", true, "Entity", "Dipakai untuk filter alias profile.", "mapping"),
			requestTextField("settlementDate", "Settlement Date", true, "Settlemet Date", "Dipakai untuk masa pajak, tahun pajak, dan tanggal pemotongan.", "mapping"),
			requestTextField("npwp", "NPWP", true, "NPWP", "NPWP penerima penghasilan dari source Excel.", "mapping"),
			requestTextField("nitku", "NITKU", true, "NITKU", "ID TKU penerima penghasilan.", "mapping"),
			requestTextField("facility", "Fasilitas", false, "Fasilitas", "Dipakai jika logic fasilitas dan tarif aktif.", "mapping"),
			requestTextField("taxObjectCode", "Kode Objek Pajak", true, "Kode Objek Pajak", "Kode objek pajak untuk output Coretax.", "mapping"),
			requestTextField("taxBase", "DPP", true, "(Rp)Total Invoice (Exc VAT)", "Nilai dasar pengenaan pajak.", "mapping"),
			requestTextField("withholdingRate", "Tarif / WHT", true, "WHT", "Tarif pemotongan dari source Excel.", "mapping"),
			requestTextField("taxInvoiceNumber", "Faktur Pajak No", false, "Faktur Pajak No", "Dipakai untuk jenis dan nomor dokumen referensi.", "mapping"),
			requestTextField("referenceNumber", "Invoice / Kwitansi No", true, "Invoice / Kwitansi No", "Fallback nomor dokumen referensi jika faktur pajak kosong.", "mapping"),
			requestTextField("referenceDate", "FP DATE", true, "FP DATE", "Tanggal dokumen referensi.", "mapping"),
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
	return specutil.IntOptions(max, func(label string, value string) RequestConfigFieldOption {
		return RequestConfigFieldOption{Label: label, Value: value}
	})
}

func requestTextField(
	key string,
	label string,
	required bool,
	defaultValue string,
	description string,
	group string,
) RequestConfigFieldSpec {
	return RequestConfigFieldSpec{
		Key:          key,
		Label:        label,
		Required:     required,
		DefaultValue: defaultValue,
		Description:  description,
		Kind:         "text",
		Group:        group,
	}
}

func requestSelectField(
	key string,
	label string,
	required bool,
	defaultValue string,
	description string,
	group string,
	options []RequestConfigFieldOption,
) RequestConfigFieldSpec {
	return RequestConfigFieldSpec{
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
