package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/sieryo/invoice-extractor/internal/app"
	"github.com/sieryo/invoice-extractor/internal/database"
	"github.com/sieryo/invoice-extractor/internal/pkg/logger"
)

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatalf("Failed to get user config directory: %v", err)
	}

	appDir := filepath.Join(configDir, "invoice-extractor")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		log.Fatalf("Failed to create app directory: %v", err)
	}

	dbPath := filepath.Join(appDir, "app.db")
	logPath := filepath.Join(appDir, "app.log")

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	slogger := logger.Setup(env, logPath)

	slogger.Info("Starting Invoice Extractor", "env", env, "db_path", dbPath, "log_path", logPath)

	db := database.NewSQLite(dbPath)
	defer db.Close()

	if err := database.RunMigration(db, "migrations/001_init.sql"); err != nil {
		slogger.Error("Failed to run migrations", "error", err)
		log.Fatal(err)
	}

	_ = app.New(db, slogger)
}
