package app

import (
	"database/sql"
	"log/slog"

	"github.com/sieryo/invoice-extractor/internal/app/auth"
	repository "github.com/sieryo/invoice-extractor/internal/infra/persistence/sqlite"
)

type App struct {
	AuthService *auth.AuthService
	Logger      *slog.Logger
}

func New(db *sql.DB, logger *slog.Logger) *App {
	// repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// services
	authService := auth.NewService(userRepo, sessionRepo, logger)

	return &App{
		AuthService: authService,
		Logger:      logger,
	}
}
