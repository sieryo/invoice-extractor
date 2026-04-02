package bukpot

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

const bukpotActionProfileSchemaVersion = "1"

type ActionProfileKey string

const (
	ActionProfileBPPURenameBukpot     ActionProfileKey = "bukpot_bppu_rename_bukpot"
	ActionProfileBP21RenameBukpot     ActionProfileKey = "bukpot_bp21_rename_bukpot"
	ActionProfileBPA1RenameBukpot     ActionProfileKey = "bukpot_bpa1_rename_bukpot"
	ActionProfileBPPURenameByCategory ActionProfileKey = "bukpot_bppu_rename_by_category"
	ActionProfileBP21RenameByCategory ActionProfileKey = "bukpot_bp21_rename_by_category"
)

type ActionProfileSpec struct {
	SchemaVersion  string                     `json:"schemaVersion"`
	ConfigKey      string                     `json:"configKey"`
	Label          string                     `json:"label"`
	Description    string                     `json:"description,omitempty"`
	CollectionKind string                     `json:"collectionKind"`
	ActionType     string                     `json:"actionType"`
	Sections       []configlayout.SectionSpec `json:"sections,omitempty"`
	Fields         []ActionProfileFieldSpec   `json:"fields"`
}

type ActionProfileFieldSpec struct {
	Key          string                        `json:"key"`
	Label        string                        `json:"label"`
	Required     bool                          `json:"required"`
	DefaultValue string                        `json:"defaultValue,omitempty"`
	Description  string                        `json:"description,omitempty"`
	Kind         string                        `json:"kind,omitempty"`
	Suggestions  []ActionProfileSuggestionSpec `json:"suggestions,omitempty"`
	HelpText     string                        `json:"helpText,omitempty"`
	Placeholder  string                        `json:"placeholder,omitempty"`
}

