package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// LoadDotEnvIfExists loads .env for local/dev convenience.
// Missing file is ignored, so production remains unaffected.
func LoadDotEnvIfExists() error {
	candidates := []string{
		".env",
		filepath.Join("backend", ".env"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return err
		}

		// Load does not override existing process env by default.
		return godotenv.Load(path)
	}

	return nil
}

