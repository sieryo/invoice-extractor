package app

import (
	"database/sql"
	"log/slog"

	"github.com/sieryo/invoice-extractor/internal/repository"
	"github.com/sieryo/invoice-extractor/internal/services"
)

type App struct {
	AuthService *services.AuthService
	Logger      *slog.Logger
}

func New(db *sql.DB, logger *slog.Logger) *App {
	// repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// services
	authService := services.NewAuth(userRepo, sessionRepo, logger)

	return &App{
		AuthService: authService,
		Logger:      logger,
	}
}
