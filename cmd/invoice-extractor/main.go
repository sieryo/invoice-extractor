package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/sieryo/invoice-extractor/internal/app"
	"github.com/sieryo/invoice-extractor/internal/config"
	"github.com/sieryo/invoice-extractor/internal/database"
	"github.com/sieryo/invoice-extractor/internal/netutil"
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

	cfg := config.Load()

	ilogger := logger.Setup(environment, logPath)

	db := database.NewSQLite(dbPath)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	defer database.CloseSQLite(db)

	if err := database.RunMigrations(db, "migrations"); err != nil {
		ilogger.Error("migration failed", "error", err)
		log.Fatal(err)
	}

	appContainer := app.New(db, ilogger, appDir)
	server := http.NewServer(appContainer)

	port, err := netutil.FindAvailablePort(cfg.Port, cfg.PortMax)
	if err != nil {
		ilogger.Error("no available port", "error", err)
		log.Fatal(err)
	}

	addr := fmt.Sprintf(":%d", port)
	url := fmt.Sprintf("http://localhost:%d", port)

	ilogger.Info("HTTP server starting", "addr", addr)
	logger.ServerStarted(url, environment)

	log.Fatal(server.Listen(addr))
}