type ActionProfileSuggestionSpec struct {
	Token       string `json:"token"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
}

type ActionProfile struct {
	SchemaVersion string               `json:"schemaVersion"`
	ConfigKey     string               `json:"configKey"`
	Fields        []ActionProfileField `json:"fields"`
}

type ActionProfileField struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
	Kind        string `json:"kind,omitempty"`
}

type ActionProfileStatus struct {
	Configured    bool     `json:"configured"`
	MissingFields []string `json:"missingFields,omitempty"`
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	SchemaVersion string   `json:"schemaVersion"`
}

type ActionProfileService struct {
	rootDir string
}

func NewActionProfileService(rootDir string) *ActionProfileService {
	return &ActionProfileService{rootDir: rootDir}
}

func ResolveActionProfileKey(
	collectionKind string,
	actionType string,
) (ActionProfileKey, bool) {
	switch collectionKind {
	case "bukpot_bppu":
		switch strings.TrimSpace(actionType) {
		case "rename_by_category":
			return ActionProfileBPPURenameByCategory, true
		case "rename_bukpot":
			return ActionProfileBPPURenameBukpot, true
		}
	case "bukpot_bp21":
		switch strings.TrimSpace(actionType) {
		case "rename_by_category":
			return ActionProfileBP21RenameByCategory, true
		case "rename_bukpot":
			return ActionProfileBP21RenameBukpot, true
		}
	case "bukpot_bpa1":
		if strings.TrimSpace(actionType) == "rename_bukpot" {
			return ActionProfileBPA1RenameBukpot, true
		}
	}

	return "", false
}

func (s *ActionProfileService) Spec(key ActionProfileKey) ActionProfileSpec {
	return buildActionProfileSpec(key)
}

func (s *ActionProfileService) Load(profileID string, key ActionProfileKey) (ActionProfile, error) {
	seed := s.defaultConfig(key)
	path := profilepath.BukpotActionProfileJSON(s.rootDir, profileID, string(key))

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return seed, nil
		}
		return ActionProfile{}, err
	}

	var cfg ActionProfile
	if err := json.Unmarshal(b, &cfg); err != nil {
		return ActionProfile{}, err
	}

	return s.normalizeConfig(cfg, key), nil
}

func (s *ActionProfileService) Update(profileID string, key ActionProfileKey, cfg ActionProfile) (ActionProfile, error) {
	normalized := s.normalizeConfig(cfg, key)
	path := profilepath.BukpotActionProfileJSON(s.rootDir, profileID, string(key))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ActionProfile{}, err
	}
	b, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return ActionProfile{}, err
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return ActionProfile{}, err
	}
	return normalized, nil
}

func (s *ActionProfileService) Status(profileID string, key ActionProfileKey) ActionProfileStatus {
	cfg, err := s.Load(profileID, key)
	if err != nil {
		return ActionProfileStatus{
			Configured:    false,
			Code:          "CONFIG_ERROR",
			Message:       "Default profil bukpot tidak dapat dibaca saat ini.",
			SchemaVersion: bukpotActionProfileSchemaVersion,
		}
	}

	spec := s.Spec(key)
	if len(spec.Fields) == 0 {
		return ActionProfileStatus{
			Configured:    true,
			Code:          "READY",
			Message:       "Action ini belum punya parameter yang perlu disimpan di default profil.",
			SchemaVersion: bukpotActionProfileSchemaVersion,
		}
	}

	missing := make([]string, 0)
	for _, field := range cfg.Fields {
		if field.Required && strings.TrimSpace(field.Value) == "" {
			missing = append(missing, field.Label)
		}
	}

	if len(missing) > 0 {
		return ActionProfileStatus{
			Configured:    false,
			MissingFields: missing,
			Code:          "NOT_READY",
			Message:       "Default profil bukpot belum lengkap.",
			SchemaVersion: bukpotActionProfileSchemaVersion,
		}
	}

	return ActionProfileStatus{
		Configured:    true,
		Code:          "READY",
		Message:       "Default profil bukpot siap digunakan.",
		SchemaVersion: bukpotActionProfileSchemaVersion,
	}
}

func buildActionProfileSpec(key ActionProfileKey) ActionProfileSpec {
	spec := ActionProfileSpec{
		SchemaVersion: bukpotActionProfileSchemaVersion,
		ConfigKey:     string(key),
		Sections:      []configlayout.SectionSpec{},
		Fields:        []ActionProfileFieldSpec{},
	}

	switch key {
	case ActionProfileBPPURenameBukpot:
		spec.Label = "BPPU Rename Bukpot"
		spec.Description = "Nilai default parameter untuk action Rename Bukpot pada collection BPPU."
		spec.CollectionKind = "bukpot_bppu"
		spec.ActionType = "rename_bukpot"
		spec.Sections = []configlayout.SectionSpec{
			bukpotActionSection("rename", "Pengaturan nama file default untuk action Rename Bukpot.", "filenameTemplate", "onlyNormalStatus"),
		}
		spec.Fields = []ActionProfileFieldSpec{
			buildFilenameTemplateFieldSpec(
				"Template default untuk rename bukpot BPPU.",
				spec.CollectionKind,
			),
			buildOnlyNormalStatusFieldSpec(),
		}
	case ActionProfileBP21RenameBukpot:
		spec.Label = "BP21 Rename Bukpot"
		spec.Description = "Nilai default parameter untuk action Rename Bukpot pada collection BP21."
		spec.CollectionKind = "bukpot_bp21"
		spec.ActionType = "rename_bukpot"
		spec.Sections = []configlayout.SectionSpec{
			bukpotActionSection("rename", "Pengaturan nama file default untuk action Rename Bukpot.", "filenameTemplate", "onlyNormalStatus"),
		}
		spec.Fields = []ActionProfileFieldSpec{
			buildFilenameTemplateFieldSpec(
				"Template default untuk rename bukpot BP21.",
				spec.CollectionKind,
			),
			buildOnlyNormalStatusFieldSpec(),
		}
	case ActionProfileBPA1RenameBukpot:
		spec.Label = "BPA1 Rename Bukpot"
		spec.Description = "Nilai default parameter untuk action Rename Bukpot pada collection BPA1."
		spec.CollectionKind = "bukpot_bpa1"
		spec.ActionType = "rename_bukpot"
		spec.Sections = []configlayout.SectionSpec{
			bukpotActionSection("rename", "Pengaturan nama file default untuk action Rename Bukpot.", "filenameTemplate", "onlyNormalStatus"),
		}
		spec.Fields = []ActionProfileFieldSpec{
			buildFilenameTemplateFieldSpec(
				"Template default untuk rename bukpot BPA1.",
				spec.CollectionKind,
			),
			buildOnlyNormalStatusFieldSpec(),
		}
	case ActionProfileBPPURenameByCategory:
		spec.Label = "BPPU Rename by Category"
		spec.Description = "Nilai default parameter untuk action Rename by Category pada collection BPPU."
		spec.CollectionKind = "bukpot_bppu"
		spec.ActionType = "rename_by_category"
		spec.Sections = []configlayout.SectionSpec{
			bukpotActionSection("filter", "Pengaturan default filter dokumen untuk action Rename by Category.", "onlyNormalStatus"),
		}
		spec.Fields = []ActionProfileFieldSpec{
			buildOnlyNormalStatusFieldSpec(),
		}
	case ActionProfileBP21RenameByCategory:
		spec.Label = "BP21 Rename by Category"
		spec.Description = "Nilai default parameter untuk action Rename by Category pada collection BP21."
		spec.CollectionKind = "bukpot_bp21"
		spec.ActionType = "rename_by_category"
		spec.Sections = []configlayout.SectionSpec{
			bukpotActionSection("filter", "Pengaturan default filter dokumen untuk action Rename by Category.", "onlyNormalStatus"),
		}
		spec.Fields = []ActionProfileFieldSpec{
			buildOnlyNormalStatusFieldSpec(),
		}
	}

	return spec
}

func (s *ActionProfileService) defaultConfig(key ActionProfileKey) ActionProfile {
	spec := s.Spec(key)
	fields := make([]ActionProfileField, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		fields = append(fields, ActionProfileField{
			Key:         item.Key,
			Label:       item.Label,
			Required:    item.Required,
			Value:       item.DefaultValue,
			Description: item.Description,
			Kind:        item.Kind,
		})
	}

	return ActionProfile{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Fields:        fields,
	}
}

func buildFilenameTemplateFieldSpec(
	description string,
	collectionKind string,
) ActionProfileFieldSpec {
	return ActionProfileFieldSpec{
		Key:          "filenameTemplate",
		Label:        "Template Nama File",
		Required:     true,
		DefaultValue: "{{nomorBuktiPotong}} - {{namaPenerima}}",
		Description:  description,
		Kind:         "template",
		Suggestions:  buildBukpotTemplateSuggestions(collectionKind),
		HelpText:     "Gunakan placeholder yang tersedia. Ekstensi .pdf akan ditambahkan otomatis.",
		Placeholder:  "{{nomorBuktiPotong}} - {{namaPenerima}}",
	}
}

func buildOnlyNormalStatusFieldSpec() ActionProfileFieldSpec {
	return ActionProfileFieldSpec{
		Key:          "onlyNormalStatus",
		Label:        "Hanya Include Status Normal",
		Required:     false,
		DefaultValue: "true",
		Description:  "Jika aktif, bukpot DIBATALKAN atau PEMBETULAN tidak diproses.",
		Kind:         "checkbox",
	}
}

func bukpotActionSection(
	key string,
	description string,
	fieldKeys ...string,
) configlayout.SectionSpec {
	return configlayout.SectionSpec{
		Key:         key,
		Title:       specutil.ParameterActionSectionTitle,
		Description: description,
		Columns:     1,
		FieldKeys:   append([]string(nil), fieldKeys...),
	}
}

func buildBukpotTemplateSuggestions(collectionKind string) []ActionProfileSuggestionSpec {
	suggestions := []ActionProfileSuggestionSpec{
		{Token: "nomorBuktiPotong", Label: "Nomor Bukti Potong", Example: "{{nomorBuktiPotong}}"},
		{Token: "namaPenerima", Label: "Nama Penerima", Example: "{{namaPenerima}}"},
		{Token: "sifatPemotongan", Label: "Sifat Pemotongan", Example: "{{sifatPemotongan}}"},
		{Token: "statusBukti", Label: "Status Bukti", Example: "{{statusBukti}}"},
		{Token: "npwpNikPenerima", Label: "NPWP/NIK Penerima", Example: "{{npwpNikPenerima}}"},
		{Token: "namaPemotong", Label: "Nama Pemotong", Example: "{{namaPemotong}}"},
		{Token: "npwpNikPemotong", Label: "NPWP/NIK Pemotong", Example: "{{npwpNikPemotong}}"},
		{Token: "documentTag", Label: "Tag Dokumen", Example: "{{documentTag}}"},
		{Token: "sourceName", Label: "Nama File Asal", Example: "{{sourceName}}"},
	}

	switch collectionKind {
	case "bukpot_bppu", "bukpot_bp21":
		suggestions = append(suggestions,
			ActionProfileSuggestionSpec{Token: "dokumenReferensiNomor", Label: "Nomor Dokumen Referensi", Example: "{{dokumenReferensiNomor}}"},
			ActionProfileSuggestionSpec{Token: "dokumenReferensiJenis", Label: "Jenis Dokumen Referensi", Example: "{{dokumenReferensiJenis}}"},
			ActionProfileSuggestionSpec{Token: "dokumenReferensiTanggal", Label: "Tanggal Dokumen Referensi", Example: "{{dokumenReferensiTanggal}}"},
			ActionProfileSuggestionSpec{Token: "masaPajak", Label: "Masa Pajak", Example: "{{masaPajak}}"},
		)
	case "bukpot_bpa1":
		suggestions = append(suggestions,
			ActionProfileSuggestionSpec{Token: "periodePenghasilan", Label: "Periode Penghasilan", Example: "{{periodePenghasilan}}"},
			ActionProfileSuggestionSpec{Token: "posisi", Label: "Posisi", Example: "{{posisi}}"},
			ActionProfileSuggestionSpec{Token: "statusPtkp", Label: "Status PTKP", Example: "{{statusPtkp}}"},
		)
	}

	return suggestions
}

func (s *ActionProfileService) normalizeConfig(cfg ActionProfile, key ActionProfileKey) ActionProfile {
	spec := s.Spec(key)
	byKey := make(map[string]ActionProfileField, len(cfg.Fields))
	for _, field := range cfg.Fields {
		normalized := strings.TrimSpace(field.Key)
		if normalized == "" {
			continue
		}
		byKey[normalized] = field
	}

	fields := make([]ActionProfileField, 0, len(spec.Fields))
	for _, item := range spec.Fields {
		field := ActionProfileField{
			Key:         item.Key,
			Label:       item.Label,
			Required:    item.Required,
			Value:       item.DefaultValue,
			Description: item.Description,
			Kind:        item.Kind,
		}
		if current, ok := byKey[item.Key]; ok {
			switch item.Kind {
			case "checkbox":
				field.Value = normalizeBoolString(current.Value, item.DefaultValue)
			case "number":
				field.Value = normalizeNumberString(current.Value, item.DefaultValue)
			default:
				value := strings.TrimSpace(current.Value)
				if value != "" {
					field.Value = value
				} else if !item.Required {
					field.Value = ""
				}
			}
		}
		fields = append(fields, field)
	}

	return ActionProfile{
		SchemaVersion: spec.SchemaVersion,
		ConfigKey:     spec.ConfigKey,
		Fields:        fields,
	}
}

func normalizeBoolString(raw string, fallback string) string {
	value := strings.TrimSpace(strings.ToLower(raw))
	switch value {
	case "true", "1", "yes", "on":
		return "true"
	case "false", "0", "no", "off":
		return "false"
	default:
		return normalizeBoolString(fallback, "false")
	}
}

func normalizeNumberString(raw string, fallback string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return strings.TrimSpace(fallback)
	}
	if _, err := strconv.Atoi(value); err != nil {
		return strings.TrimSpace(fallback)
	}
	return value
}
