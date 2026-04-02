package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type AppSettingsStatus struct {
	RestartRequired bool         `json:"restartRequired"`
	RuntimeFeatures FeatureFlags `json:"runtimeFeatures"`
	SavedFeatures   FeatureFlags `json:"savedFeatures"`
	Message         string       `json:"message"`
}

type SettingsService struct {
	rootDir         string
	runtimeFeatures FeatureFlags
	mu              sync.RWMutex
}

func NewSettingsService(rootDir string, runtimeFeatures FeatureFlags) *SettingsService {
	return &SettingsService{
		rootDir:         rootDir,
		runtimeFeatures: runtimeFeatures,
	}
}

func (s *SettingsService) Load() (AppSettings, error) {
	settings := loadSettings(s.rootDir)
	return settings, nil
}

func (s *SettingsService) Update(payload AppSettings) (AppSettings, error) {
	normalized := AppSettings{
		SchemaVersion: appSettingsSchemaVersion,
		Features: FeatureFlags{
			EnableCashflowXLSX: payload.Features.EnableCashflowXLSX,
		},
	}

	if err := os.MkdirAll(filepath.Dir(SettingsPath(s.rootDir)), 0o755); err != nil {
		return AppSettings{}, err
	}

	b, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return AppSettings{}, err
	}
	if err := os.WriteFile(SettingsPath(s.rootDir), b, 0o644); err != nil {
		return AppSettings{}, err
	}
	s.mu.Lock()
	s.runtimeFeatures = normalized.Features
	s.mu.Unlock()
	return normalized, nil
}

func (s *SettingsService) Status() AppSettingsStatus {
	saved := loadSettings(s.rootDir)
	runtime := s.CurrentFeatures()
	restartRequired := saved.Features != runtime

	message := "Pengaturan modul aplikasi sudah aktif."
	if restartRequired {
		message = "Perubahan modul tersimpan. Restart backend untuk menerapkan perubahan."
	}

	return AppSettingsStatus{
		RestartRequired: restartRequired,
		RuntimeFeatures: runtime,
		SavedFeatures:   saved.Features,
		Message:         strings.TrimSpace(message),
	}
}

func (s *SettingsService) CurrentFeatures() FeatureFlags {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtimeFeatures
}
