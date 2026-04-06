package moduleactivation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/profilepath"
)

const schemaVersion = "1"

const (
	ModuleInvoice       = "invoice"
	ModuleTaxInvoice    = "tax_invoice"
	ModuleBukpot        = "bukpot"
	ModuleRequestBukpot = "request_bukpot"
	ModuleCashflow      = "cashflow"
)

type RuntimeFeatures struct {
	EnableCashflowXLSX bool
}

type ModuleSpec struct {
	Key             string   `json:"key"`
	Label           string   `json:"label"`
	Description     string   `json:"description,omitempty"`
	CollectionKinds []string `json:"collectionKinds,omitempty"`
	ConfigGroups    []string `json:"configGroups,omitempty"`
}

type ModulePreference struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Description string   `json:"description,omitempty"`
	Enabled     bool     `json:"enabled"`
	Available   bool     `json:"available"`
	Message     string   `json:"message,omitempty"`
	CollectionKinds []string `json:"collectionKinds,omitempty"`
	ConfigGroups []string `json:"configGroups,omitempty"`
}

type ModuleSettings struct {
	SchemaVersion string          `json:"schemaVersion"`
	Modules       map[string]bool `json:"modules"`
}

type Service struct {
	rootDir  string
	features RuntimeFeatures
}

func NewService(rootDir string, features RuntimeFeatures) *Service {
	return &Service{
		rootDir:  rootDir,
		features: features,
	}
}

func Specs() []ModuleSpec {
	return []ModuleSpec{
		{
			Key:             ModuleInvoice,
			Label:           "Invoice",
			Description:     "Menampilkan collection invoice PDF dan konfigurasi buyer/template yang terkait.",
			CollectionKinds: []string{"invoice_company"},
			ConfigGroups:    []string{"invoice"},
		},
		{
			Key:             ModuleTaxInvoice,
			Label:           "Tax Invoice Coretax",
			Description:     "Menampilkan collection tax invoice Coretax, FP keluaran/masukan Coretax, dan konfigurasi MYOB yang terkait.",
			CollectionKinds: []string{"tax_invoice_coretax", "fp_keluaran_coretax", "fp_keluaran_retur_coretax", "fp_masukan_coretax"},
			ConfigGroups:    []string{"tax_invoice"},
		},
		{
			Key:             ModuleBukpot,
			Label:           "Bukpot",
			Description:     "Menampilkan collection bukpot PDF dan default profile action rename yang terkait.",
			CollectionKinds: []string{"bukpot_bppu", "bukpot_bp21", "bukpot_bpa1"},
			ConfigGroups:    []string{"bukpot"},
		},
		{
			Key:             ModuleRequestBukpot,
			Label:           "Request Bukpot",
			Description:     "Menampilkan collection request bukpot XLSX dan default profile mapping yang terkait.",
			CollectionKinds: []string{"bukpot_request_gst_deduction_mt"},
			ConfigGroups:    []string{"request_bukpot"},
		},
		{
			Key:             ModuleCashflow,
			Label:           "Cashflow",
			Description:     "Menampilkan collection cashflow XLSX, default profile action, dan registry MYOB terkait.",
			CollectionKinds: []string{"cashflow_import"},
			ConfigGroups:    []string{"cashflow"},
		},
	}
}

func ModuleForCollectionKind(collectionKind string) (ModuleSpec, bool) {
	target := strings.TrimSpace(strings.ToLower(collectionKind))
	if target == "" {
		return ModuleSpec{}, false
	}
	for _, item := range Specs() {
		for _, kind := range item.CollectionKinds {
			if strings.EqualFold(strings.TrimSpace(kind), target) {
				return item, true
			}
		}
	}
	return ModuleSpec{}, false
}

func ModuleForConfigGroup(group string) (ModuleSpec, bool) {
	target := strings.TrimSpace(strings.ToLower(group))
	if target == "" {
		return ModuleSpec{}, false
	}
	for _, item := range Specs() {
		for _, configGroup := range item.ConfigGroups {
			if strings.EqualFold(strings.TrimSpace(configGroup), target) {
				return item, true
			}
		}
	}
	return ModuleSpec{}, false
}

