package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/sieryo/invoice-extractor/internal/app"
	"github.com/sieryo/invoice-extractor/internal/database"
	"github.com/sieryo/invoice-extractor/internal/pkg/logger"
	"github.com/sieryo/invoice-extractor/internal/transport/http"
)

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	appDir := filepath.Join(configDir, "invoice-extractor")
	_ = os.MkdirAll(appDir, 0755)

	dbPath := filepath.Join(appDir, "app.db")
	logPath := filepath.Join(appDir, "app.log")

	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	logger := logger.Setup(env, logPath)

	db := database.NewSQLite(dbPath)

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	defer database.CloseSQLite(db)

	if err := database.RunMigrations(db, "migrations"); err != nil {
		logger.Error("migration failed", "error", err)
		log.Fatal(err)
	}

	appContainer := app.New(db, logger, appDir)

	server := http.NewServer(appContainer)
	logger.Info("HTTP server starting", "addr", ":8080")

	log.Fatal(server.Listen(":8080"))
}
