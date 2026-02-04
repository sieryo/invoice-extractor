package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port    int
	PortMax int
}

func Load() Config {
	return Config{
		Port:    getEnvInt("APP_PORT", 8080),
		PortMax: getEnvInt("APP_PORT_MAX", 8100),
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
