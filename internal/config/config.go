package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Port     int
	PortMax  int
	Features FeatureFlags
}

type FeatureFlags struct {
	EnableCashflowXLSX bool
}

type AppSettings struct {
	SchemaVersion string       `json:"schemaVersion"`
	Features      FeatureFlags `json:"features"`
}

const appSettingsSchemaVersion = "1"

func Load(rootDir string) Config {
	settings := loadSettings(rootDir)

	return Config{
		Port:    getEnvInt("APP_PORT", 8080),
		PortMax: getEnvInt("APP_PORT_MAX", 8100),
		Features: FeatureFlags{
			EnableCashflowXLSX: getEnvBool("APP_ENABLE_CASHFLOW_XLSX", settings.Features.EnableCashflowXLSX),
		},
	}
}

func SettingsPath(rootDir string) string {
	return filepath.Join(rootDir, "app_settings.json")
}

func loadSettings(rootDir string) AppSettings {
	settings := AppSettings{
		SchemaVersion: appSettingsSchemaVersion,
		Features: FeatureFlags{
			EnableCashflowXLSX: false,
		},
	}

	if strings.TrimSpace(rootDir) == "" {
		return settings
	}

	b, err := os.ReadFile(SettingsPath(rootDir))
	if err != nil {
		return settings
	}

	var payload AppSettings
	if err := json.Unmarshal(b, &payload); err != nil {
		return settings
	}

	if payload.SchemaVersion == "" {
		payload.SchemaVersion = appSettingsSchemaVersion
	}

	return payload
}

// func getEnv(key, def string) string {
// 	if v := os.Getenv(key); v != "" {
// 		return v
// 	}
// 	return def
// }

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getEnvBool(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch v {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}