func (s *Service) Load(profileID string) (ModuleSettings, error) {
	path := profilepath.ProfileModulesJSON(s.rootDir, profileID)
	payload := ModuleSettings{
		SchemaVersion: schemaVersion,
		Modules:       s.defaultModules(),
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return payload, nil
		}
		return ModuleSettings{}, err
	}

	var stored ModuleSettings
	if err := json.Unmarshal(b, &stored); err != nil {
		return ModuleSettings{}, err
	}

	if stored.SchemaVersion == "" {
		stored.SchemaVersion = schemaVersion
	}

	modules := s.defaultModules()
	for key, enabled := range stored.Modules {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" {
			continue
		}
		if _, ok := modules[normalizedKey]; !ok {
			continue
		}
		modules[normalizedKey] = enabled && s.isAvailable(normalizedKey)
	}
	stored.Modules = modules
	stored.SchemaVersion = schemaVersion
	return stored, nil
}

func (s *Service) Update(profileID string, input ModuleSettings) (ModuleSettings, error) {
	normalized := ModuleSettings{
		SchemaVersion: schemaVersion,
		Modules:       s.defaultModules(),
	}
	for key := range normalized.Modules {
		normalized.Modules[key] = normalizeModuleBool(input.Modules, key) && s.isAvailable(key)
	}

	if err := os.MkdirAll(filepath.Dir(profilepath.ProfileModulesJSON(s.rootDir, profileID)), 0o755); err != nil {
		return ModuleSettings{}, err
	}

	b, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return ModuleSettings{}, err
	}
	if err := os.WriteFile(profilepath.ProfileModulesJSON(s.rootDir, profileID), b, 0o644); err != nil {
		return ModuleSettings{}, err
	}
	return normalized, nil
}

func (s *Service) Preferences(profileID string) ([]ModulePreference, error) {
	settings, err := s.Load(profileID)
	if err != nil {
		return nil, err
	}

	items := make([]ModulePreference, 0, len(Specs()))
	for _, spec := range Specs() {
		enabled := settings.Modules[spec.Key]
		preference := ModulePreference{
			Key:             spec.Key,
			Label:           spec.Label,
			Description:     spec.Description,
			Enabled:         enabled,
			Available:       s.isAvailable(spec.Key),
			CollectionKinds: append([]string(nil), spec.CollectionKinds...),
			ConfigGroups:    append([]string(nil), spec.ConfigGroups...),
		}
		if !preference.Available {
			preference.Message = "Modul ini belum tersedia pada backend saat ini."
		} else if !preference.Enabled {
			preference.Message = "Modul disembunyikan dari create collection dan halaman konfigurasi."
		}
		items = append(items, preference)
	}

	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})
	return items, nil
}

func (s *Service) IsEnabledForCollectionKind(profileID string, collectionKind string) (bool, ModuleSpec, error) {
	module, ok := ModuleForCollectionKind(collectionKind)
	if !ok {
		return true, ModuleSpec{}, nil
	}
	settings, err := s.Load(profileID)
	if err != nil {
		return false, ModuleSpec{}, err
	}
	return settings.Modules[module.Key], module, nil
}

func (s *Service) IsEnabledForConfigGroup(profileID string, group string) (bool, ModuleSpec, error) {
	module, ok := ModuleForConfigGroup(group)
	if !ok {
		return true, ModuleSpec{}, nil
	}
	settings, err := s.Load(profileID)
	if err != nil {
		return false, ModuleSpec{}, err
	}
	return settings.Modules[module.Key], module, nil
}

func (s *Service) defaultModules() map[string]bool {
	return map[string]bool{
		ModuleInvoice:       true,
		ModuleTaxInvoice:    true,
		ModuleBukpot:        true,
		ModuleRequestBukpot: true,
		ModuleCashflow:      s.features.EnableCashflowXLSX,
	}
}

func (s *Service) isAvailable(key string) bool {
	switch strings.TrimSpace(key) {
	case ModuleCashflow:
		return s.features.EnableCashflowXLSX
	default:
		return true
	}
}

func normalizeModuleBool(items map[string]bool, key string) bool {
	if items == nil {
		return false
	}
	value, ok := items[key]
	if !ok {
		return false
	}
	return value
}
