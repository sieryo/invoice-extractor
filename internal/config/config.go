package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port    int
	PortMax int
	Features FeatureFlags
}

type FeatureFlags struct {
	EnableCashflowXLSX bool
}

func Load() Config {
	return Config{
		Port:    getEnvInt("APP_PORT", 8080),
		PortMax: getEnvInt("APP_PORT_MAX", 8100),
		Features: FeatureFlags{
			EnableCashflowXLSX: getEnvBool("APP_ENABLE_CASHFLOW_XLSX", false),
		},
	}
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
