package app

import (
	"database/sql"

	"github.com/sieryo/invoice-extractor/internal/repository"
	"github.com/sieryo/invoice-extractor/internal/services"
)

type App struct {
	AuthService *services.AuthService
}

func New(db *sql.DB) *App {
	// repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// services
	authService := services.NewAuth(userRepo, sessionRepo)

	return &App{
		AuthService: authService,
	}
}
